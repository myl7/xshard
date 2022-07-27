package pkg

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/gob"
	"fmt"
	"net"

	"github.com/myl7/mingchain/third_party/pbft"
	"github.com/niclabs/tcrsa"
)

type ValidatorAddrBook struct {
	CoordinatorAddr string
	NodeAddrs       []string
	LeaderAddr      string
}

type ValidatorConfig struct {
	NodeID      int
	Port        int
	PrivKey     rsa.PrivateKey
	PubKey      rsa.PublicKey
	PTSKeyShare *tcrsa.KeyShare
	PTSKeyMeta  tcrsa.KeyMeta
	Addrs       ValidatorAddrBook
}

type Validator struct {
	C    ValidatorConfig
	pbft pbft.Pbft
}

func NewValidator(c ValidatorConfig, pc pbft.NodeInfo) *Validator {
	return &Validator{
		C:    c,
		pbft: *pbft.NewPBFT(pc),
	}
}

func (v *Validator) Listen() {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", v.C.Port))
	if err != nil {
		panic(err)
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			panic(err)
		}

		go v.Handle(conn)
	}
}

func (v *Validator) Handle(conn net.Conn) {
	defer conn.Close()

	d := gob.NewDecoder(conn)
	var msg Msg

	err := d.Decode(&msg)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Handling %s msg\n", msg.Type)

	switch msg.Type {
	case "preprepare":
		v.HandlePrePrepare(msg)
	case "prepare":
		v.HandlePrepare(msg)
	case "commit":
		v.HandleCommit(msg)
	}
}

func (v *Validator) HandlePrePrepare(msg Msg) {
	pp := msg.Data.(pbft.PrePrepare)
	v.pbft.HandlePrePrepare(&pp, v.broadcast)
}

func (v *Validator) HandlePrepare(msg Msg) {
	pp := msg.Data.(pbft.Prepare)
	v.pbft.HandlePrepare(&pp, v.broadcast)
}

func (v *Validator) HandleCommit(msg Msg) {
	c := msg.Data.(pbft.Commit)

	reply := func(m pbft.Message) {
		id := m.ID
		bufB := make([]byte, len(m.Content))
		copy(bufB, m.Content)

		var block Block
		GobDec(bufB, &block)

		blockBHash := sha256.Sum256(m.Content)
		blockBPKCS1, err := tcrsa.PrepareDocumentHash(v.C.PTSKeyMeta.PublicKey.Size(), crypto.SHA256, blockBHash[:])
		if err != nil {
			panic(err)
		}

		sigShare, err := v.C.PTSKeyShare.Sign(blockBPKCS1, crypto.SHA256, &v.C.PTSKeyMeta)
		if err != nil {
			panic(err)
		}

		msg := Msg{
			Type: "reply",
			Data: BlockResult{
				ID:    id,
				Block: block,
				Sig:   sigShare,
			},
			PubKey: v.C.PubKey,
		}

		msgB := GobEnc(msg)

		TcpSend(v.C.Addrs.LeaderAddr, msgB)
	}

	v.pbft.HandleCommit(&c, reply)
}

func (v *Validator) broadcast(objType string, obj any) {
	msg := Msg{
		Type:   objType,
		Data:   obj,
		PubKey: v.C.PubKey,
	}

	msgB := GobEnc(msg)

	tcpBroadcast(v.C.Addrs.NodeAddrs, msgB, &v.C.NodeID)
}
