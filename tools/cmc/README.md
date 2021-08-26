# 命令行工具
<span id="section_cmc"></span>

## 简介
cmc`(ChainMaker Client)`是ChainMaker提供的命令行工具，用于和ChainMaker链进行交互以及生成证书等功能。cmc基于go语言编写，通过使用ChainMaker的go语言sdk（使用grpc协议）达到和ChainMaker链进行交互的目的。<br>
cmc的详细日志请查看`./sdk.log`

## 编译&配置

cmc工具的编译方式如下：

```sh
# 创建工作目录
$ mkdir ~/chainmaker
# 编译cmc
$ cd ~/chainmaker
$ git clone -b v2.0.0 https://git.chainmaker.org.cn/chainmaker/chainmaker-go.git
$ cd chainmaker-go/tools/cmc
$ go mod download
$ go build
# 启动测试链
$ cd ~/chainmaker/chainmaker-go/tools
$ git clone -b master https://git.chainmaker.org.cn/chainmaker/chainmaker-cryptogen.git
$ cd chainmaker-cryptogen && make
$ cd ~/chainmaker/chainmaker-go/scripts
$ ./prepare.sh 4 1 # 这里以4组织，每组织1共识节点为例。使用默认参数，一直回车。
$ ./build_release.sh
$ ./cluster_quick_start.sh normal
# 配置测试数据
$ cd ~/chainmaker
$ cp -rf ./chainmaker-go/build/crypto-config ./chainmaker-go/tools/cmc/testdata/ # 使用chainmaker-cryptogen生成的测试链的证书
# 配置完成
$ cd ~/chainmaker/chainmaker-go/tools/cmc
$ ./cmc --help
```

<span id="sdkConfig"></span>
## 自定义配置

cmc 依赖 sdk-go 配置文件。<br>
编译&配置 步骤使用的是 [SDK配置模版](https://git.chainmaker.org.cn/chainmaker/sdk-go/-/blob/master/testdata/sdk_config.yml) <br>
可通过修改 ~/chainmaker/chainmaker-go/tools/cmc/testdata/sdk_config.yml 实现自定义配置。<br>
比如 `user_key_file_path`,`user_crt_file_path`,`user_sign_key_file_path`,`user_sign_crt_file_path`<br>
这四个参数可设置为普通用户或admin用户的证书/私钥路径。设置后cmc将会以对应用户身份与链建立连接。<br>
其他详细配置项请参看 ~/chainmaker/chainmaker-go/tools/cmc/testdata/sdk_config.yml 中的注解。<br>

## 功能

cmc提供功能如下:

- [私钥管理](#keyManage)：私钥生成功能
- [证书管理](#certManage)：包括生成ca证书、生成crl列表、生成csr、颁发证书、根据证书获取节点Id等功能
- [交易功能](#sendRequest)：主要包括链管理、用户合约发布、升级、吊销、冻结、调用、查询等功能
- [查询链上数据](#queryOnChainData)：查询链上block和transaction
- [链配置](#chainConfig)：查询及更新链配置
- [归档&恢复功能](#archive)：将链上数据转移到独立存储上，归档后的数据具备可查询、可恢复到链上的特性

### 示例

<span id="keyManage"></span>
#### 私钥管理

  生成私钥, 目前支持的算法有 SM2 ECC_P256 未来将支持更多算法。

  **参数说明**：

  ```sh
  $ ./cmc key gen -h 
  Private key generate
  Usage:
    cmc key gen [flags]
  
  Flags:
    -a, --algo string   specify key generate algorithm
    -h, --help          help for gen
    -n, --name string   specify storage name
    -p, --path string   specify storage path
  ```

  **示例：**

  ```sh
  $ ./cmc key gen -a ECC_P256 -n ca.key -p ./
  ```

<span id="certManage"></span>
#### 证书管理
  - 生成ca证书

    生成ca证书之前需要先生成ca的私钥文件，然后在生成ca证书时使用-k参数指定生成的私钥文件路径

    **参数说明**

    ```sh
    $ ./cmc cert ca -h 
    Create certificate authority crtificate
    Usage:
      cmc cert ca [flags]

    Flags:
      -c, --cn string         specify common name
      -H, --hash string       specify hash algorithm
      -h, --help              help for ca
      -k, --key-path string   specify key path
      -n, --name string       specify storage name
      -o, --org string        specify organization
      -O, --ou string         specify organizational unit
      -p, --path string       specify storage path
    ```

    **示例**

    ```sh
    $ ./cmc cert ca -c wx-org1.chainmaker.org -k ca.key -H sha256 --ou root-ca -n ca.crt -p ./ --org wx-org1
    ```

  - 生成crl列表

    crl列表用于撤消证书请求，首先将要撤消的证书生成一个crl列表，然后再发送请求到链上

    **参数说明**

    ```sh
    $ ./cmc cert crl -h  
    create cert crl
    
    Usage:
      cmc cert crl [flags]
    
    Flags:
      -C, --ca-cert-path string   specify certificate authority certificate path
      -K, --ca-key-path string    specify certificate authority key path
          --crl-path string       specify crl file path
          --crt-path string       specify crt file path
      -h, --help                  help for crl
    ```

    **示例**

    ```sh
    $ ./cmc cert crl -C ./ca.crt -K ca.key --crl-path=./client1.crl --crt-path=../sdk/testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.crt
    ```
    
  - 生成csr文件

    **参数说明**
    
    ```sh
    $ ./cmc cert csr -h
    Create certificate request
    
    Usage:
      cmc cert csr [flags]
    
    Flags:
      -c, --cn string         specify common name
      -h, --help              help for csr
      -k, --key-path string   specify key path
      -n, --name string       specify storage name
      -o, --org string        specify organization
      -O, --ou string         specify organizational unit
      -p, --path string       specify storage path
    ```
    
    **示例**
    
    ```sh
    $ ./cmc cert csr -c wx-org1.chainmaker.org -k ../sdk/testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.key -n client1.csr -p ./ --ou root-ca -o wx-org1
    ```
    
    

  - 颁发证书

    **参数说明**

    ```sh
    $ ./cmc cert issue -h
    Issue certificate
    
    Usage:
      cmc cert issue [flags]
    
    Flags:
      -C, --ca-cert-path string   specify certificate authority certificate path
      -K, --ca-key-path string    specify certificate authority key path
      -r, --csr-path string       specify certificate request path
      -H, --hash string           specify hash algorithm
      -h, --help                  help for issue
          --is-ca                 specify is certificate authority
      -n, --name string           specify storage name
      -p, --path string           specify storage path
    ```

    **示例**

    ```sh
    $ ./cmc cert issue -C ca.crt -K ca.key -r client1.csr -H sha256 -n client1.crt -p ./
    ```

  - 根据证书获取节点Id

    **参数说明**

    ```sh
    $ ./cmc cert nid -h
    Get node id of node cert
    
    Usage:
      cmc cert nid [flags]
    
    Flags:
      -h, --help                    help for nid
          --node-cert-path string   specify node cert path
    ```

    **示例**

    ```sh
    $ ./cmc cert nid --node-cert-path=../sdk/testdata/crypto-config/wx-org1.chainmaker.org/node/consensus1/consensus1.tls.crt
    node id : QmcQHCuAXaFkbcsPUj7e37hXXfZ9DdN7bozseo5oX4qiC4
    ```

<span id="sendRequest"></span>
#### 交易功能
##### 用户合约
  cmc的交易功能用来发送交易和链进行交互，主要参数说明如下：

  ```sh
    sdk配置文件flag
    --sdk-conf-path：指定cmc使用sdk的配置文件路径

    admin签名者flags，此类flag的顺序及个数必须保持一致，且至少传入一个admin
    --admin-crt-file-paths: admin签名者的tls crt文件的路径列表. 单签模式下只需要填写一个即可, 离线多签模式下多个需要用逗号分割
        比如 ./wx-org1.chainmaker.org/admin1.tls.crt,./wx-org2.chainmaker.org/admin1.tls.crt
    --admin-key-file-paths: admin签名者的tls key文件的路径列表. 单签模式下只需要填写一个即可, 离线多签模式下多个需要用逗号分割
        比如 ./wx-org1.chainmaker.org/admin1.tls.key,./wx-org2.chainmaker.org/admin1.tls.key

    覆盖sdk配置flags，不传则使用sdk配置，如果想覆盖sdk的配置，则以下六个flag都必填
    --org-id: 指定发送交易的用户所属的组织Id, 会覆盖sdk配置文件读取的配置
    --chain-id: 指定链Id, 会覆盖sdk配置文件读取的配置
    --user-tlscrt-file-path: 指定发送交易的用户tls证书文件路径, 会覆盖sdk配置文件读取的配置
    --user-tlskey-file-path: 指定发送交易的用户tls私钥路径, 会覆盖sdk配置文件读取的配置
    --user-signcrt-file-path: 指定发送交易的用户sign证书文件路径, 会覆盖sdk配置文件读取的配置
    --user-signkey-file-path: 指定发送交易的用户sign私钥路径, 会覆盖sdk配置文件读取的配置

    其他flags
    --byte-code-path：指定合约的wasm文件路径
    --contract-name：指定合约名称
    --method：指定调用的合约方法名称
    --runtime-type：指定合约执行虚拟机环境，包含：GASM、EVM、WASMER、WXVM、NATIVE
    --version：指定合约的版本号，在发布和升级合约时使用
    --sync-result：指定是否同步等待交易执行结果，默认为false，如果设置为true，在发送完交易后会主动查询交易执行结果
    --params：指定发布合约或调用合约时的参数信息
    --concurrency：指定调用合约并发的go routine，用于压力测试
    --total-count-per-goroutine：指定单个go routine发送的交易数量，用于压力测试，和--concurrency配合使用
    --block-height：指定区块高度
    --tx-id：指定交易Id
    --with-rw-set：指定获取区块时是否附带读写集，默认是false
    --abi-file-path：调用evm合约时需要指定被调用合约的abi文件路径，如：--abi-file-path=./testdata/balance-evm-demo/ledger_balance.abi
  ```

  - 创建wasm合约
  
    ```sh
    $ ./cmc client contract user create \
    --contract-name=fact \
    --runtime-type=WASMER \
    --byte-code-path=./testdata/claim-wasm-demo/rust-fact-2.0.0.wasm \
    --version=1.0 \
    --sdk-conf-path=./testdata/sdk_config.yml \
    --admin-key-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.tls.key,./testdata/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.tls.key,./testdata/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.tls.key \
    --admin-crt-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.tls.crt,./testdata/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.tls.crt,./testdata/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.tls.crt \
    --sync-result=true \
    --params="{}"
    ```
    
      > 如下返回表示成功：
      >
      > response: message:"OK" contract_result:<result:"\n\004fact\022\0031.0\030\002*<\n\026wx-org1.chainmaker.org\020\001\032 F]\334,\005O\200\272\353\213\274\375nT\026%K\r\314\362\361\253X\356*2\377\216\250kh\031" message:"OK" > tx_id:"991a1c00369e4b76853dadf410182bcdfc86062f8cf1478f93482ba9000191d7"



      注：智能合约编写参见：[智能合约开发](./智能合约.md)

  - 创建evm合约
  
    ```sh
    $ ./cmc client contract user create \
    --contract-name=balance001 \
    --runtime-type=EVM \
    --byte-code-path=./testdata/balance-evm-demo/ledger_balance.bin \
    --version=1.0 \
    --sdk-conf-path=./testdata/sdk_config.yml \
    --admin-key-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.tls.key,./testdata/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.tls.key,./testdata/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.tls.key \
    --admin-crt-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.tls.crt,./testdata/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.tls.crt,./testdata/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.tls.crt \
    --sync-result=true
    ```
    
      > 如下返回表示成功：
      >
      > EVM contract name in hex: 532c238cec7071ce8655aba07e50f9fb16f72ca1 
      > response: message:"OK" contract_result:<result:"\n(532c238cec7071ce8655aba07e50f9fb16f72ca1\022\0031.0\030\005*<\n\026wx-org1.chainmaker.org\020\001\032 F]\334,\005O\200\272\353\213\274\375nT\026%K\r\314\362\361\253X\356*2\377\216\250kh\031" message:"OK" > tx_id:"e2af1241ff464d47b869a69ce8a615df50da57d3faff4754ad6e45b9f914b938"


      注：智能合约编写参见：[智能合约开发](./智能合约.md)

  - 调用wasm合约
  
    ```sh
    $ ./cmc client contract user invoke \
    --contract-name=fact \
    --method=save \
    --sdk-conf-path=./testdata/sdk_config.yml \
    --params="{\"faile_name\":\"name007\",\"file_hash\":\"ab3456df5799b87c77e7f88\",\"time\":\"6543234\"}" \
    --sync-result=true
    ```
  
      > 如下返回表示成功：
      >
      > INVOKE contract resp, [code:0]/[msg:OK]/[contractResult:gas_used:12964572 contract_event:<topic:"topic_vx" tx_id:"7c9e98befbb64cec916765d760d4def5aa26f8bac78d419c9018b8d220e7f041" contract_name:"fact" contract_version:"1.0" event_data:"ab3456df5799b87c77e7f88" event_data:"" event_data:"6543234" > ]/[txId:7c9e98befbb64cec916765d760d4def5aa26f8bac78d419c9018b8d220e7f041]

  - 调用evm合约
    
    evm的 --params 是一个数组json格式。如下updateBalance有两个形参第一个是uint256类型，第二个是address类型。<br>
    10000对应第一个形参uint256的具体值，0xa166c92f4c8118905ad984919dc683a7bdb295c1对应第二个形参address的具体值。
  
    ```sh
    $ ./cmc client contract user invoke \
    --contract-name=balance001 \
    --method=updateBalance \
    --sdk-conf-path=./testdata/sdk_config.yml \
    --params="[{\"uint256\": \"10000\"},{\"address\": \"0xa166c92f4c8118905ad984919dc683a7bdb295c1\"}]" \
    --sync-result=true \
    --abi-file-path=./testdata/balance-evm-demo/ledger_balance.abi
    ```
  
      > 如下返回表示成功：
      >
      > EVM contract name in hex: 532c238cec7071ce8655aba07e50f9fb16f72ca1
      > INVOKE contract resp, [code:0]/[msg:OK]/[contractResult:result:"[]" gas_used:5888 ]/[txId:4f25f47518b14e6b92ce184dc6ed84f594341567050b4023ae1686a47e2e22ec]


  - 查询合约
  
    ```sh
    $ ./cmc client contract user get \
    --contract-name=fact \
    --method=find_by_file_hash \
    --sdk-conf-path=./testdata/sdk_config.yml \
    --params="{\"file_hash\":\"ab3456df5799b87c77e7f88\"}"
    ```
  
      > 如下返回表示成功：
      >
      >  QUERY contract resp: message:"SUCCESS" contract_result:<result:"{\"file_hash\":\"ab3456df5799b87c77e7f88\",\"file_name\":\"\",\"time\":\"6543234\"}" gas_used:24354672 > tx_id:"25716b955ebd4a258c4bd6b6f682f1341dfe97e4bd18495c864992f1618a2003"

  - 升级合约
  
    ```sh
    $ ./cmc client contract user upgrade \
    --contract-name=fact \
    --runtime-type=WASMER \
    --byte-code-path=./testdata/claim-wasm-demo/rust-fact-2.0.0.wasm \
    --version=2.0 \
    --sdk-conf-path=./testdata/sdk_config.yml \
    --admin-key-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.tls.key,./testdata/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.tls.key,./testdata/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.tls.key \
    --admin-crt-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.tls.crt,./testdata/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.tls.crt,./testdata/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.tls.crt \
    --org-id=wx-org1.chainmaker.org \
    --sync-result=true \
    --params="{}"
    ```
  
      > 如下返回表示成功：其中result结果为用户自定义，每个合约可能不一样，也可能没有。
      >
      > upgrade user contract params:[]
      > upgrade contract resp: message:"OK" contract_result:<result:"\n\004fact\022\0032.0\030\002*<\n\026wx-org1.chainmaker.org\020\001\032 F]\334,\005O\200\272\353\213\274\375nT\026%K\r\314\362\361\253X\356*2\377\216\250kh\031" message:"OK" > tx_id:"d89df9fcd87f4071972fdabdf3003a349250a94893fb43899eac4d68e7855d52" 

  - 冻结合约
  
    ```sh
    $ ./cmc client contract user freeze \
    --contract-name=fact \
    --sdk-conf-path=./testdata/sdk_config.yml \
    --admin-key-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.tls.key,./testdata/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.tls.key,./testdata/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.tls.key \
    --admin-crt-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.tls.crt,./testdata/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.tls.crt,./testdata/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.tls.crt \
    --org-id=wx-org1.chainmaker.org \
    --sync-result=true
    ```
  
    > 如下返回表示成功：冻结后的合约再去执行查询、调用合约则会失败
    >
    > freeze contract resp: message:"OK" contract_result:<result:"{\"name\":\"fact\",\"version\":\"3.0\",\"runtime_type\":2,\"status\":1,\"creator\":{\"org_id\":\"wx-org1.chainmaker.org\",\"member_type\":1,\"member_info\":\"Rl3cLAVPgLrri7z9blQWJUsNzPLxq1juKjL/jqhraBk=\"}}" message:"OK" > tx_id:"09841775173548ad9a8a39e2987a4f5115d59d50dd3448e8b09a83624dee5367"

  - 解冻合约
  
    ```sh
    $ ./cmc client contract user unfreeze \
    --contract-name=fact \
    --sdk-conf-path=./testdata/sdk_config.yml \
    --admin-key-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.tls.key,./testdata/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.tls.key,./testdata/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.tls.key \
    --admin-crt-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.tls.crt,./testdata/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.tls.crt,./testdata/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.tls.crt \
    --org-id=wx-org1.chainmaker.org \
    --sync-result=true
    ```

    > 如下返回表示成功：解冻后的合约可正常使用
    >
    > unfreeze contract resp: message:"OK" contract_result:<result:"{\"name\":\"fact\",\"version\":\"3.0\",\"runtime_type\":2,\"creator\":{\"org_id\":\"wx-org1.chainmaker.org\",\"member_type\":1,\"member_info\":\"Rl3cLAVPgLrri7z9blQWJUsNzPLxq1juKjL/jqhraBk=\"}}" message:"OK" > tx_id:"fccf024450c140dea999cc46ad24d381a679ce2142bd48b2a829abcd4f099866"

  - 吊销合约
  
    ```sh
    $ ./cmc client contract user revoke \
    --contract-name=fact \
    --sdk-conf-path=./testdata/sdk_config.yml \
    --admin-key-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.tls.key,./testdata/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.tls.key,./testdata/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.tls.key \
    --admin-crt-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.tls.crt,./testdata/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.tls.crt,./testdata/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.tls.crt \
    --org-id=wx-org1.chainmaker.org \
    --sync-result=true
    ```
  
    > 如下返回表示成功：吊销合约后，不可恢复，且不能对该合约执行任何操作，包括查询。
    > 
    > revoke contract resp: message:"OK" contract_result:<result:"{\"name\":\"fact\",\"version\":\"3.0\",\"runtime_type\":2,\"status\":2,\"creator\":{\"org_id\":\"wx-org1.chainmaker.org\",\"member_type\":1,\"member_info\":\"Rl3cLAVPgLrri7z9blQWJUsNzPLxq1juKjL/jqhraBk=\"}}" message:"OK" > tx_id:"d971b57cf12c46ff8fe0d4f5897634c644fb802998f44360bb130f27ff54a10a"

##### 系统合约
###### DPoS 计算用户地址
<span id="chainConfig.addrFromCert"></span>
  - 用户证书计算出用户的地址

    ```sh
    $ ./cmc cert userAddr --cert-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.crt
    ```

###### DPoS-ERC20 系统合约

<span id="chainConfig.dposMint"></span>
  - 增发Token

    ```sh
    $ ./cmc client contract system mint \
    --amount=100000000 \
    --address=ADZTrzVF9SuvQqmn9YTAJiwnLCnXonMTj6Bq1HRiwVnR \
    --sdk-conf-path=./testdata/sdk_config.yml \
    --org-id=wx-org1.chainmaker.org \
    --chain-id=chain1 \
    --user-tlscrt-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.crt \
    --user-tlskey-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.key \
    --user-signcrt-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.crt \
    --user-signkey-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.key \
    --sync-result=true
    ```

<span id="chainConfig.dposTransfer"></span>

  - 转账

    ```sh
    $ ./cmc client contract system transfer \
    --amount=100000000 \
    --address=ADZTrzVF9SuvQqmn9YTAJiwnLCnXonMTj6Bq1HRiwVnR \
    --sdk-conf-path=./testdata/sdk_config.yml \
    --org-id=wx-org1.chainmaker.org \
    --chain-id=chain1 \
    --user-tlscrt-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.crt \
    --user-tlskey-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.key \
    --user-signcrt-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.crt \
    --user-signkey-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.key
    ```

<span id="chainConfig.dposBalanceOf"></span>
  - 查询余额

    ```sh
    $ ./cmc client contract system balance-of \
    --address=ADZTrzVF9SuvQqmn9YTAJiwnLCnXonMTj6Bq1HRiwVnR \
    --sdk-conf-path=./testdata/sdk_config.yml \
    --org-id=wx-org1.chainmaker.org \
    --chain-id=chain1 \
    --user-tlscrt-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.crt \
    --user-tlskey-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.key \
    --user-signcrt-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.crt \
    --user-signkey-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.key
    ```

<span id="chainConfig.dposOwner"></span>
  - 查询合约管理地址

    ```sh
    $ ./cmc client contract system owner \
    --sdk-conf-path=./testdata/sdk_config.yml \
    --org-id=wx-org1.chainmaker.org \
    --chain-id=chain1 \
    --user-tlscrt-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.crt \
    --user-tlskey-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.key \
    --user-signcrt-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.crt \
    --user-signkey-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.key
    ```

<span id="chainConfig.dposDecimals"></span>
  - 查询ERC20合约的精度

    ```sh
    $ ./cmc client contract system decimals \
    --sdk-conf-path=./testdata/sdk_config.yml \
    --org-id=wx-org1.chainmaker.org \
    --chain-id=chain1 \
    --user-tlscrt-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.crt \
    --user-tlskey-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.key \
    --user-signcrt-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.crt \
    --user-signkey-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.key
    ```

<span id="chainConfig.dposTotalAmount"></span>
  - 查询 Token 总供应量

    ```sh
    $ ./cmc client contract system total \
    --sdk-conf-path=./testdata/sdk_config.yml \
    --org-id=wx-org1.chainmaker.org \
    --chain-id=chain1 \
    --user-tlscrt-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.crt \
    --user-tlskey-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.key \
    --user-signcrt-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.crt \
    --user-signkey-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.key
    ```

###### DPoS-Stake 系统合约

<span id="chainConfig.dposCandidates"></span>
  - 查询所有的候选人

    ```sh
    $ ./cmc client contract system all-candidates \
    --sdk-conf-path=./testdata/sdk_config.yml \
    --org-id=wx-org1.chainmaker.org \
    --chain-id=chain1 \
    --user-tlscrt-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.crt \
    --user-tlskey-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.key \
    --user-signcrt-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.crt \
    --user-signkey-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.key
    ```

<span id="chainConfig.dposValidatorInfo"></span>

  - 查询指定验证人的信息

    ```sh
    $ ./cmc client contract system get-validator \
    --address=ADZTrzVF9SuvQqmn9YTAJiwnLCnXonMTj6Bq1HRiwVnR \
    --sdk-conf-path=./testdata/sdk_config.yml \
    --org-id=wx-org1.chainmaker.org \
    --chain-id=chain1 \
    --user-tlscrt-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.crt \
    --user-tlskey-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.key \
    --user-signcrt-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.crt \
    --user-signkey-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.key
    ```

<span id="chainConfig.dposDelegate"></span>
  - 抵押权益到验证人

    ```sh
    $ ./cmc client contract system delegate \
    --address=ADZTrzVF9SuvQqmn9YTAJiwnLCnXonMTj6Bq1HRiwVnR \
    --amount=100000000 \
    --sdk-conf-path=./testdata/sdk_config.yml \
    --org-id=wx-org5.chainmaker.org \
    --chain-id=chain1 \
    --user-tlscrt-file-path=./testdata/crypto-config/wx-org5.chainmaker.org/user/client1/client1.tls.crt \
    --user-tlskey-file-path=./testdata/crypto-config/wx-org5.chainmaker.org/user/client1/client1.tls.key \
    --user-signcrt-file-path=./testdata/crypto-config/wx-org5.chainmaker.org/user/client1/client1.sign.crt \
    --user-signkey-file-path=./testdata/crypto-config/wx-org5.chainmaker.org/user/client1/client1.sign.key \
    --sync-result=true
    ```

<span id="chainConfig.dposUserDelegations"></span>
  - 查询指定地址的抵押信息

    ```sh
    $ ./cmc client contract system get-delegations-by-address \
    --address=ADZTrzVF9SuvQqmn9YTAJiwnLCnXonMTj6Bq1HRiwVnR \
    --sdk-conf-path=./testdata/sdk_config.yml \
    --org-id=wx-org1.chainmaker.org \
    --chain-id=chain1 \
    --user-tlscrt-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.crt \
    --user-tlskey-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.key \
    --user-signcrt-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.crt \
    --user-signkey-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.key
    ```

<span id="chainConfig.dposUserDelagationWithValidator"></span>
  - 查询指定地址的抵押信息

    ```sh
    $ ./cmc client contract system get-user-delegation-by-validator \
    --delegator=ADZTrzVF9SuvQqmn9YTAJiwnLCnXonMTj6Bq1HRiwVnR \
    --validator=ADZTrzVF9SuvQqmn9YTAJiwnLCnXonMTj6Bq1HRiwVnR \
    --sdk-conf-path=./testdata/sdk_config.yml \
    --org-id=wx-org1.chainmaker.org \
    --chain-id=chain1 \
    --user-tlscrt-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.crt \
    --user-tlskey-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.key \
    --user-signcrt-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.crt \
    --user-signkey-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.key
    ```

<span id="chainConfig.dposUndelegate"></span>
  - 从验证人解除抵押的权益

    ```sh
    $ ./cmc client contract system undelegate \
    --address=ADZTrzVF9SuvQqmn9YTAJiwnLCnXonMTj6Bq1HRiwVnR \
    --amount=100000000 \
    --sdk-conf-path=./testdata/sdk_config.yml \
    --org-id=wx-org5.chainmaker.org \
    --chain-id=chain1 \
    --user-tlscrt-file-path=./testdata/crypto-config/wx-org5.chainmaker.org/user/client1/client1.tls.crt \
    --user-tlskey-file-path=./testdata/crypto-config/wx-org5.chainmaker.org/user/client1/client1.tls.key \
    --user-signcrt-file-path=./testdata/crypto-config/wx-org5.chainmaker.org/user/client1/client1.sign.crt \
    --user-signkey-file-path=./testdata/crypto-config/wx-org5.chainmaker.org/user/client1/client1.sign.key \
    --sync-result=true
    ```

<span id="chainConfig.dposEpochInfoByID"></span>

  - 查询指定世代信息

    ```sh
    $ ./cmc client contract system read-epoch-by-id \
    --epoch-id=1 \
    --sdk-conf-path=./testdata/sdk_config.yml \
    --org-id=wx-org1.chainmaker.org \
    --chain-id=chain1 \
    --user-tlscrt-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.crt \
    --user-tlskey-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.key \
    --user-signcrt-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.crt \
    --user-signkey-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.key
    ```

<span id="chainConfig.dposCurrEpochInfo"></span>
  - 查询当前世代信息

    ```sh
    $ ./cmc client contract system read-latest-epoch \
    --sdk-conf-path=./testdata/sdk_config.yml \
    --org-id=wx-org1.chainmaker.org \
    --chain-id=chain1 \
    --user-tlscrt-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.crt \
    --user-tlskey-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.key \
    --user-signcrt-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.crt \
    --user-signkey-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.key
    ```

<span id="chainConfig.dposSetNodeID"></span>
  - Stake合约中设置验证人的NodeID

    ```sh
    $ ./cmc client contract system set-node-id \
    --node-id="QmWwNupMzs2GWyPXaUK3BvgvuZN74qxyz3rHaGioWDLX3D" \
    --sdk-conf-path=./testdata/sdk_config.yml \
    --org-id=wx-org5.chainmaker.org \
    --chain-id=chain1 \
    --user-tlscrt-file-path=./testdata/crypto-config/wx-org5.chainmaker.org/user/client1/client1.tls.crt \
    --user-tlskey-file-path=./testdata/crypto-config/wx-org5.chainmaker.org/user/client1/client1.tls.key \
    --user-signcrt-file-path=./testdata/crypto-config/wx-org5.chainmaker.org/user/client1/client1.sign.crt \
    --user-signkey-file-path=./testdata/crypto-config/wx-org5.chainmaker.org/user/client1/client1.sign.key \
    --sync-result=true
    ```

<span id="chainConfig.dposGetNodeID"></span>

  - Stake合约中查询验证人的NodeID

    ```sh
    $ ./cmc client contract system get-node-id \
    --address=7E9czQBNz99iBfy4EDb7SUB9HxV4rQZjiXcnwBb3UFYk \
    --sdk-conf-path=./testdata/sdk_config.yml \
    --org-id=wx-org1.chainmaker.org \
    --chain-id=chain1 \
    --user-tlscrt-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.crt \
    --user-tlskey-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.key \
    --user-signcrt-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.crt \
    --user-signkey-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.key
    ```

<span id="chainConfig.dposMinSelfDelegation"></span>
  - 查询验证人节点的最少自我抵押数量

    ```sh
    $ ./cmc client contract system min-self-delegation \
    --sdk-conf-path=./testdata/sdk_config.yml \
    --org-id=wx-org1.chainmaker.org \
    --chain-id=chain1 \
    --user-tlscrt-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.crt \
    --user-tlskey-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.key \
    --user-signcrt-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.crt \
    --user-signkey-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.key
    ```

    

<span id="chainConfig.dposValidatorNumInEpoch"></span>
  - 查询世代中的验证人数

    ```sh
    $ ./cmc client contract system validator-number \
    --sdk-conf-path=./testdata/sdk_config.yml \
    --org-id=wx-org1.chainmaker.org \
    --chain-id=chain1 \
    --user-tlscrt-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.crt \
    --user-tlskey-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.key \
    --user-signcrt-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.crt \
    --user-signkey-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.key
    ```

<span id="chainConfig.dposBlockNumEachEpoch"></span>
  - 查询世代中的区块数量

    ```sh
    $ ./cmc client contract system epoch-block-number \
    --sdk-conf-path=./testdata/sdk_config.yml \
    --org-id=wx-org1.chainmaker.org \
    --chain-id=chain1 \
    --user-tlscrt-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.crt \
    --user-tlskey-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.key \
    --user-signcrt-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.crt \
    --user-signkey-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.key
    ```

<span id="chainConfig.dposSystemStakeAddr"></span>
  - 查询Stake合约的系统地址

    ```sh
    $ ./cmc client contract system system-address \
    --sdk-conf-path=./testdata/sdk_config.yml \
    --org-id=wx-org1.chainmaker.org \
    --chain-id=chain1 \
    --user-tlscrt-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.crt \
    --user-tlskey-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.key \
    --user-signcrt-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.crt \
    --user-signkey-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.key
    ```

<span id="chainConfig.dposCompeleteUndeleteBlockNum"></span>
  - 查询收到解质押退款间隔的世代数

    ```sh
    $ ./cmc client contract system unbonding-epoch-number \
    --sdk-conf-path=./testdata/sdk_config.yml \
    --org-id=wx-org1.chainmaker.org \
    --chain-id=chain1 \
    --user-tlscrt-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.crt \
    --user-tlskey-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.key \
    --user-signcrt-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.crt \
    --user-signkey-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.key
    ```

<span id="queryOnChainData"></span>
#### 查询链上数据

  查询链上block和transaction 主要参数说明如下：

  ```sh
    --sdk-conf-path：指定cmc使用sdk的配置文件路径
    --chain-id：指定链Id
  ```
  - 根据区块高度查询链上未归档区块
    
    ```sh
    ./cmc query block-by-height [blockheight] \
    --chain-id=chain1 \
    --sdk-conf-path=./testdata/sdk_config.yml
    ```

  - 根据区块hash查询链上未归档区块

    ```sh
    ./cmc query block-by-hash [blockhash] \
    --chain-id=chain1 \
    --sdk-conf-path=./testdata/sdk_config.yml
    ```

  - 根据txid查询链上未归档区块

    ```sh
    ./cmc query block-by-txid [txid] \
    --chain-id=chain1 \
    --sdk-conf-path=./testdata/sdk_config.yml
    ```

  - 根据txid查询链上未归档tx

    ```sh
    ./cmc query tx [txid] \
    --chain-id=chain1 \
    --sdk-conf-path=./testdata/sdk_config.yml
    ```

<span id="chainConfig"></span>
#### 链配置

  查询及更新链配置 主要参数说明如下：

  ```sh
    sdk配置文件flag
    --sdk-conf-path：指定cmc使用sdk的配置文件路径

    admin签名者flags，此类flag的顺序及个数必须保持一致，且至少传入一个admin
    --admin-crt-file-paths: admin签名者的tls crt文件的路径列表. 单签模式下只需要填写一个即可, 离线多签模式下多个需要用逗号分割
        比如 ./wx-org1.chainmaker.org/admin1.tls.crt,./wx-org2.chainmaker.org/admin1.tls.crt
    --admin-key-file-paths: admin签名者的tls key文件的路径列表. 单签模式下只需要填写一个即可, 离线多签模式下多个需要用逗号分割
        比如 ./wx-org1.chainmaker.org/admin1.tls.key,./wx-org2.chainmaker.org/admin1.tls.key

    覆盖sdk配置flags，不传则使用sdk配置，如果想覆盖sdk的配置，则以下六个flag都必填
    --org-id: 指定发送交易的用户所属的组织Id, 会覆盖sdk配置文件读取的配置
    --chain-id: 指定链Id, 会覆盖sdk配置文件读取的配置
    --user-tlscrt-file-path: 指定发送交易的用户tls证书文件路径, 会覆盖sdk配置文件读取的配置
    --user-tlskey-file-path: 指定发送交易的用户tls私钥路径, 会覆盖sdk配置文件读取的配置
    --user-signcrt-file-path: 指定发送交易的用户sign证书文件路径, 会覆盖sdk配置文件读取的配置
    --user-signkey-file-path: 指定发送交易的用户sign私钥路径, 会覆盖sdk配置文件读取的配置

    --block-interval: 出块时间 单位ms
    --trust-root-org-id: 增加/删除/更新组织证书时指定的组织Id
    --trust-root-path: 增加/删除/更新组织证书时指定的组织CA根证书文件目录
    --node-id: 增加/删除/更新共识节点Id时指定的节点Id
    --node-ids: 增加/更新共识节点Org时指定的节点Id列表
    --node-org-id: 增加/删除/更新共识节点Id,Org时指定节点的组织Id 
  ```

<span id="chainConfig.query"></span>
  - 查询链配置

    ```sh
    ./cmc client chainconfig query \
    --sdk-conf-path=./testdata/sdk_config.yml
    ```

<span id="chainConfig.updateBlockInterval"></span>
  - 更新出块时间

    ```sh
    ./cmc client chainconfig block updateblockinterval \
    --sdk-conf-path=./testdata/sdk_config.yml \
    --admin-key-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.tls.key,./testdata/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.tls.key,./testdata/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.tls.key \
    --admin-crt-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.tls.crt,./testdata/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.tls.crt,./testdata/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.tls.crt \
    --block-interval 1000
    ```

<span id="chainConfig.addOrgRootCA"></span>
  - 增加组织根证书

    ```sh
    ./cmc client chainconfig trustroot add \
    --sdk-conf-path=./testdata/sdk_config.yml \
    --org-id=wx-org1.chainmaker.org \
    --user-tlscrt-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.crt \
    --user-tlskey-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.key \
    --user-signcrt-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.crt \
    --user-signkey-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.key \
    --admin-crt-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.tls.crt,./testdata/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.tls.crt,./testdata/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.tls.crt \
    --admin-key-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.tls.key,./testdata/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.tls.key,./testdata/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.tls.key \
    --trust-root-org-id=wx-org5.chainmaker.org \
    --trust-root-path=./testdata/crypto-config/wx-org5.chainmaker.org/ca/ca.crt
    ```

<span id="chainConfig.delOrgRootCA"></span>
  - 删除组织根证书

    ```sh
    ./cmc client chainconfig trustroot remove \
    --sdk-conf-path=./testdata/sdk_config.yml \
    --org-id=wx-org1.chainmaker.org \
    --user-tlscrt-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.crt \
    --user-tlskey-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.key \
    --user-signcrt-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.crt \
    --user-signkey-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.key \
    --admin-crt-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.tls.crt,./testdata/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.tls.crt,./testdata/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.tls.crt \
    --admin-key-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.tls.key,./testdata/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.tls.key,./testdata/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.tls.key \
    --trust-root-org-id=wx-org5.chainmaker.org \
    --trust-root-path=./testdata/crypto-config/wx-org5.chainmaker.org/ca/ca.crt
    ```

<span id="chainConfig.updateOrgRootCA"></span>
  - 更新组织根证书

    ```sh
    ./cmc client chainconfig trustroot update \
    --sdk-conf-path=./testdata/sdk_config.yml \
    --org-id=wx-org1.chainmaker.org \
    --user-tlscrt-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.crt \
    --user-tlskey-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.key \
    --user-signcrt-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.crt \
    --user-signkey-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.key \
    --admin-crt-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.tls.crt,./testdata/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.tls.crt,./testdata/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.tls.crt \
    --admin-key-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.tls.key,./testdata/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.tls.key,./testdata/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.tls.key \
    --trust-root-org-id=wx-org5.chainmaker.org \
    --trust-root-path=./testdata/crypto-config/wx-org5.chainmaker.org/ca/ca.crt
    ```

<span id="chainConfig.addConsensusNodeOrg"></span>
  - 添加共识节点Org

    ```sh
    ./cmc client chainconfig consensusnodeorg add \
    --sdk-conf-path=./testdata/sdk_config.yml \
    --org-id=wx-org1.chainmaker.org \
    --user-tlscrt-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.crt \
    --user-tlskey-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.key \
    --user-signcrt-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.crt \
    --user-signkey-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.key \
    --admin-crt-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.tls.crt,./testdata/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.tls.crt,./testdata/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.tls.crt \
    --admin-key-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.tls.key,./testdata/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.tls.key,./testdata/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.tls.key \
    --node-ids=QmcQHCuAXaFkbcsPUj7e37hXXfZ9DdN7bozseo5oX4qiC4,QmaWrR72CbT51nFVpNDS8NaqUZjVuD4Ezf8xcHcFW9SJWF \
    --node-org-id=wx-org5.chainmaker.org
    ```

<span id="chainConfig.delConsensusNodeOrg"></span>
  - 删除共识节点Org

    ```sh
    ./cmc client chainconfig consensusnodeorg remove \
    --sdk-conf-path=./testdata/sdk_config.yml \
    --org-id=wx-org1.chainmaker.org \
    --user-tlscrt-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.crt \
    --user-tlskey-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.key \
    --user-signcrt-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.crt \
    --user-signkey-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.key \
    --admin-crt-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.tls.crt,./testdata/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.tls.crt,./testdata/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.tls.crt \
    --admin-key-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.tls.key,./testdata/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.tls.key,./testdata/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.tls.key \
    --node-org-id=wx-org5.chainmaker.org
    ```

<span id="chainConfig.updateConsensusNodeOrg"></span>
  - 更新共识节点Org

    ```sh
    ./cmc client chainconfig consensusnodeorg update \
    --sdk-conf-path=./testdata/sdk_config.yml \
    --org-id=wx-org1.chainmaker.org \
    --user-tlscrt-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.crt \
    --user-tlskey-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.key \
    --user-signcrt-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.crt \
    --user-signkey-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.key \
    --admin-crt-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.tls.crt,./testdata/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.tls.crt,./testdata/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.tls.crt \
    --admin-key-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.tls.key,./testdata/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.tls.key,./testdata/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.tls.key \
    --node-ids=QmcQHCuAXaFkbcsPUj7e37hXXfZ9DdN7bozseo5oX4qiC4,QmaWrR72CbT51nFVpNDS8NaqUZjVuD4Ezf8xcHcFW9SJWF \
    --node-org-id=wx-org5.chainmaker.org
    ```

<span id="chainConfig.addConsensusNodeId"></span>
  - 添加共识节点Id

    ```sh
    ./cmc client chainconfig consensusnodeid add \
    --sdk-conf-path=./testdata/sdk_config.yml \
    --org-id=wx-org1.chainmaker.org \
    --user-tlscrt-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.crt \
    --user-tlskey-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.key \
    --user-signcrt-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.crt \
    --user-signkey-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.key \
    --admin-crt-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.tls.crt,./testdata/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.tls.crt,./testdata/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.tls.crt \
    --admin-key-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.tls.key,./testdata/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.tls.key,./testdata/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.tls.key \
    --node-id=QmcQHCuAXaFkbcsPUj7e37hXXfZ9DdN7bozseo5oX4qiC4 \
    --node-org-id=wx-org1.chainmaker.org
    ```

<span id="chainConfig.delConsensusNodeId"></span>
  - 删除共识节点Id

    ```sh
    ./cmc client chainconfig consensusnodeid remove \
    --sdk-conf-path=./testdata/sdk_config.yml \
    --org-id=wx-org1.chainmaker.org \
    --user-tlscrt-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.crt \
    --user-tlskey-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.key \
    --user-signcrt-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.crt \
    --user-signkey-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.key \
    --admin-crt-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.tls.crt,./testdata/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.tls.crt,./testdata/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.tls.crt \
    --admin-key-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.tls.key,./testdata/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.tls.key,./testdata/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.tls.key \
    --node-id=QmcQHCuAXaFkbcsPUj7e37hXXfZ9DdN7bozseo5oX4qiC4 \
    --node-org-id=wx-org1.chainmaker.org
    ```

<span id="chainConfig.updateConsensusNodeId"></span>
  - 更新共识节点Id

    ```sh
    ./cmc client chainconfig consensusnodeid update \
    --sdk-conf-path=./testdata/sdk_config.yml \
    --org-id=wx-org1.chainmaker.org \
    --user-tlscrt-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.crt \
    --user-tlskey-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.key \
    --user-signcrt-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.crt \
    --user-signkey-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.key \
    --admin-crt-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.tls.crt,./testdata/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.tls.crt,./testdata/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.tls.crt \
    --admin-key-file-paths=./testdata/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.tls.key,./testdata/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.tls.key,./testdata/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.tls.key \
    --node-id=QmXxeLkNTcvySPKMkv3FUqQgpVZ3t85KMo5E4cmcmrexrC \
    --node-id-old=QmcQHCuAXaFkbcsPUj7e37hXXfZ9DdN7bozseo5oX4qiC4 \
    --node-org-id=wx-org1.chainmaker.org
    ```

  - Mint

    ```sh
    ./cmc client contract system mint \
    --amount=100000000 \
    --address=8qMDuPfyw7GrMfncziGjA1Uaz9LdqPZZon5zXF3bLJ5Q \
    --sdk-conf-path=./testdata/sdk_config.yml \
    --org-id=wx-org1.chainmaker.org \
    --chain-id=chain1 \
    --user-tlscrt-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.crt \
    --user-tlskey-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.key \
    --user-signcrt-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.crt \
    --user-signkey-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.key \
    --sync-result=true
    ```

  - Banalce

    ```sh
    ./cmc client contract system balance-of \
    --address=Fgtec5CoPaZ39mhQzn4xnuN4ZRMeXMtJB3JZsfamt3C1 \
    --sdk-conf-path=./testdata/sdk_config.yml \
    --org-id=wx-org1.chainmaker.org \
    --chain-id=chain1 \
    --user-tlscrt-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.crt \
    --user-tlskey-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.key \
    --user-signcrt-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.crt \
    --user-signkey-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.key
    ```

  - Set Node ID

    ```sh
    ./cmc client contract system set-node-id \
    --node-id="QmYGWh39qNsF4PdsoUoZkLPMm6Hds6VbEQhSu2Q1z6PTsc" \
    --sdk-conf-path=./testdata/sdk_config.yml \
    --org-id=wx-org5.chainmaker.org \
    --chain-id=chain1 \
    --user-tlscrt-file-path=./testdata/crypto-config/wx-org5.chainmaker.org/user/client1/client1.tls.crt \
    --user-tlskey-file-path=./testdata/crypto-config/wx-org5.chainmaker.org/user/client1/client1.tls.key \
    --user-signcrt-file-path=./testdata/crypto-config/wx-org5.chainmaker.org/user/client1/client1.sign.crt \
    --user-signkey-file-path=./testdata/crypto-config/wx-org5.chainmaker.org/user/client1/client1.sign.key \
    --sync-result=true
    ```

  - Get Node ID

    ```sh
    ./cmc client contract system get-node-id \
    --address=Fgtec5CoPaZ39mhQzn4xnuN4ZRMeXMtJB3JZsfamt3C1 \
    --sdk-conf-path=./testdata/sdk_config.yml \
    --org-id=wx-org1.chainmaker.org \
    --chain-id=chain1 \
    --user-tlscrt-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.crt \
    --user-tlskey-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.key \
    --user-signcrt-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.crt \
    --user-signkey-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.key
    ```
    
  - Delegate
    
    ```sh
    ./cmc client contract system delegate \
    --address=Fgtec5CoPaZ39mhQzn4xnuN4ZRMeXMtJB3JZsfamt3C1 \
    --amount=100000000 \
    --sdk-conf-path=./testdata/sdk_config.yml \
    --org-id=wx-org5.chainmaker.org \
    --chain-id=chain1 \
    --user-tlscrt-file-path=./testdata/crypto-config/wx-org5.chainmaker.org/user/client1/client1.tls.crt \
    --user-tlskey-file-path=./testdata/crypto-config/wx-org5.chainmaker.org/user/client1/client1.tls.key \
    --user-signcrt-file-path=./testdata/crypto-config/wx-org5.chainmaker.org/user/client1/client1.sign.crt \
    --user-signkey-file-path=./testdata/crypto-config/wx-org5.chainmaker.org/user/client1/client1.sign.key \
    --sync-result=true
    ```

  - Undelegate

    ```sh
    ./cmc client contract system undelegate \
    --address=Fgtec5CoPaZ39mhQzn4xnuN4ZRMeXMtJB3JZsfamt3C1 \
    --amount=100000000 \
    --sdk-conf-path=./testdata/sdk_config.yml \
    --org-id=wx-org5.chainmaker.org \
    --chain-id=chain1 \
    --user-tlscrt-file-path=./testdata/crypto-config/wx-org5.chainmaker.org/user/client1/client1.tls.crt \
    --user-tlskey-file-path=./testdata/crypto-config/wx-org5.chainmaker.org/user/client1/client1.tls.key \
    --user-signcrt-file-path=./testdata/crypto-config/wx-org5.chainmaker.org/user/client1/client1.sign.crt \
    --user-signkey-file-path=./testdata/crypto-config/wx-org5.chainmaker.org/user/client1/client1.sign.key \
    --sync-result=true
    ```

  - 查询所有验证人信息

    ```sh
    ./cmc client contract system all-candidates \
    --sdk-conf-path=./testdata/sdk_config.yml \
    --org-id=wx-org1.chainmaker.org \
    --chain-id=chain1 \
    --user-tlscrt-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.crt \
    --user-tlskey-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.key \
    --user-signcrt-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.crt \
    --user-signkey-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.key
    ```

  - 查询指定验证人信息
    
    ```sh
    ./cmc client contract system get-validator \
    --address=Fgtec5CoPaZ39mhQzn4xnuN4ZRMeXMtJB3JZsfamt3C1 \
    --sdk-conf-path=./testdata/sdk_config.yml \
    --org-id=wx-org1.chainmaker.org \
    --chain-id=chain1 \
    --user-tlscrt-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.crt \
    --user-tlskey-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.key \
    --user-signcrt-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.crt \
    --user-signkey-file-path=./testdata/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.key
    ```

<span id="archive"></span>
#### 归档&恢复功能

  cmc的归档功能是指将链上数据转移到独立存储上，归档后的数据具备可查询、可恢复到链上的特性。<br>
  为了保持数据一致性和防止误操作，cmc实现了分布式锁，同一时刻只允许一个cmc进程进行转储。<br>
  cmc支持增量转储和恢复、断点中继转储和恢复，中途退出不影响数据一致性。<br><br>
  主要参数说明如下：

  ```sh
    --sdk-conf-path：指定cmc使用sdk的配置文件路径
    --chain-id：指定链Id
    --type：指定链下独立存储类型，如 --type=mysql 默认mysql，目前只支持mysql
    --dest：指定链下独立存储目标地址，mysql类型的格式如 --dest=user:password:localhost:port
    --target：指定转储目标区块高度，在达到这个高度后停止转储(包括这个块) --target=100
        也可指定转存目标日期，转储在此日期之前的所有区块 --target="2021-06-01 15:01:41"
    --blocks：指定本次要转储的块数量，注意：对于target和blocks这两个参数，cmc会就近原则采用先符合条件的参数
    --start-block-height：指定链数据恢复时的起始区块高度，如设置为100，则从已转储并且未恢复的最大区块开始降序恢复链数据至第100区块
    --secret-key：指定密码，用于链数据转储和链数据恢复时数据一致性校验，转储和恢复时密码需要一致
  ```
  - 根据时间转储，将链上数据转移到独立存储上，需要权限：sdk配置文件中设置与归档节点同组织的[admin用户](#sdkConfig)

    ```sh
    ./cmc archive dump --type=mysql \
    --dest=root:password:localhost:3306 \
    --target="2021-06-01 15:01:41" \
    --blocks=10000 \
    --chain-id=chain1 \
    --sdk-conf-path=./testdata/sdk_config.yml \
    --secret-key=mypassword
    ```

  - 根据区块高度转储，将链上数据转移到独立存储上，需要权限：sdk配置文件中设置与归档节点同组织的[admin用户](#sdkConfig)

    ```sh
    ./cmc archive dump --type=mysql \
    --dest=root:password:localhost:3306 \
    --target=100 \
    --blocks=10000 \
    --chain-id=chain1 \
    --sdk-conf-path=./testdata/sdk_config.yml \
    --secret-key=mypassword
    ```

  - 恢复，将链下的链数据恢复到链上，需要权限：sdk配置文件中设置与归档节点同组织的[admin用户](#sdkConfig)

    ```sh
    ./cmc archive restore --type=mysql \
    --dest=root:password:localhost:3306 \
    --start-block-height=0 \
    --chain-id=chain1 \
    --sdk-conf-path=./testdata/sdk_config.yml \
    --secret-key=mypassword
    ```

  - 根据区块高度查询链下已归档区块

    ```sh
    ./cmc archive query block-by-height [blockheight] \
    --chain-id=chain1 \
    --sdk-conf-path=./testdata/sdk_config.yml \
    --type=mysql \
    --dest=root:password:localhost:3306
    ```

  - 根据区块hash查询链下已归档区块

    ```sh
    ./cmc archive query block-by-hash [blockhash] \
    --chain-id=chain1 \
    --sdk-conf-path=./testdata/sdk_config.yml \
    --type=mysql \
    --dest=root:password:localhost:3306
    ```

  - 根据txid查询链下已归档区块

    ```sh
    ./cmc archive query block-by-txid [txid] \
    --chain-id=chain1 \
    --sdk-conf-path=./testdata/sdk_config.yml \
    --type=mysql \
    --dest=root:password:localhost:3306
    ```

  - 根据txid查询链下已归档tx

    ```sh
    ./cmc archive query tx [txid] \
    --chain-id=chain1 \
    --sdk-conf-path=./testdata/sdk_config.yml \
    --type=mysql \
    --dest=root:password:localhost:3306
    ```

  <br><br>