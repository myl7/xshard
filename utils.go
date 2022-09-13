package mingchain

import (
	"encoding/csv"
	"encoding/gob"
	"io"
	"math"
	"net"
	"strings"
	"time"
)

func getListenAddr(addr string) string {
	return ":" + strings.Split(addr, ":")[1]
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
	m := int(math.Log2(float64(n))) // Neighbor num
	if m%2 != 0 {
		m--
	}
	neighbors := make([]int, m)
	for j := 0; j < m; j++ {
		neighbors[j] = int(math.Abs(float64(int(math.Pow(2, float64(j))+float64(i))))) % n
	}
	return neighbors
}

func checkHostEq(a string, b string) bool {
	return strings.Split(a, ":")[0] == strings.Split(b, ":")[0]
}
