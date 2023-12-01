from random import randint
import sys
import csv

TXN = 94228894
TXN_SELECTED = 1000000

window = TXN // TXN_SELECTED
i = 0  # Seq in window
j = 0  # Selected seq
k = 0  # Processed rows
row_list = []
csv_reader = csv.DictReader(sys.stdin)
csv_writer = csv.DictWriter(sys.stdout, fieldnames=csv_reader.fieldnames)
csv_writer.writeheader()
for row in csv_reader:
    if row['fromIsContract'] == '0' and row['toIsContract'] == '0' and row['to'] != 'None' and row['from'] != row['to']:
        i += 1
        row_list.append(row)
        k += 1
        if i >= window:
            x = randint(0, window - 1)
            csv_writer.writerow(row_list[x])
            row_list = []
            i = 0
            j += 1
        if k % 1000000 == 0:
            print(f'Processed {k} rows', file=sys.stderr)
print(f'Selected {j} rows', file=sys.stderr)
