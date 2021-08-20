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
START=$1
END=$2

function check_env(){
    if [[ ! -n $START ]];then
            echo "start node index is empty"
            exit 1
    fi
    if [[ ! -n $END ]];then
            echo "end node index is empty"
            exit 1
    fi

    if  [ ${START} -lt 1 ] || [ ${END} -gt 16 ] ;then
            echo "start or end exceed "
            echo "start nodes ["$START"~"$END"]"
            exit 1
    fi
    echo "start nodes ["$START"~"$END"]"
}

function start_all() {
    for file in `ls $RELEASE_PATH`
    do
        if [[ $file == chainmaker* ]] && [[ $file == *gz ]]; then
                index=`echo ${file} | grep -Eo "org[0-9]+" | grep -Eo "[0-9]+"`
                if [ ${index} -ge >= $START ] && [ ${index} -le $END ] ;then
                        echo $file
                        tar -zxvf $RELEASE_PATH/$file -C $RELEASE_PATH
                        dir_name=`echo $file | grep -Eo ".*chainmaker.org"`
                        echo $RELEASE_PATH/$dir_name
                        if [ -d $RELEASE_PATH/$dir_name ]; then
                            echo "START ==> " $RELEASE_PATH/$dir_name
                            cd $RELEASE_PATH/$dir_name/bin && ./restart.sh && cd - > /dev/null
                        fi
                fi
        fi
    done
}

check_env
start_all
