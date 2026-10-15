//go:build integration

package xshard

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"database/sql"
	"encoding/gob"
	"io"
	"net"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	pbft "github.com/myl7/pbft/pkg"
	"github.com/myl7/tcrsa"
	log "github.com/sirupsen/logrus"
)

// Test-only construction uses the existing Node/PBFT callbacks; production
// constructors and experiment control flow are intentionally left unchanged.
func clusterTestNode(config *NodeConfig, db *sql.DB) *Node {
	replicaPubkeys := make([][]byte, len(config.NodePubKeys[config.ShardID]))
	for i, key := range config.NodePubKeys[config.ShardID] {
		replicaPubkeys[i] = pbft.GobEnc(key)
	}
	pbft.InitDB(db)

	pbft := &pbft.Handler{
		StateMachine: pbft.StateMachine{
			State: nil,
			Transform: func(state any, op any) (nextState any, res any) {
				block := op.(Block)

				h := sha256.New()
				err := gob.NewEncoder(h).Encode(block)
				if err != nil {
					panic(err)
				}

				blockBPKCS1, err := tcrsa.PrepareDocumentHash(config.TcKeyMetas[config.ShardID].PublicKey.Size(), crypto.SHA256, h.Sum(nil))
				if err != nil {
					panic(err)
				}

				sigShare, err := config.TcKeyShare.Sign(blockBPKCS1, crypto.SHA256, config.TcKeyMetas[config.ShardID])
				if err != nil {
					panic(err)
				}

				br := BlockResult{
					Block: block,
					Sig:   sigShare,
				}

				log.Debug("node compute block result")

				return state, br
			},
		},
		NetFuncSet: pbft.NetFuncSet{
			NetSend: func(id int, msg any) {
				log.Fatal("no request forwarding to primary")
			},
			NetReply: func(client string, msg any) {
				log.Debug("node send reply msg")
				tcpSend(client, Msg{
					Head: []string{"reply"},
					Body: msg,
				})
			},
			NetBroadcast: func(id int, msg any) {
				gossipID := uuid.NewString()
				var msgType string
				switch msg.(type) {
				case pbft.PrePrepareMsg:
					msgType = "preprepare"
					log.Debug("node send preprepare msg")
				case pbft.WithSig[pbft.Prepare]:
					msgType = "prepare"
					log.Debug("node send prepare msg")
				case pbft.WithSig[pbft.Commit]:
					msgType = "commit"
					log.Debug("node send commit msg")
				}
				m := Msg{
					Head: []string{"gossip", gossipID, "pbft", msgType},
					Body: msg,
				}
				for _, nid := range config.NeighborIDs {
					tcpSend(config.NodeAddrs[config.ShardID][nid], m)
				}
			},
		},
		GetPubkeyFuncSet: pbft.GetPubkeyFuncSet{
			GetClientPubkey: func(client string) []byte {
				return pbft.GobEnc(config.NodePubKeys[config.ShardID][0])
			},
			ReplicaPubkeys: replicaPubkeys,
		},
		DigestFuncSet: pbft.DigestFuncSet{
			Hash: func(data any) []byte {
				h := sha256.New()
				err := gob.NewEncoder(h).Encode(data)
				if err != nil {
					panic(err)
				}
				return h.Sum(nil)
			},
		},
		PubkeyFuncSet: pbft.PubkeyFuncSet{
			PubkeySign: func(digest []byte, privkey []byte) []byte {
				var key *rsa.PrivateKey
				pbft.GobDec(privkey, &key)
				sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest)
				if err != nil {
					panic(err)
				}
				return sig
			},
			PubkeyVerify: func(sig []byte, digest []byte, pubkey []byte) error {
				var key *rsa.PublicKey
				pbft.GobDec(pubkey, &key)
				return rsa.VerifyPKCS1v15(key, crypto.SHA256, digest, sig)
			},
		},
		F:              len(config.NodeAddrs[0]) / 3,
		ID:             config.InShardID,
		Privkey:        pbft.GobEnc(config.PrivKey),
		DB:             db,
		DBSerdeFuncSet: *pbft.NewDBSerdeFuncSetDefault(),
	}
	pbft.Init()

	return &Node{
		NodeConfig:      *config,
		PBFT:            pbft,
		gossipFwdedMap:  make(map[string]bool),
		packBlockChan:   make(chan bool),
		waitingPool:     make(map[string]*TxWithMetrics),
		waitingPoolDone: make(map[string]bool),
		readyPool:       make(map[string]*TxWithMetrics),
		processingPool:  make(map[string]*TxWithMetrics),
		replyMap:        make(map[string][]*tcrsa.SigShare),
		repliedMap:      make(map[string]bool),
		deadChan:        make(chan bool),
	}
}

// Exercise getNeighbors, real gob/TCP, RSA authentication, PBFT, and threshold
// signing together. It is a small successful-work validation, not a benchmark.
func TestGossipPBFTCluster(t *testing.T) {
	const n = 4
	// The legacy generator returns a modulus one bit shorter than requested.
	// Request 1025 only for this test's 1024-bit key; deployment defaults stay 2048.
	shares, meta, err := tcrsa.NewKey(1025, 3, n, nil)
	if err != nil {
		t.Fatal(err)
	}
	sink, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer sink.Close()
	go func() {
		for {
			c, err := sink.Accept()
			if err != nil {
				return
			}
			go func() { defer c.Close(); io.Copy(io.Discard, c) }()
		}
	}()
	listeners := make([]net.Listener, n)
	info := PublicInfo{CoorAddr: sink.Addr().String(), NodeAddrs: make([][]string, 1), NodePubKeys: make([][]*rsa.PublicKey, 1), TcKeyMetas: []*tcrsa.KeyMeta{meta}}
	info.NodeAddrs[0] = make([]string, n)
	info.NodePubKeys[0] = make([]*rsa.PublicKey, n)
	keys := make([]*rsa.PrivateKey, n)
	for i := 0; i < n; i++ {
		listeners[i], err = net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		keys[i], err = rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			t.Fatal(err)
		}
		info.NodeAddrs[0][i] = listeners[i].Addr().String()
		info.NodePubKeys[0][i] = &keys[i].PublicKey
	}
	nodes := make([]*Node, n)
	dbs := make([]*sql.DB, n)
	var servers sync.WaitGroup
	var lock sync.Mutex
	last := time.Now()
	counts := make([]map[string]int, n)
	replies := map[string]map[int]bool{}
	capped := false
	stopping := false
	failures := make(chan error, 32)
	for i := 0; i < n; i++ {
		dbs[i], err = sql.Open("sqlite3", filepath.Join(t.TempDir(), "node.db")+"?_busy_timeout=5000")
		if err != nil {
			t.Fatal(err)
		}
		dbs[i].SetMaxOpenConns(1)
		nodes[i] = clusterTestNode(&NodeConfig{PublicInfo: info, InShardID: i, PrivKey: keys[i], TcKeyShare: shares[i], NeighborIDs: getNeighbors(n, i)}, dbs[i])
		nodes[i].packBlockChan = make(chan bool, 10)
		counts[i] = map[string]int{}
		servers.Add(1)
		go func(i int) {
			defer servers.Done()
			for {
				c, err := listeners[i].Accept()
				if err != nil {
					return
				}
				func() {
					defer c.Close()
					c.SetReadDeadline(time.Now().Add(3 * time.Second))
					var msg Msg
					if err := gob.NewDecoder(c).Decode(&msg); err != nil {
						failures <- err
						return
					}
					lock.Lock()
					last = time.Now()
					dispatch := !stopping
					if msg.Head[0] == "gossip" {
						counts[i][msg.Head[1]]++
						if counts[i][msg.Head[1]] > 3 {
							capped = true
							dispatch = false
						}
					}
					if msg.Head[0] == "reply" {
						r := msg.Body.(pbft.WithSig[pbft.Reply]).Body
						br := r.Result.(BlockResult)
						id := string(br.Hash)
						if replies[id] == nil {
							replies[id] = map[int]bool{}
						}
						replies[id][r.Replica] = true
					}
					lock.Unlock()
					// Distinct logical upstream hosts avoid the loopback-only same-host filter.
					if dispatch {
						nodes[i].handleMsg(msg, &gossipTestConn{Conn: c, addr: &net.TCPAddr{IP: net.IPv4(192, 0, 2, 200), Port: 8000}})
					}
				}()
			}
		}(i)
	}
	defer func() {
		// Stop new protocol dispatch and drain already scheduled sends before closing
		// resources; production callbacks intentionally retain their original API.
		lock.Lock()
		stopping = true
		last = time.Now()
		lock.Unlock()
		drainDeadline := time.Now().Add(5 * time.Second)
		for {
			lock.Lock()
			quiet := time.Since(last)
			lock.Unlock()
			if quiet >= time.Second || time.Now().After(drainDeadline) {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		for _, l := range listeners {
			l.Close()
		}
		servers.Wait()
		for _, db := range dbs {
			db.Close()
		}
	}()
	for seq := 0; seq < 3; seq++ {
		tx := Tx{Hash: []byte{byte(seq + 1)}, FromShards: []int{0}, ToShards: []int{0}}
		block := Block{Txs: []Tx{tx}}
		block.GenHash()
		nodes[0].processingPoolLock.Lock()
		nodes[0].processingPool[string(tx.Hash)] = &TxWithMetrics{Tx: &tx, InPoolTimestamp: time.Now().UnixNano()}
		nodes[0].processingPoolLock.Unlock()
		r := pbft.Request{Op: block, Client: info.NodeAddrs[0][0], Timestamp: time.Now().UnixNano()}
		h := sha256.New()
		gob.NewEncoder(h).Encode(r)
		sig, err := rsa.SignPKCS1v15(rand.Reader, keys[0], crypto.SHA256, h.Sum(nil))
		if err != nil {
			t.Fatal(err)
		}
		nodes[0].PBFT.HandleRequest(pbft.WithSig[pbft.Request]{Body: r, Sig: sig})
		select {
		case <-nodes[0].packBlockChan:
		case <-time.After(10 * time.Second):
			t.Fatalf("block %d did not finish", seq)
		}
		deadline := time.Now().Add(10 * time.Second)
		for {
			select {
			case err := <-failures:
				t.Fatal(err)
			default:
			}
			lock.Lock()
			all := len(replies[string(block.Hash)]) == n
			quiet := time.Since(last)
			bad := capped
			lock.Unlock()
			if bad {
				t.Fatal("gossip exceeded the bounded harness budget")
			}
			if all && quiet >= 5*tcpSendManualDelay {
				break
			}
			if time.Now().After(deadline) {
				t.Fatalf("block %d did not receive all replica replies or settle", seq)
			}
			time.Sleep(5 * time.Millisecond)
		}
	}
	lock.Lock()
	defer lock.Unlock()
	if capped {
		t.Fatal("message budget was exceeded")
	}
	total := 0
	for _, m := range counts {
		for _, count := range m {
			total += count
		}
	}
	t.Logf("3 blocks completed with all 4 replica replies; %d gossip receives; no message cap triggered", total)
}
