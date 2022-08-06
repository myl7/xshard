import re
import json

N = 20

objs = []
with open(f'log_{N}_7') as f:
    for line in f:
        if re.match(r'^[^r]', line):
            continue
        line = line.removeprefix('report: ')
        obj = json.loads(line)
        objs.append(obj)

with open(f'log_{N}_7.json', 'w') as f:
    json.dump(objs, f)
