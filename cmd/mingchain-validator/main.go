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
	nodeID := flag.Int("node-id", 0, "Node ID")
	gCfgPath := flag.String("gcfg-path", "", "Global config path")
	sCfgPath := flag.String("scfg-path", "", "Shard-specified config path")
	flag.Parse()

	if *shardID == -1 {
		panic("shard-id is required")
	}
	if *nodeID == 0 {
		panic("node-id is required")
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

	validatorCfg := pkg.ValidatorConfig{
		NodeID:      *nodeID,
		Port:        8000,
		PrivKey:     sCfg.PrivKeys[*nodeID],
		PubKey:      sCfg.PrivKeys[*nodeID].PublicKey,
		PTSKeyShare: sCfg.PTSKeyShares[*nodeID],
		PTSKeyMeta:  sCfg.PTSKeyMeta,
		Addrs: pkg.ValidatorAddrBook{
			CoordinatorAddr: pkg.Cfg.CoordinatorAddr,
			NodeAddrs:       sCfg.NodeAddrs,
			LeaderAddr:      sCfg.NodeAddrs[0],
		},
	}

	var nodePubKeys [][]byte
	for _, k := range sCfg.PrivKeys {
		nodePubKeys = append(nodePubKeys, pkg.PbftRsaPubEnc(k.PublicKey))
	}

	pbftCfg := pbft.NodeInfo{
		NodeID:      strconv.Itoa(*nodeID),
		RsaPrivKey:  pkg.PbftRsaKeyEnc(sCfg.PrivKeys[*nodeID]),
		RsaPubKey:   pkg.PbftRsaPubEnc(sCfg.PrivKeys[*nodeID].PublicKey),
		NodePubKeys: nodePubKeys,
		NodeNum:     pkg.Cfg.NodeNum,
	}

	validator := pkg.NewValidator(validatorCfg, pbftCfg)

	log.Println("Starting")

	validator.Listen()
}
