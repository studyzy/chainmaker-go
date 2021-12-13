
# how to use

step1. change `crypto_config[0].count=n` in file `chainmaker-cryptogen/config/crypto_config_template.yml`
<br> change `IMAGE` in file create_docker_compose_yml.sh

`crypto_config_template.yml`
```yaml
crypto_config:
  - domain: chainmaker.org
    host_name: wx-org
    count: 5                # change this
```
`create_docker_compose_yml.sh`
```yaml
P2P_PORT=$1
RPC_PORT=$2
NODE_COUNT=$3
CONFIG_DIR=$4
SERVER_COUNT=$5
IMAGE="chainmakerofficial/chainmaker:v2.1.0" # change this
```

step2. prepare 
```sh
cd chainmaker-go/script 
./prepare.sh 4 1
```

step3. change and execute 
```sh
cd chainmaker-go
cp -r build/config script/docker/multi_node/
cd script/docker/multi_node
./create_docker_compose_yml.sh 11301 12301 20 ./config 6
```
step4. run docker with compose 
```sh
docker-compose -f docker-compose.yml up -d
# or 
./start docker-compose.yml
```

note: stop docker with compose
```sh
docker-compose -f docker-compose.yml down
# or
./stop docker-compose.yml
```
