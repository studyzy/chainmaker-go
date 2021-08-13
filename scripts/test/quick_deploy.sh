#!/usr/bin/env bash
#
# Copyright (C) BABEC. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
#



# chainmaker-go/scripts
cd ..
./cluster_quick_stop.sh clean
sleep 1
mv ../build ../build-bak
echo -e "\nINFO\n" | ./prepare.sh 4 1
./build_release.sh
./cluster_quick_start.sh normal
sleep 1
ps -ef|grep chainmaker


# chainmaker-go/build
cd ../build
mkdir bak
mv release/*.gz bak/

# chainmaker-go/bin
cd ../bin
rm -rf testdata
mkdir testdata
cp ../../sdk-go/testdata/sdk_config.yml testdata/
cp -r ../build/crypto-config/ testdata/
cd ..

# chainmaker-go/scripts/test
cd scripts/test
#./cmc_test.sh
./send_tool_test.sh