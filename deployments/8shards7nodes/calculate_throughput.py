import json

N = 8

with open(f'log_{N}_7.json') as f:
    txes = json.load(f)
sharded_txes = [[] for _ in range(N)]
for tx in txes:
    if tx['type'] == 'on_chain' and tx['block_hash']:
        sharded_txes[int(tx['shard'])].append(tx)

s = [sum([int(tx['tx_num']) for tx in sharded_txes[i]]) / ((int(sharded_txes[i][-1]['timestamp']) -
                                                            int(sharded_txes[i][0]['timestamp'])) / 1000000000) for i in range(N)]
print(s)
print(sum(s))
