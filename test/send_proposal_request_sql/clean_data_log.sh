#
# Copyright (C) BABEC. All rights reserved.
# Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
rm -rf ../../data/*
rm -rf ../../log/*/*
rm -rf ../../main/panic*.log
rm -rf ../../bin/panic*.log
rm -rf ../../../cmdata

dsn="-uroot -p123 -P3306 -h127.0.0.1"
#dsn="-uroot -ppassw0rd -P3307 -h192.168.1.35"

for((i=1;i<=4;i++))
do
    mysql $dsn -e "show databases like 'org${i}_%'" |grep -v org${i}_% | xargs -I{} mysql $dsn -e "drop database {}"
done
mysql $dsn -e "show databases;"
ps -fe|grep chainmaker|grep -v grep|grep start
