## 历史数据库

### 状态变更历史
记录了每个合约的每个状态的变更历史，只支持PutState的状态变更历史，不支持SQL语句的状态变更历史。
状态历史表的主键为：
ContractName+StateKey+BlockHeight+TxId
### 账户发起交易历史
记录了每个账户发起的TxId的历史，其主键为：
AccountId+BlockHeight+ TxId
### 合约被调用历史
记录了每个合约在哪个TxId中被调谁用了。其主键为：
ContractName+BlockHeight+ TxId+AccountId
### SQL DB
数据库名： chainId+"_historydb"