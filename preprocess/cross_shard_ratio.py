import csv
import sys
import logging

logging.basicConfig(level=logging.INFO)

# We use Ethereum data some they are always SISO.
# But here we will leave the place to handle MIMO.

n = 0
cross_n = 0
r = csv.DictReader(sys.stdin)
for row in r:
    n += 1
    fromShards = [row['fromShard']]
    toShards = [row['toShard']]
    if len(fromShards) != 1 or len(toShards) != 1 or fromShards[0] != toShards[0]:
        cross_n += 1

logging.info('cross_n: %d, n: %d, ratio: %f', cross_n, n, cross_n / n)
