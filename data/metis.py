import pickle
import sys

import numpy as np
import pymetis

N = int(sys.argv[1])
OUTPUT_FILE = sys.argv[2]

with open('graph/xadj.pkl', 'rb') as f:
    xadj = pickle.load(f).astype(np.int32)
with open('graph/adjncy.pkl', 'rb') as f:
    adjncy = pickle.load(f).astype(np.int32)
with open('graph/vweights.pkl', 'rb') as f:
    vweights = pickle.load(f).astype(np.int32)
with open('graph/eweights.pkl', 'rb') as f:
    eweights = pickle.load(f).astype(np.int32)

print('Start')
res = pymetis.part_graph(N, xadj=xadj, adjncy=adjncy, vweights=vweights, eweights=eweights)
print('Done')

with open(OUTPUT_FILE, 'wb') as f:
    pickle.dump(res, f)
