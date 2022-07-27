import csv

accounts = set()
i = 0
with open('tx.csv') as f:
    c = csv.DictReader(f)
    for r in c:
        accounts.add(r['from'])
        accounts.add(r['to'])
        i += 1
        if i % 1000000 == 0:
            print(f'Processed tx num: {i}')
print(f'Account num: {len(accounts)}')
