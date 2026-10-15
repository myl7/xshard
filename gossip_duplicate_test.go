package mingchain

import "testing"

func TestGossipDuplicateKeepsOriginalDelivery(t *testing.T) {
	nd := &Node{hasStarted: true, gossipFwdedMap: make(map[string]bool), readyPool: make(map[string]*TxWithMetrics)}
	tx := Tx{Hash: []byte("tx"), FromShards: []int{0}, ToShards: []int{0}}
	msg := Msg{Head: []string{"gossip", "duplicate", "request"}, Body: []Tx{tx}}
	nd.handleGossip(msg, nil)
	first := nd.readyPool["tx"]
	if first == nil {
		t.Fatal("first delivery was not processed")
	}
	nd.handleGossip(msg, nil)
	if nd.readyPool["tx"] != first {
		t.Fatal("duplicate gossip processed the transaction again")
	}
}
