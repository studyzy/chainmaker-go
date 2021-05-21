#
# Copyright (C) BABEC. All rights reserved.
# Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

export LD_LIBRARY_PATH=$(dirname $PWD)/:$LD_LIBRARY_PATH
export PATH=$(dirname $PWD)/prebuilt/linux:$(dirname $PWD)/prebuilt/win64:$PATH
export WASMER_BACKTRACE=1
cd ../../main

pid=`ps -ef | grep chainmaker | grep "\-c ../config" | grep -v grep |  awk  '{print $2}'`
if [ -z ${pid} ];then
    nohup ./chainmaker start -c ../config/wx-org1-solo-sql/chainmaker.yml local-tbft > ../log/org1/panic.log &
    echo "wx-org1 chainmaker is startting, pls check log..."
else
    echo "wx-org1 chainmaker is already started"
fi


sleep 2
ps -ef|grep chainmaker