#
# Copyright (C) BABEC. All rights reserved.
# Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

export LD_LIBRARY_PATH=$(dirname $PWD)/lib:$LD_LIBRARY_PATH
export WASMER_BACKTRACE=1
pid=`ps -ef | grep chainmaker | grep "\-c ../config/{org_id}/chainmaker.yml" | grep -v grep | awk  '{print $2}'`
if [ ! -z ${pid} ];then
    kill -9 $pid
fi
sleep 2
#nohup ./chainmaker start -c ../config/{org_id}/chainmaker.yml > /dev/null 2>&1 &
nohup ./chainmaker start -c ../config/{org_id}/chainmaker.yml > panic.log &
echo "chainmaker is restartting, pls check log..."
