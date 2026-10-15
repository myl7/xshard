package mingchain

import (
	"encoding/gob"
	"net"
	"testing"
	"time"
)

func TestSetupRegistrationHasListeningReceiver(t *testing.T) {
	// The setup protocol binds port 8000; do not interfere with a running node.
	probe, err := net.Listen("tcp", ":8000")
	if err != nil {
		t.Skipf("setup port unavailable: %v", err)
	}
	probe.Close()
	coor, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer coor.Close()
	if err := coor.(*net.TCPListener).SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	nd := &tmpNode{coorAddr: coor.Addr().String(), configChan: make(chan *NodeConfig, 1)}
	done := make(chan struct{})
	go func() { defer close(done); nd.listen() }()
	sendConfig := func() error {
		c, err := net.DialTimeout("tcp", "127.0.0.1:8000", time.Second)
		if err != nil {
			return err
		}
		defer c.Close()
		if err := c.SetDeadline(time.Now().Add(time.Second)); err != nil {
			return err
		}
		return gob.NewEncoder(c).Encode(Msg{Head: []string{"setup", "config"}, Body: NodeConfig{ShardID: 2, InShardID: 3}})
	}
	t.Cleanup(func() {
		select {
		case <-done:
			return
		default:
		}
		// Release Accept even if an assertion fails before the coordinator reply.
		if err := sendConfig(); err != nil {
			t.Errorf("release setup listener: %v", err)
			return
		}
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Error("setup listener did not stop")
		}
	})
	c, err := coor.Accept()
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if err := c.SetDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	var registration Msg
	if err := gob.NewDecoder(c).Decode(&registration); err != nil {
		t.Fatal(err)
	}
	if len(registration.Head) != 2 || registration.Head[0] != "setup" || registration.Head[1] != "addr" || registration.Body != "" {
		t.Fatalf("registration changed: %+v", registration)
	}
	// Reply immediately: registration must only happen after the listener binds.
	if err := sendConfig(); err != nil {
		t.Fatalf("registered without a usable configuration listener: %v", err)
	}
	select {
	case cfg := <-nd.configChan:
		if cfg.ShardID != 2 || cfg.InShardID != 3 {
			t.Fatalf("configuration changed: %+v", cfg)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("configuration was not delivered")
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("setup listener did not finish")
	}
}
