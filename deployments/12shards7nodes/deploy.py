import subprocess

ips = [
    ['54.86.148.25', '54.89.7.136', '18.206.212.204', '54.90.227.56',
        '3.88.110.151', '54.161.1.84', '54.144.98.180', '54.147.29.87',
        '54.227.36.132', '54.196.228.252', '54.90.248.118', '54.83.83.44'],
    ['34.229.218.1', '50.16.162.29', '3.80.86.4', '52.90.1.101', '3.84.184.65', '18.212.203.44'],
    ['50.16.146.230', '34.228.71.33', '54.160.182.59', '52.90.149.200', '54.226.168.49', '54.242.144.221'],
    ['54.87.30.15', '34.228.68.80', '54.90.203.202', '54.89.140.7', '34.224.90.77', '34.226.194.50'],
    ['54.242.239.230', '52.73.113.102', '54.210.184.99', '52.91.158.69', '3.85.204.129', '52.87.253.200'],
    ['34.228.17.34', '54.196.36.237', '184.73.12.105', '18.209.14.238', '54.90.115.68', '3.91.76.210'],
    ['52.23.231.140', '18.234.170.62', '52.87.175.133', '3.90.190.100', '54.147.1.151', '54.89.200.233'],
    ['18.234.65.240', '35.173.249.56', '35.173.129.50', '54.159.12.201', '35.172.220.188', '54.196.130.163'],
    ['3.89.60.34', '18.205.234.193', '54.175.42.224', '54.91.28.52', '18.232.136.149', '54.90.132.44'],
    ['100.26.110.246', '54.204.224.152', '54.91.194.200', '54.226.123.70', '100.26.143.200', '54.235.0.146'],
    ['3.85.129.58', '174.129.64.76', '52.90.59.98', '54.221.176.127', '54.227.109.11', '34.207.115.113'],
    ['34.229.205.164', '54.160.144.114', '54.221.132.250', '54.90.74.206', '54.84.42.195', '18.234.211.144'],
    ['54.226.208.203', '34.228.7.11', '54.144.202.68', '34.234.78.54', '34.224.60.127', '3.95.175.156'],
]

SSH_PROXY = ''

RSCP_CMD = f'rsync -e "ssh -i ~/.ssh/mingchain.pem{SSH_PROXY}" -P '
SSH_CMD = f'ssh -i ~/.ssh/mingchain.pem{SSH_PROXY} '

SKIP_COPY = False

N = 12
N_OLD = 8

if not SKIP_COPY:
    for i, ip in enumerate(ips[0]):
        if i >= N_OLD:
            subprocess.run(
                SSH_CMD + f'ubuntu@{ip} "wget -nv -O mingchain-leader https://mingchain.s3.amazonaws.com/mingchain-leader && chmod a+x mingchain-leader"', shell=True)
            subprocess.run(
                RSCP_CMD + f'gcfg.json {i}.bin ubuntu@{ip}:/home/ubuntu/', shell=True)
        else:
            subprocess.run(RSCP_CMD + f'gcfg.json ubuntu@{ip}:/home/ubuntu/', shell=True)
        print(f'Leader {i} is copied')
    for i in range(N):
        shard_ips = ips[i + 1]
        for j, ip in enumerate(shard_ips):
            if i >= N_OLD:
                subprocess.run(
                    SSH_CMD + f'ubuntu@{ip} "wget -nv -O mingchain-validator https://mingchain.s3.amazonaws.com/mingchain-validator && chmod a+x mingchain-validator"', shell=True)
                subprocess.run(
                    RSCP_CMD + f'gcfg.json {i}.bin ubuntu@{ip}:/home/ubuntu/', shell=True)
            else:
                subprocess.run(
                    RSCP_CMD + f'gcfg.json ubuntu@{ip}:/home/ubuntu/', shell=True)
            print(f'Validator {i}-{j+1} is copied')
print('Start to run command')
for i, ip in enumerate(ips[0]):
    subprocess.run(
        SSH_CMD + f'ubuntu@{ip} "nohup ./mingchain-leader -shard-id={i} -gcfg-path=gcfg.json -scfg-path={i}.bin &> log &"', shell=True)
    print(f'Leader {i} is running')
for i in range(N):
    shard_ips = ips[i + 1]
    for j, ip in enumerate(shard_ips):
        subprocess.run(
            SSH_CMD + f'ubuntu@{ip} "nohup ./mingchain-validator -shard-id={i} -node-id={j+1} -gcfg-path=gcfg.json -scfg-path={i}.bin &> log &"', shell=True)
        print(f'Validator {i}-{j+1} is running')
