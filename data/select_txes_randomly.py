from random import randint
import sys

i = -1
j = 0
k = randint(0, 2488)
for line in sys.stdin:
    line = line.removesuffix('\n')
    if i == -1:
        print(line)
    if i % 2489 == k:
        print(line)
        j += 1
        k = randint(0, 2488)
    if j >= 100000:
        break
    i += 1
