package main

import (
	"flag"
	"log"
	"os"
	"strconv"

	"github.com/myl7/mingchain/pkg"
	"github.com/myl7/mingchain/third_party/pbft"
)

func main() {
	shardID := flag.Int("shard-id", -1, "Shard ID")
	gCfgPath := flag.String("gcfg-path", "", "Global config path")
	sCfgPath := flag.String("scfg-path", "", "Shard-specified config path")
	flag.Parse()

	if *shardID == -1 {
		panic("shard-id is required")
	}
	if *gCfgPath == "" {
		panic("gcfg-path is required")
	}
	if *sCfgPath == "" {
		panic("scfg-path is required")
	}

	gCfgB, err := os.ReadFile(*gCfgPath)
	if err != nil {
		panic(err)
	}

	pkg.LoadConfig(gCfgB)

	sCfgB, err := os.ReadFile(*sCfgPath)
	if err != nil {
		panic(err)
	}

	var sCfg pkg.ShardConfig
	pkg.GobDec(sCfgB, &sCfg)

	leaderCfg := pkg.LeaderConfig{
		NodeID:      0,
		LeaderID:    *shardID,
		Port:        pkg.ShardConfigPort,
		PrivKey:     sCfg.PrivKeys[0],
		PubKey:      sCfg.PrivKeys[0].PublicKey,
		PTSKeyShare: sCfg.PTSKeyShares[0],
		PTSKeyMeta:  sCfg.PTSKeyMeta,
		Addrs: pkg.LeaderAddrBook{
			CoordinatorAddr: pkg.Cfg.CoordinatorAddr,
			NodeAddrs:       sCfg.NodeAddrs,
			LeaderAddrs:     pkg.Cfg.LeaderAddrs,
		},
	}

	var nodePubKeys [][]byte
	for _, k := range sCfg.PrivKeys {
		nodePubKeys = append(nodePubKeys, pkg.PbftRsaPubEnc(k.PublicKey))
	}

	pbftCfg := pbft.NodeInfo{
		NodeID:      strconv.Itoa(0),
		RsaPrivKey:  pkg.PbftRsaKeyEnc(sCfg.PrivKeys[0]),
		RsaPubKey:   pkg.PbftRsaPubEnc(sCfg.PrivKeys[0].PublicKey),
		NodePubKeys: nodePubKeys,
		NodeNum:     pkg.Cfg.NodeNum,
	}

	leader := pkg.NewLeader(leaderCfg, pbftCfg)

	log.Println("Starting")

	go leader.PackBlock()

	leader.Listen()
}
