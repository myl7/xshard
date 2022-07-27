package pkg

import (
	"crypto/rsa"
	"encoding/gob"
	"encoding/hex"

	"github.com/myl7/mingchain/third_party/pbft"
	"github.com/niclabs/tcrsa"
	merkletree "github.com/wealdtech/go-merkletree"
)

type Msg struct {
	Type   string
	Data   interface{}
	PubKey rsa.PublicKey
}

type Tx struct {
	BlockID   uint64
	TxHash    string
	From      string
	FromShard int
	To        string
	ToShard   int
	Value     uint64
	IsSubTx   bool
}

type Block struct {
	Txes   []Tx
	TxHash string
	PubKey rsa.PublicKey
}

func (b *Block) GenTxHash() {
	txHashes := make([][]byte, len(b.Txes))
	for i, tx := range b.Txes {
		txHashes[i] = []byte(tx.TxHash)
	}

	tree, err := merkletree.New(txHashes)
	if err != nil {
		panic(err)
	}

	b.TxHash = hex.EncodeToString(tree.Root())
}

type BlockResult struct {
	ID    string
	Block Block
	Sig   *tcrsa.SigShare
}

type CrossShardBlock struct {
	Block     Block
	Sig       tcrsa.Signature
	PTSPubKey rsa.PublicKey
}

func init() {
	gob.Register(Tx{})
	gob.Register(Block{})
	gob.Register(BlockResult{})
	gob.Register(CrossShardBlock{})
	gob.Register(pbft.PrePrepare{})
	gob.Register(pbft.Prepare{})
	gob.Register(pbft.Commit{})
}
