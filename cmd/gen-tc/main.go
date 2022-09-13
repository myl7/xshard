package main

import (
	"encoding/gob"
	"flag"
	"fmt"
	"os"

	"github.com/myl7/tcrsa"
	log "github.com/sirupsen/logrus"
)

func main() {
	shardNum := flag.Int("shardN", 0, "shard num")
	inShardNum := flag.Int("inShardN", 0, "in-shard node num. must be 3f + 1.")
	tcK := flag.Int("tcK", 0, "threshold of tcrsa. at least half of in-shard node num.")
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

	var tcKeys []TcKeys

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
		f, err := os.Create(fmt.Sprintf("tc%d-%dx%d", *inShardNum, *tcK, *shardNum))
		if err != nil {
			panic(err)
		}
		defer f.Close()

		err = gob.NewEncoder(f).Encode(tcKeys)
		if err != nil {
			panic(err)
		}
	}()
}

type TcKeys struct {
	K      int
	L      int
	Meta   *tcrsa.KeyMeta
	Shares []*tcrsa.KeyShare
}
