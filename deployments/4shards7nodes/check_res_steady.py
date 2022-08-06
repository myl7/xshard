import json

import matplotlib.pyplot as plt

TIMESTAMP_BASE = 1659014675204

with open('log_4_7.json') as f:
    txes = json.load(f)
sharded_txes = [[], [], [], []]
for tx in txes:
    if tx['type'] == 'on_chain':
        sharded_txes[int(tx['shard'])].append(tx)
plt.plot(
    range(len(sharded_txes[0])), [int(tx['timestamp']) / 1000000 - TIMESTAMP_BASE for tx in sharded_txes[0]], color='r', label="shard0")
plt.plot(
    range(len(sharded_txes[1])), [int(tx['timestamp']) / 1000000 - TIMESTAMP_BASE for tx in sharded_txes[1]], color='g', label="shard1")
plt.plot(
    range(len(sharded_txes[2])), [int(tx['timestamp']) / 1000000 - TIMESTAMP_BASE for tx in sharded_txes[2]], color='b', label="shard2")
plt.plot(
    range(len(sharded_txes[3])), [int(tx['timestamp']) / 1000000 - TIMESTAMP_BASE for tx in sharded_txes[3]], color='y', label="shard3")
plt.show()
