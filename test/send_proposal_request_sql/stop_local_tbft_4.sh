#
# Copyright (C) BABEC. All rights reserved.
# Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

pid=`ps -ef | grep chainmaker | grep "local-tbft" | grep -v grep |  awk  '{print $2}'`
for p in $pid
do
    kill $p
    echo "kill $p"
done

ps -ef|grep chainmaker | grep "local-tbft"
