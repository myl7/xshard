import csv
import sys

N = int(sys.argv[1])

i = 0
j = 0
with open(f'tx_10w_{N}shards.csv') as f:
    c = csv.DictReader(f)
    for tx in c:
        if tx['to'] == 'None':
            j += 1
            continue
        from_shard = int(((int(tx['from'][:2], 16) & 0b11111000) / 32 * 20))
        to_shard = int(((int(tx['to'][:2], 16) & 0b11111000) / 32 * 20))
        if from_shard != to_shard:
            i += 1
print(i)
print(j)
