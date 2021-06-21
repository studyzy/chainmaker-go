#
# Copyright (C) BABEC. All rights reserved.
# Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

export LD_LIBRARY_PATH=$(dirname $PWD)/:$LD_LIBRARY_PATH
export PATH=$(dirname $PWD)/prebuilt/linux:$(dirname $PWD)/prebuilt/win64:$PATH
export WASMER_BACKTRACE=1
cd ../../bin

pid=`ps -ef | grep chainmaker | grep "\-c ../config-sql/wx-org1/chainmaker.yml local-tbft" | grep -v grep |  awk  '{print $2}'`
if [ -z ${pid} ];then
    nohup ./chainmaker start -c ../config-sql/wx-org1/chainmaker.yml local-tbft > panic1.log 2>&1 &
    echo "wx-org1 chainmaker is startting, pls check log..."
else
    echo "wx-org1 chainmaker is already started"
fi

pid2=`ps -ef | grep chainmaker | grep "\-c ../config-sql/wx-org2/chainmaker.yml local-tbft" | grep -v grep |  awk  '{print $2}'`
if [ -z ${pid2} ];then
    nohup ./chainmaker start -c ../config-sql/wx-org2/chainmaker.yml local-tbft > panic2.log 2>&1 &
    echo "wx-org2 chainmaker is startting, pls check log..."
else
    echo "wx-org2 chainmaker is already started"
fi



pid3=`ps -ef | grep chainmaker | grep "\-c ../config-sql/wx-org3/chainmaker.yml local-tbft" | grep -v grep |  awk  '{print $2}'`
if [ -z ${pid3} ];then
    nohup ./chainmaker start -c ../config-sql/wx-org3/chainmaker.yml local-tbft > panic3.log 2>&1 &
    echo "wx-org3 chainmaker is startting, pls check log..."
else
    echo "wx-org3 chainmaker is already started"
fi


pid4=`ps -ef | grep chainmaker | grep "\-c ../config-sql/wx-org4/chainmaker.yml local-tbft" | grep -v grep |  awk  '{print $2}'`
if [ -z ${pid4} ];then
    nohup ./chainmaker start -c ../config-sql/wx-org4/chainmaker.yml local-tbft > panic4.log 2>&1 &
    echo "wx-org4 chainmaker is startting, pls check log..."
else
    echo "wx-org4 chainmaker is already started"
fi

# nohup ./chainmaker start -c ../config-sql/wx-org5/chainmaker.yml local-tbft > panic.log &

sleep 4
ps -ef|grep chainmaker
