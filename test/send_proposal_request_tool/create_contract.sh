#/bin/bash

contract_name=$1
server_ip="127.0.0.1"
server_port=12301
tools_path=/mnt/d/develop/workspace/chainMaker/chainmaker-go/test/send_proposal_request_tool
project_path=/mnt/d/develop/workspace/chainMaker/chainmaker-go

if [ $# != 1 ];then
  echo "input param error as:  ./create_contract.sh contractName"
  exit -1
fi


${tools_path}/send_proposal_request_tool createContract  \
--run-time=2  \
--wasm-path=${project_path}/test/wasm/rust-counter-1.1.1.wasm   \
--pairs="[]"   \
--ip=127.0.0.1  \
--port=12301  \
--user-key=${project_path}/config/wx-org1/certs/user/client1/client1.sign.key  \
--user-crt=${project_path}/config/wx-org1/certs/user/client1/client1.sign.crt  \
--ca-path=${project_path}/config/wx-org1/certs/ca/wx-org1.chainmaker.org  \
--use-tls=true  \
--chain-id=chain1  \
--org-id=wx-org1.chainmaker.org  \
--org-ids=wx-org1.chainmaker.org,wx-org2.chainmaker.org,wx-org3.chainmaker.org,wx-org4.chainmaker.org  \
--contract-name=${contract_name}  \
--admin-sign-crts=${project_path}/config/wx-org1/certs/user/admin1/admin1.sign.crt,${project_path}/config/wx-org2/certs/user/admin1/admin1.sign.crt,${project_path}/config/wx-org3/certs/user/admin1/admin1.sign.crt,${project_path}/config/wx-org4/certs/user/admin1/admin1.sign.crt  \
--admin-sign-keys=${project_path}/config/wx-org1/certs/user/admin1/admin1.sign.key,${project_path}/config/wx-org2/certs/user/admin1/admin1.sign.key,${project_path}/config/wx-org3/certs/user/admin1/admin1.sign.key,${project_path}/config/wx-org4/certs/user/admin1/admin1.sign.key






