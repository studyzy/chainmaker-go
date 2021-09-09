
# how to use

step1. change `crypto_config[0].count=n` in file `chainmaker-cryptogen/config/crypto_config_template.yml`
<br> change `IMAGE` in file create_docker_conpose_yml.sh

step2. prepare 
<br>`cd chainmaker-go/script && ./prepare.sh 4 1`

step3. change and execute 
<br>`cp -r chainmaker-go/build/config chainmaker-go/script/docker/multi_node/`
<br>`cd chainmaker-go/script/docker/multi_node ` 
<br>`./create_docker_conpose_yml.sh 11301 12301 20 ./config 6`

step4. run docker with compose 
<br>`docker-compose -f docker-compose.yml up -d` 
<br>or `./start docker-compose.yml`

note: stop docker with compose
<br>`docker-compose -f docker-compose.yml down`
<br>or `./stop docker-compose.yml`