#
# Copyright (C) BABEC. All rights reserved.
# Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
rm -rf ../../data
rm -rf ../../log
rm -rf ../../main/panic.log
rm -rf ../../../cmdata
rm -rf ./*.log
rm -rf ./data
#用户名密码在$HOME/.my.cnf 这个文件中，如果没有请创建。内容如下：
#[mysql]
#host=127.0.0.1
#user=root
#password=123
dsn="-h127.0.0.1 -P3306"

for((i=1;i<=4;i++))
do
    mysql $dsn -e "show databases like 'org${i}_%'" |grep -v org${i}_% | xargs -I{} mysql $dsn -e "drop database {}"
done
mysql $dsn -e "show databases;"
