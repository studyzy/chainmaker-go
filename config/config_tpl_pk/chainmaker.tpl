#
# Copyright (C) BABEC. All rights reserved.
# Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

# [*] represents the item cannot be modified after startup

# The blockchain auth type, shoudle be consistent with auth type in each chain config (e.g., bc1.yml)
# The auth type can be permissionedWithCert, permissionedWithKey, public.
# By default it is permissionedWithCert.
# permissionedWithCert: permissioned blockchain, using x.509 certificate to identify members.
# permissionedWithKey: permissioned blockchain, using public key to identify members.
# public: public blockchain, using public key to identify members.
auth_type: "public" # [*]

# Logger settings
log:
  # Logger configuration file path
  config_file: ../config/{org_path}/log.yml

# Chains the node currently knows
blockchain:
  # chain id and its genesis block file path
#  - chainId: chain1
#    genesis: ../config/{org_path1}/chainconfig/bc1.yml
#  - chainId: chain2
#    genesis: ../config/{org_path2}/chainconfig/bc2.yml
#  - chainId: chain3
#    genesis: ../config/{org_path3}/chainconfig/bc3.yml
#  - chainId: chain4
#    genesis: ../config/{org_path4}/chainconfig/bc4.yml

# Blockchain node settings
node:
  # Private key file path
  priv_key_file:     ../config/{org_path}/certs/{node_cert_path}.key  # [*]

  # PKCS#11 crypto settings
  pkcs11:
    # Enable it or not
    enabled: false  # [*]

    # Path to the so file of pkcs11 interface
    library: /usr/local/lib64/pkcs11/libupkcs11.so

    # Label for the slot to be used
    label: HSM

    # HSM Password
    password: 11111111

    # Size of HSM session cache, default to 10
    session_cache_size: 10

    # Hash algorithm used to compute SKI.
    # It can be SHA256 or SM3.
    hash: "SHA256"  # [*]

# Network Settings
net:
  # Network provider, can be libp2p or liquid.
  # libp2p: using libp2p components to build the p2p module.
  # liquid: a new p2p module we build from 0 to 1.
  # This item must be consistent across the blockchain network.
  provider: LibP2P

  # The address and port the node listens on.
  # By default, it uses 0.0.0.0 to listen on all network interfaces.
  listen_addr: /ip4/0.0.0.0/tcp/{net_port}

  # Max stream of a connection.
  # peer_stream_pool_size: 100

  # Max number of peers the node can connect.
  # max_peer_count_allow: 20

  # The strategy for eliminating node when the count of connecting peers reach the max value.
  # It could be: 1 Random, 2 FIFO, 3 LIFO. The default strategy is LIFO.
  # peer_elimination_strategy: 3

  # The seeds peer list used to join in the network when starting.
  # The connection supervisor will try to dial seed peer whenever the connection is broken.
  # Example ip format: "/ip4/127.0.0.1/tcp/11301/p2p/"+nodeid
  # Example dns format："/dns/cm-node1.org/tcp/11301/p2p/"+nodeid
  seeds:

  # Network tls settings.
  tls:
    # Enable tls or not. Currently it can only be true...
    enabled: true

    # TLS private key file path.
    priv_key_file: ../config/{org_path}/certs/{net_pk_path}.key

    # TLS Certificate file path.
    cert_file: ../config/{org_path}/certs/{net_cert_path}.crt

  # The blacklisted peers in p2p network.
  # blacklist:
      # The addresses in blacklist.
      # The address format can be ip or ip+port.
      # addresses:
      #   - "127.0.0.1:11301"
      #   - "192.168.1.8"

      # The node ids in blacklist.
      # node_ids:
      #   - "QmeyNRs2DwWjcHTpcVHoUSaDAAif4VQZ2wQDQAUNDP33gH"

# Transaction pool settings
# Other txpool settings can be found in tx_Pool_config.go
txpool:
  # txpool type, can be signle or batch.
  # By default the txpool type is single.
  pool_type: "single"

  # Max transaction count in txpool.
  # If txpool is full, the following transactions will be discarded.
  max_txpool_size: 50000

  # Max config transaction count in config txpool.
  max_config_txpool_size: 10

  # Interval of creating a transaction batch, only for batch txpool, in ms.
  # batch_create_timeout: 200


# RPC service setting
rpc:
  # RPC type, can only be grpc now
  provider: grpc  # [*]

  # RPC port
  port: {rpc_port}

  # Interval of checking trust root changes, in seconds.
  # If changed, the rpc server's root certificate pool will also change.
  # Only valid if tls is enabled.
  # The minium value is 10s.
  check_chain_conf_trust_roots_change_interval: 60

  # Rate limit related settings
  # Here we use token bucket to limit rate.
  ratelimit:
    # If rate limit is enabled.
    enabled: false

    # Rate limit type
    # 0: limit globally, 1: limit by ip
    type: 0

    # Token number added to bucket per second.
    # -1: unlimited, by default is 10000.
    token_per_second: -1

    # Token bucket size.
    # -1: unlimited, by default is 10000.
    token_bucket_size: -1

  # Rate limit settings for subscriber
  subscriber:
    ratelimit:
      token_per_second: 100
      token_bucket_size: 100

  # RPC TLS settings
  tls:
    # TLS mode, can be disable, oneway, twoway.
    mode:           twoway

    # RPC TLS private key file path
    priv_key_file:  ../config/{org_path}/certs/{rpc_cert_path}.key

    # RPC TLS public key file path
    cert_file:      ../config/{org_path}/certs/{rpc_cert_path}.crt

  # RPC blacklisted ip addresses
  blacklist:
    addresses:
      # - "127.0.0.1"

# Monitor related settings
monitor:
  # If monitor service is enabled or not
  enabled: false

  # Monitor service port
  port: {monitor_port}

# PProf Settings
pprof:
  # If pprof is enabled or not
  enabled: false

  # PProf port
  port: {pprof_port}

# Consensus related settings
consensus:
  raft:
    # We should take a snapshot after how many blocks.
    # If raft nodes change, a snapshot is taken immediately.
    snap_count: 10

    # Saving wal asynchronously or not.
    async_wal_save: true

    # Min time unit in rate election and heartbeat.
    ticker: 1

# Scheduler related settings
scheduler:
  # whether log the txRWSet map in debug mode
  rwset_log: false

# Storage config settings
# Contains blockDb、stateDb、historyDb、resultDb、contractEventDb
#
# blockDb: block transaction data,                          support leveldb、mysql、badgerdb
# stateDb: world state data,                                support leveldb、mysql、badgerdb
# historyDb: world state change history of transactions,    support leveldb、mysql、badgerdb
# resultDb: transaction execution results data,             support leveldb、mysql、badgerdb
# contractEventDb: contract emit event data, support        support mysql
#
# provider、sqldb_type cannot be changed after startup
# store_path、dsn the content cannot be changed after startup
storage:
  # Default store path
  store_path: ../data/{org_id}/ledgerData1 # [*]

  # Prefix for mysql db name
  # db_prefix: org1_

  # Minimum block height not allowed to be archived
  unarchive_block_height: 300000

  # Symmetric encryption algorithm for writing data to disk. can be sm4 or aes
  # encryptor: sm4    # [*]

  # Symmetric encryption key:16 bytes key
  # If pkcs11 is enabled, it is the keyID
  # encrypt_key: "1234567890123456"

  # Block db config
  blockdb_config:
    # Databases type support leveldb、sql、badgerdb
    provider: leveldb # [*]
    # Provider used leveldb must be set leveldb_config
    leveldb_config:
      # LevelDb store path
      store_path: ../data/{org_id}/block

    # Example for sql provider
    # Databases type support leveldb、sql、badgerdb
    # provider: sql # [*]
    # Provider used sql must be set sqldb_config
    # sqldb_config:
      # Sql db type, can be mysql、sqlite. sqlite only for test
      # sqldb_type: mysql # # [*]
      # Mysql connection info, the database name is not required. such as:  root:admin@tcp(127.0.0.1:3306)/
      # dsn: root:password@tcp(127.0.0.1:3306)/

    # Example for badgerdb provider
    # Databases type support leveldb、sql、badgerdb
    # provider: badgerdb
    # Provider used badgerdb must be set badgerdb_config
    # badgerdb_config:
      # BadgerDb store path
      # store_path: ../data/wx-org1.chainmaker.org/history
      # Whether compression is enabled for stored data, default is 0: disabled
      # compression: 0
      # Key and value are stored separately when value is greater than this byte, default is 1024 * 10
      # value_threshold: 256
      # Number of key value pairs written in batch. default is 128
      # write_batch_size: 1024

  # State db config
  statedb_config:
    provider: leveldb
    leveldb_config:
      store_path: ../data/{org_id}/state

  # History db config
  historydb_config:
    provider: leveldb
    leveldb_config:
      store_path: ../data/{org_id}/history

  # Result db config
  resultdb_config:
    provider: leveldb
    leveldb_config:
      store_path: ../data/{org_id}/result

  # Disable db config, If it is set to false, MySQL needs to be contract_eventdb_config
  disable_contract_eventdb: true
  # Contract event db config
  contract_eventdb_config:
    # Event db only support sql
    provider: sql
    # Sql db config
    sqldb_config:
      # Event db only support mysql
      sqldb_type: mysql
      # Mysql connection info, such as:  root:admin@tcp(127.0.0.1:3306)/
      dsn: root:password@tcp(127.0.0.1:3306)/
