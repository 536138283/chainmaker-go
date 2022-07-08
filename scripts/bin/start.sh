#!/bin/bash
#
# Copyright (C) BABEC. All rights reserved.
# Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

export LD_LIBRARY_PATH=$(dirname $PWD)/lib:$LD_LIBRARY_PATH
export PATH=$(dirname $PWD)/lib:$PATH
export WASMER_BACKTRACE=1
ulimit -n 655360

function parse_yaml {
   local prefix=$2
   local s='[[:space:]]*' w='[a-zA-Z0-9_]*' fs=$(echo @|tr @ '\034')
   sed -ne "s|^\($s\):|\1|" \
        -e "s|^\($s\)\($w\)$s:$s[\"']\(.*\)[\"']$s\$|\1$fs\2$fs\3|p" \
        -e "s|^\($s\)\($w\)$s:$s\(.*\)$s\$|\1$fs\2$fs\3|p"  $1 |
   awk -F$fs '{
      indent = length($1)/2;
      vname[indent] = $2;
      for (i in vname) {if (i > indent) {delete vname[i]}}
      if (length($3) > 0) {
         vn=""; for (i=0; i<indent; i++) {vn=(vn)(vname[i])("_")}
         printf("%s%s%s=\"%s\"\n", "'$prefix'",vn, $2, $3);
      }
   }'
}

config_file="../config/{org_id}/chainmaker.yml"
# config_file="../../config/wx-org1-solo/chainmaker.yml"

# if clean existed container(can be -y/-f/force)
FORCE_CLEAN=$1
# if start vm go(can be -a/-alone)
START_WITHOUT_VM_GO=$2

eval $(parse_yaml "$config_file" "chainmaker_")


# if enable docker vm service and use unix domain socket, run a vm docker container
function start_docker_vm() {
  image_name="chainmakerofficial/chainmaker-vm-docker-go:v2.3.0"

  container_name=VM-GO-{org_id}
  echo "start docker vm service container: $container_name"
  #check container exists
  exist=$(docker ps -f name="$container_name" --format '{{.Names}}')
  if [ "$exist" ]; then
    echo "$container_name already RUNNING, please stop it first."
    exit 1
  fi

  exist=$(docker ps -a -f name="$container_name" --format '{{.Names}}')
  if [ "$exist" ]; then
    echo "$container_name already exists(STOPPED)"
    if [ "$FORCE_CLEAN" == "-f" ] || [ "$FORCE_CLEAN" == "force" ] || [ "$FORCE_CLEAN" == "-y" ]; then
      echo "remove it:"
      docker rm $container_name
    else
      read -r -p "remove it and start a new container, default: yes (y|n): " need_rm
      if [ "$need_rm" == "no" ] || [ "$need_rm" == "n" ]; then
        exit 0
      else
        docker rm $container_name
      fi
    fi
  fi

  # concat mount_path and log_path for container to mount
  mount_path=$chainmaker_vm_go_data_mount_path
  log_path=$chainmaker_vm_go_log_mount_path
  if [[ "${mount_path:0:1}" != "/" ]];then
    mount_path=$(pwd)/$mount_path
  fi
  if [[ "${log_path:0:1}" != "/" ]];then
    log_path=$(pwd)/$log_path
  fi

  mkdir -p "$mount_path"
  mkdir -p "$log_path"

  enable_vm_go=$chainmaker_vm_go_enable
  protocol=$chainmaker_vm_go_protocol
  vm_go_log_level=$chainmaker_vm_go_log_level
  runtime_server_port=$chainmaker_vm_go_runtime_server_port
  contract_engine_port=$chainmaker_vm_go_contract_engine_port
  rpc_timeout=$chainmaker_vm_go_dial_timeout
  rpc_max_send_size=$chainmaker_vm_go_max_send_msg_size
  rpc_max_recv_size=$chainmaker_vm_go_max_recv_msg_size
  log_in_console=$chainmaker_vm_go_log_in_console

  if [[ $enable_vm_go = "true" &&  $start_now != "false" ]]
  then

    if [[ $protocol = "uds" ]]
    then
      echo "docker vm protocol: unix domain socket"

      docker run -itd \
      -v "$mount_path":/mount \
      -v "$log_path":/log \
      -e CHAIN_RPC_PROTOCOL="0" \
      -e MAX_SEND_MSG_SIZE="$rpc_max_send_size" \
      -e MAX_RECV_MSG_SIZE="$rpc_max_recv_size" \
      -e MAX_CONN_TIMEOUT="$rpc_timeout" \
      -e DOCKERVM_CONTRACT_ENGINE_LOG_LEVEL="$vm_go_log_level" \
      -e DOCKERVM_SANDBOX_LOG_LEVEL="$vm_go_log_level" \
      -e DOCKERVM_LOG_IN_CONSOLE="$log_in_console" \
      --name VM-GO-{org_id} \
      --privileged $image_name
    else
      # $protocol = "tcp"
      echo "docker vm protocol: tcp"

        EXPOSE_PORT=$contract_engine_port

        docker run -itd \
        --net=host \
        -v "$mount_path":/mount \
        -v "$log_path":/log \
        -e CHAIN_RPC_PROTOCOL="1" \
        -e CHAIN_RPC_PORT="$contract_engine_port" \
        -e SANDBOX_RPC_PORT="$runtime_server_port" \
        -e MAX_SEND_MSG_SIZE="$rpc_max_send_size" \
        -e MAX_RECV_MSG_SIZE="$rpc_max_recv_size" \
        -e MAX_CONN_TIMEOUT="$rpc_timeout" \
        -e DOCKERVM_CONTRACT_ENGINE_LOG_LEVEL="$vm_go_log_level" \
        -e DOCKERVM_SANDBOX_LOG_LEVEL="$vm_go_log_level" \
        -e DOCKERVM_LOG_IN_CONSOLE="$log_in_console" \
        --name VM-GO-{org_id} \
        --privileged $image_name
    fi
  fi
  retval="$?"
  if [ $retval -ne 0 ]; then
    echo "Fail to run docker vm."
    exit 1
  fi

  echo "waiting for docker vm container to warm up..."
  sleep 5
}


pid=$(ps -ef | grep chainmaker | grep "\-c ../config/{org_id}/chainmaker.yml" | grep -v grep |  awk  '{print $2}')
if [ -z "${pid}" ];then

    # check if need to start docker vm service.
    enable_vm_go=$chainmaker_vm_enable_dockervm
    protocol=$chainmaker_vm_go_protocol
    if [[ $enable_vm_go == "true" &&  ("$START_WITHOUT_VM_GO" == "-s" ||  "$START_WITHOUT_VM_GO" == "start") ]]
    then
      start_docker_vm
    fi

    # start chainmaker
    #nohup ./chainmaker start -c ../config/{org_id}/chainmaker.yml > /dev/null 2>&1 &
    nohup ./chainmaker start -c ../config/{org_id}/chainmaker.yml > panic.log 2>&1 &
    echo "chainmaker is startting, pls check log..."
else
    echo "chainmaker is already started"
fi
