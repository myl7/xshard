package mingchain

import (
	"encoding/gob"
	"net"
	"strings"
	"time"
)

func getListenAddr(addr string) string {
	return strings.Split(addr, ":")[1]
}

const tcpSendManualDelay = time.Millisecond * 100

func tcpSend(addr string, msg Msg) {
	time.Sleep(tcpSendManualDelay)

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	err = gob.NewEncoder(conn).Encode(msg)
	if err != nil {
		panic(err)
	}
}
