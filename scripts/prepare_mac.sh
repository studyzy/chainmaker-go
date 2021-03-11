#!/usr/bin/env bash
#
# Copyright (C) BABEC. All rights reserved.
# Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

set -e

NODE_CNT=$1
CHAIN_CNT=$2
P2P_PORT_PREFIX=$3
RPC_PORT_PREFIX=$4

CURRENT_PATH=$(pwd)
PROJECT_PATH=$(dirname "${CURRENT_PATH}")
BUILD_PATH=${PROJECT_PATH}/build
CONFIG_TPL_PATH=${PROJECT_PATH}/config/config_tpl
BUILD_CRYPTO_CONFIG_PATH=${BUILD_PATH}/crypto-config
BUILD_CONFIG_PATH=${BUILD_PATH}/config
CRYPTOGEN_TOOL_PATH=${PROJECT_PATH}/tools/chainmaker-cryptogen
CRYPTOGEN_TOOL_BIN=${CRYPTOGEN_TOOL_PATH}/bin/chainmaker-cryptogen
CRYPTOGEN_TOOL_CONF=${CRYPTOGEN_TOOL_PATH}/config/crypto_config_template.yml

function show_help() {
    echo "Usage:  "
    echo "  prepare.sh node_cnt(1/4/7/10/13) chain_cnt(1-4) p2p_port_prefix(default:11300) rpc_port_prefix(default:12300)"
    echo "    eg1: prepare.sh 4 1"
    echo "    eg2: prepare.sh 4 1 11300 12300"
}

if [ ! $# -eq 2 ] && [ ! $# -eq 3 ] && [ ! $# -eq 4 ]; then
    echo "invalid params"
    show_help
    exit 1
fi

function check_params() {
    echo "begin check params..."
    if  [[ ! -n $NODE_CNT ]] ;then
        echo "node cnt is empty"
        show_help
        exit 1
    fi

    if  [ ! $NODE_CNT -eq 1 ] && [ ! $NODE_CNT -eq 4 ] && [ ! $NODE_CNT -eq 7 ]&& [ ! $NODE_CNT -eq 10 ]&& [ ! $NODE_CNT -eq 13 ];then
        echo "node cnt should be 1 or 4 or 7 or 10 or 13"
        show_help
        exit 1
    fi

    if  [[ ! -n $CHAIN_CNT ]] ;then
        echo "chain cnt is empty"
        show_help
        exit 1
    fi

    if  [ ${CHAIN_CNT} -lt 1 ] || [ ${CHAIN_CNT} -gt 4 ] ;then
        echo "chain cnt should be 1 - 4"
        show_help
        exit 1
    fi

    if  [[ ! -n $P2P_PORT_PREFIX ]] ;then
        P2P_PORT_PREFIX=11300
    fi

    if  [ ${P2P_PORT_PREFIX} -ge 60000 ] || [ ${P2P_PORT_PREFIX} -le 10000 ];then
        echo "p2p_port_prefix should >=10000 && <=60000"
        show_help
        exit 1
    fi

    if  [[ ! -n $RPC_PORT_PREFIX ]] ;then
        RPC_PORT_PREFIX=12300
    fi

    if  [ ${RPC_PORT_PREFIX} -ge 60000 ] || [ ${RPC_PORT_PREFIX} -le 10000 ];then
        echo "rpc_port_prefix should >=10000 && <=60000"
        show_help
        exit 1
    fi
}

function generate_certs() {
    echo "begin generate certs, cnt: ${NODE_CNT}"
    mkdir -p ${BUILD_PATH}
    cd "${BUILD_PATH}"
    if [ -d crypto-config ]; then
        mkdir -p backup/backup_certs
        mv crypto-config  backup/backup_certs/crypto-config_$(date "+%Y%m%d%H%M%S")
    fi

    cp $CRYPTOGEN_TOOL_CONF crypto_config.yml

    sed -i "" "s%count: 4%count: ${NODE_CNT}%g" crypto_config.yml

    ${CRYPTOGEN_TOOL_BIN} generate -c ./crypto_config.yml
}

function generate_config() {
    LOG_LEVEL="INFO"
    CONSENSUS_TYPE=1
    MONITOR_PORT_PREFIX=14320
    PPROF_PORT_PREFIX=24320
    TRUSTED_PORT_PREFIX=13300

    read -p "input consensus type(default 1/tbft): " tmp
    if  [ ! -z "$tmp" ] ;then
        CONSENSUS_TYPE=$tmp
    fi
    read -p "input log level(default INFO): " tmp
    if  [ ! -z "$tmp" ] ;then
        LOG_LEVEL=$tmp
    fi

    cd "${BUILD_PATH}"
    if [ -d config ]; then
        mkdir -p backup/backup_config
        mv config  backup/backup_config/config_$(date "+%Y%m%d%H%M%S")
    fi


    mkdir -p ${BUILD_PATH}/config
    cd ${BUILD_PATH}/config

    for ((i = 1; i < $NODE_CNT + 1; i = i + 1)); do
        echo "begin generate node$i config..."
        mkdir -p ${BUILD_PATH}/config/node$i
        mkdir -p ${BUILD_PATH}/config/node$i/chainconfig
        cp $CONFIG_TPL_PATH/log.yml node$i
        sed -i "" "s%{log_level}%$LOG_LEVEL%g" node$i/log.yml
        cp $CONFIG_TPL_PATH/chainmaker.yml node$i

        sed -i "" "s%{net_port}%$(($P2P_PORT_PREFIX+$i))%g" node$i/chainmaker.yml
        sed -i "" "s%{rpc_port}%$(($RPC_PORT_PREFIX+$i))%g" node$i/chainmaker.yml
        sed -i "" "s%{monitor_port}%$(($MONITOR_PORT_PREFIX+$i))%g" node$i/chainmaker.yml
        sed -i "" "s%{pprof_port}%$(($PPROF_PORT_PREFIX+$i))%g" node$i/chainmaker.yml
        sed -i "" "s%{trusted_port}%$(($TRUSTED_PORT_PREFIX+$i))%g" node$i/chainmaker.yml

        for ((j = 1; j < $CHAIN_CNT + 1; j = j + 1)); do
            sed -i "" "s%#\(.*\)- chainId: chain${j}%\1- chainId: chain${j}%g" node$i/chainmaker.yml
            sed -i "" "s%#\(.*\)genesis: ../config/{org_path$j}/chainconfig/bc${j}.yml%\1genesis: ../config/{org_path$j}/chainconfig/bc${j}.yml%g" node$i/chainmaker.yml

            if  [ $NODE_CNT -eq 1 ]; then
                cp $CONFIG_TPL_PATH/chainconfig/bc_solo.yml node$i/chainconfig/bc$j.yml
                sed -i "" "s%{consensus_type}%0%g" node$i/chainconfig/bc$j.yml
            elif [ $NODE_CNT -eq 4 ] || [ $NODE_CNT -eq 7 ]; then
                cp $CONFIG_TPL_PATH/chainconfig/bc_4_7.yml node$i/chainconfig/bc$j.yml
                sed -i "" "s%{consensus_type}%$CONSENSUS_TYPE%g" node$i/chainconfig/bc$j.yml
            else
                cp $CONFIG_TPL_PATH/chainconfig/bc_10_13.yml node$i/chainconfig/bc$j.yml
                sed -i "" "s%{consensus_type}%$CONSENSUS_TYPE%g" node$i/chainconfig/bc$j.yml
            fi

            sed -i "" "s%{chain_id}%chain$j%g" node$i/chainconfig/bc$j.yml
            sed -i "" "s%{org_top_path}%$file%g" node$i/chainconfig/bc$j.yml

            if  [ $NODE_CNT -eq 7 ] || [ $NODE_CNT -eq 13 ]; then
                sed -i "" "s%#\(.*\)- org_id:%\1- org_id:%g" node$i/chainconfig/bc$j.yml
                sed -i "" "s%#\(.*\)address:%\1address:%g" node$i/chainconfig/bc$j.yml
                sed -i "" "s%#\(.*\)root:%\1root:%g" node$i/chainconfig/bc$j.yml
                sed -i "" "s%#\(.*\)- \"%\1- \"%g" node$i/chainconfig/bc$j.yml
            fi

            for ((k = 1; k < $NODE_CNT + 1; k = k + 1)); do
                sed -i "" "s%{org${k}_port}%$(($P2P_PORT_PREFIX+$k))%g" node$i/chainconfig/bc$j.yml
            done

            c=0
            for file in `ls -tr $BUILD_CRYPTO_CONFIG_PATH`
            do
                c=$(($c+1))
                sed -i "" "s%{org${c}_id}%$file%g" node$i/chainconfig/bc$j.yml

                peerId=`cat $BUILD_CRYPTO_CONFIG_PATH/$file/node/consensus1/consensus1.nodeid`
                sed -i "" "s%{org${c}_peerid}%$peerId%g" node$i/chainconfig/bc$j.yml

                for ((k = 1; k < $NODE_CNT + 1; k = k + 1)); do
                    mkdir -p $BUILD_CONFIG_PATH/node$k/certs/ca/$file
                    cp $BUILD_CRYPTO_CONFIG_PATH/$file/ca/ca.crt $BUILD_CONFIG_PATH/node$k/certs/ca/$file
                done

                if  [ $c -eq $i ]; then
                    sed -i "" "s%{org_path}%$file%g" node$i/chainconfig/bc$j.yml
                    sed -i "" "s%{node_cert_path}%node\/consensus1\/consensus1.sign%g" node$i/chainmaker.yml
                    sed -i "" "s%{net_cert_path}%node\/consensus1\/consensus1.tls%g" node$i/chainmaker.yml
                    sed -i "" "s%{rpc_cert_path}%node\/consensus1\/consensus1.tls%g" node$i/chainmaker.yml
                    sed -i "" "s%{org_id}%$file%g" node$i/chainmaker.yml
                    sed -i "" "s%{org_path}%$file%g" node$i/chainmaker.yml
                    sed -i "" "s%{org_path$j}%$file%g" node$i/chainmaker.yml

                    cp -r $BUILD_CRYPTO_CONFIG_PATH/$file/node $BUILD_CONFIG_PATH/node$i/certs
                    cp -r $BUILD_CRYPTO_CONFIG_PATH/$file/user $BUILD_CONFIG_PATH/node$i/certs
                fi

            done
        done
    done
}

check_params
generate_certs
generate_config
