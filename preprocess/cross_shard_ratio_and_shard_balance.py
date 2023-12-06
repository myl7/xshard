import csv
import sys
import logging
import os

logging.basicConfig(level=logging.INFO)

SWITCH_TO_RANDOM = bool(os.environ.get('SWITCH_TO_RANDOM', None))

# We use Ethereum data some they are always SISO.
# But here we will leave the place to handle MIMO.

# Shard num
PART_N = 8
part_counter = [0 for _ in range(PART_N)]


def get_random_shard(addr):
    addr_s = addr.removeprefix('0x')
    part_16 = int(addr_s[0], 16)
    return part_16 % PART_N


n = 0
cross_n = 0
r = csv.DictReader(sys.stdin)
for row in r:
    n += 1
    fromShards = [row['fromShard']]
    toShards = [row['toShard']]

    if SWITCH_TO_RANDOM:
        fromShards = [get_random_shard(row['from'])]
        toShards = [get_random_shard(row['to'])]

    is_cross = False
    if len(fromShards) != 1 or len(toShards) != 1 or fromShards[0] != toShards[0]:
        is_cross = True
    if is_cross:
        cross_n += 1
    if is_cross:
        for shard in set(fromShards + toShards):
            part_counter[int(shard)] += 1
    else:
        part_counter[int(fromShards[0])] += 1

logging.info('cross_n: %d, n: %d, ratio: %f', cross_n, n, cross_n / n)
logging.info('part_counter: %s', part_counter)
