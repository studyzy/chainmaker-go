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
START_TYPE=$1

function split_window() {
    tmux splitw -v -p 50
    tmux splitw -h -p 50
    tmux selectp -t 0
    tmux splitw -h -p 50
}

function cluster_start() {
    echo "===> Staring chainmaker cluster"

    if [[ $START_TYPE == "normal" ]] ; then
        start_normal
    else
        start_tmux
    fi

}

function prepare() {
    for file in `ls $RELEASE_PATH`
    do
        if [[ $file == chainmaker* ]] && [[ $file == *gz ]]; then
            tar -zxvf $RELEASE_PATH/$file -C $RELEASE_PATH
        fi
    done
}

function start_normal() {
    cd $RELEASE_PATH
    for file in `ls $RELEASE_PATH`
    do
        if [ -d $file ]; then
            echo "START ==> " $RELEASE_PATH/$file
            cd $file/bin && ./restart.sh && cd - > /dev/null
        fi
    done
}

function start_tmux() {
    tmux new -d -s chainmaker > /dev/null 2>&1 || (tmux kill-session -t chainmaker && tmux new -d -s chainmaker) 
    split_window
    cnt=0
    once=1
    for file in `ls $RELEASE_PATH`
    do
        if [ -d $RELEASE_PATH/$file ]; then
            echo "START ==> " $RELEASE_PATH/$file
            tmux selectp -t $((cnt % 4))
            tmux send-keys "export PS1=\"\[\e[32m\]($(date +%Y-%m-%d) \t) <node$(($cnt+1)) \W> \[\e[m\]\" && cd $RELEASE_PATH/$file/bin && ./restart.sh && reset && ./chainmaker version && echo \"sleep 5s...\" && sleep 5 && echo -e \"\\n>>> show last line log <<<\" && tail -n 1 ../log/system.log && echo -e \"\\n>>> show process list <<<\" && ps axo pid,cmd | grep -v grep | grep \"chainmaker start\"" C-m

            if [ $once -eq 1 ] && [[ $(($cnt/4)) -eq 1 ]]; then
                tmux new-window
                tmux selectw -t 1
                split_window
                once=0
            fi

            cnt=$(($cnt+1))
        fi
    done

    tmux selectw -t 0
    tmux attach-session -t chainmaker
}

prepare
cluster_start
