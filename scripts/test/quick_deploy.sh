#!/usr/bin/env bash
#
# Copyright (C) BABEC. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
#


# chainmaker-go
cd ../../
mv build build-bak


# chainmaker-go/scripts
cd scripts
./cluster_quick_stop.sh clean
echo -e "\nINFO\n" | ./prepare.sh 4 1
./build_release.sh
./cluster_quick_start.sh normal

# chainmaker-go/build
cd ../build
mkdir bak
mv release/*.gz bak/

# chainmaker-go
cd ..
make cmc
# chainmaker-go/bin
cd bin
rm -rf testdata
mkdir testdata
cp ../../sdk-go/testdata/sdk_config.yml testdata/
cp -r ../build/crypto-config/ testdata/

sleep 1
ps -ef|grep chainmaker
sleep 1

## create contract
./cmc client contract user create \
				--contract-name=fact \
				--runtime-type=WASMER \
				--byte-code-path=../test/wasm/rust-fact-2.0.0.wasm \
				--version=1.0 \
				--sdk-conf-path=./testdata/sdk_config.yml \
				--admin-org-ids=wx-org1.chainmaker.org \
				--admin-key-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.tls.key \
				--admin-crt-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.tls.crt \
				--org-id=wx-org1.chainmaker.org \
				--sync-result=true \
				--params="{}"

./cmc client contract user invoke \
				--contract-name=fact \
				--method=save \
				--sdk-conf-path=./testdata/sdk_config.yml \
				--params="{\"faile_name\":\"name007\",\"file_hash\":\"ab3456df5799b87c77e7f88\",\"time\":\"6543234\"}" \
				--org-id=wx-org1.chainmaker.org \
				--sync-result=true

./cmc client contract user get \
				--contract-name=fact \
				--method=find_by_file_hash \
				--sdk-conf-path=./testdata/sdk_config.yml \
				--params="{\"file_hash\":\"ab3456df5799b87c77e7f88\"}" \
				--org-id=wx-org1.chainmaker.org \
