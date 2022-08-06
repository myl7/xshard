import csv
import sys

N = int(sys.argv[1])

i = 0
with open(f'tx_10w_{N}shards.csv') as f:
    c = csv.DictReader(f)
    for tx in c:
        if tx['fromShard'] != tx['toShard']:
            i += 1
print(i)
