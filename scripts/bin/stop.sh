#
# Copyright (C) BABEC. All rights reserved.
# Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

pid=`ps -ef | grep chainmaker | grep "\-c ../config/{org_id}/chainmaker.yml" | grep -v grep |  awk  '{print $2}'`
if [ ! -z ${pid} ];then
    kill $pid
fi

enable_dockervm=`grep 'enable_dockervm:' ../config/{org_id}/chainmaker.yml | awk '{print $2}'`
if [ ${enable_dockervm} == "true" ];then
  docker_go_container_name=`grep 'dockervm_container_name:' ../config/{org_id}/chainmaker.yml | tail -n1 | awk '{print $2}'`
  docker_container_lists=(`docker ps -a | grep ${docker_go_container_name} | awk '{print $1}'`)
  for container_id in ${docker_container_lists[*]}
  do
    docker stop ${container_id}
    docker rm ${container_id}
  done
fi

echo "chainmaker is stopped"
