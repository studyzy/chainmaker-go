## 使用CMC工具测试合约
```shell script
cd chainmaker-go/tools/cmc
go mod download
go build

#拷贝sdk的配置文件和示例里cmc命令行需要使用的文件
cp -r ../sdk/testdata ./
mv testdata/crypto-config testdata/crypto-config-bak
cp -r ../../config/crypto-config/ testdata/crypto-config
```




### rust fact

```sh
# 创建合约
./cmc client contract user create \
--byte-code-path=/home/taifu/workspace/chainmaker-go/test/wasm/rust-fact-1.1.1.wasm \
--runtime-type=WASMER \
--admin-key-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.tls.key \
--admin-crt-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.tls.crt \
--client-crt-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.crt \
--client-key-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.key \
--org-id=wx-org1.chainmaker.org \
--contract-name=fact_004 \
--sdk-conf-path=./testdata/sdk_config.yml \
--version=1.0 \
--sync-result=true \

# 执行合约
./cmc client contract user invoke \
--contract-name=fact_004 \
--method=save \
--org-id=wx-org1.chainmaker.org \
--client-crt-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.crt \
--client-key-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.key \
--sdk-conf-path=./testdata/sdk_config.yml \
--sync-result=true \
--params="{\"time\":\"123\",\"file_hash\":\"2352B3523FB3F2B2FB2E254AA5B6\",\"file_name\":\"name.png\"}"

# 查询合约
./cmc client contract user get \
--contract-name=fact_004 \
--method=find_by_file_hash \
--sdk-conf-path=./testdata/sdk_config.yml \
--org-id=wx-org1.chainmaker.org \
--client-crt-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.crt \
--client-key-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.key \
--params="{\"time\":\"123\",\"file_hash\":\"2352B3523FB3F2B2FB2E254AA5B6\",\"file_name\":\"name.png\"}"
```



### go fact

```sh
# 创建合约
./cmc client contract user create \
--byte-code-path=/home/taifu/workspace/chainmaker-go/test/wasm/go-fact-1.1.1.wasm \
--runtime-type=GASM \
--admin-key-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.tls.key \
--admin-crt-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.tls.crt \
--client-crt-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.crt \
--client-key-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.key \
--org-id=wx-org1.chainmaker.org \
--contract-name=fact_003 \
--sdk-conf-path=./testdata/sdk_config.yml \
--version=1.0 \
--sync-result=true \

# 执行合约
./cmc client contract user invoke \
--contract-name=fact_003 \
--method=save \
--org-id=wx-org1.chainmaker.org \
--client-crt-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.crt \
--client-key-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.key \
--sdk-conf-path=./testdata/sdk_config.yml \
--sync-result=true \
--params="{\"time\":\"123\",\"file_hash\":\"2352B3523FB3F2B2FB2E254AA5B6\",\"file_name\":\"name.png\"}"

# 查询合约
./cmc client contract user get \
--contract-name=fact_003 \
--method=find_by_file_hash \
--sdk-conf-path=./testdata/sdk_config.yml \
--org-id=wx-org1.chainmaker.org \
--client-crt-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.crt \
--client-key-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.key \
--params="{\"time\":\"123\",\"file_hash\":\"2352B3523FB3F2B2FB2E254AA5B6\",\"file_name\":\"name.png\"}"
```

### go counter

```sh
# 创建合约
./cmc client contract user create \
--byte-code-path=/home/taifu/workspace/chainmaker-go/test/wasm/go-counter-1.1.1.wasm \
--runtime-type=GASM \
--admin-key-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.tls.key \
--admin-crt-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.tls.crt \
--client-crt-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.crt \
--client-key-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.key \
--org-id=wx-org1.chainmaker.org \
--contract-name=counter001 \
--sdk-conf-path=./testdata/sdk_config.yml \
--version=1.0 \
--sync-result=true \

# 执行合约
./cmc client contract user invoke \
--contract-name=counter001 \
--method=increase \
--org-id=wx-org1.chainmaker.org \
--client-crt-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.crt \
--client-key-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.key \
--sdk-conf-path=./testdata/sdk_config.yml \
--sync-result=true \

# 查询合约
./cmc client contract user get \
--contract-name=counter001 \
--method=query \
--sdk-conf-path=./testdata/sdk_config.yml \
--org-id=wx-org1.chainmaker.org \
--client-crt-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.crt \
--client-key-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.key \

```



### rust counter

```sh
# 创建合约
./cmc client contract user create \
--byte-code-path=/home/taifu/workspace/chainmaker-go/test/wasm/rust-counter-1.1.1.wasm \
--runtime-type=WASMER \
--admin-key-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.tls.key \
--admin-crt-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.tls.crt \
--client-crt-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.crt \
--client-key-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.key \
--org-id=wx-org1.chainmaker.org \
--contract-name=counter002 \
--sdk-conf-path=./testdata/sdk_config.yml \
--version=1.0 \
--sync-result=true \

# 执行合约
./cmc client contract user invoke \
--contract-name=counter002 \
--method=increase \
--org-id=wx-org1.chainmaker.org \
--client-crt-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.crt \
--client-key-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.key \
--sdk-conf-path=./testdata/sdk_config.yml \
--sync-result=true \

# 查询合约
./cmc client contract user get \
--contract-name=counter002 \
--method=query \
--sdk-conf-path=./testdata/sdk_config.yml \
--org-id=wx-org1.chainmaker.org \
--client-crt-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.crt \
--client-key-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.key \

```
