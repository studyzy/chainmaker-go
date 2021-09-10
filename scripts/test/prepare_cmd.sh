#!/usr/bin/env bash
#
# Copyright (C) BABEC. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
## deploy ChainMaker and test

alreadyBuild=$1

CURRENT_PATH=$(pwd)
SCRIPT_PATH=$(dirname "${CURRENT_PATH}")
PROJECT_PATH=$(dirname "${SCRIPT_PATH}")

if [ "${alreadyBuild}" != "true" ]; then
  echo "build cmc start..."
  cd $PROJECT_PATH
  make cmc
fi


cd $PROJECT_PATH/bin
pwd
rm -rf testdata
mkdir testdata
cp $PROJECT_PATH/../sdk-go/testdata/sdk_config.yml testdata/
cp -r $PROJECT_PATH/build/crypto-config/ testdata/