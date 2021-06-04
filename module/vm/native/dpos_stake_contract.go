/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package native

import (
	"chainmaker.org/chainmaker-go/logger"
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/protocol"
	"crypto/sha256"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/mr-tron/base58"
	"github.com/shopspring/decimal"
	"github.com/syndtr/goleveldb/leveldb/util"
	"math/big"
)

// Key: validatorPrefix + ValidatorAddress
func toValidatorKey(ValidatorAddress string) string {
	return commonPb.StakePrefix_Prefix_Validator.String() + ValidatorAddress
}

// Key: delegationPrefix + DelegatorAddress + ValidatorAddress
func toDelegationKey(DelegatorAddress, ValidatorAddress string) string {
	return commonPb.StakePrefix_Prefix_Delegation.String() + DelegatorAddress + ValidatorAddress
}

// Key：epochPrefix + EpochID
func toEpochKey(epochID string) string {
	return commonPb.StakePrefix_Prefix_Epoch_Record.String() + epochID
}

// Key：unbondPrefix + DelegatorID + ValidatorID
func toUnbondingDelegationKey(DelegatorID, ValidatorID string) string {
	return commonPb.StakePrefix_Prefix_Unbond.String() + DelegatorID + ValidatorID
}

func newValidator(validatorAddress string) *commonPb.Validator {
	return &commonPb.Validator{
		ValidatorAddress: validatorAddress,
		Jailed: false,
		Status: commonPb.BondStatus_Unbonded,
		Tokens: "0",
		DelegatorShares: "0",
		UnbondingEpochID: 0,
		UnbondingCompletionEpochID: 0,
		SelfDelegation: "0",
	}
}

func newDelegation(delegatorAddress, validatorAddress string, shares string) *commonPb.Delegation {
	return &commonPb.Delegation{
		DelegatorAddress: delegatorAddress,
		ValidatorAddress: validatorAddress,
		Shares: shares,
	}
}

func newUnbondingDelegation(DelegatorAddress, ValidatorAddress string) *commonPb.UnbondingDelegation {
	UnbondingDelegationEntry := make([]*commonPb.UnbondingDelegationEntry, 0)
	return &commonPb.UnbondingDelegation{
		DelegatorAddress: DelegatorAddress,
		ValidatorAddress: ValidatorAddress,
		Entries: UnbondingDelegationEntry,
	}
}

func newUnbondingDelegationEntry(CreationEpochID, UnbondedEpochID uint64, InitialBalance, Balance string) *commonPb.UnbondingDelegationEntry {
	return &commonPb.UnbondingDelegationEntry{
		CreationEpochID: CreationEpochID,
		UnbondedEpochID: UnbondedEpochID,
		CompletionEpochID: -1,
		InitialBalance: InitialBalance,
		Balance: Balance,
	}
}

// main implement here
type DPosStakeContract struct {
	methods map[string]ContractFunc
	log     *logger.CMLogger
}

func (d *DPosStakeContract) getMethod(methodName string) ContractFunc {
	return d.methods[methodName]
}

func newDPosStakeContract(log *logger.CMLogger) *DPosStakeContract {
	return &DPosStakeContract{
		log:     log,
		methods: registerDPosStakeContractMethods(log),
	}
}

func registerDPosStakeContractMethods(log *logger.CMLogger) map[string]ContractFunc {
	queryMethodMap := make(map[string]ContractFunc, 64)
	// implement
	DPosStakeRuntime := &DPosStakeRuntime{log: log}
	queryMethodMap[commonPb.DPoSStakeContractFunction_GET_ALL_VALIDATOR.String()] = DPosStakeRuntime.GetAllValidator
	queryMethodMap[commonPb.DPoSStakeContractFunction_DELEGATE.String()] = DPosStakeRuntime.Delegation
	queryMethodMap[commonPb.DPoSStakeContractFunction_UNDELEGATE.String()] = DPosStakeRuntime.Undelegation
	queryMethodMap[commonPb.DPoSStakeContractFunction_READ_EPOCH_BY_ID.String()] = DPosStakeRuntime.ReadEpochByID
	queryMethodMap[commonPb.DPoSStakeContractFunction_READ_LATEST_EPOCH.String()] = DPosStakeRuntime.ReadLatestEpochByID

	return queryMethodMap
}

type DPosStakeRuntime struct {
	log *logger.CMLogger
}

// * GetAllValidator() []ValidatorAddress		// 返回所有满足最低抵押条件验证人
// return ValidatorVector
func (s *DPosStakeRuntime) GetAllValidator(context protocol.TxSimContext, params map[string]string) ([]byte, error) {
	// 获取验证人数据
	vc, err := getAllValidatorByPrefix(context, commonPb.StakePrefix_Prefix_Validator.String())
	if err != nil {
		s.log.Error("get validator address error")
		return nil, err
	}

	// 过滤
	minSelfDelegation, err := getMinSelfDelegation(context)
	if err != nil {
		s.log.Error("get min self delegation error: ", err.Error())
		return nil, err
	}
	collection := &commonPb.ValidatorVector{}
	for _, v := range vc {
		if v == nil {
			s.log.Errorf("validator [%s] is nil", v)
			continue
		}
		value, err := stringToBigInt(v.SelfDelegation)
		if err != nil {
			s.log.Errorf("convert self delegate string to integer error, amount: %s", v.SelfDelegation)
			return nil, fmt.Errorf("convert self delegate string to integer error, amount: %s", v.SelfDelegation)
		}
		if v.Jailed == true || value.Cmp(minSelfDelegation) == -1 || v.Status != commonPb.BondStatus_Bonded {
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

// * Delegation(to string, amount string) (delegation, error)		// 创建抵押，更新验证人，如果MsgSender是给自己，即给自己抵押，则创建验证人
// @params["to"] 		抵押的目标验证人
// @params["amount"]	抵押数量，带decimal
// return Delegation
func (s *DPosStakeRuntime) Delegation(context protocol.TxSimContext, params map[string]string) ([]byte, error) {
	to      := params["to"]		// delegate target
	amount  := params["amount"]	// amount must be a integer

	// 获取最小抵押的全局变量
	minSelfDelegation, err := getMinSelfDelegation(context)
	if err != nil {
		s.log.Error("get min self delegation error: ", err.Error())
		return nil, err
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
		selfDelegationValue, err := stringToBigInt(v.SelfDelegation)
		if err != nil {
			s.log.Errorf("get or create validator error: ", err.Error())
			return nil, err
		}
		if v.Status == commonPb.BondStatus_Unbonded && selfDelegationValue.Cmp(minSelfDelegation) == 1 {
			updateValidatorStatus(v, commonPb.BondStatus_Bonded)
		}
	}

	// 跨合约转账
	// 获取 runtime 对象
	erc20RunTime := NewDPoSRuntime(s.log)
	// stake 地址
	stakeAddrHash := sha256.Sum256([]byte(commonPb.ContractName_SYSTEM_CONTRACT_DPOS_ERC20.String()))
	stakeAddr := base58.Encode(stakeAddrHash[:])
	// prepare params
	transferParams := map[string]string{
		"to": stakeAddr,
		"value": amount,
	}
	_, err = erc20RunTime.Transfer(context, transferParams)
	if err != nil {
		s.log.Errorf("cross call contract ERC20, method transfer error: ", err.Error())
		return nil, err
	}

	// 写入存储
	err = save(context, toDelegationKey(from, to), d)
	if err != nil {
		s.log.Errorf("save delegate error: ", err.Error())
		return nil, err
	}
	err = save(context, toValidatorKey(to), v)
	if err != nil {
		s.log.Errorf("save validator error: ", err.Error())
		return nil, err
	}

	// return Delegate info
	return proto.Marshal(d)
}

// * Undelegation(from string, amount string) bool	// 解除抵押，更新验证人
//@params["from"] 		解质押的验证人
//@params["amount"] 	解质押数量
//return
func (s *DPosStakeRuntime) Undelegation(context protocol.TxSimContext, params map[string]string) ([]byte, error) {
	undelegateValidatorAddress := params["from"]
	amount := params["amount"]
	// read epoch
	bz, err := s.ReadLatestEpochByID(context, nil)
	if err != nil {
		s.log.Errorf("undelegate read latest epoch error")
		return nil, err
	}
	epoch := &commonPb.Epoch{}
	err = proto.Unmarshal(bz, epoch)
	if err != nil {
		s.log.Errorf("undelegate read latest epoch error")
		return nil, err
	}
	// parse sender
	sender, err := loadSenderAddress(context) // Use ERC20 parse method
	if err != nil {
		s.log.Errorf("get sender address error: ", err.Error())
		return nil, err
	}
	// read balance
	erc20RunTime := NewDPoSRuntime(s.log)
	bz, err = erc20RunTime.BalanceOf(context, map[string]string{"owner": sender} )
	if err != nil {
		s.log.Errorf("get sender balance error: ", err.Error())
		return nil, err
	}
	currentBalance, err := stringToBigInt(string(bz))
	if err != nil {
		s.log.Errorf("get sender balance error: ", err.Error())
		return nil, err
	}
	amountValue, err := stringToBigInt(amount)
	if err != nil {
		s.log.Errorf("get sender balance error: ", err.Error())
		return nil, err
	}
	total := &big.Int{}
	total.Add(amountValue, currentBalance)
	// new entry
	entry := newUnbondingDelegationEntry(epoch.EpochID, epoch.EpochID + 1, currentBalance.String(), total.String())
	// update delegation
	ud, err := getOrCreateUnbondingDelegation(context, sender, undelegateValidatorAddress)
	if err != nil {
		s.log.Errorf("get or create unbonding delegation error: ", err.Error())
		return nil, err
	}
	ud.Entries = append(ud.Entries, entry)
	// get UnbondingDelegationQueue
	udq, err := getUnbondingDelegationQueue(context)
	if err != nil {
		s.log.Errorf("get unbonding delegation queue error: ", err.Error())
		return nil, err
	}
	udq.Queue = append(udq.Queue, ud)
	// save UnbondingDelegationQueue
	err = save(context, commonPb.StakePrefix_Prefix_UnbondingDelegationQueue.String(), udq)
	if err != nil {
		s.log.Errorf("get unbonding delegation queue error: ", err.Error())
		return nil, err
	}
	return proto.Marshal(ud)
}

// * ReadEpochByID() []ValidatorAddress				// 读取当前世代数据
// return Epoch
func (s *DPosStakeRuntime) ReadLatestEpochByID(context protocol.TxSimContext, params map[string]string) ([]byte, error) {
	bz, err := context.Get(commonPb.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String(), []byte(commonPb.StakePrefix_Prefix_Curr_Epoch.String()))
	if err != nil {
		return nil, err
	}
	return bz, nil
}

// * ReadEpochByID() []ValidatorAddress				// 读取指定ID的世代数据
//@params["epoch_id"] 查询的世代ID
//return Epoch
func (s *DPosStakeRuntime) ReadEpochByID(context protocol.TxSimContext, params map[string]string) ([]byte, error) {
	epochID := params["epoch_id"]

	bz, err := context.Get(commonPb.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String(), []byte(toEpochKey(epochID)))
	if err != nil {
		return nil, err
	}
	return bz, nil
}

// 获取或创建 validator
func getOrCreateValidator(context protocol.TxSimContext, delegatorAddress, validatorAddress string) (*commonPb.Validator, error) {
	bz, err := context.Get(commonPb.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String(), []byte(toValidatorKey(validatorAddress)))
	if err != nil {
		return nil, err
	}
	v := &commonPb.Validator{}
	if len(bz) > 0 {
		err := proto.Unmarshal(bz, v)
		if err != nil {
			return nil, err
		}
		if delegatorAddress == validatorAddress && v.Status != commonPb.BondStatus_Unbonding {
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
			v = newValidator(validatorAddress)
		}
		return v, nil
	}
}

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

func updateValidatorStatus(validator *commonPb.Validator, status commonPb.BondStatus) {
	validator.Status = status
}

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
		percentage := decimal.NewFromBigInt(amountValue, 0).Div(decimal.NewFromBigInt(tokensValue, 0))
		newShare = percentage.Mul(decimal.NewFromBigInt(sharesValue, 0)).BigInt()
	} else if tokensValue.Cmp(big.NewInt(0)) == -1 {
		return nil, fmt.Errorf("validator's token amount is less than 0, token amount: %s", tokensValue.String())
	}
	return newShare, nil
}

// 获取或创建 delegation
func getOrCreateDelegation(context protocol.TxSimContext, delegatorAddress, validatorAddress string) (*commonPb.Delegation, error) {
	bz, err := context.Get(commonPb.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String(), []byte(toDelegationKey(delegatorAddress, validatorAddress)))
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

func updateDelegateShares(delegate *commonPb.Delegation, shares *big.Int) error {
	sharesValue, err := stringToBigInt(delegate.Shares)
	if err != nil {

	}
	total := &big.Int{}
	total.Add(shares, sharesValue)
	if total.Cmp(big.NewInt(0)) == -1 {
		return fmt.Errorf("delegate share update result less than 0")
	}
	delegate.Shares = total.String()
	return nil
}

// 获取或创建 delegation
func getOrCreateUnbondingDelegation(context protocol.TxSimContext, delegatorAddress, validatorAddress string) (*commonPb.UnbondingDelegation, error) {
	bz, err := context.Get(commonPb.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String(), []byte(toUnbondingDelegationKey(delegatorAddress, validatorAddress)))
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
		ud = newUnbondingDelegation(delegatorAddress, validatorAddress)
	}
	return ud, nil
}

// 返回所有验证人
func getAllValidatorByPrefix(context protocol.TxSimContext, prefix string) ([]*commonPb.Validator, error) {
	// 获取所有验证人数据
	iterRange := util.BytesPrefix([]byte(prefix))
	iter, err := context.Select(commonPb.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String(), iterRange.Start, iterRange.Limit)
	if err != nil {
		return nil, err
	}
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

// 获得最少抵押数量的基础配置
func getMinSelfDelegation(context protocol.TxSimContext) (*big.Int, error) {
	bz, err := context.Get(commonPb.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String(), []byte(commonPb.StakePrefix_Prefix_MinSelfDelegation.String()))
	if err != nil {
		return nil, err
	}
	v, err := stringToBigInt(string(bz))
	if err != nil {
		return nil, err
	}
	return v, nil
}

func getUnbondingDelegationQueue(context protocol.TxSimContext) (*commonPb.UnbondingDelegationQueue, error) {
	bz, err := context.Get(commonPb.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String(), []byte(commonPb.StakePrefix_Prefix_UnbondingDelegationQueue.String()))
	if err != nil {
		return nil, err
	}
	udq := &commonPb.UnbondingDelegationQueue{}
	err = proto.Unmarshal(bz, udq)
	if err != nil {
		return nil, err
	}
	return udq, nil
}

func save(context protocol.TxSimContext, key string, m proto.Message) error {
	bz, err := proto.Marshal(m)
	if err != nil {
		return err
	}
	err = context.Put(commonPb.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String(), []byte(key), bz)
	if err != nil {
		return err
	}
	return nil
}

func stringToBigInt(amount string) (*big.Int, error) {
	v := &big.Int{}
	v, ok := v.SetString(amount, 0)
	if !ok {
		return nil, fmt.Errorf("convert amount to big int error: %s", amount)
	}
	return v, nil
}