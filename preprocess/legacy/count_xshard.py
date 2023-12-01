import sys
import csv

SHARDS = [8, 16, 24, 32, 40, 48, 56, 64]
SHARD_BITN = [3, 4, 5, 5, 6, 6, 6, 6]


def get_monoxide_shard(addr, shard_num):
    addr = addr.removeprefix('0x')
    bitn = SHARD_BITN[SHARDS.index(shard_num)]
    if bitn == 3:
        split = (int(addr[0], 16) & 0b1110) >> 1
    elif bitn == 4:
        split = int(addr[0], 16)
    elif bitn == 5:
        split = (int(addr[:2], 16) & 0b11111000) >> 3
    elif bitn == 6:
        split = (int(addr[:2], 16) & 0b11111100) >> 2
    else:
        raise RuntimeError('Invalid bitn')
    return int(split / 2 ** bitn * shard_num)


csv_writer = csv.DictWriter(sys.stdout, fieldnames=['shardNum', 'monoxide', 'metis'])
csv_writer.writeheader()
for shard in SHARDS:
    monoxide_n = 0
    metis_n = 0
    with open(f'data/tx_part{shard}.csv') as f:
        csv_reader = csv.DictReader(f)
        i = 0
        for row in csv_reader:
            i += 1
            if i % 100000 == 0:
                print(f'Processed part{shard} {i} rows', file=sys.stderr)
            if row['fromShard'] != row['toShard']:
                metis_n += 1
            if get_monoxide_shard(row['from'], shard) != get_monoxide_shard(row['to'], shard):
                monoxide_n += 1
        print(f'Processed part{shard}', file=sys.stderr)
    csv_writer.writerow({'shardNum': shard, 'monoxide': monoxide_n, 'metis': metis_n})
