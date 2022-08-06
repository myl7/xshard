package pkg

import (
	"crypto/rsa"
	"encoding/json"

	"github.com/niclabs/tcrsa"
)

type Config struct {
	ShardNum        int      `json:"shard_num"`
	NodeNum         int      `json:"node_num"`
	BlockSize       int      `json:"block_size"`
	BlockInterval   int      `json:"block_interval"`
	CoordinatorAddr string   `json:"coordinator_addr"`
	LeaderAddrs     []string `json:"leader_addrs"`
}

var Cfg Config

func LoadConfig(b []byte) {
	err := json.Unmarshal(b, &Cfg)
	if err != nil {
		panic(err)
	}
}

const ShardConfigPort = 8000

type ShardConfig struct {
	ShardID      int
	NodeAddrs    []string
	PrivKeys     []rsa.PrivateKey
	PTSKeyShares tcrsa.KeyShareList
	PTSKeyMeta   tcrsa.KeyMeta
}
