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
RELEASE_PATH=${PROJECT_PATH}/build/release
VERSION=V1.0.0
ARG1=$1

function cluster_stop() {
    echo "===> Stoping chainmaker cluster"

    if [[ $ARG1 == "clean" ]] ; then
        clean
    fi
    stop_all
}

function stop_all() {
    cd $RELEASE_PATH
    for file in `ls $RELEASE_PATH`
    do
        if [ -d $file ]; then
            echo "STOP ==> " $RELEASE_PATH/$file
            cd $file/bin && ./stop.sh && cd - > /dev/null
        fi
    done
}

function clean() {
    cd $RELEASE_PATH
    for file in `ls $RELEASE_PATH`
    do
        if [ -d $file ]; then
            echo "CLEAN ==> " $RELEASE_PATH/$file/data
            cd $file && rm -rf data && cd - > /dev/null
        fi
    done
}

cluster_stop
