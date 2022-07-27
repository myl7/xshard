package pbft

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
)

//<REQUEST,o,t,c>
type Request struct {
	Message
	Timestamp int64
	//相当于clientID
	ClientAddr string
}

//<<PRE-PREPARE,v,n,d>,m>
type PrePrepare struct {
	RequestMessage Request
	Digest         string
	SequenceID     int
	Sign           []byte
}

//<PREPARE,v,n,d,i>
type Prepare struct {
	Digest     string
	SequenceID int
	NodeID     string
	Sign       []byte
}

//<COMMIT,v,n,D(m),i>
type Commit struct {
	Digest     string
	SequenceID int
	NodeID     string
	Sign       []byte
}

//<REPLY,v,t,c,i,r>
type Reply struct {
	MessageID int
	NodeID    string
	Result    bool
}

type Message struct {
	Content []byte
	ID      string
}

//对消息详情进行摘要
func getDigest(request Request) string {
	b, err := json.Marshal(request)
	if err != nil {
		log.Panic(err)
	}
	hash := sha256.Sum256(b)
	//进行十六进制字符串编码
	return hex.EncodeToString(hash[:])
}
