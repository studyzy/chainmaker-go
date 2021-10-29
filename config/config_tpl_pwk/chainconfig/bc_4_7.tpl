#
# Copyright (C) BABEC. All rights reserved.
# Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

chain_id: {chain_id}                # 链标识
version: v2.1.0                     # 链版本
sequence: 0                         # 配置版本
auth_type: "permissionedWithKey"    # 认证类型

crypto:
  hash: SHA256

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

# snapshot module
snapshot:
  enable_evidence: false # enable the evidence support

# scheduler module
scheduler:
  enable_evidence: false # enable the evidence support

#共识配置
consensus:
  # 共识类型(0-SOLO,1-TBFT,2-MBFT,3-HOTSTUFF,4-RAFT,5-DPOS,10-POW)
  type: {consensus_type}
  # 共识节点列表，组织必须出现在trust_roots的org_id中，每个组织可配置多个共识节点，节点地址采用libp2p格式
  # 其中node_id为chainmaker.yml中 node.priv_key_file对应的nodeid
  nodes:
    - org_id: "{org1_id}"
      node_id:
        - "{org1_peerid}"
    - org_id: "{org2_id}"
      node_id:
        - "{org2_peerid}"
    - org_id: "{org3_id}"
      node_id:
        - "{org3_peerid}"
    - org_id: "{org4_id}"
      node_id:
        - "{org4_peerid}"
#    - org_id: "{org5_id}"
#      node_id:
#        - "{org5_peerid}"
#    - org_id: "{org6_id}"
#      node_id:
#        - "{org6_peerid}"
#    - org_id: "{org7_id}"
#      node_id:
#        - "{org7_peerid}"
  ext_config: # 扩展字段，记录难度、奖励等其他类共识算法配置
    - key: aa
      value: chain01_ext11

# 信任组织和根证书
trust_roots:
  - org_id: "{org1_id}"
    root:
      - "../config/{org_path}/keys/admin/{org1_id}/admin.pem"
  - org_id: "{org2_id}"
    root:
      - "../config/{org_path}/keys/admin/{org2_id}/admin.pem"
  - org_id: "{org3_id}"
    root:
      - "../config/{org_path}/keys/admin/{org3_id}/admin.pem"
  - org_id: "{org4_id}"
    root:
      - "../config/{org_path}/keys/admin/{org4_id}/admin.pem"
#  - org_id: "{org5_id}"
#    root:
#      - "../config/{org_path}/keys/admin/{org5_id}/admin.pem"
#  - org_id: "{org6_id}"
#    root:
#      - "../config/{org_path}/keys/admin/{org6_id}/admin.pem"
#  - org_id: "{org7_id}"
#    root:
#      - "../config/{org_path}/keys/admin/{org7_id}/admin.pem"

# 权限配置（只能整体添加、修改、删除）
resource_policies:
  - resource_name: CHAIN_CONFIG-NODE_ID_UPDATE
    policy:
      rule: SELF # 规则（ANY，MAJORITY...，全部大写，自动转大写）
      org_list: # 组织名称（组织名称，区分大小写）
      role_list: # 角色名称（role，自动转大写）
        - admin
  - resource_name: CHAIN_CONFIG-TRUST_ROOT_ADD
    policy:
      rule: MAJORITY
      org_list:
      role_list:
        - admin

disabled_native_contract:
  - CONTRACT_NAME
