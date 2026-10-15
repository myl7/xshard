package mingchain

import (
	"bytes"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	pbft "github.com/myl7/pbft/pkg"
	"github.com/myl7/tcrsa"
)

func gobCompatibilityMessages() []Msg {
	tx := Tx{Hash: []byte{1, 2}, FromShards: []int{1}, ToShards: []int{2}, IsSubTx: true, Extra: []byte{3}}
	block := Block{Hash: []byte{4, 5}, Txs: []Tx{tx}}
	result := BlockResult{Block: block, Sig: &tcrsa.SigShare{Xi: []byte{6}, C: []byte{7}, Z: []byte{8}, Id: 2}}
	publicKey := rsa.PublicKey{N: big.NewInt(15), E: 3}
	config := NodeConfig{
		PublicInfo: PublicInfo{
			NodeAddrs:   [][]string{{"127.0.0.1:8001"}},
			NodePubKeys: [][]*rsa.PublicKey{{&publicKey}},
			TcKeyMetas:  []*tcrsa.KeyMeta{{PublicKey: &publicKey, K: 2, L: 3}},
			CoorAddr:    "127.0.0.1:8000", BlockSize: 10, BlockInterval: 100,
		},
		ShardID: 1, InShardID: 2,
		PrivKey:    &rsa.PrivateKey{PublicKey: publicKey, D: big.NewInt(3), Primes: []*big.Int{big.NewInt(3), big.NewInt(5)}},
		TcKeyShare: &tcrsa.KeyShare{Id: 2, Si: []byte{9}}, NeighborIDs: []int{0, 1},
	}
	request := pbft.WithSig[pbft.Request]{Body: pbft.Request{Op: block, Timestamp: 123, Client: "client"}, Sig: []byte{10}}
	return []Msg{
		{Head: []string{"request"}, Body: []Tx{tx}},
		{Head: []string{"block"}, Body: block},
		{Head: []string{"result"}, Body: result},
		{Head: []string{"crossshard"}, Body: CrossShardBlockResult{Block: block, FromShard: 1, Sig: tcrsa.Signature{11}}},
		{Head: []string{"setup", "config"}, Body: config},
		{Head: []string{"setup", "ready"}, Body: SetupReady{ShardID: 1, InShardID: 2}},
		{Head: []string{"pbft", "request"}, Body: request},
		{Head: []string{"pbft", "preprepare"}, Body: pbft.PrePrepareMsg{
			PP: pbft.WithSig[pbft.PrePrepare]{Body: pbft.PrePrepare{View: 0, Seq: 1, Digest: []byte{12}}, Sig: []byte{13}}, Req: request,
		}},
		{Head: []string{"pbft", "reply"}, Body: pbft.WithSig[pbft.Reply]{
			Body: pbft.Reply{View: 0, Timestamp: 123, Client: "client", Replica: 2, Result: result}, Sig: []byte{14},
		}},
	}
}

func checkGobCompatibilityMessages(t *testing.T, reader io.Reader) {
	t.Helper()
	decoder := gob.NewDecoder(reader)
	for i, want := range gobCompatibilityMessages() {
		var got Msg
		if err := decoder.Decode(&got); err != nil {
			t.Fatalf("message %d: %v", i, err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("message %d (%v) changed after decoding: got %#v, want %#v", i, want.Head, got, want)
		}
	}
	var extra Msg
	if err := decoder.Decode(&extra); err != io.EOF {
		t.Fatalf("after final message: got %v, want EOF", err)
	}
}

func TestDecodeLegacyGobMessages(t *testing.T) {
	data, err := os.ReadFile("testdata/gobcompat/legacy.gob")
	if err != nil {
		t.Fatal(err)
	}
	checkGobCompatibilityMessages(t, bytes.NewReader(data))
}

func TestLegacyDecoderAcceptsCurrentGobMessages(t *testing.T) {
	var encoded bytes.Buffer
	encoder := gob.NewEncoder(&encoded)
	for _, msg := range gobCompatibilityMessages() {
		if err := encoder.Encode(msg); err != nil {
			t.Fatal(err)
		}
	}
	// The old schema runs in another process so it cannot use this package's
	// registrations. Compare decoded values, not process-global gob type IDs.
	command := exec.Command(filepath.Join(runtime.GOROOT(), "bin", "go"), "run", "./testdata/gobcompat")
	command.Stdin = &encoded
	var stderr bytes.Buffer
	command.Stderr = &stderr
	output, err := command.Output()
	if err != nil {
		t.Fatalf("legacy decoder: %v\n%s", err, &stderr)
	}
	checkGobCompatibilityMessages(t, bytes.NewReader(output))
}

func TestGobSigningPreimagesRemainCompatible(t *testing.T) {
	if os.Getenv("XSHARD_GOB_SIGNING_CHECK") != "1" {
		// Production initializes gob in a fixed order. Start a fresh process so
		// other tests cannot allocate additional type IDs before this check.
		command := exec.Command(os.Args[0], "-test.run=^TestGobSigningPreimagesRemainCompatible$")
		command.Env = append(os.Environ(), "XSHARD_GOB_SIGNING_CHECK=1")
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("isolated signing preimages: %v\n%s", err, output)
		}
		return
	}
	data, err := os.ReadFile("testdata/gobcompat/legacy-signing.json")
	if err != nil {
		t.Fatal(err)
	}
	var want map[string]string
	if err := json.Unmarshal(data, &want); err != nil {
		t.Fatal(err)
	}
	messages := gobCompatibilityMessages()
	values := map[string]any{
		"request": messages[6].Body.(pbft.WithSig[pbft.Request]).Body,
		"reply":   messages[8].Body.(pbft.WithSig[pbft.Reply]).Body,
	}
	for _, name := range []string{"request", "reply"} {
		hash := sha256.New()
		if err := gob.NewEncoder(hash).Encode(values[name]); err != nil {
			t.Fatal(err)
		}
		if got := fmt.Sprintf("%x", hash.Sum(nil)); got != want[name] {
			t.Errorf("%s signing preimage digest = %s, want legacy %s", name, got, want[name])
		}
	}
}
