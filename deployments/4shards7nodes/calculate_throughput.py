import json

import matplotlib.pyplot as plt

with open('log_4_7.json') as f:
    txes = json.load(f)
sharded_txes = [[], [], [], []]
for tx in txes:
    if tx['type'] == 'on_chain' and tx['block_hash']:
        sharded_txes[int(tx['shard'])].append(tx)

s0 = sum([int(tx['tx_num']) for tx in sharded_txes[0]]) / \
    ((int(sharded_txes[0][-1]['timestamp']) - int(sharded_txes[0][0]['timestamp'])) / 1000000000)
s1 = sum([int(tx['tx_num']) for tx in sharded_txes[1]]) / \
    ((int(sharded_txes[1][-1]['timestamp']) - int(sharded_txes[1][0]['timestamp'])) / 1000000000)
s2 = sum([int(tx['tx_num']) for tx in sharded_txes[2]]) / \
    ((int(sharded_txes[2][-1]['timestamp']) - int(sharded_txes[2][0]['timestamp'])) / 1000000000)
s3 = sum([int(tx['tx_num']) for tx in sharded_txes[3]]) / \
    ((int(sharded_txes[3][-1]['timestamp']) - int(sharded_txes[3][0]['timestamp'])) / 1000000000)
print([s0, s1, s2, s3])
print(s0 + s1 + s2 + s3)
