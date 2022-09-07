package mingchain

import (
	"encoding/gob"

	pbft "github.com/myl7/pbft/pkg"
	"github.com/myl7/tcrsa"
	log "github.com/sirupsen/logrus"
	merkletree "github.com/wealdtech/go-merkletree"
)

type Msg struct {
	Head []string
	Body any
}

type Tx struct {
	Hash      []byte
	FromShard int
	ToShard   int
	IsSubTx   bool
	Extra     []byte
}

type Block struct {
	Hash []byte
	Txs  []Tx
}

func (b *Block) GenHash() {
	if len(b.Txs) == 0 {
		return
	}

	hashList := make([][]byte, len(b.Txs))
	for i, tx := range b.Txs {
		hashList[i] = tx.Hash
	}

	tree, err := merkletree.New(hashList)
	if err != nil {
		panic(err)
	}

	b.Hash = tree.Root()
}

type BlockResult struct {
	Block
	Sig *tcrsa.SigShare
}

type CrossShardBlockResult struct {
	Block
	FromShard int
	Sig       tcrsa.Signature
}

type SetupReady struct {
	ShardID   int
	InShardID int
}

func init() {
	gob.Register(log.Fields{})
	gob.Register(pbft.PrePrepareMsg{})
	gob.Register(pbft.WithSig[pbft.Prepare]{})
	gob.Register(pbft.WithSig[pbft.Commit]{})
	gob.Register(Block{})
	gob.Register(BlockResult{})
	gob.Register(CrossShardBlockResult{})
	gob.Register(NodeConfig{})
	gob.Register(SetupReady{})
}
