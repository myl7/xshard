package main

import (
	"bytes"
	"encoding/gob"
	"encoding/hex"

	"github.com/myl7/mingchain"
)

func main() {
	hash, _ := hex.DecodeString("6509fae57546a5d1cb13a8cd05d93ecc5fc7d0e28d9567082985bfebc3db68a2")
	tx := mingchain.Tx{
		Hash:      hash,
		FromShard: 0,
		ToShard:   1,
		IsSubTx:   false,
	}
	blk := mingchain.Block{}
	for i := 0; i < 100; i++ {
		blk.Txs = append(blk.Txs, mingchain.Tx{
			Hash:      hash,
			FromShard: 1,
			ToShard:   2,
			IsSubTx:   false,
		})
	}
	blk.GenHash()
	b := GobEnc(tx)
	println("tx size:", len(b))
	b = GobEnc(blk)
	println("block size:", len(b))
}

func GobEnc(obj any) []byte {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(obj)
	if err != nil {
		panic(err)
	}
	return buf.Bytes()
}
