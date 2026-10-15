package mingchain

import (
	"encoding/gob"
	"net"
	"sync"
	"testing"
	"time"
)

// Keep the original TCP forwarding and delay. Cap dispatches in the harness so
// broken duplicate handling cannot make this cyclic test run without a bound.
func TestGossipRingStopsAfterFirstDelivery(t *testing.T) {
	const n = 4
	listeners := make([]net.Listener, n)
	addrs := make([]string, n)
	for i := range listeners {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		listeners[i] = l
		addrs[i] = l.Addr().String()
	}
	nodes := make([]*Node, n)
	for i := range nodes {
		nodes[i] = &Node{
			NodeConfig: NodeConfig{PublicInfo: PublicInfo{NodeAddrs: [][]string{addrs}}, NeighborIDs: []int{(i + 1) % n}},
			hasStarted: true, gossipFwdedMap: make(map[string]bool), readyPool: make(map[string]*TxWithMetrics),
		}
	}
	var lock sync.Mutex
	counts := make([]int, n)
	last := time.Now()
	failures := make(chan error, n)
	var servers sync.WaitGroup
	for i := range nodes {
		servers.Add(1)
		go func(i int) {
			defer servers.Done()
			for {
				conn, err := listeners[i].Accept()
				if err != nil {
					return
				}
				func() {
					defer conn.Close()
					conn.SetReadDeadline(time.Now().Add(2 * time.Second))
					var msg Msg
					if err := gob.NewDecoder(conn).Decode(&msg); err != nil {
						failures <- err
						return
					}
					lock.Lock()
					counts[i]++
					count := counts[i]
					last = time.Now()
					lock.Unlock()
					if count <= 2 {
						// Model distinct upstream hosts while the test sockets use loopback.
						remote := &gossipTestConn{Conn: conn, addr: &net.TCPAddr{IP: net.IPv4(192, 0, 2, byte((i+n-1)%n+1)), Port: 8000}}
						nodes[i].handleMsg(msg, remote)
					}
				}()
			}
		}(i)
	}
	defer func() {
		for _, l := range listeners {
			l.Close()
		}
		servers.Wait()
	}()
	tx := Tx{Hash: []byte("ring"), FromShards: []int{0}, ToShards: []int{0}}
	msg := Msg{Head: []string{"gossip", "ring-id", "request"}, Body: []Tx{tx}}
	nodes[0].handleMsg(msg, &gossipTestConn{addr: &net.TCPAddr{IP: net.IPv4(192, 0, 2, n), Port: 8000}})
	nodes[0].readyPoolLock.Lock()
	first := nodes[0].readyPool["ring"]
	nodes[0].readyPoolLock.Unlock()
	deadline := time.Now().Add(4 * time.Second)
	for {
		select {
		case err := <-failures:
			t.Fatal(err)
		default:
		}
		lock.Lock()
		total := 0
		for _, count := range counts {
			total += count
		}
		quiet := time.Since(last)
		lock.Unlock()
		if total >= n && quiet >= 3*tcpSendManualDelay {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("gossip did not settle within the message/time budget")
		}
		time.Sleep(5 * time.Millisecond)
	}
	lock.Lock()
	got := append([]int(nil), counts...)
	lock.Unlock()
	for i, count := range got {
		if count != 1 {
			t.Fatalf("node %d received the same ID %d times; ring counts=%v", i, count, got)
		}
	}
	nodes[0].readyPoolLock.Lock()
	same := nodes[0].readyPool["ring"] == first
	nodes[0].readyPoolLock.Unlock()
	if !same {
		t.Fatal("looped message was locally processed again")
	}
	for i, nd := range nodes {
		nd.readyPoolLock.Lock()
		delivered := nd.readyPool["ring"] != nil
		nd.readyPoolLock.Unlock()
		if !delivered {
			t.Fatalf("node %d missed the transaction", i)
		}
	}
}
