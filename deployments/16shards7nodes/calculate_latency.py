import json

import matplotlib.pyplot as plt

N = 16

with open(f'log_{N}_7.json') as f:
    txes = json.load(f)
sharded_txes = [[] for _ in range(N)]
for tx in txes:
    if tx['block_hash']:
        sharded_txes[int(tx['shard'])].append(tx)

latencies = [{} for _ in range(N)]

for i in range(N):
    for tx in sharded_txes[i]:
        if tx['type'] == 'send_block':
            latencies[i][tx['block_hash']] = [int(tx['timestamp']), -1]
        else:
            r = latencies[i].get(tx['block_hash'], None)
            if r is None or r[1] != -1:
                continue
            latencies[i][tx['block_hash']] = [r[0], int(tx['timestamp'])]

ls = [[(v[1] - v[0]) / 1000000000 for _, v in latencies[i].items() if v[1] != -1] for i in range(N)]
l = [sum(l) / len(l) for l in ls]

print(l)
print(sum(l) / len(l))
