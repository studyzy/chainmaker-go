/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package native

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"strconv"

	"chainmaker.org/chainmaker-go/logger"
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/protocol"
	"github.com/golang/protobuf/proto"
	"github.com/mr-tron/base58"
	"github.com/syndtr/goleveldb/leveldb/util"
)

const (
	prefixValidator    = "V/"
	prefixDelegation   = "D/"
	prefixUnDelegation = "U/"

	KeyNodeIDFormat              = "N/%s"
	KeyRevNodeFormat             = "NR/%s"
	KeyValidatorFormat           = "V/%s"
	KeyDelegationFormat          = "D/%s/%s"
	KeyEpochFormat               = "E/%s"
	KeyUnbondingDelegationFormat = "U/%s/%s/%s"

	KeyCurrentEpoch                   = "CE"
	KeyMinSelfDelegation              = "MSD"
	KeyCompletionUnbondingEpochNumber = "CUEN" // 这个 number 不能改动，若改动，会遇到 unbonding entry 数组中 completionEpochID 不同的情况
	KeyEpochValidatorNumber           = "EVN"
	KeyEpochBlockNumber               = "EBN"
)

// ToValidatorKey - Key: V + "/" + ValidatorAddress
func ToValidatorKey(ValidatorAddress string) []byte {
	return []byte(fmt.Sprintf(KeyValidatorFormat, ValidatorAddress))
}

// ToValidatorPrefix - Key: V + "/"
func ToValidatorPrefix() []byte {
	return []byte(fmt.Sprintf(prefixValidator))
}

// ToDelegationKey - Key: D + "/" + DelegatorAddress + "/" + ValidatorAddress
func ToDelegationKey(DelegatorAddress, ValidatorAddress string) []byte {
	return []byte(fmt.Sprintf(KeyDelegationFormat, DelegatorAddress, ValidatorAddress))
}

// ToDelegationPrefix - Key: D + "/" + DelegatorAddress
func ToDelegationPrefix(DelegatorAddress string) []byte {
	return []byte(prefixDelegation + DelegatorAddress)
}

// ToEpochKey - Key：E + "/" + EpochID
func ToEpochKey(epochID string) []byte {
	return []byte(fmt.Sprintf(KeyEpochFormat, epochID))
}

// ToNodeIDKey - Key：N + "/" + NodeID
func ToNodeIDKey(addr string) []byte {
	return []byte(fmt.Sprintf(KeyNodeIDFormat, addr))
}

// ToUnbondingDelegationKey - Key：U + "/" + BigEndian(EpochID) + "/" + DelegatorAddress + "/" + ValidatorAddress
func ToUnbondingDelegationKey(epochID uint64, delegatorAddress, validatorAddress string) []byte {
	epochBz := make([]byte, 8)
	binary.BigEndian.PutUint64(epochBz, epochID)
	return []byte(fmt.Sprintf(KeyUnbondingDelegationFormat, epochBz, delegatorAddress, validatorAddress))
}

// ToUnbondingDelegationPrefix - Key：U + "/" + BigEndian(EpochID)
func ToUnbondingDelegationPrefix(epochID uint64) []byte {
	epochBz := make([]byte, 8)
	binary.BigEndian.PutUint64(epochBz, epochID)
	return []byte(prefixUnDelegation + string(epochBz))
}

func ToReverseNodeIDKey(nodeID string) []byte {
	return []byte(fmt.Sprintf(KeyRevNodeFormat, nodeID))
}

// StakeContractAddr - convert stake contract name string to address: base58.Encode(sha256(stake_contract_name))
func StakeContractAddr() string {
	stakeAddrHash := sha256.Sum256([]byte(commonPb.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String()))
	stakeAddr := base58.Encode(stakeAddrHash[:])
	return stakeAddr
}

func newValidator(validatorAddress string) *commonPb.Validator {
	return &commonPb.Validator{
		ValidatorAddress:           validatorAddress,
		Jailed:                     false,
		Status:                     commonPb.BondStatus_Unbonded,
		Tokens:                     "0",
		DelegatorShares:            "0",
		UnbondingEpochID:           math.MaxInt64,
		UnbondingCompletionEpochID: math.MaxInt64,
		SelfDelegation:             "0",
	}
}

func newDelegation(delegatorAddress, validatorAddress string, shares string) *commonPb.Delegation {
	return &commonPb.Delegation{
		DelegatorAddress: delegatorAddress,
		ValidatorAddress: validatorAddress,
		Shares:           shares,
	}
}

func newUnbondingDelegation(EpochID uint64, DelegatorAddress, ValidatorAddress string) *commonPb.UnbondingDelegation {
	UnbondingDelegationEntry := make([]*commonPb.UnbondingDelegationEntry, 0)
	return &commonPb.UnbondingDelegation{
		EpochID:          strconv.Itoa(int(EpochID)),
		DelegatorAddress: DelegatorAddress,
		ValidatorAddress: ValidatorAddress,
		Entries:          UnbondingDelegationEntry,
	}
}

func newUnbondingDelegationEntry(CreationEpochID, CompletionEpochID uint64, amount string) *commonPb.UnbondingDelegationEntry {
	return &commonPb.UnbondingDelegationEntry{
		CreationEpochID:   CreationEpochID,
		CompletionEpochID: CompletionEpochID,
		Amount:            amount,
	}
}

// main implement here
type DPoSStakeContract struct {
	methods map[string]ContractFunc
	log     *logger.CMLogger
}

func (d *DPoSStakeContract) getMethod(methodName string) ContractFunc {
	return d.methods[methodName]
}

func newDPoSStakeContract(log *logger.CMLogger) *DPoSStakeContract {
	return &DPoSStakeContract{
		log:     log,
		methods: registerDPoSStakeContractMethods(log),
	}
}

func registerDPoSStakeContractMethods(log *logger.CMLogger) map[string]ContractFunc {
	methodMap := make(map[string]ContractFunc, 64)
	// implement
	DPoSStakeRuntime := &DPoSStakeRuntime{log: log}
	methodMap[commonPb.DPoSStakeContractFunction_GET_ALL_VALIDATOR.String()] = DPoSStakeRuntime.GetAllValidator
	methodMap[commonPb.DPoSStakeContractFunction_GET_VALIDATOR_BY_ADDRESS.String()] = DPoSStakeRuntime.GetValidatorByAddress
	methodMap[commonPb.DPoSStakeContractFunction_DELEGATE.String()] = DPoSStakeRuntime.Delegate
	methodMap[commonPb.DPoSStakeContractFunction_GET_DELEGATIONS_BY_ADDRESS.String()] = DPoSStakeRuntime.GetDelegationsByAddress
	methodMap[commonPb.DPoSStakeContractFunction_GET_USER_DELEGATION_BY_VALIDATOR.String()] = DPoSStakeRuntime.GetUserDelegationByValidator
	methodMap[commonPb.DPoSStakeContractFunction_UNDELEGATE.String()] = DPoSStakeRuntime.UnDelegate
	methodMap[commonPb.DPoSStakeContractFunction_READ_EPOCH_BY_ID.String()] = DPoSStakeRuntime.ReadEpochByID
	methodMap[commonPb.DPoSStakeContractFunction_READ_LATEST_EPOCH.String()] = DPoSStakeRuntime.ReadLatestEpoch
	methodMap[commonPb.DPoSStakeContractFunction_SET_NODE_ID.String()] = DPoSStakeRuntime.SetNodeID
	methodMap[commonPb.DPoSStakeContractFunction_GET_NODE_ID.String()] = DPoSStakeRuntime.GetNodeID
	return methodMap
}

type DPoSStakeRuntime struct {
	log *logger.CMLogger
}

// SetNodeID - 系统加入节点绑定自身身份
func (s *DPoSStakeRuntime) SetNodeID(context protocol.TxSimContext, params map[string]string) ([]byte, error) {
	// check params
	err := checkParams(params, "node_id")
	if err != nil {
		return nil, err
	}
	nodeID := params["node_id"]
	// get message sender
	sender, err := loadSenderAddress(context) // Use ERC20 parse method
	if err != nil {
		s.log.Errorf("get sender address error: ", err.Error())
		return nil, err
	}
	// construct kv pair
	// key: 		N/{sender}	V: nodeID
	// reverse key: NR/{nodeID}	V: sender
	reverseKey := ToReverseNodeIDKey(nodeID)
	key := ToNodeIDKey(sender)
	// check reverseKey, check nodeIDKey is unnecessary
	if val, err := context.Get(commonPb.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String(), reverseKey); err != nil {
		s.log.Errorf("query state from context failed, reason: %s", err)
		return nil, fmt.Errorf("query state from context failed, reason: %s", err)
	} else if len(val) > 0 {
		s.log.Errorf("the nodeID:[%s] has been set by [%s]", nodeID, val)
		return nil, fmt.Errorf("the nodeID:[%s] has been set by [%s]", nodeID, val)
	}
	// save to context
	if err := context.Put(commonPb.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String(), key, []byte(nodeID)); err != nil {
		s.log.Errorf("context put key: [%s], value: [&s] error, error: ", key, nodeID, err)
		return nil, err
	}
	if err := context.Put(commonPb.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String(), reverseKey, []byte(sender)); err != nil {
		s.log.Errorf("context put key: [%s], value: [&s] error, error: ", reverseKey, sender, err)
		return nil, err
	}
	return nil, nil
}

// GetNodeID - 系统节点自身身份查询
func (s *DPoSStakeRuntime) GetNodeID(context protocol.TxSimContext, params map[string]string) ([]byte, error) {
	// check params
	err := checkParams(params, "address")
	if err != nil {
		return nil, err
	}
	address := params["address"]
	// construct kv pair
	// key: 		N/{sender}	V: nodeID
	key := ToNodeIDKey(address)
	// check reverseKey, check nodeIDKey is unnecessary
	if val, err := context.Get(commonPb.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String(), key); err != nil {
		s.log.Errorf("query nodeID from context failed, error: %s", err)
		return nil, fmt.Errorf("query nodeID from context failed, error: %s", err)
	} else if len(val) <= 0 {
		s.log.Errorf("[%s] the address's nodeID has not been set", address)
		return nil, fmt.Errorf("[%s] the address's nodeID has not been set", address)
	} else {
		return val, nil
	}
}

// GetAllValidator() []ValidatorAddress		// 返回所有满足最低抵押条件验证人
// return ValidatorVector
func (s *DPoSStakeRuntime) GetAllValidator(context protocol.TxSimContext, params map[string]string) ([]byte, error) {
	// 获取验证人数据
	vc, err := getAllValidatorByPrefix(context, ToValidatorPrefix())
	if err != nil {
		s.log.Error("get validator address error")
		return nil, err
	}

	// 过滤
	collection := &commonPb.ValidatorVector{}
	for _, v := range vc {
		if v == nil {
			s.log.Errorf("validator is nil", v)
			continue
		}
		cmp, err := compareMinSelfDelegation(context, v.SelfDelegation)
		if err != nil {
			s.log.Errorf("compare min self delegation error, amount: %s", v.SelfDelegation)
			return nil, fmt.Errorf("convert self delegate string to integer error, amount: %s, err: %s", v.SelfDelegation, err.Error())
		}
		if v.Jailed == true || cmp == -1 || v.Status != commonPb.BondStatus_Bonded {
			continue
		}
		collection.Vector = append(collection.Vector, v.ValidatorAddress)
	}

	// 序列化
	bz, err := proto.Marshal(collection)
	if err != nil {
		s.log.Errorf("marshal validator collection error: ", err.Error())
		return nil, err
	}
	return bz, nil
}

// GetValidatorByAddress() Validator		// 返回所有满足最低抵押条件验证人
// @params["address"]
// return Validator
func (s *DPoSStakeRuntime) GetValidatorByAddress(context protocol.TxSimContext, params map[string]string) ([]byte, error) {
	// check params
	err := checkParams(params, "address")
	if err != nil {
		return nil, err
	}

	address := params["address"]
	// 获取验证人数据
	bz, err := getValidatorBytes(context, address)
	if err != nil {
		s.log.Errorf("get validator error, address: %s", address)
		return nil, err
	}
	return bz, nil
}

// * Delegate(to string, amount string) (delegation, error)		// 创建抵押，更新验证人，如果MsgSender是给自己，即给自己抵押，则创建验证人
// @params["to"] 		抵押的目标验证人
// @params["amount"]	抵押数量，乘上erc20合约 decimal 的结果值，比如用户认知的1USD，系统内是 1 * 10 ^ 18
// return Delegation
func (s *DPoSStakeRuntime) Delegate(context protocol.TxSimContext, params map[string]string) ([]byte, error) {
	// check params
	err := checkParams(params, "to", "amount")
	if err != nil {
		return nil, err
	}

	to := params["to"]         // delegate target
	amount := params["amount"] // amount must be a integer

	// check amount
	if !assertStringAmountPositive(amount) {
		s.log.Errorf("amount is less than 0")
		return nil, fmt.Errorf("amount is less than 0")
	}
	// 解析交易发送方地址
	from, err := loadSenderAddress(context) // Use ERC20 parse method
	if err != nil {
		s.log.Errorf("get sender address error: ", err.Error())
		return nil, err
	}

	// 查看 validator 是否存在
	v, err := getOrCreateValidator(context, from, to)
	if err != nil {
		s.log.Errorf("get or create validator error: ", err.Error())
		return nil, err
	}

	// 获取或者创建 Delegation
	d, err := getOrCreateDelegation(context, from, to)
	if err != nil {
		s.log.Errorf("get or create delegation error: ", err.Error())
		return nil, err
	}

	// 计算抵押获得的 share
	shares, err := calcShareByAmount(v.Tokens, v.DelegatorShares, amount)
	if err != nil {
		s.log.Errorf("calculate share by amount error: ", err.Error())
		return nil, err
	}

	// 更新 delegation 的 share
	err = updateDelegateShares(d, shares)
	if err != nil {
		s.log.Errorf("update delegate share error", err.Error())
		return nil, err
	}

	// 更新 validator
	err = updateValidatorShares(v, shares)
	if err != nil {
		s.log.Errorf("update shares error: ", err.Error())
		return nil, err
	}
	err = updateValidatorTokens(v, amount)
	if err != nil {
		s.log.Errorf("update tokens error: ", err.Error())
		return nil, err
	}
	if from == to {
		err = updateValidatorSelfDelegate(v, amount)
		if err != nil {
			s.log.Errorf("update self delegate error: ", err.Error())
			return nil, err
		}
		cmp, err := compareMinSelfDelegation(context, v.SelfDelegation)
		if err != nil {
			s.log.Errorf("compare min self delegation error: ", err.Error())
			return nil, err
		}
		if v.Status != commonPb.BondStatus_Bonded && cmp >= 0 {
			updateValidatorStatus(v, commonPb.BondStatus_Bonded)
		} else if v.Status != commonPb.BondStatus_Bonded && cmp == -1 {
			updateValidatorStatus(v, commonPb.BondStatus_Unbonding)
		}
	}

	// 跨合约转账
	// 获取 runtime 对象
	erc20RunTime := NewDPoSRuntime(s.log)
	// stake 地址
	stakeAddr := StakeContractAddr()
	// prepare params
	transferParams := map[string]string{
		"to":    stakeAddr,
		"value": amount,
	}
	_, err = erc20RunTime.Transfer(context, transferParams)
	if err != nil {
		s.log.Errorf("cross call contract ERC20, method transfer error: ", err.Error())
		return nil, err
	}

	// 写入存储
	err = save(context, ToDelegationKey(from, to), d)
	if err != nil {
		s.log.Errorf("save delegate error: ", err.Error())
		return nil, err
	}
	err = save(context, ToValidatorKey(to), v)
	if err != nil {
		s.log.Errorf("save validator error: ", err.Error())
		return nil, err
	}

	// return Delegate info
	return proto.Marshal(d)
}

// GetDelegationsByAddress() []Delegation		// 返回所有 Delegation
// @params["address"]
// return DelegationInfo
func (s *DPoSStakeRuntime) GetDelegationsByAddress(context protocol.TxSimContext, params map[string]string) ([]byte, error) {
	// check params
	err := checkParams(params, "address")
	if err != nil {
		return nil, err
	}

	address := params["address"]
	// 获取验证人数据
	di, err := getDelegationsByAddress(context, address)
	if err != nil {
		s.log.Errorf("get delegation of address [%s] error, error: %s", address, err.Error())
		return nil, err
	}
	return proto.Marshal(di)
}

// GetUserDelegationByValidator() Delegation		// 返回所有 Delegation
// @params["validator_address"]
// return Delegation
func (s *DPoSStakeRuntime) GetUserDelegationByValidator(context protocol.TxSimContext, params map[string]string) ([]byte, error) {
	// check params
	err := checkParams(params, "delegator_address", "validator_address")
	if err != nil {
		return nil, err
	}

	delegatorAddress := params["delegator_address"]
	validatorAddress := params["validator_address"]

	// 获取验证人数据
	bz, err := getDelegationBytes(context, delegatorAddress, validatorAddress)
	if err != nil {
		s.log.Errorf("get delegation of address [%s] error, error: %s", validatorAddress, err.Error())
		return nil, err
	}
	return bz, nil
}

// Undelegation(from string, amount string) bool	// 解除抵押，更新验证人
// @params["from"] 		解质押的验证人
// @params["amount"] 	解质押数量，1 * 10 ^ 18
// return UnbondingDelegation
func (s *DPoSStakeRuntime) UnDelegate(context protocol.TxSimContext, params map[string]string) ([]byte, error) {
	// check params
	err := checkParams(params, "from", "amount")
	if err != nil {
		return nil, err
	}

	undelegateValidatorAddress := params["from"]
	amount := params["amount"]

	// check amount
	if !assertStringAmountPositive(amount) {
		s.log.Errorf("amount is less than 0")
		return nil, fmt.Errorf("amount is less than 0")
	}
	// read epoch
	bz, err := s.ReadLatestEpoch(context, nil)
	if err != nil {
		s.log.Errorf("undelegate read latest epoch error")
		return nil, err
	}
	// read epoch
	epoch := &commonPb.Epoch{}
	err = proto.Unmarshal(bz, epoch)
	if err != nil {
		s.log.Errorf("undelegate unmarshal latest epoch error")
		return nil, err
	}
	// parse sender
	sender, err := loadSenderAddress(context) // Use ERC20 parse method
	if err != nil {
		s.log.Errorf("get sender address error: ", err.Error())
		return nil, err
	}
	// check epoch undelegate amount
	shares, err := s.checkEpochUnDelegateAmount(context, sender, undelegateValidatorAddress, amount)
	if err != nil {
		s.log.Errorf("check epoch undelegate amount error: ", err.Error())
		return nil, err
	}
	// get completion epochID
	epochNum, err := getUnbondingEpochNumber(context)
	if err != nil {
		s.log.Errorf("get completion epoch error: ", err.Error())
		return nil, err
	}
	completeEpoch := epochNum + epoch.EpochID
	// new entry
	entry := newUnbondingDelegationEntry(epoch.EpochID, completeEpoch, amount)
	// update delegation
	ud, err := getOrCreateUnbondingDelegation(context, completeEpoch, sender, undelegateValidatorAddress)
	if err != nil {
		s.log.Errorf("get or create unbonding delegation error: ", err.Error())
		return nil, err
	}
	ud.Entries = append(ud.Entries, entry)

	// update validator
	v, err := getValidator(context, undelegateValidatorAddress)
	if err != nil {
		s.log.Errorf("get validator [%s] error: ", undelegateValidatorAddress, err.Error())
		return nil, err
	}
	negShare := &big.Int{}
	negShare.Mul(shares, big.NewInt(-1))
	err = updateValidatorShares(v, negShare)
	if err != nil {
		s.log.Errorf("update validator [%s] share error: ", undelegateValidatorAddress, err.Error())
		return nil, err
	}
	err = updateValidatorTokens(v, "-"+amount)
	if err != nil {
		s.log.Errorf("update validator [%s] tokens error: ", undelegateValidatorAddress, err.Error())
		return nil, err
	}
	// 如果是 验证人自身 解除抵押
	if sender == undelegateValidatorAddress {
		err = updateValidatorSelfDelegate(v, "-"+amount)
		if err != nil {
			s.log.Errorf("update validator [%s] self delegation error: ", undelegateValidatorAddress, err.Error())
			return nil, err
		}
		// compare self delegation
		cmp, err := compareMinSelfDelegation(context, v.SelfDelegation)
		if err != nil {
			s.log.Errorf("compare min self delegation error: ", err.Error())
			return nil, err
		}
		if cmp == -1 {
			if v.SelfDelegation == "0" {
				updateValidatorStatus(v, commonPb.BondStatus_Unbonded)
			} else {
				updateValidatorStatus(v, commonPb.BondStatus_Unbonding)
			}
		}
	}

	// update delegation
	d, err := getDelegation(context, sender, undelegateValidatorAddress)
	if err != nil {
		s.log.Errorf("get delegation error: ", err.Error())
		return nil, err
	}
	err = updateDelegateShares(d, negShare)
	if err != nil {
		s.log.Errorf("update delegate shares error: ", err.Error())
		return nil, err
	}

	// save
	err = save(context, ToUnbondingDelegationKey(completeEpoch, sender, undelegateValidatorAddress), ud)
	if err != nil {
		s.log.Errorf("save unbonding delegation error: ", err.Error())
		return nil, err
	}
	err = save(context, ToValidatorKey(undelegateValidatorAddress), v)
	if err != nil {
		s.log.Errorf("save validator error: ", err.Error())
		return nil, err
	}
	err = save(context, ToDelegationKey(sender, undelegateValidatorAddress), d)
	if err != nil {
		s.log.Errorf("save delegation error: ", err.Error())
		return nil, err
	}

	return proto.Marshal(ud)
}

// ReadEpochByID() []ValidatorAddress				// 读取当前世代数据
// return Epoch
func (s *DPoSStakeRuntime) ReadLatestEpoch(context protocol.TxSimContext, params map[string]string) ([]byte, error) {
	bz, err := context.Get(commonPb.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String(), []byte(KeyCurrentEpoch))
	if err != nil {
		return nil, err
	}
	return bz, nil
}

// ReadEpochByID() []ValidatorAddress				// 读取指定ID的世代数据
// @params["epoch_id"] 查询的世代ID
// return Epoch
func (s *DPoSStakeRuntime) ReadEpochByID(context protocol.TxSimContext, params map[string]string) ([]byte, error) {
	// check params
	err := checkParams(params, "epoch_id")
	if err != nil {
		return nil, err
	}

	epochID := params["epoch_id"]
	if !assertStringAmountPositive(epochID) {
		s.log.Errorf("epoch_id less than 0")
		return nil, fmt.Errorf("epoch_id less than 0")
	}
	bz, err := context.Get(commonPb.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String(), ToEpochKey(epochID))
	if err != nil {
		return nil, err
	}
	return bz, nil
}

// 获取或创建 validator
func getOrCreateValidator(context protocol.TxSimContext, delegatorAddress, validatorAddress string) (*commonPb.Validator, error) {
	bz, err := context.Get(commonPb.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String(), ToValidatorKey(validatorAddress))
	if err != nil {
		return nil, err
	}
	v := &commonPb.Validator{}
	if len(bz) > 0 {
		err := proto.Unmarshal(bz, v)
		if err != nil {
			return nil, err
		}
		if delegatorAddress == validatorAddress {
			return v, nil
		}
		if v.Status != commonPb.BondStatus_Bonded || v.Jailed == true {
			return nil, fmt.Errorf("validator in wrong status, jailed: %v, status: %s", v.Jailed, v.Status)
		}
		return v, nil
	} else {
		// 新建 validator 判断
		if delegatorAddress != validatorAddress {
			// 如果是新建 validator, 抵押人被抵押人必须是同一个人
			return nil, fmt.Errorf("no such validator, validator address: %s", validatorAddress)
		} else {
			key := ToNodeIDKey(validatorAddress)
			if bz, err := context.Get(commonPb.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String(), key); err != nil || len(bz) == 0 {
				return nil, fmt.Errorf("not set validator nodeID, you should first set nodeID with validator")
			}
			v = newValidator(validatorAddress)
		}
		return v, nil
	}
}

// 返回 validator 字节数据
func getValidatorBytes(context protocol.TxSimContext, validatorAddress string) ([]byte, error) {
	bz, err := context.Get(commonPb.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String(), ToValidatorKey(validatorAddress))
	if err != nil {
		return nil, err
	}
	if len(bz) <= 0 {
		return nil, fmt.Errorf("no susch validator as address: %s", validatorAddress)
	}
	return bz, nil
}

// 返回 validator 对象
func getValidator(context protocol.TxSimContext, validatorAddress string) (*commonPb.Validator, error) {
	bz, err := context.Get(commonPb.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String(), ToValidatorKey(validatorAddress))
	if err != nil {
		return nil, err
	}
	if len(bz) <= 0 {
		return nil, fmt.Errorf("no susch validator as address: %s", validatorAddress)
	}
	v := &commonPb.Validator{}
	err = proto.Unmarshal(bz, v)
	if err != nil {
		return nil, err
	}
	return v, nil
}

// 更新 validator shares 字段
func updateValidatorShares(validator *commonPb.Validator, shares *big.Int) error {
	validatorSharesValue, err := stringToBigInt(validator.DelegatorShares)
	if err != nil {
		return err
	}
	total := &big.Int{}
	total.Add(shares, validatorSharesValue)
	if total.Cmp(big.NewInt(0)) == -1 {
		return fmt.Errorf("share update result less than 0")
	}
	validator.DelegatorShares = total.String()
	return nil
}

// 更新 validator tokens 字段
func updateValidatorTokens(validator *commonPb.Validator, amount string) error {
	tokensValue, err := stringToBigInt(validator.Tokens)
	if err != nil {
		return err
	}
	amountValue, err := stringToBigInt(amount)
	if err != nil {
		return err
	}

	total := &big.Int{}
	total.Add(amountValue, tokensValue)
	if total.Cmp(big.NewInt(0)) == -1 {
		return fmt.Errorf("token update result less than 0")
	}
	validator.Tokens = total.String()
	return nil
}

// 更新 validator selfDelegation 字段
func updateValidatorSelfDelegate(validator *commonPb.Validator, amount string) error {
	selfDelegationValue, err := stringToBigInt(validator.SelfDelegation)
	if err != nil {
		return err
	}
	amountValue, err := stringToBigInt(amount)
	if err != nil {
		return err
	}

	total := &big.Int{}
	total.Add(amountValue, selfDelegationValue)
	if total.Cmp(big.NewInt(0)) == -1 {
		return fmt.Errorf("self delegation update result less than 0")
	}
	validator.SelfDelegation = total.String()
	return nil
}

// 更新 valudator status 字段
func updateValidatorStatus(validator *commonPb.Validator, status commonPb.BondStatus) {
	validator.Status = status
}

// 根据 amount 计算 share
func calcShareByAmount(tokens string, shares string, amount string) (*big.Int, error) {
	// 将 amount 转换成 int
	var err error
	tokensValue, err := stringToBigInt(tokens)
	if err != nil {
		return nil, err
	}
	sharesValue, err := stringToBigInt(shares)
	if err != nil {
		return nil, err
	}
	amountValue, err := stringToBigInt(amount)
	if err != nil {
		return nil, err
	}

	// 计算 amount 对应的 share 数量
	newShare := &big.Int{}
	if tokensValue.Cmp(big.NewInt(0)) == 0 && sharesValue.Cmp(big.NewInt(0)) == 0 {
		newShare = amountValue
	} else if tokensValue.Cmp(big.NewInt(0)) == 1 {
		// 计算 shares 的数量， new_shares = shares * amount / tokens
		x := newShare.Mul(sharesValue, amountValue)
		newShare = x.Div(x, tokensValue)
		//percentage := decimal.NewFromBigInt(amountValue, 0).Div(decimal.NewFromBigInt(tokensValue, 0))
		//newShare = percentage.Mul(decimal.NewFromBigInt(sharesValue, 0)).BigInt()
	} else if tokensValue.Cmp(big.NewInt(0)) == -1 {
		return nil, fmt.Errorf("validator's token amount is less than 0, token amount: %s", tokensValue.String())
	}
	return newShare, nil
}

// 根据 share 计算 amount
func calcAmountByShare(tokens string, shares string, share string) (*big.Int, error) {
	// 将 amount 转换成 int
	var err error
	tokensValue, err := stringToBigInt(tokens)
	if err != nil {
		return nil, err
	}
	sharesValue, err := stringToBigInt(shares)
	if err != nil {
		return nil, err
	}
	shareValue, err := stringToBigInt(share)
	if err != nil {
		return nil, err
	}

	// 计算 share 对应的 amount 数量
	newAmount := &big.Int{}
	if sharesValue.Cmp(big.NewInt(0)) < 1 {
		return nil, fmt.Errorf("shares less or equal to 0")
	} else {
		// 计算 amount 的数量, amount = share / shares * tokens
		y := newAmount.Mul(sharesValue, tokensValue)
		newAmount = y.Div(shareValue, y)
	}
	return newAmount, nil
}

// 获取或创建 delegation
func getOrCreateDelegation(context protocol.TxSimContext, delegatorAddress, validatorAddress string) (*commonPb.Delegation, error) {
	bz, err := context.Get(commonPb.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String(), ToDelegationKey(delegatorAddress, validatorAddress))
	if err != nil {
		return nil, err
	}
	d := &commonPb.Delegation{}
	if len(bz) > 0 {
		err = proto.Unmarshal(bz, d)
		if err != nil {
			return nil, err
		}
	} else {
		d = newDelegation(delegatorAddress, validatorAddress, "0")
	}
	return d, nil
}

// 获取或创建 delegation
func getDelegationsByAddress(context protocol.TxSimContext, delegatorAddress string) (*commonPb.DelegationInfo, error) {
	// 获取地址所有抵押数据
	iterRange := util.BytesPrefix(ToDelegationPrefix(delegatorAddress))
	// TODO search scope has no memory data
	iter, err := context.Select(commonPb.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String(), iterRange.Start, iterRange.Limit)
	if err != nil {
		return nil, err
	}
	defer iter.Release()

	delegationVector := make([]*commonPb.Delegation, 0)
	for iter.Next() {
		kv, err := iter.Value()
		if err != nil {
			return nil, err
		}
		d := &commonPb.Delegation{}
		err = proto.Unmarshal(kv.GetValue(), d)
		if err != nil {
			return nil, err
		}
		delegationVector = append(delegationVector, d)
	}

	if len(delegationVector) == 0 {
		return nil, fmt.Errorf("address: [%s] has no delegation", delegatorAddress)
	}

	di := &commonPb.DelegationInfo{}
	di.Infos = delegationVector

	return di, nil
}

// 返回 delegation 信息
func getDelegation(context protocol.TxSimContext, delegatorAddress, validatorAddress string) (*commonPb.Delegation, error) {
	bz, err := context.Get(commonPb.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String(), ToDelegationKey(delegatorAddress, validatorAddress))
	if err != nil {
		return nil, err
	}
	d := &commonPb.Delegation{}
	if len(bz) > 0 {
		err = proto.Unmarshal(bz, d)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("no such delegation, delegatorAddress: [%s], validatorAddress: [%s]", delegatorAddress, validatorAddress)
	}
	return d, nil
}

// 返回 delegation 字节数据
func getDelegationBytes(context protocol.TxSimContext, delegatorAddress, validatorAddress string) ([]byte, error) {
	bz, err := context.Get(commonPb.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String(), ToDelegationKey(delegatorAddress, validatorAddress))
	if err != nil {
		return nil, err
	}
	if len(bz) <= 0 {
		return nil, fmt.Errorf("no susch validator as address: %s", validatorAddress)
	}
	return bz, nil
}

// 更新 delegate share 字段
func updateDelegateShares(delegate *commonPb.Delegation, shares *big.Int) error {
	sharesValue, err := stringToBigInt(delegate.Shares)
	if err != nil {
		return err
	}
	total := &big.Int{}
	total.Add(shares, sharesValue)
	if total.Cmp(big.NewInt(0)) == -1 {
		return fmt.Errorf("delegate share update result less than 0")
	}
	delegate.Shares = total.String()
	return nil
}

// 获取或创建 unbonding delegation
func getOrCreateUnbondingDelegation(context protocol.TxSimContext, epochID uint64, delegatorAddress, validatorAddress string) (*commonPb.UnbondingDelegation, error) {
	bz, err := context.Get(commonPb.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String(), ToUnbondingDelegationKey(epochID, delegatorAddress, validatorAddress))
	if err != nil {
		return nil, err
	}
	ud := &commonPb.UnbondingDelegation{}
	if len(bz) > 0 {
		err = proto.Unmarshal(bz, ud)
		if err != nil {
			return nil, err
		}
	} else {
		ud = newUnbondingDelegation(epochID, delegatorAddress, validatorAddress)
	}
	return ud, nil
}

// 获取 unbonding delegation
func getUnbondingDelegation(context protocol.TxSimContext, epochID uint64, delegatorAddress, validatorAddress string) (*commonPb.UnbondingDelegation, error) {
	bz, err := context.Get(commonPb.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String(), ToUnbondingDelegationKey(epochID, delegatorAddress, validatorAddress))
	if err != nil {
		return nil, err
	}
	ud := &commonPb.UnbondingDelegation{}
	if len(bz) > 0 {
		err = proto.Unmarshal(bz, ud)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("no such unbonding delegation, epochID: [%d], delegatorAddress: [%s],validatorAddress: [%s]", epochID, delegatorAddress, validatorAddress)
	}
	return ud, nil
}

// 检查 当前 epoch 总的 undelegation 数量
func (s *DPoSStakeRuntime) checkEpochUnDelegateAmount(context protocol.TxSimContext, delegatorAddress string, validatorAddress string, amount string) (*big.Int, error) {
	// get delegate
	d, err := getDelegation(context, delegatorAddress, validatorAddress)
	if err != nil {
		s.log.Errorf("get delegation error, delegationAddress: [%s], validatorAddress: [%s]", delegatorAddress, validatorAddress)
		return nil, err
	}
	// get validator
	v, err := getValidator(context, validatorAddress)
	if err != nil {
		s.log.Errorf("get validator error, validatorAddress: [%s]", validatorAddress)
		return nil, err
	}
	// calc share by amount
	shareAmount, err := calcShareByAmount(v.Tokens, v.DelegatorShares, amount)
	if err != nil {
		s.log.Errorf("calculate share by amount error: ", err)
		return nil, err
	}
	// convert share string to big int
	delegateShareValue, err := stringToBigInt(d.Shares)
	if err != nil {
		s.log.Errorf("convert delegate share string to big int error: ", err)
		return nil, err
	}
	if delegateShareValue.Cmp(shareAmount) == -1 {
		return nil, fmt.Errorf("delegate less than undelegate amount")
	}
	return shareAmount, nil
}

// 返回所有验证人
func getAllValidatorByPrefix(context protocol.TxSimContext, prefix []byte) ([]*commonPb.Validator, error) {
	// 获取所有验证人数据
	iterRange := util.BytesPrefix(prefix)
	iter, err := context.Select(commonPb.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String(), iterRange.Start, iterRange.Limit)
	if err != nil {
		return nil, err
	}
	defer iter.Release()
	validatorVector := make([]*commonPb.Validator, 0)
	for iter.Next() {
		kv, err := iter.Value()
		if err != nil {
			return nil, err
		}
		v := &commonPb.Validator{}
		err = proto.Unmarshal(kv.GetValue(), v)
		if err != nil {
			return nil, err
		}
		validatorVector = append(validatorVector, v)
	}

	return validatorVector, nil
}

// 返回 unstake 完成需要的 epoch 数
func getUnbondingEpochNumber(context protocol.TxSimContext) (uint64, error) {
	bz, err := context.Get(commonPb.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String(), []byte(KeyCompletionUnbondingEpochNumber))
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint64(bz), nil
}

// 获得最少抵押数量的基础配置
func getMinSelfDelegation(context protocol.TxSimContext) (*big.Int, error) {
	bz, err := context.Get(commonPb.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String(), []byte(KeyMinSelfDelegation))
	if err != nil {
		return nil, err
	}
	m, ok := big.NewInt(0).SetString(string(bz), 10)
	if !ok {
		return nil, fmt.Errorf("invalid minSelfDelegation in stakeContract config")
	}
	return m, nil
}

// 保存 message 对象
func save(context protocol.TxSimContext, key []byte, m proto.Message) error {
	bz, err := proto.Marshal(m)
	if err != nil {
		return err
	}
	err = context.Put(commonPb.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String(), key, bz)
	if err != nil {
		return err
	}
	return nil
}

// 将 数字字符串 转换为 big.Int
func stringToBigInt(amount string) (*big.Int, error) {
	v := &big.Int{}
	v, ok := v.SetString(amount, 10) // only support 10 count base number
	if !ok {
		return nil, fmt.Errorf("convert amount to big int error: %s", amount)
	}
	return v, nil
}

// 检查入参
func checkParams(params map[string]string, keys ...string) error {
	if params == nil {
		return fmt.Errorf("params is nil")
	}
	for _, key := range keys {
		if params[key] == "" {
			return fmt.Errorf("params has no such key: [%s]", key)
		}
	}
	return nil
}

// selfDelegation > minSelfDelegation 	: 1
// selfDelegation == minSelfDelegation 	: 0
// selfDelegation < minSelfDelegation 	: -1
func compareMinSelfDelegation(context protocol.TxSimContext, selfDelegation string) (int, error) {
	minSelfDelegation, err := getMinSelfDelegation(context)
	if err != nil {
		return 0, err
	}
	selfDelegationValue, err := stringToBigInt(selfDelegation)
	if err != nil {
		return 0, err
	}
	return selfDelegationValue.Cmp(minSelfDelegation), nil
}

// check amount params
// amount > 0 return true else false
func assertStringAmountPositive(amount string) bool {
	amountValue, err := stringToBigInt(amount)
	if err != nil {
		return false
	}
	return amountValue.Cmp(big.NewInt(0)) >= 0
}
