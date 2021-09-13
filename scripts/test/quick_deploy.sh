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
CURRENT_PATH=$(pwd)
SCRIPT_PATH=$(dirname "${CURRENT_PATH}")
PROJECT_PATH=$(dirname "${SCRIPT_PATH}")

function start_chainmaker() {
  cd $SCRIPT_PATH
  ./cluster_quick_stop.sh clean
  echo -e "\n\ngenerate certs and config..."
  echo -e "\nINFO\n" | ./prepare.sh 4 2
  echo -e "\n\nbuild release..."
  ./build_release.sh
  echo -e "\n\nstart chainmaker..."
  ./cluster_quick_start.sh normal
  sleep 1

  echo "chainmaker process..."
  ps -ef | grep chainmaker
  chainmaker_count=$(ps -ef | grep chainmaker | wc -l)
  if [ $chainmaker_count -lt 4 ]; then
    echo "build error"
    exit
  fi

  # backups *.gz
  cd $PROJECT_PATH/build
  mkdir bak
  mv release/*.gz bak/
}
start_chainmaker

cd $CURRENT_PATH
./prepare_cmd.sh alreadyBuild
cd $CURRENT_PATH
./cmc_test.sh alreadyBuild
