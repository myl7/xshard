import subprocess

ips = [
    ['54.86.148.25', '54.89.7.136', '18.206.212.204', '54.90.227.56'],
    ['34.229.218.1', '50.16.162.29', '3.80.86.4', '52.90.1.101', '3.84.184.65', '18.212.203.44'],
    ['50.16.146.230', '34.228.71.33', '54.160.182.59', '52.90.149.200', '54.226.168.49', '54.242.144.221'],
    ['54.87.30.15', '34.228.68.80', '54.90.203.202', '54.89.140.7', '34.224.90.77', '34.226.194.50'],
    ['54.242.239.230', '52.73.113.102', '54.210.184.99', '52.91.158.69', '3.85.204.129', '52.87.253.200']
]

SSH_PROXY = ''

RSCP_CMD = f'rsync -e "ssh -i ~/.ssh/mingchain.pem{SSH_PROXY}" -P '
SSH_CMD = f'ssh -i ~/.ssh/mingchain.pem{SSH_PROXY} '

SKIP_COPY = False

if not SKIP_COPY:
    for i, ip in enumerate(ips[0]):
        subprocess.run(
            RSCP_CMD + f'gcfg.json ubuntu@{ip}:/home/ubuntu/', shell=True)
        print(f'Leader {i} is copied')
    for i in range(4):
        shard_ips = ips[i + 1]
        for j, ip in enumerate(shard_ips):
            subprocess.run(
                RSCP_CMD + f'gcfg.json ubuntu@{ip}:/home/ubuntu/', shell=True)
            print(f'Validator {i}-{j+1} is copied')
print('Start to run command')
for i, ip in enumerate(ips[0]):
    subprocess.run(
        SSH_CMD + f'ubuntu@{ip} "nohup ./mingchain-leader -shard-id={i} -gcfg-path=gcfg.json -scfg-path={i}.bin &> log &"', shell=True)
    print(f'Leader {i} is running')
for i in range(4):
    shard_ips = ips[i + 1]
    for j, ip in enumerate(shard_ips):
        subprocess.run(
            SSH_CMD + f'ubuntu@{ip} "nohup ./mingchain-validator -shard-id={i} -node-id={j+1} -gcfg-path=gcfg.json -scfg-path={i}.bin &> log &"', shell=True)
        print(f'Validator {i}-{j+1} is running')
