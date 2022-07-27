package main

import (
	"crypto/rand"
	"crypto/rsa"
	"flag"
	"os"
	"strings"

	"github.com/myl7/mingchain/pkg"
	"github.com/niclabs/tcrsa"
)

func main() {
	shardID := flag.Int("shard-id", -1, "Shard ID")
	nodeNum := flag.Int("node-num", 0, "Total node num")
	sigThreshold := flag.Int("sig-threshold", 0, "Threshold signature least num to verify")
	nodeAddrsS := flag.String("node-addrs", "", "Node addrs splitted by comma")
	output := flag.String("output", "", "Output filename")
	flag.Parse()

	if *shardID == -1 {
		panic("shard-id is required")
	}
	if *nodeNum == 0 {
		panic("node-num is required")
	}
	if *sigThreshold == 0 {
		panic("sig-threshold is required")
	}
	if *nodeAddrsS == "" {
		panic("node-addrs is required")
	}
	if *output == "" {
		panic("output is required")
	}

	nodeAddrs := strings.Split(*nodeAddrsS, ",")

	var keys []rsa.PrivateKey
	for i := 0; i < *nodeNum; i++ {
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			panic(err)
		}

		keys = append(keys, *key)
	}

	ptsKeyShares, ptsKeyMeta, err := tcrsa.NewKey(2048, uint16(*sigThreshold), uint16(*nodeNum), nil)
	if err != nil {
		panic(err)
	}

	c := pkg.ShardConfig{
		ShardID:      *shardID,
		NodeAddrs:    nodeAddrs,
		PrivKeys:     keys,
		PTSKeyShares: ptsKeyShares,
		PTSKeyMeta:   *ptsKeyMeta,
	}

	b := pkg.GobEnc(c)

	err = os.WriteFile(*output, b, 0644)
	if err != nil {
		panic(err)
	}
}
