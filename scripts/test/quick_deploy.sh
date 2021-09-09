#!/usr/bin/env bash
#
# Copyright (C) BABEC. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
## deploy ChainMaker and test

alreadyBuild=$1
if [ ! -d "../../tools/chainmaker-cryptogen" ]; then
  echo "not found chainmaker-go/tools/chainmaker-cryptogen"
  echo "  you can use "
  echo "              cd chainmaker-go/tools"
  echo "              ln -s ../../chainmaker-cryptogen ."
  echo "              cd chainmaker-cryptogen && make"
  exit 0
fi

# chainmaker-go/scripts
# backups & build release & start ChainMaker
cd ..
./cluster_quick_stop.sh clean
sleep 1
echo -e "\nINFO\n" | ./prepare.sh 4 1
./build_release.sh
./cluster_quick_start.sh normal
sleep 1
echo "chainmaker process"
ps -ef|grep chainmaker


# chainmaker-go/build
# backups *.gz
cd ../build
mkdir bak
mv release/*.gz bak/

# chainmaker-go/bin
# prepare sdk config & crypto config
cd ../bin
rm -rf testdata
mkdir testdata
cp ../../chainmaker-sdk-go/testdata/sdk_config.yml testdata/
cp -r ../build/crypto-config/ testdata/
cd ..

# chainmaker-go/scripts/test
cd scripts/test
./cmc_test.sh alreadyBuild
