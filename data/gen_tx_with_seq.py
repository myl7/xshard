import sqlite3
import csv

db = sqlite3.connect('db.sqlite3')
cursor = db.cursor()

f_in = open('tx_10w.csv')
cr = csv.DictReader(f_in)
fieldnames = cr.fieldnames
fieldnames.append('fromSeq')
fieldnames.append('toSeq')
f_out = open('tx_10w_with_seq.csv', 'w')
cw = csv.DictWriter(f_out, fieldnames=fieldnames)
cw.writeheader()

for row in cr:
    tx_hash = row['transactionHash'].removeprefix('0x')
    from_addr = row['from'].removeprefix('0x')
    to_addr = row['to'].removeprefix('0x')

    sql = '''SELECT `seq` FROM `vs` WHERE `id` = ?'''
    cursor.execute(sql, (from_addr,))
    from_seq = cursor.fetchone()[0]
    cursor.execute(sql, (to_addr,))
    to_seq = cursor.fetchone()[0]

    row['transactionHash'] = tx_hash
    row['from'] = from_addr
    row['to'] = to_addr
    row['fromSeq'] = from_seq
    row['toSeq'] = to_seq
    cw.writerow(row)
