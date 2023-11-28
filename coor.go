package mingchain

// Coordinator

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/csv"
	"encoding/gob"
	"encoding/hex"
	"flag"
	"io"
	"net"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/myl7/tcrsa"
	log "github.com/sirupsen/logrus"
)

type CoorNode struct {
	PublicInfo
	TxRate        int
	resultLog     *log.Logger
	keys          []*rsa.PrivateKey
	tcKeys        []TcKeys
	nodeReady     []bool
	nodeReadyLock sync.Mutex
}

func (nd *CoorNode) Listen() {
	l, err := net.Listen("tcp", getListenAddr(nd.CoorAddr))
	if err != nil {
		panic(err)
	}

	for {
		c, err := l.Accept()
		if err != nil {
			panic(err)
		}

		go nd.handle(c)
	}
}

func (nd *CoorNode) handle(c net.Conn) {
	defer c.Close()

	d := gob.NewDecoder(c)
	var msg Msg
	err := d.Decode(&msg)
	if err != nil {
		// There will be wired spiders following the route to access the listener
		log.WithField("remoteAddr", c.RemoteAddr()).Warn("invalid client")
		return
	}

	nd.handleMsg(msg)
}

func (nd *CoorNode) handleMsg(msg Msg) {
	switch msg.Head[0] {
	case "report":
		nd.handleReport(msg)
	case "setup":
		nd.handleSetup(msg)
	default:
		log.WithField("head", msg.Head).Error("invalid msg")
	}
}

func (nd *CoorNode) handleReport(msg Msg) {
	nd.resultLog.WithFields(msg.Body.(log.Fields)).Info(strings.Join(msg.Head, " "))
}

func (nd *CoorNode) handleSetup(msg Msg) {
	switch msg.Head[1] {
	case "ready":
		nd.handleSetupReady(msg)
	default:
		log.WithField("head", msg.Head).Error("invalid msg")
	}
}

func (nd *CoorNode) handleSetupReady(msg Msg) {
	nd.nodeReadyLock.Lock()
	defer nd.nodeReadyLock.Unlock()

	ready := msg.Body.(SetupReady)
	nd.nodeReady[ready.ShardID*len(nd.NodeAddrs[0])+ready.InShardID] = true
	done := true
	for _, ok := range nd.nodeReady {
		if !ok {
			done = false
			break
		}
	}
	if done {
		go nd.sendTxs()
	}
}

func (nd *CoorNode) SetupNodes() {
	for i := 0; i < len(nd.NodeAddrs); i++ {
		go func(i int) {
			for j := 0; j < len(nd.NodeAddrs[0]); j++ {
				neighbors := getNeighbors(len(nd.NodeAddrs[0]), j)
				go tcpSendRetry(nd.NodeAddrs[i][j], Msg{
					Head: []string{"setup", "config"},
					Body: NodeConfig{
						PublicInfo:  nd.PublicInfo,
						ShardID:     i,
						InShardID:   j,
						PrivKey:     nd.keys[i*len(nd.NodeAddrs[0])+j],
						TcKeyShare:  nd.tcKeys[i].Shares[j],
						NeighborIDs: neighbors,
					},
				}, 100)
			}
			log.WithField("shardID", i).Info("coor setup config ok")
		}(i)
	}
}

func (nd *CoorNode) sendTxs() {
	txFile, err := os.Open("tx.csv")
	if err != nil {
		panic(err)
	}
	defer txFile.Close()

	cr := csv.NewReader(txFile)

	fields, err := cr.Read()
	if err != nil {
		panic(err)
	}

	var txs []Tx
	for {
		row, err := readCsvRow(cr, fields)
		if err != nil {
			if err == io.EOF {
				break
			}

			panic(err)
		}

		fromShard, err := strconv.Atoi(row["fromShard"])
		if err != nil {
			panic(err)
		}
		toShard, err := strconv.Atoi(row["toShard"])
		if err != nil {
			panic(err)
		}
		txHash := strings.TrimPrefix(row["transactionHash"], "0x")
		hash, err := hex.DecodeString(txHash)
		if err != nil {
			panic(err)
		}

		tx := Tx{
			Hash:       hash,
			FromShards: []int{fromShard},
			ToShards:   []int{toShard},
			IsSubTx:    false,
		}
		txs = append(txs, tx)
	}

	i := 0
	for i < len(txs) {
		// Counted by s
		time.Sleep(1 * time.Second)

		txToSent := make([][]Tx, len(nd.NodeAddrs))

		for j := 0; j < nd.TxRate; j++ {
			tx := txs[i]

			for _, shard := range tx.FromShards {
				txToSent[shard] = append(txToSent[shard], tx)
			}

			subTx := tx
			subTx.IsSubTx = true
			for _, shard := range tx.ToShards {
				if !slices.Contains(tx.FromShards, shard) {
					txToSent[shard] = append(txToSent[shard], subTx)
				}
			}

			i++
			if i >= len(txs) {
				break
			}
		}

		log.Info("coor send txs started")

		go func() {
			for shard, txs := range txToSent {
				if len(txs) == 0 {
					continue
				}

				msg := Msg{
					Head: []string{"request"},
					Body: txs,
				}
				go tcpSend(nd.NodeAddrs[shard][0], msg)
			}
		}()
	}

	log.Info("coor sent txs ok")

	for {
		time.Sleep(60 * time.Second)
	}
}

// Notice that this will access commandlne args
func NewCoorNode() *CoorNode {
	shardNum := flag.Int("shardN", 0, "shard num")
	inShardNum := flag.Int("inShardN", 0, "in-shard node num. must be 3f + 1.")
	tcK := flag.Int("tcK", 0, "threshold of tcrsa. at least half of in-shard node num.")
	blockSize := flag.Int("blockSize", 0, "block size. unit: tx num.")
	blockInterval := flag.Int("blockInterval", 0, "block interval. unit: ms.")
	coorAddr := flag.String("coorAddr", "", "coordinator listen addr")
	resultFile := flag.String("resultFile", "", "result log file")
	txRate := flag.Int("txRate", 0, "tx rate. unit: tx num/s.")
	debug := flag.Bool("debug", false, "debug logging")
	flag.Parse()

	if *shardNum == 0 {
		log.WithField("shardN", *shardNum).Fatal("invalid commandline arg")
	}
	if *inShardNum == 0 {
		log.WithField("inShardN", *inShardNum).Fatal("invalid commandline arg")
	}
	if *tcK == 0 {
		log.WithField("tcK", *tcK).Fatal("invalid commandline arg")
	}
	if *blockSize == 0 {
		log.WithField("blockSize", *blockSize).Fatal("invalid commandline arg")
	}
	if *blockInterval == 0 {
		log.WithField("blockInterval", *blockInterval).Fatal("invalid commandline arg")
	}
	if *coorAddr == "" {
		log.WithField("coorAddr", *coorAddr).Fatal("invalid commandline arg")
	} else if !strings.Contains(*coorAddr, ":") {
		*coorAddr += ":8000"
	}
	if *resultFile == "" {
		*resultFile = "result"
	}
	if *txRate == 0 {
		log.WithField("txRate", *txRate).Fatal("invalid commandline arg")
	}

	if *debug {
		log.SetLevel(log.DebugLevel)
	}

	var keys []*rsa.PrivateKey

	// RSA cache
	func() {
		f, err := os.Open("rsa")
		if err != nil {
			return
		}
		defer f.Close()

		err = gob.NewDecoder(f).Decode(&keys)
		if err != nil {
			panic(err)
		}
	}()

	for i := len(keys); i < (*inShardNum * *shardNum); i++ {
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			panic(err)
		}

		keys = append(keys, key)
	}

	func() {
		f, err := os.Create("rsa")
		if err != nil {
			panic(err)
		}
		defer f.Close()

		err = gob.NewEncoder(f).Encode(keys)
		if err != nil {
			panic(err)
		}
	}()

	var tcKeys []TcKeys

	// tcrsa cache
	func() {
		f, err := os.Open("tc")
		if err != nil {
			return
		}
		defer f.Close()

		err = gob.NewDecoder(f).Decode(&tcKeys)
		if err != nil {
			panic(err)
		}
	}()

	if tcKeys[0].K != *tcK || tcKeys[0].L != *inShardNum || len(tcKeys) < *shardNum {
		log.Fatal("unmatched tc config file")
	}
	if len(tcKeys) > *shardNum {
		tcKeys = tcKeys[:*shardNum]
	}

	resultLog := log.New()
	resultLog.SetFormatter(&log.JSONFormatter{})

	resultLogFile, err := os.Create(*resultFile)
	if err != nil {
		panic(err)
	}

	resultLog.Out = resultLogFile

	log.Info("coor setup local prepare ok")

	publicInfo := PublicInfo{
		CoorAddr:      *coorAddr,
		BlockSize:     *blockSize,
		BlockInterval: *blockInterval,
	}

	addrs := make([]string, *shardNum**inShardNum)
	addrChan := make(chan string, *shardNum**inShardNum)

	tmpNd := &tmpCoorNode{
		PublicInfo: publicInfo,
		AddrChan:   addrChan,
		N:          *shardNum * *inShardNum,
	}
	tmpNd.listen()

	for i := 0; i < *shardNum**inShardNum; i++ {
		addr := <-addrChan
		addrs[i] = addr
	}

	for i := 0; i < *shardNum; i++ {
		publicInfo.TcKeyMetas = append(publicInfo.TcKeyMetas, tcKeys[i].Meta)
		publicInfo.NodeAddrs = append(publicInfo.NodeAddrs, addrs[i**inShardNum:(i+1)**inShardNum])
		shardPubKeys := make([]*rsa.PublicKey, *inShardNum)
		for j := 0; j < *inShardNum; j++ {
			shardPubKeys[j] = &keys[i**inShardNum+j].PublicKey
		}
		publicInfo.NodePubKeys = append(publicInfo.NodePubKeys, shardPubKeys)
	}

	log.Info("coor setup public info ok")

	return &CoorNode{
		PublicInfo: publicInfo,
		TxRate:     *txRate,
		resultLog:  resultLog,
		keys:       keys,
		tcKeys:     tcKeys,
		nodeReady:  make([]bool, *shardNum**inShardNum),
	}
}

type TcKeys struct {
	K      int
	L      int
	Meta   *tcrsa.KeyMeta
	Shares []*tcrsa.KeyShare
}

type tmpCoorNode struct {
	PublicInfo
	AddrChan chan string
	N        int
	i        int
}

func (nd *tmpCoorNode) listen() {
	l, err := net.Listen("tcp", getListenAddr(nd.CoorAddr))
	if err != nil {
		panic(err)
	}
	defer l.Close()

	for {
		done := func() bool {
			c, err := l.Accept()
			if err != nil {
				panic(err)
			}
			defer c.Close()

			d := gob.NewDecoder(c)
			var msg Msg
			err = d.Decode(&msg)
			if err != nil {
				// There will be wired spiders following the route to access the listener
				log.WithField("remoteAddr", c.RemoteAddr()).Warn("invalid client")
				return false
			}

			if msg.Head[0] != "setup" || msg.Head[1] != "addr" {
				log.WithField("head", msg.Head).Warn("invalid setup addr msg")
				return false
			}

			addr := msg.Body.(string)
			if !strings.Contains(addr, ":") {
				host := strings.Split(c.RemoteAddr().String(), ":")[0]
				addr = host + ":8000"
			}
			nd.AddrChan <- addr
			log.WithFields(log.Fields{
				"i":    nd.i,
				"addr": addr,
			}).Debug("coor setup addr recv")
			nd.i++
			return nd.i >= nd.N
		}()
		if done {
			break
		}
	}
}
