#!/usr/bin/env bash
#
# Copyright (C) BABEC. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
## deploy ChainMaker and test

node_count=$1
chain_count=$2
alreadyBuild=$3
if [ ! -d "../../tools/chainmaker-cryptogen" ]; then
  echo "not found chainmaker-go/tools/chainmaker-cryptogen"
  echo "  you can use "
  echo "              cd chainmaker-go/tools"
  echo "              ln -s ../../chainmaker-cryptogen ."
  echo "              cd chainmaker-cryptogen && make"
  exit 0
fi
CURRENT_PATH=$(pwd)
SCRIPT_PATH=$(dirname "${CURRENT_PATH}")
PROJECT_PATH=$(dirname "${SCRIPT_PATH}")

if  [[ ! -n $node_count ]] ;then
    echo "node cnt is empty"
    exit 1
fi
if  [ ! $node_count -eq 1 ] && [ ! $node_count -eq 4 ] && [ ! $node_count -eq 7 ]&& [ ! $node_count -eq 10 ]&& [ ! $node_count -eq 13 ]&& [ ! $node_count -eq 16 ];then
    echo "node cnt should be 1 or 4 or 7 or 10 or 13"
    exit 1
fi

function start_chainmaker() {
  cd $SCRIPT_PATH
  ./cluster_quick_stop.sh clean
  echo -e "\n\n【generate】 certs and config..."
  echo -e "\nINFO\n\n\n" | ./prepare_pk.sh $node_count $chain_count
  echo -e "\n\n【build】 release..."
  ./build_release.sh
  echo -e "\n\n【start】 chainmaker..."
  ./cluster_quick_start.sh normal
  sleep 1

  echo "【chainmaker】 process..."
  ps -ef | grep chainmaker
  chainmaker_count=$(ps -ef | grep chainmaker | wc -l)
  if [ $chainmaker_count -lt 4 ]; then
    echo "build error"
    exit
  fi

  # backups *.gz
  cd $PROJECT_PATH/build
  mkdir -p bak
  mv release/*.gz bak/
}

function prepare_cmc() {
  if [ "${alreadyBuild}" != "true" ]; then
    echo "【build】 cmc start..."
    cd $PROJECT_PATH
    make cmc
  fi

  echo "【prepare】 cmc cert and sdk..."
  cd $PROJECT_PATH/bin
  pwd
  rm -rf testdata
  mkdir testdata
  cp $PROJECT_PATH/tools/cmc/testdata/sdk_config_pk.yml testdata/
  cp -r $PROJECT_PATH/build/crypto-config/ testdata/
}

function cmc_test() {
  echo "【cmc】 send tx..."
  cd $PROJECT_PATH/bin
  pwd
  ## create contract
  ./cmc client contract user create \
    --contract-name=fact \
    --runtime-type=WASMER \
    --byte-code-path=../test/wasm/rust-fact-2.0.0.wasm \
    --version=1.0 \
    --sdk-conf-path=./testdata/sdk_config_pk.yml \
    --sync-result=true \
    --params="{}"

  ## invoke tx
  ./cmc client contract user invoke \
    --contract-name=fact \
    --method=save \
    --sdk-conf-path=./testdata/sdk_config_pk.yml \
    --params="{\"file_name\":\"name007\",\"file_hash\":\"ab3456df5799b87c77e7f88\",\"time\":\"6543234\"}" \
    --sync-result=true

  ## query tx
  ./cmc client contract user get \
    --contract-name=fact \
    --method=find_by_file_hash \
    --sdk-conf-path=./testdata/sdk_config_pk.yml \
    --params="{\"file_hash\":\"ab3456df5799b87c77e7f88\"}"
}

function cat_log() {
  grep "ERROR\|put block" $PROJECT_PATH/build/release/chainmaker-v2.1.0_alpha-wx-org1.chainmaker.org/log/system.log
}

start_chainmaker
prepare_cmc
cmc_test
cat_log