#
# Copyright (C) BABEC. All rights reserved.
# Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

pid=`ps -ef | grep chainmaker | grep "\-c ../config/{org_id}/chainmaker.yml" | grep -v grep |  awk  '{print $2}'`
if [ ! -z ${pid} ];then
    kill -9 $pid
fi

docker_go_container_name=`grep -A3 'docker:' ../config/{org_id}/chainmaker.yml | tail -n1 | awk '{print $2}'`
docker_container_exist=`docker ps -a | grep ${docker_go_container_name}`

if  [[ -n $docker_container_exist ]] ;then
    docker stop ${docker_go_container_name}
    docker rm ${docker_go_container_name}
fi

echo "chainmaker is stopped"
