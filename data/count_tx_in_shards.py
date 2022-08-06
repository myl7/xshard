import csv
import sys

sharded_txes = [0 for _ in range(20)]
with open(sys.argv[1]) as f:
    c = csv.DictReader(f)
    for tx in c:
        shard = int(tx['toShard'])
        sharded_txes[shard] += 1
print(sharded_txes)
