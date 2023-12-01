import pickle
import ctypes
import argparse

parser = argparse.ArgumentParser()
parser.add_argument('nparts', type=int)
args = parser.parse_args()

with open('data/graph.pkl', 'rb') as f:
    graph = pickle.load(f)
try:
    metis = ctypes.CDLL('libmetis.so')
except OSError:
    metis = ctypes.CDLL('libmetis.so.5')
metis_f = metis.METIS_PartGraphKway

nvtxs = ctypes.c_int(graph['nvtxs'])
ncon = ctypes.c_int(graph['ncon'])
xadj = (ctypes.c_int * len(graph['xadj']))(*graph['xadj'])
adjncy = (ctypes.c_int * len(graph['adjncy']))(*graph['adjncy'])
vwgt = (ctypes.c_int * len(graph['vwgt']))(*graph['vwgt'])
vsize = None
adjwgt = (ctypes.c_int * len(graph['adjwgt']))(*graph['adjwgt'])
nparts = ctypes.c_int(args.nparts)
tpwgts = None
ubvec = None
options = None
objval = ctypes.c_int()
part = (ctypes.c_int * graph['nvtxs'])()
ret = metis_f(
    ctypes.byref(nvtxs),
    ctypes.byref(ncon),
    ctypes.byref(xadj),
    ctypes.byref(adjncy),
    ctypes.byref(vwgt),
    vsize,
    ctypes.byref(adjwgt),
    ctypes.byref(nparts),
    tpwgts,
    ubvec,
    options,
    ctypes.byref(objval),
    ctypes.byref(part)
)
if int(ret) != 1:
    raise RuntimeError(f'METIS failed: ret = {ret}')
res = {
    'objval': int(objval),
    'part': part[:],
}
with open(f'data/part{args.nparts}.pkl', 'wb') as f:
    pickle.dump(res, f)
