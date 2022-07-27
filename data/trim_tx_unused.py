import csv

with open('/run/media/myl/6417-EBD3/tx.csv') as rf:
    with open('tx_trimmed.csv', 'w') as wf:
        i = 0
        rc = csv.DictReader(rf)
        fieldnames = ['block_id', 'timestamp', 'tx_hash', 'from', 'to', 'value']
        wc = csv.DictWriter(wf, fieldnames=fieldnames)
        wc.writeheader()

        for r in rc:
            w = {
                'block_id': r['blockNumber'],
                'timestamp': r['timestamp'],
                'tx_hash': r['transactionHash'],
                'from': r['from'].removeprefix('0x'),
                'to': r['to'].removeprefix('0x'),
                'value': r['value'],
            }
            wc.writerow(w)

            i += 1
            if i % 1000000 == 0:
                print(f'Processed tx num: {i}')
print('Done')
