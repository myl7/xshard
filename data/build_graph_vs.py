import sqlite3
import pickle

import numpy as np

V_NUM = 35928379

db = sqlite3.connect('db.sqlite3')
cursor = db.cursor()

xadj = np.zeros(V_NUM + 1)
vweights = np.zeros(V_NUM)
sql = '''SELECT `seq`, `vweight`, `neighbor_num` FROM `vs`'''
cursor.execute(sql)
i = 0
for row in cursor.fetchall():
    seq, vweight, neighbor_num = row
    xadj[seq] = neighbor_num
    vweights[seq - 1] = vweight
    i += 1
    if i % 1000000 == 0:
        print(f'Processed {i} rows')
for j in range(1, V_NUM + 1):
    xadj[j] += xadj[j - 1]

with open('graph/xadj.pkl', 'wb') as f:
    pickle.dump(xadj, f)
with open('graph/vweights.pkl', 'wb') as f:
    pickle.dump(vweights, f)
