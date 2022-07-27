package main

import (
	"crypto/rand"
	"crypto/rsa"
	"os"

	"github.com/myl7/mingchain/pkg"
	"github.com/niclabs/tcrsa"
)

func main() {
	var keys []rsa.PrivateKey
	for i := 0; i < 4; i++ {
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			panic(err)
		}

		keys = append(keys, *key)
	}

	ptsKeyShares, ptsKeyMeta, err := tcrsa.NewKey(2048, 3, 4, nil)
	if err != nil {
		panic(err)
	}

	c := pkg.ShardConfig{
		ShardID:      0,
		NodeAddrs:    []string{"172.31.27.117:8000", "172.31.27.117:8001", "172.31.27.117:8002", "172.31.27.117:8003"},
		PrivKeys:     keys,
		PTSKeyShares: ptsKeyShares,
		PTSKeyMeta:   *ptsKeyMeta,
	}

	b := pkg.GobEnc(c)

	err = os.WriteFile("scfg_dev_shard1.bin", b, 0644)
	if err != nil {
		panic(err)
	}
}
