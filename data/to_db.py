import csv
import sqlite3

db = sqlite3.connect('db.sqlite3')
cursor = db.cursor()

i = 0
with open('tx.csv') as f:
    c = csv.DictReader(f)
    for r in c:
        v_from = r['from'].removeprefix('0x')
        v_to = r['to'].removeprefix('0x')

        sql = '''INSERT OR IGNORE INTO `vs` (`id`) VALUES (?)'''
        cursor.execute(sql, (v_from,))
        cursor.execute(sql, (v_to,))

        sql = '''INSERT OR IGNORE INTO `es` (`from`, `to`) VALUES (?, ?)'''
        cursor.execute(sql, (v_from, v_to))
        sql = '''UPDATE `es` SET `eweight` = `eweight` + 1 WHERE `from` = ? AND `to` = ?'''
        cursor.execute(sql, (v_from, v_to))

        i += 1
        if i % 1000000 == 0:
            print(f'Processed tx num: {i}')
            db.commit()
print('Done')

db.commit()
db.close()
