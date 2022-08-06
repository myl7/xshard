import csv
import pickle
import sys

input_pkl = sys.argv[1]
if not input_pkl.endswith('.pkl'):
    print('Invalid metis result input')
    exit(1)
output_csv = sys.argv[2]
if not output_csv.endswith('.csv'):
    print('Invalid tx csv output')
    exit(1)

f_in = open('tx_10w_with_seq.csv')
cr = csv.DictReader(f_in)
fieldnames = cr.fieldnames
fieldnames.append('fromShard')
fieldnames.append('toShard')
f_out = open(output_csv, 'w')
cw = csv.DictWriter(f_out, fieldnames=fieldnames)
cw.writeheader()

with open(input_pkl, 'rb') as f:
    shard_table = pickle.load(f)[1]

for row in cr:
    from_seq = int(row['fromSeq'])
    to_seq = int(row['toSeq'])

    from_shard = shard_table[from_seq]
    to_shard = shard_table[to_seq]

    row['fromShard'] = from_shard
    row['toShard'] = to_shard
    cw.writerow(row)
