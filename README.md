# X-shard

## Get Started

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
