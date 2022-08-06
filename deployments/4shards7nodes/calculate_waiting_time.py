import json
import sys

N = 4

with open(f'{sys.argv[1]}.json') as f:
    txes = json.load(f)
sharded_txes = [[] for _ in range(N)]
for tx in txes:
    if tx['type'] == 'on_chain' and tx['block_hash']:
        sharded_txes[int(tx['shard'])].append(tx)

normal_delay_total_list = [sum([int(tx['normal_delay_total']) / 1000000000
                               for tx in sharded_txes[i]]) for i in range(N)]
cross_delay_total_list = [sum([int(tx['cross_delay_total']) / 1000000000 for tx in sharded_txes[i]]) for i in range(N)]
normal_tx_num_list = [sum([int(tx['tx_num']) - int(tx['cross_tx_num']) for tx in sharded_txes[i]]) for i in range(N)]
cross_tx_num_list = [sum([int(tx['cross_tx_num']) for tx in sharded_txes[i]]) for i in range(N)]

normal_delay = [normal_delay_total_list[i] / normal_tx_num_list[i] for i in range(N)]
cross_delay = [cross_delay_total_list[i] / cross_tx_num_list[i] for i in range(N)]
delay = [(normal_delay_total_list[i] + cross_delay_total_list[i]) /
         (normal_tx_num_list[i] + cross_tx_num_list[i]) for i in range(N)]

print(normal_delay)
print(sum(normal_delay) / N)
print(cross_delay)
print(sum(cross_delay) / N)
print(delay)
print(sum(delay) / N)
