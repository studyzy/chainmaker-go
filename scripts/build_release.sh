#!/usr/bin/env bash
#
# Copyright (C) BABEC. All rights reserved.
# Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

set -e

CURRENT_PATH=$(pwd)
PROJECT_PATH=$(dirname "${CURRENT_PATH}")
BUILD_PATH=${PROJECT_PATH}/build
RELEASE_PATH=${PROJECT_PATH}/build/release
BACKUP_PATH=${PROJECT_PATH}/build/backup
BUILD_CRYPTO_CONFIG_PATH=${BUILD_PATH}/crypto-config
BUILD_CONFIG_PATH=${BUILD_PATH}/config
VERSION=V1.0.0
DATETIME=$(date "+%Y%m%d%H%M%S")
PLATFORM=$(uname -m)

function check_env() {
    if  [ ! -d $BUILD_CONFIG_PATH ] ;then
        echo $BUILD_CONFIG_PATH" is missing"
        exit 1
    fi

    if  [ ! -d $BUILD_CRYPTO_CONFIG_PATH ] ;then
        echo $BUILD_CRYPTO_CONFIG_PATH" is missing"
        exit 1
    fi
}

function build() {
    cd $PROJECT_PATH
    make
}

function package() {
    if [ -d $RELEASE_PATH ]; then
        mkdir -p $BACKUP_PATH/backup_release
        mv $RELEASE_PATH $BACKUP_PATH/backup_release/release_$(date "+%Y%m%d%H%M%S")
    fi

    mkdir -p $RELEASE_PATH
    cd $RELEASE_PATH
    tar -zcvf crypto-config-$DATETIME.tar.gz ../crypto-config

    c=0
    for file in `ls -tr $BUILD_CRYPTO_CONFIG_PATH`
    do
        c=$(($c+1))
        chainmaker_file=chainmaker-$VERSION-$file
        mkdir $chainmaker_file
        mkdir $chainmaker_file/bin
        mkdir $chainmaker_file/lib
        mkdir -p $chainmaker_file/config/$file
        mkdir $chainmaker_file/log
        cp $PROJECT_PATH/bin/chainmaker   $chainmaker_file/bin
        cp $CURRENT_PATH/bin/start.sh     $chainmaker_file/bin
        cp $CURRENT_PATH/bin/stop.sh      $chainmaker_file/bin
        cp $CURRENT_PATH/bin/restart.sh   $chainmaker_file/bin
#        cp -r $PROJECT_PATH/main/libwasmer_runtime_c_api.so     $chainmaker_file/bin/libwasmer.so
#        chmod 644 $chainmaker_file/bin/libwasmer.so
        cp -r $PROJECT_PATH/main/libwasmer_runtime_c_api.so     $chainmaker_file/lib/libwasmer.so
        chmod 644 $chainmaker_file/lib/*
        cp -r $BUILD_CONFIG_PATH/node$c/* $chainmaker_file/config/$file
        sed -i "s%{org_id}%$file%g"       $chainmaker_file/bin/start.sh
        sed -i "s%{org_id}%$file%g"       $chainmaker_file/bin/stop.sh
        sed -i "s%{org_id}%$file%g"       $chainmaker_file/bin/restart.sh
        tar -zcvf chainmaker-$VERSION-$file-$DATETIME-$PLATFORM.tar.gz $chainmaker_file
        rm -rf $chainmaker_file
    done
}

check_env
build
package
