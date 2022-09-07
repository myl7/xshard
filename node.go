package mingchain

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/gob"
	"flag"
	"net"
	"strings"
	"sync"
	"time"

	pbft "github.com/myl7/pbft/pkg"
	"github.com/myl7/tcrsa"
	log "github.com/sirupsen/logrus"
)

type PublicInfo struct {
	// [][0] are for leaders
	// Leader [i][0] can only use [i][] & [][0]
	// Validator [i][j] can only use [i][]
	NodeAddrs   [][]string
	NodePubKeys [][]*rsa.PublicKey
	TcKeyMetas  []*tcrsa.KeyMeta
	// Coordinator addr
	CoorAddr      string
	BlockSize     int
	BlockInterval int
}

type NodeConfig struct {
	PublicInfo
	ShardID int
	// 0 are for the leader
	InShardID   int
	PrivKey     *rsa.PrivateKey
	TcKeyShare  *tcrsa.KeyShare
	NeighborIDs []int
}

type TxWithMetrics struct {
	Tx              *Tx
	InPoolTimestamp int64
}

type Node struct {
	NodeConfig
	PBFT           *pbft.Handler
	gossipFwdedMap map[string]bool
	hasStarted     bool
	hasStartedLock sync.Mutex
	// Leader only fields
	packBlockChan chan bool
	// Lock them following the list order
	waitingPool        map[string]*TxWithMetrics
	waitingPoolLock    sync.Mutex
	readyPool          map[string]*TxWithMetrics
	readyPoolLock      sync.Mutex
	processingPool     map[string]*TxWithMetrics
	processingPoolLock sync.Mutex
	replyMap           map[string][]*tcrsa.SigShare
	repliedMap         map[string]bool
	replyLock          sync.Mutex
}

func (nd *Node) Listen() {
	l, err := net.Listen("tcp", getListenAddr(nd.NodeAddrs[nd.ShardID][nd.InShardID]))
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

func (nd *Node) handle(c net.Conn) {
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

func (nd *Node) handleMsg(msg Msg) {
	switch msg.Head[0] {
	case "gossip":
		nd.handleGossip(msg)
	case "pbft":
		nd.handlePBFT(msg)
	default:
		if nd.InShardID == 0 {
			nd.handleMsgAsLeader(msg)
		} else {
			log.WithField("head", msg.Head).Error("invalid msg")
		}
	}
}

func (nd *Node) handleMsgAsLeader(msg Msg) {
	switch msg.Head[0] {
	case "request":
		nd.handleRequest(msg)
	case "reply":
		nd.handleReply(msg)
	case "crossshard":
		nd.handleCrossShard(msg)
	default:
		log.WithField("head", msg.Head).Error("invalid msg")
	}
}

func (nd *Node) handleGossip(msg Msg) {
	id := msg.Head[1]
	if nd.gossipFwdedMap[id] {
		return
	}
	nd.gossipFwdedMap[id] = true

	go func() {
		// Forward
		for _, nid := range nd.NeighborIDs {
			tcpSend(nd.NodeAddrs[nd.ShardID][nid], msg)
		}
	}()

	msg.Head = msg.Head[2:]
	nd.handleMsg(msg)
}

func (nd *Node) handlePBFT(msg Msg) {
	switch msg.Head[1] {
	case "preprepare":
		ppMsg := msg.Body.(pbft.PrePrepareMsg)
		nd.PBFT.HandlePrePrepare(ppMsg)
	case "prepare":
		pMsg := msg.Body.(pbft.WithSig[pbft.Prepare])
		nd.PBFT.HandlePrepare(pMsg)
	case "commit":
		cMsg := msg.Body.(pbft.WithSig[pbft.Commit])
		nd.PBFT.HandleCommit(cMsg)
	default:
		log.WithField("head", msg.Head).Error("invalid msg")
	}
}

func (nd *Node) handleRequest(msg Msg) {
	func() {
		nd.hasStartedLock.Lock()
		defer nd.hasStartedLock.Unlock()

		if !nd.hasStarted {
			nd.hasStarted = true
			nd.packBlockChan <- true
		}
	}()

	txs := msg.Body.([]Tx)
	for _, tx := range txs {
		if tx.FromShard == nd.ShardID {
			if tx.ToShard == nd.ShardID || tx.IsSubTx {
				func() {
					nd.readyPoolLock.Lock()
					defer nd.readyPoolLock.Unlock()

					nd.readyPool[string(tx.Hash)] = &TxWithMetrics{
						Tx:              &tx,
						InPoolTimestamp: time.Now().UnixNano(),
					}
				}()
			} else {
				log.WithField("tx", tx).Error("invalid tx")
			}
		} else if tx.ToShard == nd.ShardID {
			func() {
				nd.waitingPoolLock.Lock()
				defer nd.waitingPoolLock.Unlock()

				nd.waitingPool[string(tx.Hash)] = &TxWithMetrics{
					Tx:              &tx,
					InPoolTimestamp: time.Now().UnixNano(),
				}
			}()
		} else {
			log.WithField("tx", tx).Error("invalid tx")
		}
	}
}

func (nd *Node) handleReply(msg Msg) {
	br := msg.Body.(BlockResult)

	var sigs []*tcrsa.SigShare
	done := func() bool {
		nd.replyLock.Lock()
		defer nd.replyLock.Unlock()

		if nd.repliedMap[string(br.Block.Hash)] {
			return false
		}

		// IGNORED: We do not send duplicated replies so the uniqueness check is skipped
		sigs = nd.replyMap[string(br.Block.Hash)]
		sigs = append(sigs, br.Sig)
		nd.replyMap[string(br.Block.Hash)] = sigs

		if len(sigs) >= int(nd.TcKeyMetas[nd.ShardID].K) {
			nd.repliedMap[string(br.Block.Hash)] = true
			return true
		}
		return false
	}()
	if !done {
		return
	}

	// No need to store the blockchain since we do not use it
	// TODO: Report

	func() {
		nd.processingPoolLock.Lock()
		defer nd.processingPoolLock.Unlock()

		for _, tx := range br.Block.Txs {
			delete(nd.processingPool, string(tx.Hash))
		}
	}()
	nd.packBlockChan <- false

	csShards := make(map[int]bool)
	for _, tx := range br.Block.Txs {
		if tx.IsSubTx {
			csShards[tx.ToShard] = true
		}
	}

	h := sha256.New()
	err := gob.NewEncoder(h).Encode(br.Block)
	if err != nil {
		panic(err)
	}

	blockBPKCS1, err := tcrsa.PrepareDocumentHash(nd.TcKeyMetas[nd.ShardID].PublicKey.Size(), crypto.SHA256, h.Sum(nil))
	if err != nil {
		panic(err)
	}

	var sigShares tcrsa.SigShareList = sigs
	sig, err := sigShares.Join(blockBPKCS1, nd.TcKeyMetas[nd.ShardID])
	if err != nil {
		panic(err)
	}

	csbr := CrossShardBlockResult{
		Block:     br.Block,
		FromShard: nd.ShardID,
		Sig:       sig,
	}
	csbrMsg := Msg{
		Head: []string{"crossshard"},
		Body: csbr,
	}

	for shard := range csShards {
		if shard == nd.ShardID {
			continue
		}

		go func(shard int) {
			tcpSend(nd.NodeAddrs[shard][0], csbrMsg)
		}(shard)
	}
}

func (nd *Node) handleCrossShard(msg Msg) {
	csbr := msg.Body.(CrossShardBlockResult)
	h := sha256.New()
	gob.NewEncoder(h).Encode(csbr.Block)

	err := rsa.VerifyPKCS1v15(nd.TcKeyMetas[csbr.FromShard].PublicKey, crypto.SHA256, h.Sum(nil), csbr.Sig)
	if err != nil {
		panic(err)
	}

	func() {
		nd.waitingPoolLock.Lock()
		defer nd.waitingPoolLock.Unlock()
		nd.readyPoolLock.Lock()
		defer nd.readyPoolLock.Unlock()

		for _, tx := range csbr.Block.Txs {
			if tx.IsSubTx {
				delete(nd.waitingPool, string(tx.Hash))
				nd.readyPool[string(tx.Hash)] = &TxWithMetrics{
					Tx:              &tx,
					InPoolTimestamp: time.Now().UnixNano(),
				}
			}
		}
	}()
}

func (nd *Node) PackBlock() {
	for {
		first := <-nd.packBlockChan

		var txs []Tx
		if first {
			txs = []Tx{}
		} else {
			for {
				var txs []Tx
				func() {
					nd.readyPoolLock.Lock()
					defer nd.readyPoolLock.Unlock()
					nd.processingPoolLock.Lock()
					defer nd.processingPoolLock.Unlock()

					i := 0
					for k, txWithMetrics := range nd.readyPool {
						tx := txWithMetrics.Tx
						txs = append(txs, *tx)
						delete(nd.readyPool, k)
						// TODO: Report
						nd.processingPool[k] = &TxWithMetrics{
							Tx:              tx,
							InPoolTimestamp: time.Now().UnixNano(),
						}
						i += 1
						if i >= nd.BlockSize {
							break
						}
					}
				}()

				if len(txs) > 0 {
					break
				}
				time.Sleep(time.Millisecond * time.Duration(nd.BlockInterval))
			}
		}

		block := Block{Txs: txs}
		block.GenHash()

		rMsg := pbft.Request{
			Op:        block,
			Timestamp: time.Now().UnixNano(),
			Client:    nd.NodeAddrs[nd.ShardID][nd.InShardID],
		}
		h := sha256.New()
		err := gob.NewEncoder(h).Encode(rMsg)
		if err != nil {
			panic(err)
		}

		sig, err := rsa.SignPKCS1v15(rand.Reader, nd.PrivKey, crypto.SHA256, h.Sum(nil))
		if err != nil {
			panic(err)
		}

		go nd.PBFT.HandleRequest(pbft.WithSig[pbft.Request]{Body: rMsg, Sig: sig})
	}
}

func NewNode() *CoorNode {
	coorAddr := flag.String("coorAddr", "", "coordinator listen addr")
	flag.Parse()

	if *coorAddr == "" {
		log.WithField("coorAddr", *coorAddr).Fatal("invalid commandline arg")
	} else if !strings.Contains(*coorAddr, ":") {
		*coorAddr += ":8000"
	}

	// TODO
	return nil
}
