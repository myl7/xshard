package mingchain

import (
	"encoding/gob"
	"io"

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
	gob.NewEncoder(io.Discard).Encode(log.Fields{})
	gob.Register(pbft.PrePrepareMsg{})
	gob.NewEncoder(io.Discard).Encode(pbft.PrePrepareMsg{})
	gob.Register(pbft.WithSig[pbft.Request]{})
	gob.NewEncoder(io.Discard).Encode(pbft.WithSig[pbft.Request]{})
	gob.Register(pbft.WithSig[pbft.Prepare]{})
	gob.NewEncoder(io.Discard).Encode(pbft.WithSig[pbft.Prepare]{})
	gob.Register(pbft.WithSig[pbft.Commit]{})
	gob.NewEncoder(io.Discard).Encode(pbft.WithSig[pbft.Commit]{})
	gob.Register(pbft.WithSig[pbft.Reply]{})
	gob.NewEncoder(io.Discard).Encode(pbft.WithSig[pbft.Reply]{})
	gob.Register([]Tx{})
	gob.NewEncoder(io.Discard).Encode([]Tx{})
	gob.Register(Block{})
	gob.NewEncoder(io.Discard).Encode(Block{})
	gob.Register(BlockResult{})
	gob.NewEncoder(io.Discard).Encode(BlockResult{})
	gob.Register(CrossShardBlockResult{})
	gob.NewEncoder(io.Discard).Encode(CrossShardBlockResult{})
	gob.Register(NodeConfig{})
	gob.NewEncoder(io.Discard).Encode(NodeConfig{})
	gob.Register(SetupReady{})
	gob.NewEncoder(io.Discard).Encode(SetupReady{})
}
