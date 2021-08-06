#!/usr/bin/env bash
#
# Copyright (C) BABEC. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

alreadyBuild=$1

# chainmaker-go
cd ../../

if [ "${alreadyBuild}" != "true" ]; then
  echo "build cmc start..."
  make cmc
fi
cd bin
## create contract
./cmc client contract user create \
--contract-name=fact \
--runtime-type=WASMER \
--byte-code-path=../test/wasm/rust-fact-2.0.0.wasm \
--version=1.0 \
--sdk-conf-path=./testdata/sdk_config.yml \
--admin-key-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.tls.key,./testdata/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.tls.key,./testdata/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.tls.key \
--admin-crt-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.tls.crt,./testdata/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.tls.crt,./testdata/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.tls.crt \
--org-id=wx-org1.chainmaker.org \
--sync-result=true \
--params="{}"

## invoke tx
./cmc client contract user invoke \
--contract-name=fact \
--method=save \
--sdk-conf-path=./testdata/sdk_config.yml \
--params="{\"faile_name\":\"name007\",\"file_hash\":\"ab3456df5799b87c77e7f88\",\"time\":\"6543234\"}" \
--org-id=wx-org1.chainmaker.org \
--sync-result=true

## query tx
./cmc client contract user get \
--contract-name=fact \
--method=find_by_file_hash \
--sdk-conf-path=./testdata/sdk_config.yml \
--params="{\"file_hash\":\"ab3456df5799b87c77e7f88\"}" \
--org-id=wx-org1.chainmaker.org \
