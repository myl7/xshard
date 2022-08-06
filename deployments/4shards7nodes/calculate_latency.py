import json

import matplotlib.pyplot as plt

with open('log_4_7.json') as f:
    txes = json.load(f)
sharded_txes = [[], [], [], []]
for tx in txes:
    if tx['block_hash']:
        sharded_txes[int(tx['shard'])].append(tx)

latencies = [{}, {}, {}, {}]

for i in range(4):
    for tx in sharded_txes[i]:
        if tx['type'] == 'send_block':
            latencies[i][tx['block_hash']] = [int(tx['timestamp']), -1]
        else:
            r = latencies[i].get(tx['block_hash'], None)
            if r is None or r[1] != -1:
                continue
            latencies[i][tx['block_hash']] = [r[0], int(tx['timestamp'])]

l0s = [(v[1] - v[0]) / 1000000000 for _, v in latencies[0].items() if v[1] != -1]
l1s = [(v[1] - v[0]) / 1000000000 for _, v in latencies[1].items() if v[1] != -1]
l2s = [(v[1] - v[0]) / 1000000000 for _, v in latencies[2].items() if v[1] != -1]
l3s = [(v[1] - v[0]) / 1000000000 for _, v in latencies[3].items() if v[1] != -1]

l0 = sum(l0s) / len(l0s)
l1 = sum(l1s) / len(l1s)
l2 = sum(l2s) / len(l2s)
l3 = sum(l3s) / len(l3s)

print(l0, l1, l2, l3)
print((l0 + l1 + l2 + l3) / 4)
