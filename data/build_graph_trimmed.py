import sqlite3
import pickle

import numpy as np

V_NUM = -1
E_NUM = -1

db = sqlite3.connect('db.sqlite3')
cursor = db.cursor()

xadj = np.zeros(V_NUM + 1)
vweights = np.zeros(V_NUM)
adjncy = np.zeros(E_NUM)
eweights = np.zeros(E_NUM)

i = 0
for j in range(1, V_NUM + 1):
    sql = '''\
SELECT `seq` - 1, SUM(`eweight`) FROM (
    SELECT `vst`.`seq` AS `seq`, `eweight` FROM `es`
        JOIN `vs` `vsf` ON `vsf`.`id` = `es`.`from`
        JOIN `vs` `vst` ON `vst`.`id` = `es`.`to`
        WHERE `vsf`.`seq` = ? AND `vst`.`seq` != ?
    UNION SELECT `vsf`.`seq` AS `seq`, `eweight` FROM `es`
        JOIN `vs` `vsf` ON `vsf`.`id` = `es`.`from`
        JOIN `vs` `vst` ON `vst`.`id` = `es`.`to`
        WHERE `vsf`.`seq` != ? AND `vst`.`seq` = ?
) GROUP BY `seq`
'''
    cursor.execute(sql, (j, j, j, j))
    vweight = 0
    for row in cursor.fetchall():
        idx, eweight = row
        adjncy[i] = idx
        eweights[i] = eweight
        vweight += eweight
        i += 1
    xadj[j] = i

    sql = '''\
SELECT `eweight` FROM `es`
    JOIN `vs` `vsf` ON `vsf`.`id` = `es`.`from`
    JOIN `vs` `vst` ON `vst`.`id` = `es`.`to`
WHERE `vsf`.`seq` = ? AND `vst`.`seq` = ?
'''
    cursor.execute(sql, (j, j))
    vweight += cursor.fetchone()[0]
    vweights[j - 1] = vweight

    if (j - 1) % 1000000 == 0 and j - 1 > 0:
        print(f'Processed {j - 1} rows')
print('Done')

with open('graph/xadj.pkl', 'wb') as f:
    pickle.dump(xadj, f)
with open('graph/adjncy.pkl', 'wb') as f:
    pickle.dump(adjncy, f)
with open('graph/eweights.pkl', 'wb') as f:
    pickle.dump(eweights, f)
with open('graph/vweights.pkl', 'wb') as f:
    pickle.dump(vweights, f)
