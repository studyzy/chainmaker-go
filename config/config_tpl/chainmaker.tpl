#
# Copyright (C) BABEC. All rights reserved.
# Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

log:
  config_file: ../config/{org_path}/log.yml          # config file of logger configuration.

blockchain:
#  - chainId: chain1
#    genesis: ../config/{org_path1}/chainconfig/bc1.yml
#  - chainId: chain2
#    genesis: ../config/{org_path2}/chainconfig/bc2.yml
#  - chainId: chain3
#    genesis: ../config/{org_path3}/chainconfig/bc3.yml
#  - chainId: chain4
#    genesis: ../config/{org_path4}/chainconfig/bc4.yml

node:
  # 节点类型：full
  type:              full
  org_id:            {org_id}
  priv_key_file:     ../config/{org_path}/certs/{node_cert_path}.key
  cert_file:         ../config/{org_path}/certs/{node_cert_path}.crt
  signer_cache_size: 1000
  cert_cache_size:   1000
  pkcs11:
    enabled: false
    library: # path to the so file of pkcs11 interface
    label: # label for the slot to be used
    password: # password to logon the HSM
    session_cache_size: 10 # size of HSM session cache, default to 10
    hash: "SHA256" # hash algorithm used to compute SKI

net:
  provider: LibP2P
  listen_addr: /ip4/0.0.0.0/tcp/{net_port}
  #  peer_stream_pool_size: 100
  #  max_peer_count_allow: 20
  #  peer_elimination_strategy: 3  # 1 Random, 2 FIFO, 3 LIFO
  #  seeds ip格式为： "/ip4/127.0.0.1/tcp/11301/p2p/"+nodeid
  #  seeds dns格式为："/dns/cm-node1.org/tcp/11301/p2p/"+nodeid
  seeds:

  tls:
    enabled: true
    priv_key_file: ../config/{org_path}/certs/{net_cert_path}.key
    cert_file: ../config/{org_path}/certs/{net_cert_path}.crt
#  blacklist:
#    addresses:
#      - "127.0.0.1:11305"
#      - "192.168.1.8"
#    node_ids:
#      - "QmeyNRs2DwWjcHTpcVHoUSaDAAif4VQZ2wQDQAUNDP33gH"
#      - "QmVSCXfPweL1GRSNt8gjcw1YQ2VcCirAtTdLKGkgGKsHqi"
#  custom_chain_trust_roots:
#    - chain_id: "chain1"
#      trust_roots:
#        - org_id: "{org_id}"
#          root: "../config/{org_path}/certs/ca/{org_id}/ca.crt"


txpool:
  max_txpool_size: 50000 # 普通交易池上限
  max_config_txpool_size: 10 # config交易池的上限
  full_notify_again_time: 30 # 交易池溢出后，再次通知的时间间隔(秒)
#  pool_type: "batch"  # single/batch：single实时进入交易池，batch批量进入交易池
#  batch_max_size: 30000 # 批次最大大小
#  batch_create_timeout: 200 # 创建批次超时时间，单位毫秒

rpc:
  provider: grpc
  port: {rpc_port}
  # 检查链配置TrustRoots证书变化时间间隔，单位：s，最小值为10s
  check_chain_conf_trust_roots_change_interval: 60
  ratelimit:
    enabled: false
    # 限速类型：0-全局限速；1-基于来源IP限速
    type: 0
    # 每秒补充令牌数，取值：-1-不受限；0-默认值（10000）
    token_per_second: -1
    # 令牌桶大小，取值：-1-不受限；0-默认值（10000）
    token_bucket_size: -1
  subscriber:
    # 历史消息订阅流控，实时消息订阅不会进行流控
    ratelimit:
      # 每秒补充令牌数，取值：-1-不受限；0-默认值（1000）
      token_per_second: 100
      # 令牌桶大小，取值：-1-不受限；0-默认值（1000）
      token_bucket_size: 100
  tls:
    # TLS模式:
    #   disable - 不启用TLS
    #   oneway  - 单向认证
    #   twoway  - 双向认证
    #mode: disable
    #mode: oneway
    mode:           twoway
    priv_key_file:  ../config/{org_path}/certs/{rpc_cert_path}.key
    cert_file:      ../config/{org_path}/certs/{rpc_cert_path}.crt

monitor:
  enabled: true
  port: {monitor_port}

pprof:
  enabled: false
  port: {pprof_port}

consensus:
  raft:
    snap_count: 10
    # 是否异步Wal文件保存，true异步保存，false同步保存
    async_wal_save: true
    # 1 time.Duration 表示1 * time.Second
    ticker: 1

storage:
  store_path: ../data/{org_id}/ledgerData1
  # 最小的不允许归档的区块高度
  unarchive_block_height: 300000
  blockdb_config:
    provider: leveldb
    leveldb_config:
      store_path: ../data/{org_id}/block
  statedb_config:
    provider: leveldb # leveldb/sql 二选一
    leveldb_config: # leveldb config
      store_path: ../data/{org_id}/state
  #    sqldb_config: # sql config，只有provider为sql的时候才需要配置和启用这个配置
  #      sqldb_type: mysql           #具体的sql db类型，目前支持mysql，sqlite
  #      dsn: root:password@tcp(127.0.0.1:3306)/  #mysql的连接信息，包括用户名、密码、ip、port等，示例：root:admin@tcp(127.0.0.1:3306)/
  historydb_config:
    provider: leveldb
    leveldb_config:
      store_path: ../data/{org_id}/history
#  historydb_config:
#    provider: rocksdb
#    rocksdb_config:
#      write_buffer_size: 64
#      db_write_buffer_size: 4
#      block_cache_size: 128
#      max_write_buffer_number: 10
#      max_background_compactions: 4
#      max_background_flushes: 2
#      bloom_filter_bits: 10
  resultdb_config:
    provider: leveldb
    leveldb_config:
      store_path: ../data/{org_id}/result
  disable_contract_eventdb: true  #是否禁止合约事件存储功能，默认为true，如果设置为false,需要配置mysql
  contract_eventdb_config:
    provider: sql                 #如果开启contract event db 功能，需要指定provider为sql
    sqldb_config:
      sqldb_type: mysql           #contract event db 只支持mysql
      dsn: root:password@tcp(127.0.0.1:3306)/  #mysql的连接信息，包括用户名、密码、ip、port等，示例：root:admin@tcp(127.0.0.1:3306)/
core:
  evidence: false
scheduler:
  rwset_log: false #whether log the txRWSet map in the debug mode
