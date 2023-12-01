import sys
import csv
import sqlite3

db = sqlite3.connect('data/db.sqlite')
cursor = db.cursor()

i = 0
csv_reader = csv.DictReader(sys.stdin)
for row in csv_reader:
    if row['fromIsContract'] == '0' and row['toIsContract'] == '0' and row['to'] != 'None' and row['from'] != row['to']:
        if row['to'] < row['from']:
            row['from'], row['to'] = row['to'], row['from']
        row['from'] = row['from'].removeprefix('0x')
        row['to'] = row['to'].removeprefix('0x')
        cursor.executemany('INSERT OR IGNORE INTO `vs` (`id`) VALUES (?)', [(row['from'],), (row['to'],)])
        cursor.execute(
            'INSERT INTO `es` (`from`, `to`) VALUES (?, ?) ON CONFLICT (`from`, `to`) DO UPDATE SET `n` = `n` + 1', (row['from'], row['to']))
        i += 1
        if i % 100000 == 0:
            db.commit()
            print(f'Processed {i} rows')
db.commit()
print(f'Processed {i} rows')
