P2P_PORT_PREFIX=$1
RPC_PORT_PREFIX=$2
NODE_COUNT=$3
CONFIG_DIR=$4
SERVER_COUNT=$5
IMAGE="chainmakerofficial/chainmaker:v1.2.4"

CURRENT_PATH=$(pwd)
CONFIG_FILE="docker-compose"
TEMPLATE_FILE="tpl_docker-compose_services.yml"

function show_help() {
    echo "Usage:  "
    echo "  create_yml.sh p2p_port_prefix(default:11300) rpc_port_prefix(default:12300) node_count config_dir(relative current dir or absolutely dir) server_count(default:1)"
    echo "    eg: ./create_docker_conpose_yml.sh 11300 12300 20 ../../../build/config 10"
    echo "    eg: ./create_docker_conpose_yml.sh 11300 12300 20 /mnt/d/develop/workspace/go/chainmaker-go/build/config 10"
}
if [ ! $# -eq 2 ] && [ ! $# -eq 3 ] && [ ! $# -eq 4 ] && [ ! $# -eq 5 ]; then
    echo "invalid params"
    show_help
    exit 1
fi

function xsed() {
    system=$(uname)

    if [ "${system}" = "Linux" ]; then
        sed -i "$@"
    else
        sed -i '' "$@"
    fi
}

function check_params() {
    if  [[ ! -n $P2P_PORT_PREFIX ]] ;then
        show_help
        exit 1
    fi

    if  [ ${P2P_PORT_PREFIX} -ge 60000 ] || [ ${P2P_PORT_PREFIX} -le 10000 ];then
        echo "p2p_port_prefix should >=10000 && <=60000"
        show_help
        exit 1
    fi

    if  [[ ! -n $RPC_PORT_PREFIX ]] ;then
        show_help
        exit 1
    fi

    if  [ ${RPC_PORT_PREFIX} -ge 60000 ] || [ ${RPC_PORT_PREFIX} -le 10000 ];then
        echo "rpc_port_prefix should >=10000 && <=60000"
        show_help
        exit 1
    fi

    if  [[ ! -n $NODE_COUNT ]] ;then
        show_help
        exit 1
    fi

    if  [[ ! -n $SERVER_COUNT ]] ;then
        SERVER_COUNT=1
    fi

    if  [[ ! -n $CONFIG_DIR ]] ;then
        CONFIG_DIR="../../../build/config"
    fi
}

function xsed() {
    system=$(uname)

    if [ "${system}" = "Linux" ]; then
        sed -i "$@"
    else
        sed -i '' "$@"
    fi
}

function generate_yml_file() {
  tmp_file="${TEMPLATE_FILE}.tmp"
  current_config_file=""
  for ((k = 1; k < $NODE_COUNT + 1; k = k + 1)); do
    surplus=$(( $(($k - 1)) % $SERVER_COUNT ))
    if [ $surplus -eq 0 ]; then
      current_config_file="${CONFIG_FILE}${k}.yml"
      rm -rf $current_config_file
      echo "generate $current_config_file"
      echo -e "version: '3'\n"  >> $current_config_file
      echo -e "services:"  >> $current_config_file
    fi
    P2P_PORT_PREFIX=$(($P2P_PORT_PREFIX+1))
    RPC_PORT_PREFIX=$(($RPC_PORT_PREFIX+1))
    if [ ! -f $tmp_file ];then
      cp $TEMPLATE_FILE $tmp_file
    fi
    node_config_dir="${CONFIG_DIR}/node${k}"
    xsed "s%{config_dir}%${node_config_dir}%g" $tmp_file
    xsed "s%{id}%${k}%g" $tmp_file
    xsed "s%{image}%${IMAGE}%g" $tmp_file
    xsed "s%{rpc_port}%${RPC_PORT_PREFIX}%g" $tmp_file
    xsed "s%{p2p_port}%${P2P_PORT_PREFIX}%g" $tmp_file
    cat $tmp_file >> $current_config_file
    rm -f $tmp_file
  done

}
check_params
generate_yml_file
