# X-shard

## Get Started

The Go module is `github.com/myl7/xshard`. Clone the repository and build its
packages and commands with Go and a C compiler (required by the SQLite dependency):

```sh
git clone https://github.com/myl7/xshard.git
cd xshard
go build ./...
```

To install the released commands into your Go binary directory:

```sh
go install github.com/myl7/xshard/cmd/...@v0.3.1
```

Library users should import `github.com/myl7/xshard`; from v0.3.1 onward, the
package name is `xshard`. Version v0.3.0 uses `mingchain`. This replaces the old `github.com/myl7/mingchain` import path.
Callers using the former default package identifier `mingchain` should use
`xshard` or explicitly alias the import as `mingchain`. Existing gob message
type names are retained for compatibility with earlier nodes.

This is a typical golang project following the [standard layout].
Folders in `cmd` can be compiled to executables.
`block-size`, `comb-tc`, `gen-tc` are helper tools and `node`, `coor` are serving programs of a X-shard deployment.

[standard layout]: https://github.com/golang-standards/project-layout

Available options for the coordinator are:

```go
shardNum := flag.Int("shardN", 0, "shard num")
inShardNum := flag.Int("inShardN", 0, "in-shard node num. must be 3f + 1.")
tcK := flag.Int("tcK", 0, "threshold of tcrsa. at least half of in-shard node num.")
blockSize := flag.Int("blockSize", 0, "block size. unit: tx num.")
blockInterval := flag.Int("blockInterval", 0, "block interval. unit: ms.")
coorAddr := flag.String("coorAddr", "", "coordinator listen addr")
resultFile := flag.String("resultFile", "", "result log file")
txRate := flag.Int("txRate", 0, "tx rate. unit: tx num/s.")
debug := flag.Bool("debug", false, "debug logging")
```

Available options for the node are:

```go
coorAddr := flag.String("coorAddr", "", "coordinator listen addr")
debug := flag.Bool("debug", false, "debug logging")
```

All address listening is on gRPC.
You may notice that the node requires much less options, because it will consult the coordinator for most required ones.

Except for the options, two files are required by the coordinator: `rsa` and `tc`.
`rsa` will be generated on the fly if not exists.
But since the generation of `tc` requires so much time, you need to use the helper tool `gen-tc` to generate it.

Available options for `gen-tc` are:

```go
shardNum := flag.Int("shardN", 0, "shard num")
inShardNum := flag.Int("inShardN", 0, "in-shard node num. must be 3f + 1.")
tcK := flag.Int("tcK", 0, "threshold of tcrsa. at least half of in-shard node num.")
```

After setting up the coordinator, now you can start the nodes to get the configuration from the coordinator and wait for incoming transactions.
We do the evaluation on AWS with hundreds of nodes.
To deploy the compiled `node` program and automatically start it, we firstly serve the compiled executable on the AWS S3.
Then we use cloud-init to automatically start all `node` programs.
The cloud-init config is at `user-data.sh` and should be put in the "User Data" box of the AWS EC2 instance creation page.

## Go maintenance checks

CI builds, vets, and runs the existing tests on Go 1.27.2 and Go 1.26.9.
Go 1.27.2 is recommended for reproduction; the Go 1.25.0 directive is preserved:

```sh
go build ./...
go vet ./...
go test -race -timeout=10m ./...
go test -tags=integration -race -run '^TestGossipPBFTCluster$' -timeout=5m .
```

The integration command exercises the existing four-replica gossip/PBFT fixture.
It covers intra-shard processing and bypasses the coordinator and block packing.
These checks do not measure paper benchmark performance. Reproducing published
measurements requires the original toolchain and experimental environment.

The build requires CGO and a C compiler for the existing SQLite dependency.
