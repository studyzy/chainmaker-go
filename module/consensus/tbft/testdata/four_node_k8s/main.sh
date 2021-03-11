printHelp() {
    echo "Usage: "
    echo " main <mode> "
    echo "  <mode> - 'start'/'stop'"
    echo "    - 'start': start network"
    echo "    - 'stop': stop network"
}

create_configmap() {
    # tar crypto materials
    tar -zcf crypto.tgz ../../../../../config/ 
    # tar config
    tar -zcf config.tgz wx-org*
    # create configmap
    kubectl --insecure-skip-tls-verify create configmap conf --from-file=crypto.tgz --from-file=config.tgz
    rm -rf crypto.tgz config.tgz
}

start_network() {
    kubectl --insecure-skip-tls-verify apply -f deployment.yaml 
}

start() {
    create_configmap
    start_network
}

stop_network() {
    kubectl --insecure-skip-tls-verify delete -f deployment.yaml 
}

stop() {
    stop_network
    # delete configmap
    kubectl --insecure-skip-tls-verify delete configmap conf
}

MODE=$1
echo ${MODE}

if [ "${MODE}" == "start" ]; then
    start
elif [ "${MODE}" == "stop" ]; then
    stop
else
    printHelp
fi
