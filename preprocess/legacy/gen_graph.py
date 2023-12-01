import sqlite3
import pickle

db = sqlite3.connect('data/db.sqlite')
cursor = db.cursor()

cursor.execute('SELECT COUNT(*) FROM vs')
vn = cursor.fetchone()[0]
cursor.execute('SELECT COUNT(*) FROM es')
en = cursor.fetchone()[0]
print(f'To process {vn} vertices and {en} edges')

graph = {}
graph['nvtxs'] = vn
graph['ncon'] = 1
graph['xadj'] = [0] * (vn + 1)
graph['adjncy'] = [0] * 2 * en
graph['vwgt'] = [0] * vn
graph['adjwgt'] = [0] * 2 * en

i = 0
cursor.execute('SELECT `id`, `seq`, `w`, `n` FROM `vs` ORDER BY `seq`')
for id, seq, w, n in cursor.fetchall():
    graph['vwgt'][seq] = w
    graph['xadj'][seq + 1] = n + graph['xadj'][seq]
    cursor.execute(
        'SELECT `seq`, `ids`.`w` FROM (SELECT `from` AS `id`, `w` FROM `es` WHERE `to` = ? UNION SELECT `to` AS `id`, `w` FROM `es` WHERE `from` = ?) AS `ids` JOIN `vs` ON `vs`.`id` = `ids`.`id`', (id, id))
    for id, w in cursor.fetchall():
        graph['adjncy'][i] = id
        graph['adjwgt'][i] = w
        i += 1
        if i % 100000 == 0:
            print(f'Processed {i} edges')
print(f'Processed {i} edges')
with open('data/graph.pkl', 'wb') as f:
    pickle.dump(graph, f)
