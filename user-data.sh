Content-Type: multipart/mixed; boundary="//"
MIME-Version: 1.0

--//
Content-Type: text/cloud-config; charset="us-ascii"
MIME-Version: 1.0
Content-Transfer-Encoding: 7bit
Content-Disposition: attachment; filename="cloud-config.txt"

#cloud-config
cloud_final_modules:
- [scripts-user, always]

--//
Content-Type: text/x-shellscript; charset="us-ascii"
MIME-Version: 1.0
Content-Transfer-Encoding: 7bit
Content-Disposition: attachment; filename="userdata.txt"

#!/bin/bash
wget -O /home/ubuntu/node https://mingchain.s3.amazonaws.com/node
chmod a+x /home/ubuntu/node
ulimit -n 65535
sysctl -w net.ipv4.tcp_tw_reuse=1
/home/ubuntu/node -coorAddr 52.1.252.8 &> /home/ubuntu/node.log
--//--
