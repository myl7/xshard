package mingchain

import (
	"encoding/csv"
	"encoding/gob"
	"io"
	"math"
	"math/rand"
	"net"
	"strings"
	"time"
)

func getListenAddr(addr string) string {
	return ":" + strings.Split(addr, ":")[1]
}

const tcpSendManualDelay = time.Millisecond * 100

// We should indeed use gRPC, but at that time I have not got familiar with it.
// We do observe connection jamming in the experiment.
// Sometimes it is a problem (too many retries) but 10 max retries is enough at most time.
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

const tcpSendRetryIntervalMax = 10

func tcpSendRetry(addr string, msg Msg, retry int) {
	var err error
	for i := 0; i < retry; i++ {
		time.Sleep(tcpSendManualDelay)

		var conn net.Conn
		conn, err = net.Dial("tcp", addr)
		if err != nil {
			time.Sleep(time.Second * time.Duration(rand.Intn(tcpSendRetryIntervalMax)+1))
			continue
		}
		defer conn.Close()

		err = gob.NewEncoder(conn).Encode(msg)
		if err != nil {
			panic(err)
		}

		return
	}

	panic(err)
}

func readCsvRow(cr *csv.Reader, fields []string) (map[string]string, error) {
	row, err := cr.Read()
	if err != nil {
		if err == io.EOF {
			return nil, err
		}

		panic(err)
	}

	obj := make(map[string]string)
	for i, value := range row {
		obj[fields[i]] = value
	}
	return obj, nil
}

func getNeighbors(n int, i int) []int {
	m := int(math.Log2(float64(n)))
	var neighbors []int
	for j := 1; j <= m; j++ {
		neighbors = append(neighbors, (int(math.Pow(2, float64(j)))-1+i)%n)
	}
	return neighbors
}

func checkHostEq(a string, b string) bool {
	return strings.Split(a, ":")[0] == strings.Split(b, ":")[0]
}
