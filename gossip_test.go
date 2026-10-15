package mingchain

import (
	"encoding/gob"
	"net"
	"reflect"
	"sync"
	"testing"
	"time"

	pbft "github.com/myl7/pbft/pkg"
)

// Block the forwarding goroutine until local dispatch has stripped its header.
// The forwarded envelope must still retain its own original gossip header.
func TestGossipForwardKeepsEnvelope(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	received := make(chan Msg, 1)
	failures := make(chan error, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			failures <- err
			return
		}
		defer conn.Close()
		conn.SetReadDeadline(time.Now().Add(3 * time.Second))
		var msg Msg
		if err := gob.NewDecoder(conn).Decode(&msg); err != nil {
			failures <- err
			return
		}
		received <- msg
	}()
	ready := make(chan struct{})
	remote := &gossipTestConn{ready: ready, addr: &net.TCPAddr{IP: net.ParseIP("192.0.2.1"), Port: 8000}}
	nd := &Node{
		NodeConfig:     NodeConfig{PublicInfo: PublicInfo{NodeAddrs: [][]string{{listener.Addr().String()}}}, NeighborIDs: []int{0}},
		gossipFwdedMap: make(map[string]bool),
	}
	sent := Msg{Head: []string{"gossip", "forward-test", "pbft", "preprepare"}, Body: pbft.PrePrepareMsg{}}
	nd.handleGossip(sent, remote)
	close(ready)
	select {
	case got := <-received:
		if !reflect.DeepEqual(got.Head, sent.Head) {
			t.Fatalf("forwarded header=%q, want %q", got.Head, sent.Head)
		}
		if !reflect.DeepEqual(got.Body, sent.Body) {
			t.Fatalf("forwarded body=%#v, want %#v", got.Body, sent.Body)
		}
	case err := <-failures:
		t.Fatal(err)
	case <-time.After(5 * time.Second):
		t.Fatal("forwarding timed out")
	}
}

type gossipTestConn struct {
	net.Conn
	ready <-chan struct{}
	addr  net.Addr
	done  *sync.WaitGroup
}

func (c *gossipTestConn) RemoteAddr() net.Addr {
	if c.ready != nil {
		<-c.ready
	}
	if c.done != nil {
		c.done.Done()
	}
	return c.addr
}

// Isolate handler/goroutine costs using the existing same-host filter. This
// avoids the fixed TCP delay and network noise; it is not a paper benchmark.
func BenchmarkGossipFilteredNeighbor(b *testing.B) {
	var forwarded sync.WaitGroup
	remote := &gossipTestConn{addr: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 8000}, done: &forwarded}
	nd := &Node{
		NodeConfig:     NodeConfig{PublicInfo: PublicInfo{NodeAddrs: [][]string{{"127.0.0.1:8001"}}}, NeighborIDs: []int{0}},
		gossipFwdedMap: make(map[string]bool),
	}
	sent := Msg{Head: []string{"gossip", "benchmark", "pbft", "preprepare"}, Body: pbft.PrePrepareMsg{}}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Each iteration follows the first-receipt path without growing the map.
		delete(nd.gossipFwdedMap, "benchmark")
		forwarded.Add(1)
		nd.handleGossip(sent, remote)
		forwarded.Wait()
	}
}
