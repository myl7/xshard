// This is an independent decoder for the pre-migration message schema.
// It deliberately does not import the Xshard implementation under test.
package main

import (
	"crypto/rsa"
	"encoding/gob"
	"fmt"
	"io"
	"os"

	pbft "github.com/myl7/pbft/pkg"
	"github.com/myl7/tcrsa"
)

type Msg struct {
	Head []string
	Body any
}

type Tx struct {
	Hash       []byte
	FromShards []int
	ToShards   []int
	IsSubTx    bool
	Extra      []byte
}

type Block struct {
	Hash []byte
	Txs  []Tx
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

type PublicInfo struct {
	NodeAddrs     [][]string
	NodePubKeys   [][]*rsa.PublicKey
	TcKeyMetas    []*tcrsa.KeyMeta
	CoorAddr      string
	BlockSize     int
	BlockInterval int
}

type NodeConfig struct {
	PublicInfo
	ShardID     int
	InShardID   int
	PrivKey     *rsa.PrivateKey
	TcKeyShare  *tcrsa.KeyShare
	NeighborIDs []int
}

type SetupReady struct {
	ShardID   int
	InShardID int
}

func main() {
	// These names are the encoding/gob defaults of the original module and
	// package, recorded before its import path changed.
	gob.RegisterName("[]mingchain.Tx", []Tx{})
	gob.RegisterName("github.com/myl7/mingchain.Block", Block{})
	gob.RegisterName("github.com/myl7/mingchain.BlockResult", BlockResult{})
	gob.RegisterName("github.com/myl7/mingchain.CrossShardBlockResult", CrossShardBlockResult{})
	gob.RegisterName("github.com/myl7/mingchain.NodeConfig", NodeConfig{})
	gob.RegisterName("github.com/myl7/mingchain.SetupReady", SetupReady{})
	gob.Register(pbft.PrePrepareMsg{})
	gob.Register(pbft.WithSig[pbft.Request]{})
	gob.Register(pbft.WithSig[pbft.Reply]{})
	decoder := gob.NewDecoder(os.Stdin)
	encoder := gob.NewEncoder(os.Stdout)
	for {
		var msg Msg
		err := decoder.Decode(&msg)
		if err == io.EOF {
			return
		}
		if err == nil {
			err = encoder.Encode(msg)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}
