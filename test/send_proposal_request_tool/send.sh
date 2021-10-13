#/bin/bash

contract_name=$1
server_ip="127.0.0.1"
server_port=12301
project_path=../..

if [ $# != 1 ];then
  echo "input param error as:  ./send.sh contractName"
  exit -1
fi


./send_proposal_request_tool parallel invoke  \
--method=increase  \
--pairs="[{\"value\": \"value_1\", \"key\": \"value\", \"unique\": false}, {\"value\": \"name_1\", \"key\": \"name\",\"randomRate\": 10}, {\"value\": \"key_1\", \"key\": \"key\",\"unique\":true}]"   \
--ip=127.0.0.1  \
--port=12301  \
--hosts=localhost:12301  \
--user-key=${project_path}/config/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.key  \
--user-keys=${project_path}/config/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.key  \
--user-crt=${project_path}/config/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.crt  \
--user-crts=${project_path}/config/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.crt  \
--ca-path=${project_path}/config/crypto-config/wx-org1.chainmaker.org/ca/  \
--use-tls=true  \
--chain-id=chain1  \
--org-id=wx-org1.chainmaker.org  \
--org-ids=wx-org1.chainmaker.org  \
--admin-sign-crts=${project_path}/config/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.sign.crt  \
--admin-sign-keys=${project_path}/config/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.sign.key  \
--contract-name=${contract_name} \
-I=wx-org1.chainmaker.org \
--sleepTime=100  \
--threadNum=10  \
--loopNum=1000  \
--timeout=30  \
--printTime=1  \
--showKey=false
