import re

with open('ips_raw.txt') as f:
    text = f.read()
    ips = re.findall(r'\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}', text)
    print(' '.join(ips))
    print(len(ips))
