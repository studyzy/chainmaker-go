#
# Copyright (C) BABEC. All rights reserved.
# Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
pid=`ps -ef | grep chainmaker | grep "ci-sql-tbft" | grep -v grep |  awk  '{print $2}'`
if test -z "$pid";  then
  echo "no process to kill"
else
  for p in $pid
  do
      kill $p
      echo "kill $p"
  done
#  ps -ef|grep chainmaker | grep "ci-sql-tbft"
fi

