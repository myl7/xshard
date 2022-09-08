package mingchain

// Coordinator

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/gob"
	"flag"
	"net"
	"os"
	"strings"

	"github.com/myl7/tcrsa"
	log "github.com/sirupsen/logrus"
)

type CoorNode struct {
	PublicInfo
	resultLog *log.Logger
	keys      []*rsa.PrivateKey
	tcKeys    []TcKeys
	nodeReady []bool
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
	nd.resultLog.WithFields(msg.Body.(log.Fields)).Info("report")
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
		go nd.SendTxs()
	}
}

func (nd *CoorNode) SetupNodes() {
	for i := 0; i < len(nd.NodeAddrs); i++ {
		go func(i int) {
			for j := 0; j < len(nd.NodeAddrs[0]); j++ {
				tcpSend(nd.NodeAddrs[i][j], Msg{
					Head: []string{"setup", "config"},
					Body: NodeConfig{
						PublicInfo:  nd.PublicInfo,
						ShardID:     i,
						InShardID:   j,
						PrivKey:     nd.keys[i*len(nd.NodeAddrs[0])+j],
						TcKeyShare:  nd.tcKeys[i].Shares[j],
						NeighborIDs: []int{}, // TODO
					},
				})
			}
			log.WithField("shardID", i).Info("coor setup config ok")
		}(i)
	}
}

func (nd *CoorNode) SendTxs() {
	// TODO
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

	if len(tcKeys) > 0 && (tcKeys[0].K != *tcK || tcKeys[0].L != *inShardNum) {
		tcKeys = nil
	}

	for i := len(tcKeys); i < *shardNum; i++ {
		shares, meta, err := tcrsa.NewKey(2048, uint16(*tcK), uint16(*inShardNum), nil)
		if err != nil {
			panic(err)
		}

		tcKeys = append(tcKeys, TcKeys{
			K:      *tcK,
			L:      *inShardNum,
			Shares: shares,
			Meta:   meta,
		})
	}

	func() {
		f, err := os.Create("tc")
		if err != nil {
			panic(err)
		}
		defer f.Close()

		err = gob.NewEncoder(f).Encode(tcKeys)
		if err != nil {
			panic(err)
		}
	}()

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
	addrChan := make(chan string)
	go func() {
		for i := 0; i < *shardNum**inShardNum; i++ {
			addr := <-addrChan
			addrs[i] = addr
		}
	}()

	tmpNd := &tmpCoorNode{
		PublicInfo: publicInfo,
		AddrChan:   addrChan,
		N:          *shardNum * *inShardNum,
	}
	tmpNd.listen()

	if addrs[*shardNum**inShardNum-1] == "" {
		log.Fatal("coor setup addr failed")
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
	// REFACTOR: Use chan buf to store
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
			nd.i++
			return nd.i >= nd.N
		}()
		if done {
			break
		}
	}
}
