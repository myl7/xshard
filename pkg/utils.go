package pkg

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/gob"
	"encoding/json"
	"encoding/pem"
	"io/ioutil"
	"net"
	"net/http"
	"time"
)

// Ser/De

func GobEnc(obj any) []byte {
	var buf bytes.Buffer
	e := gob.NewEncoder(&buf)
	err := e.Encode(obj)
	if err != nil {
		panic(err)
	}

	return buf.Bytes()
}

func GobDec(b []byte, obj any) {
	buf := bytes.NewBuffer(b)
	d := gob.NewDecoder(buf)
	err := d.Decode(obj)
	if err != nil {
		panic(err)
	}
}

func PbftRsaKeyEnc(key rsa.PrivateKey) []byte {
	derStream := x509.MarshalPKCS1PrivateKey(&key)
	block := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: derStream,
	}
	return pem.EncodeToMemory(block)
}

func PbftRsaPubEnc(pub rsa.PublicKey) []byte {
	derPkix, err := x509.MarshalPKIXPublicKey(&pub)
	if err != nil {
		panic(err)
	}

	block := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: derPkix,
	}
	return pem.EncodeToMemory(block)
}

// Network

func TcpSend(addr string, b []byte) {
	time.Sleep(time.Millisecond * 120)

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		panic(err)
	}

	_, err = conn.Write(b)
	if err != nil {
		panic(err)
	}

	conn.Close()
}

func tcpBroadcast(addrs []string, b []byte, self *int) {
	for i, addr := range addrs {
		if self != nil && i == *self {
			continue
		}

		go func(addr string) {
			TcpSend(addr, b)
		}(addr)
	}
}

func httpGet(urlS string) []byte {
	resp, err := http.Get(urlS)
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	return b
}

func reportCoordinator(msg any) {
	b, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}

	TcpSend(Cfg.CoordinatorAddr, b)
}

func countCrossTxInBlock(block Block) (int, int) {
	num := 0
	crossNum := 0
	for _, tx := range block.Txes {
		num++
		if !tx.IsSubTx && tx.FromShard != tx.ToShard {
			crossNum++
		}
	}
	return num, crossNum
}

func countDelayInBlock(block Block, onChainTimestamp int64) (int64, int64) {
	var normalDelay int64
	var crossDelay int64
	for _, tx := range block.Txes {
		if !tx.IsSubTx && tx.FromShard != tx.ToShard {
			crossDelay += onChainTimestamp - tx.InPoolTimestamp
		} else {
			normalDelay += onChainTimestamp - tx.InPoolTimestamp
		}
	}
	return normalDelay, crossDelay
}
