#
# Copyright (C) BABEC. All rights reserved.
# Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

chain_id: {chain_id}                # 链标识
version: v2.1.0_alpha                     # 链版本
sequence: 0                         # 配置版本
auth_type: "public"                 # 认证类型  permissionedWithCert / permissionedWithKey / public

crypto:
  hash: {hash_type}

# 合约支持类型的配置
contract:
  enable_sql_support: false # 合约是否支持sql，此处若为true，则chainmaker.yml中则需配置storage.statedb_config.provider=sql，否则无法启动

# 交易、区块相关配置
block:
  tx_timestamp_verify: true # 是否需要开启交易时间戳校验
  tx_timeout: 600  # 交易时间戳的过期时间(秒)
  block_tx_capacity: 100  # 区块中最大交易数
  block_size: 10  # 区块最大限制，单位MB
  block_interval: 2000 # 出块间隔，单位:ms

# core模块
core:
  tx_scheduler_timeout: 10 #  [0, 60] 交易调度器从交易池拿到交易后, 进行调度的时间
  tx_scheduler_validate_timeout: 10 # [0, 60] 交易调度器从区块中拿到交易后, 进行验证的超时时间
#  consensus_turbo_config:
#    consensus_message_turbo: true # 是否开启共识报文压缩
#    retry_time: 500 # 根据交易ID列表从交易池获取交易的重试次数
#    retry_interval: 20 # 重试间隔，单位:ms

#共识配置
consensus:
  # 共识类型(5-DPOS)
  type: {consensus_type}
  dpos_config: # DPoS
    #ERC20合约配置
    - key: erc20.total
      value: "{erc20_total}"
    - key: erc20.owner
      value: "{org1_peeraddr}"
    - key: erc20.decimals
      value: "18"
    - key: erc20.account:DPOS_STAKE
      value: "{erc20_total}"
    #Stake合约配置
    - key: stake.minSelfDelegation
      value: "2500000"
    - key: stake.epochValidatorNum
      value: "{epochValidatorNum}"
    - key: stake.epochBlockNum
      value: "10"
    - key: stake.completionUnbondingEpochNum
      value: "1"
    - key: stake.candidate:{org1_peeraddr}
      value: "2500000"
    - key: stake.candidate:{org2_peeraddr}
      value: "2500000"
    - key: stake.candidate:{org3_peeraddr}
      value: "2500000"
    - key: stake.candidate:{org4_peeraddr}
      value: "2500000"
    - key: stake.candidate:{org5_peeraddr}
      value: "2500000"
    - key: stake.candidate:{org6_peeraddr}
      value: "2500000"
    - key: stake.candidate:{org7_peeraddr}
      value: "2500000"
    - key: stake.candidate:{org8_peeraddr}
      value: "2500000"
    - key: stake.candidate:{org9_peeraddr}
      value: "2500000"
    - key: stake.candidate:{org10_peeraddr}
      value: "2500000"
    - key: stake.candidate:{org11_peeraddr}
      value: "2500000"
    - key: stake.candidate:{org12_peeraddr}
      value: "2500000"
    - key: stake.candidate:{org13_peeraddr}
      value: "2500000"
    - key: stake.candidate:{org14_peeraddr}
      value: "2500000"
    - key: stake.candidate:{org15_peeraddr}
      value: "2500000"
    - key: stake.candidate:{org16_peeraddr}
      value: "2500000"
    - key: stake.nodeID:{org1_peeraddr}
      value: "{org1_peerid}"
    - key: stake.nodeID:{org2_peeraddr}
      value: "{org2_peerid}"
    - key: stake.nodeID:{org3_peeraddr}
      value: "{org3_peerid}"
    - key: stake.nodeID:{org4_peeraddr}
      value: "{org4_peerid}"
    - key: stake.nodeID:{org5_peeraddr}
      value: "{org5_peerid}"
    - key: stake.nodeID:{org6_peeraddr}
      value: "{org6_peerid}"
    - key: stake.nodeID:{org7_peeraddr}
      value: "{org7_peerid}"
    - key: stake.nodeID:{org8_peeraddr}
      value: "{org8_peerid}"
    - key: stake.nodeID:{org9_peeraddr}
      value: "{org9_peerid}"
    - key: stake.nodeID:{org10_peeraddr}
      value: "{org10_peerid}"
    - key: stake.nodeID:{org11_peeraddr}
      value: "{org11_peerid}"
    - key: stake.nodeID:{org12_peeraddr}
      value: "{org12_peerid}"
    - key: stake.nodeID:{org13_peeraddr}
      value: "{org13_peerid}"
    - key: stake.nodeID:{org14_peeraddr}
      value: "{org14_peerid}"
    - key: stake.nodeID:{org15_peeraddr}
      value: "{org15_peerid}"
    - key: stake.nodeID:{org16_peeraddr}
      value: "{org16_peerid}"

  ext_config: # 扩展字段，记录难度、奖励等其他类共识算法配置
    - key: aa
      value: chain01_ext11

# 信任组织和根证书
trust_roots:
  - org_id: "public"
    root:
      - "../config/{org_path}/admin/admin1/admin1.pem"
      - "../config/{org_path}/admin/admin2/admin2.pem"
      - "../config/{org_path}/admin/admin3/admin3.pem"
      - "../config/{org_path}/admin/admin4/admin4.pem"

