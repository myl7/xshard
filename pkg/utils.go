package pkg

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/gob"
	"encoding/pem"
	"io/ioutil"
	"net"
	"net/http"
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
