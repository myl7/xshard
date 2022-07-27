package pkg

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/gob"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/myl7/mingchain/third_party/pbft"
	"github.com/niclabs/tcrsa"
)

type LeaderAddrBook struct {
	CoordinatorAddr string
	NodeAddrs       []string
	LeaderAddrs     []string
}

type LeaderConfig struct {
	NodeID      int
	LeaderID    int
	Port        int
	PrivKey     rsa.PrivateKey
	PubKey      rsa.PublicKey
	PTSKeyShare *tcrsa.KeyShare
	PTSKeyMeta  tcrsa.KeyMeta
	Addrs       LeaderAddrBook
}

type Leader struct {
	C               LeaderConfig
	waitingPool     TxPool
	readyPool       TxPool
	blockchain      []Block
	blockchainMutex sync.Mutex
	packBlockChan   chan struct{}
	pbft            pbft.Pbft
	replyPool       map[string][]*tcrsa.SigShare
	replyPoolMutex  sync.Mutex
	repliedPool     map[string]bool
}

func NewLeader(c LeaderConfig, pc pbft.NodeInfo) *Leader {
	return &Leader{
		C:             c,
		waitingPool:   *NewTxPool(),
		readyPool:     *NewTxPool(),
		packBlockChan: make(chan struct{}),
		replyPool:     make(map[string][]*tcrsa.SigShare),
		repliedPool:   make(map[string]bool),
		pbft:          *pbft.NewPBFT(pc),
	}
}

func (l *Leader) Listen() {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", l.C.Port))
	if err != nil {
		panic(err)
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			panic(err)
		}

		go l.Handle(conn)
	}
}

func (l *Leader) Handle(conn net.Conn) {
	defer conn.Close()

	d := gob.NewDecoder(conn)
	var msg Msg

	err := d.Decode(&msg)
	if err != nil {
		if err == io.ErrUnexpectedEOF {
			fmt.Println("Client disconnected")
			return
		}

		panic(err)
	}

	fmt.Printf("Handling %s msg\n", msg.Type)

	switch msg.Type {
	case "request":
		l.HandleRequest(msg)
	case "preprepare":
		l.HandlePrePrepare(msg)
	case "prepare":
		l.HandlePrepare(msg)
	case "commit":
		l.HandleCommit(msg)
	case "reply":
		l.HandleReply(msg)
	case "crossshard":
		l.HandleCrossShard(msg)
	}
}

func (l *Leader) HandleRequest(msg Msg) {
	tx := msg.Data.(Tx)

	if tx.FromShard == l.C.LeaderID {
		if tx.ToShard == l.C.LeaderID || tx.IsSubTx {
			size := l.readyPool.Add(tx)
			if size >= Cfg.BlockSize {
				l.packBlockChan <- struct{}{}
			}
		} else {
			fmt.Println("Received unexpected tx which should be sent to another shard")
		}
	} else if tx.ToShard == l.C.LeaderID {
		l.waitingPool.Add(tx)
	} else {
		fmt.Println("Received unexpected tx which should be sent to another shard")
	}
}

func (l *Leader) HandlePrePrepare(msg Msg) {
	pp := msg.Data.(pbft.PrePrepare)
	l.pbft.HandlePrePrepare(&pp, l.broadcast)
}

func (l *Leader) HandlePrepare(msg Msg) {
	pp := msg.Data.(pbft.Prepare)
	l.pbft.HandlePrepare(&pp, l.broadcast)
}

func (l *Leader) HandleCommit(msg Msg) {
	c := msg.Data.(pbft.Commit)

	reply := func(m pbft.Message) {
		bufB := make([]byte, len(m.Content))
		copy(bufB, m.Content)

		id := m.ID

		var block Block
		GobDec(bufB, &block)

		blockBHash := sha256.Sum256(m.Content)
		blockBPKCS1, err := tcrsa.PrepareDocumentHash(l.C.PTSKeyMeta.PublicKey.Size(), crypto.SHA256, blockBHash[:])
		if err != nil {
			panic(err)
		}

		sigShare, err := l.C.PTSKeyShare.Sign(blockBPKCS1, crypto.SHA256, &l.C.PTSKeyMeta)
		if err != nil {
			panic(err)
		}

		msg := Msg{
			Type: "reply",
			Data: BlockResult{
				ID:    id,
				Block: block,
				Sig:   sigShare,
			},
			PubKey: l.C.PubKey,
		}

		go l.HandleReply(msg)
	}

	l.pbft.HandleCommit(&c, reply)
}

func (l *Leader) PackBlock() {
	for {
		<-l.packBlockChan
		fmt.Println("Packing new block")

		txes := l.readyPool.SelectTxesForBlock()
		block := Block{
			Txes:   txes,
			PubKey: l.C.PubKey,
		}
		block.GenTxHash()

		blockB := GobEnc(block)

		r := pbft.Request{
			Message: pbft.Message{
				ID:      block.TxHash,
				Content: blockB,
			},
			Timestamp: time.Now().UnixNano(),
		}

		go l.pbft.HandleClientRequest(&r, l.broadcast)
	}
}

func (l *Leader) HandleReply(msg Msg) {
	blockRes := msg.Data.(BlockResult)

	l.replyPoolMutex.Lock()
	if l.repliedPool[blockRes.Block.TxHash] {
		l.replyPoolMutex.Unlock()
		return
	}

	// TODO: We do not send duplicated replies so the uniqueness check is skipped
	l.replyPool[blockRes.Block.TxHash] = append(l.replyPool[blockRes.Block.TxHash], blockRes.Sig)

	var sigs []*tcrsa.SigShare
	if len(l.replyPool[blockRes.Block.TxHash]) >= int(l.C.PTSKeyMeta.K) {
		sigs = l.replyPool[blockRes.Block.TxHash]
		delete(l.replyPool, blockRes.Block.TxHash)
		l.repliedPool[blockRes.Block.TxHash] = true
	}
	l.replyPoolMutex.Unlock()

	if sigs == nil {
		return
	}

	l.blockchainMutex.Lock()
	l.blockchain = append(l.blockchain, blockRes.Block)
	l.blockchainMutex.Unlock()

	l.readyPool.RemoveTxesForBlock(blockRes.Block.Txes)

	csShards := make(map[int]bool)
	for _, tx := range blockRes.Block.Txes {
		if tx.IsSubTx {
			csShards[tx.ToShard] = true
		}
	}

	blockB := GobEnc(blockRes.Block)
	blockBHash := sha256.Sum256(blockB)
	blockBPKCS1, err := tcrsa.PrepareDocumentHash(l.C.PTSKeyMeta.PublicKey.Size(), crypto.SHA256, blockBHash[:])
	if err != nil {
		panic(err)
	}

	var sigShares tcrsa.SigShareList
	for _, sig := range sigs {
		sigShares = append(sigShares, sig)
	}

	sig, err := sigShares.Join(blockBPKCS1, &l.C.PTSKeyMeta)
	if err != nil {
		panic(err)
	}

	csb := CrossShardBlock{
		Block:     blockRes.Block,
		Sig:       sig,
		PTSPubKey: *l.C.PTSKeyMeta.PublicKey,
	}

	cMsg := Msg{
		Type:   "crossshard",
		Data:   csb,
		PubKey: l.C.PubKey,
	}

	cMsgB := GobEnc(cMsg)

	for shard := range csShards {
		if shard == l.C.LeaderID {
			continue
		}

		go func(shard int) {
			TcpSend(l.C.Addrs.LeaderAddrs[shard], cMsgB)
		}(shard)
	}
}

func (l *Leader) HandleCrossShard(msg Msg) {
	csb := msg.Data.(CrossShardBlock)

	blockB := GobEnc(csb.Block)
	blockBHash := sha256.Sum256(blockB)

	err := rsa.VerifyPKCS1v15(&csb.PTSPubKey, crypto.SHA256, blockBHash[:], csb.Sig)
	if err != nil {
		panic(err)
	}

	var csTxes []Tx
	for _, tx := range csb.Block.Txes {
		if tx.IsSubTx {
			csTxes = append(csTxes, tx)
		}
	}

	removedTxes := l.waitingPool.RemoveWaitingTxesForBlock(csTxes)
	for _, tx := range removedTxes {
		size := l.readyPool.Add(tx)
		if size >= Cfg.BlockSize {
			l.packBlockChan <- struct{}{}
		}
	}
}

func (l *Leader) broadcast(objType string, obj any) {
	msg := Msg{
		Type:   objType,
		Data:   obj,
		PubKey: l.C.PubKey,
	}

	msgB := GobEnc(msg)

	for i, addr := range l.C.Addrs.NodeAddrs {
		if i == l.C.NodeID {
			continue
		}

		go func(addr string) {
			TcpSend(addr, msgB)
		}(addr)
	}
}
