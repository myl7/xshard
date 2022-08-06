import re
import sys
import json

N = 20

objs = []
with open(sys.argv[1]) as f:
    for line in f:
        if re.match(r'^[^r]', line):
            if line[-5:] == 'Done\n':
                print('Done')
                break
            continue
        line = line.removeprefix('report: ')
        try:
            obj = json.loads(line)
        except json.decoder.JSONDecodeError:
            print(line)
            raise
        objs.append(obj)

with open(f'{sys.argv[1]}.json', 'w') as f:
    json.dump(objs, f)
