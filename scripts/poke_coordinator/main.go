package main

import (
	"crypto/rand"
	"crypto/rsa"
	mrand "math/rand"
	"time"

	"github.com/myl7/mingchain/pkg"
)

var leaderAddrs = []string{"34.207.186.187:8000", "54.146.24.134:8000"}

func RandStringRunes(n int) string {
	letterRunes := []rune("0123456789abcdef")
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[mrand.Intn(len(letterRunes))]
	}
	return string(b)
}

func pokeInner() {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}

	hash := RandStringRunes(64)

	msg := pkg.Msg{
		Type: "request",
		Data: pkg.Tx{
			BlockID:   12000000,
			TxHash:    hash,
			From:      "707cd0b3d68a1cedda15854d365a80e00c6632d6",
			FromShard: 0,
			To:        "78908f29f58bc25de8e22cdbdddec9800cd5b2a1",
			ToShard:   0,
			Value:     15000000000000000,
			IsSubTx:   false,
		},
		PubKey: key.PublicKey,
	}

	b := pkg.GobEnc(msg)

	pkg.TcpSend(leaderAddrs[0], b)
}

func pokeCross() {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}

	hash := RandStringRunes(64)

	msg := pkg.Msg{
		Type: "request",
		Data: pkg.Tx{
			BlockID:   12000000,
			TxHash:    hash,
			From:      "707cd0b3d68a1cedda15854d365a80e00c6632d6",
			FromShard: 1,
			To:        "78908f29f58bc25de8e22cdbdddec9800cd5b2a1",
			ToShard:   0,
			Value:     15000000000000000,
			IsSubTx:   false,
		},
		PubKey: key.PublicKey,
	}

	subMsg := pkg.Msg{
		Type: "request",
		Data: pkg.Tx{
			BlockID:   12000000,
			TxHash:    hash,
			From:      "707cd0b3d68a1cedda15854d365a80e00c6632d6",
			FromShard: 1,
			To:        "78908f29f58bc25de8e22cdbdddec9800cd5b2a1",
			ToShard:   0,
			Value:     15000000000000000,
			IsSubTx:   true,
		},
		PubKey: key.PublicKey,
	}

	b := pkg.GobEnc(msg)

	pkg.TcpSend(leaderAddrs[0], b)

	subB := pkg.GobEnc(subMsg)

	pkg.TcpSend(leaderAddrs[1], subB)
}

func main() {
	mrand.Seed(time.Now().UnixNano())

	// pokeInner()
	pokeCross()
}
