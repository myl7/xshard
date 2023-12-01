import sys
import csv
import sqlite3
import pickle
import argparse

parser = argparse.ArgumentParser()
parser.add_argument('nparts', type=int)
args = parser.parse_args()

with open(f'data/part{args.nparts}.pkl', 'rb') as f:
    shard_map = pickle.load(f)['part']

db = sqlite3.connect('data/db.sqlite')
cursor = db.cursor()

csv_reader = csv.DictReader(sys.stdin)
fieldnames = csv_reader.fieldnames + ['fromSeq', 'toSeq', 'fromShard', 'toShard']
csv_writer = csv.DictWriter(sys.stdout, fieldnames=fieldnames)
csv_writer.writeheader()
i = 0
for row in csv_reader:
    i += 1
    if i % 100000 == 0:
        print(f'Processed {i} rows', file=sys.stderr)
    cursor.execute('SELECT seq FROM vs WHERE id = ?', (row['from'].removeprefix('0x'),))
    row['fromSeq'] = cursor.fetchone()[0]
    cursor.execute('SELECT seq FROM vs WHERE id = ?', (row['to'].removeprefix('0x'),))
    row['toSeq'] = cursor.fetchone()[0]
    row['fromShard'] = shard_map[row['fromSeq']]
    row['toShard'] = shard_map[row['toSeq']]
    csv_writer.writerow(row)
print(f'Processed {i} rows', file=sys.stderr)
