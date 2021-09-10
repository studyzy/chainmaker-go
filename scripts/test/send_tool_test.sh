#!/usr/bin/env bash
#
# Copyright (C) BABEC. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

alreadyBuild=$1

# chainmaker-go/test/send_proposal_request_tool
cd ../../test/send_proposal_request_tool

if [ "${alreadyBuild}" != "true" ]; then
  echo "build send_proposal_request_tool start..."
  go build
fi
cp -f send_proposal_request_tool ../../bin
# chainmaker-go/bin
cd ../../bin
## create contract
./send_proposal_request_tool createContract  \
--run-time=2  \
--wasm-path=../test/wasm/rust-fact-2.0.0.wasm    \
--ip=localhost  \
--port=12301  \
--user-key=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.key  \
--user-crt=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.crt  \
--ca-path=./testdata/crypto-config/wx-org1.chainmaker.org/ca  \
--use-tls=true  \
--chain-id=chain1  \
--org-id=wx-org1.chainmaker.org  \
--org-ids=wx-org1.chainmaker.org,wx-org2.chainmaker.org,wx-org3.chainmaker.org,wx-org4.chainmaker.org  \
--contract-name=demo01  \
--admin-sign-crts=./testdata/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.sign.crt,./testdata/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.sign.crt,./testdata/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.sign.crt,./testdata/crypto-config/wx-org4.chainmaker.org/user/admin1/admin1.sign.crt  \
--admin-sign-keys=./testdata/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.sign.key,./testdata/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.sign.key,./testdata/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.sign.key,./testdata/crypto-config/wx-org4.chainmaker.org/user/admin1/admin1.sign.key

sleep 3
# 调用
./send_proposal_request_tool invoke \
--method=save \
--pairs="[{\"value\": \"hash1\", \"key\": \"file_hash\"},{\"value\": \"name\", \"key\": \"file_name\"},{\"value\": \"123321\", \"key\": \"time\"}]" \
--ip=127.0.0.1 \
--port=12301 \
--user-key=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.key \
--user-crt=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.crt \
--ca-path=./testdata/crypto-config/wx-org1.chainmaker.org/ca \
--use-tls=true \
--chain-id=chain1 \
--org-id=wx-org1.chainmaker.org \
--org-ids=wx-org1.chainmaker.org,wx-org2.chainmaker.org,wx-org3.chainmaker.org,wx-org4.chainmaker.org \
--contract-name=demo01 \
--admin-sign-crts=./testdata/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.sign.crt,./testdata/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.sign.crt,./testdata/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.sign.crt,./testdata/crypto-config/wx-org4.chainmaker.org/user/admin1/admin1.sign.crt \
--admin-sign-keys=./testdata/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.sign.key,./testdata/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.sign.key,./testdata/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.sign.key,./testdata/crypto-config/wx-org4.chainmaker.org/user/admin1/admin1.sign.key

sleep 3
# 查询
./send_proposal_request_tool query \
--method=find_by_file_hash \
--pairs="[{\"value\": \"hash1\", \"key\": \"file_hash\"}]" \
--ip=127.0.0.1 \
--port=12301 \
--user-key=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.key \
--user-crt=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.crt \
--ca-path=./testdata/crypto-config/wx-org1.chainmaker.org/ca \
--use-tls=true \
--chain-id=chain1 \
--org-id=wx-org1.chainmaker.org \
--org-ids=wx-org1.chainmaker.org,wx-org2.chainmaker.org,wx-org3.chainmaker.org,wx-org4.chainmaker.org \
--contract-name=demo01 \
--admin-sign-crts=./testdata/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.sign.crt,./testdata/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.sign.crt,./testdata/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.sign.crt,./testdata/crypto-config/wx-org4.chainmaker.org/user/admin1/admin1.sign.crt \
--admin-sign-keys=./testdata/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.sign.key,./testdata/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.sign.key,./testdata/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.sign.key,./testdata/crypto-config/wx-org4.chainmaker.org/user/admin1/admin1.sign.key

