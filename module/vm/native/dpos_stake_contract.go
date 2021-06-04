/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package native

import (
	"chainmaker.org/chainmaker-go/common/serialize"
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
	"strconv"
)

const (
	validatorPrefix 			= "validator"
	delegationPrefix			= "delegation"
	epochPrefix					= "epoch"
	unbondPrefix				= "unbond"
	keyCurrentEpoch				= "currentEpoch"
	keyMinSelfDelegation		= "minSelfDelegation"
	keyValidatorNumber			= "validatorNumber"
	keyEachEpochBlockNumber		= "eachEpochBlockNumber"
	keyUnbondingDelegationQueue	= "unbondingDelegationQueue"
)

// Key: validatorPrefix + ValidatorAddress
func toValidatorKey(ValidatorAddress string) string {
	return validatorPrefix + ValidatorAddress
}

// Key: delegationPrefix + DelegatorAddress + ValidatorAddress
func toDelegationKey(DelegatorAddress, ValidatorAddress string) string {
	return delegationPrefix + DelegatorAddress + ValidatorAddress
}

// Key：epochPrefix + EPOCHID
func toEpochKey(epochID int) string {
	return epochPrefix + strconv.Itoa(epochID)
}

// Key：epochPrefix + EPOCHID
func toCurrentEpochKey() string {
	return keyCurrentEpoch
}

// Key：unbondPrefix + DelegatorID + ValidatorID
func toUnbondingDelegationKey(DelegatorID, ValidatorID string) string {
	return unbondPrefix + DelegatorID + ValidatorID
}

// Key：UnbondingDelegationQueue
func toUnbondingDelegationQueueKey() string {
	return keyUnbondingDelegationQueue
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
	return &commonPb.UnbondingDelegation{
		DelegatorAddress: DelegatorAddress,
		ValidatorAddress: ValidatorAddress,
		Entries: nil,
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

type unbondingDelegationQueue []commonPb.UnbondingDelegationEntry	 // 顺序执行 Unbond 操作，FIFO

type validatorAddressVector []string // 验证人数组

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
	queryMethodMap[commonPb.DposStakeContractFunction_GET_ALL_VALIDATOR.String()] = DPosStakeRuntime.GetAllValidator
	queryMethodMap[commonPb.DposStakeContractFunction_DELEGATE.String()] = DPosStakeRuntime.Delegation
	queryMethodMap[commonPb.DposStakeContractFunction_UNDELEGATE.String()] = DPosStakeRuntime.Undelegation
	queryMethodMap[commonPb.DposStakeContractFunction_READ_EPOCH.String()] = DPosStakeRuntime.ReadEpochByID
	queryMethodMap[commonPb.DposStakeContractFunction_READ_EPOCH.String()] = DPosStakeRuntime.ReadLatestEpochByID
	queryMethodMap[commonPb.DposStakeContractFunction_UPDATE_EPOCH.String()] = DPosStakeRuntime.UpdateEpoch

	return queryMethodMap
}

type DPosStakeRuntime struct {
	log *logger.CMLogger
}

// * GetAllValidator() []ValidatorAddress		// 返回所有满足最低抵押条件验证人
// return ValidatorVector
func (s *DPosStakeRuntime) GetAllValidator(context protocol.TxSimContext, params map[string]string) ([]byte, error) {
	// 获取验证人数据
	vc, err := getAllValidatorByPrefix(context, validatorPrefix)
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
// @to 		抵押的目标验证人
// @amount	抵押数量，带decimal
// return
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
	ok := d.updateShares(shares)
	if !ok {
		s.log.Errorf("share amount less than 0 after update, shares: %s", shares.String())
		return nil, fmt.Errorf("share amount less than 0 after update, shares: %s", shares.String())
	}

	// 更新 validator
	err = v.updateShares(shares)
	if err != nil {
		s.log.Errorf("update shares error: ", err.Error())
		return nil, err
	}
	err = v.updateTokens(amount)
	if err != nil {
		s.log.Errorf("update tokens error: ", err.Error())
		return nil, err
	}
	if from == to {
		err = v.updateSelfDelegate(amount)
		if err != nil {
			s.log.Errorf("update self delegate error: ", err.Error())
			return nil, err
		}
		if v.Status == Unbonded && v.SelfDelegation.Cmp(minSelfDelegation) == 1 {
			v.updateStatus(Bonded)
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
	return Marshal(d)
}

// * Undelegation(from string, amount int) bool	// 解除抵押，更新验证人
//@
func (s *DPosStakeRuntime) Undelegation(context protocol.TxSimContext, params map[string]string) ([]byte, error) {

	newUnbondingDelegationEntry()
	//new(UnbondingDelegationEntry)
	//update(UnbondingDelegation)
	//update(UnbondingDelegationQueue)
	//update(Validator)

}

// * ReadEpochByID() []ValidatorAddress				// 读取当前世代数据
func (s *DPosStakeRuntime) ReadLatestEpochByID(context protocol.TxSimContext, params map[string]string) ([]byte, error) {
	bz, err := context.Get(commonPb.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String(), []byte(keyCurrentEpoch))
	if err != nil {
		return nil, err
	}
	return bz, nil
}

// * ReadEpochByID() []ValidatorAddress				// 读取指定ID的世代数据
//@epoch_id 查询的世代ID
func (s *DPosStakeRuntime) ReadEpochByID(context protocol.TxSimContext, params map[string]string) ([]byte, error) {
	epochID := params["epoch_id"]

	i, err := strconv.Atoi(epochID)
	if err != nil {
		return nil, err
	}
	bz, err := context.Get(commonPb.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String(), []byte(toEpochKey(i)))
	if err != nil {
		return nil, err
	}
	return bz, nil
}

// * UpdateEpoch([]ValidatorAddress) bool		// 更新世代数据
//@epoch_id 更新的世代ID
//@proposer_vector 更新的验证人数组
func (s *DPosStakeRuntime) UpdateEpoch(context protocol.TxSimContext, params map[string]string) ([]byte, error) {
	epochIDStr := params["epoch_id"]
	proposerVector := params["proposer_vector"]

	// 检查验证人数组
	if ok := checkParamBytesType([]byte(proposerVector), []string{}); ok {
		e := &epoch{}
		epochID, err := strconv.Atoi(epochIDStr)
		if err != nil {
			return nil, err
		}
		var v []string
		err = Unmarshal([]byte(proposerVector), v)
		if err != nil {
			return nil, err
		}

		e.EpochID = epochID
		e.ProposerVector = v

		bz, err := Marshal(e)
		if err != nil {
			return nil, err
		}
		err = context.Put(commonPb.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String(), []byte(toEpochKey(epochID)), bz)
		if err != nil {
			return nil, err
		}
		return bz, nil
	} else {
		return nil, fmt.Errorf("param proposer_vector's value unmarshal to []string failed")
	}
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
		if delegatorAddress == validatorAddress {

		}
		if v.Status != Bonded || v.Jailed == true {
			return nil, fmt.Errorf("validator in wrong status, jailed: %v, status: %s", v.Jailed, v.Status)
		}
	} else {
		// 新建 validator 判断
		if delegatorAddress != validatorAddress {
			// 如果是新建 validator, 抵押人被抵押人必须是同一个人
			return nil, fmt.Errorf("no such validator, validator address: %s", validatorAddress)
		} else {
			v = newValidator(validatorAddress)
		}
	}
	return v, nil
}

func (v *validator) updateShares(shares *big.Int) error {
	total := &big.Int{}
	total.Add(shares, v.DelegatorShares)
	if total.Cmp(big.NewInt(0)) == -1 {
		return fmt.Errorf("share update result less than 0")
	}
	v.DelegatorShares = total
	return nil
}

func (v *validator) updateTokens(amount string) error {
	var val big.Int
	value, ok := val.SetString(amount, 0)
	if !ok {
		return fmt.Errorf("updateTokens convert amount string to integer error, amount: %s", amount)
	}
	total := &big.Int{}
	total.Add(value, v.Tokens)
	if total.Cmp(big.NewInt(0)) == -1 {
		return fmt.Errorf("token update result less than 0")
	}
	v.Tokens = total
	return nil
}

func (v *validator) updateSelfDelegate(amount string) error {
	var val big.Int
	value, ok := val.SetString(amount, 0)
	if !ok {
		return fmt.Errorf("convert amount string to integer error, amount: %s", amount)
	}
	total := &big.Int{}
	total.Add(value, v.SelfDelegation)
	if total.Cmp(big.NewInt(0)) == -1 {
		return fmt.Errorf("self delegation update result less than 0")
	}
	v.SelfDelegation = total
	return nil
}

func (v *validator) updateStatus(status BondStatus) {
	v.Status = status
}

func calcShareByAmount(tokens *big.Int, shares *big.Int, amount string) (*big.Int, error) {
	// 将 amount 转换成 int
	var val big.Int
	value, ok := val.SetString(amount, 0)
	if !ok {
		return nil, fmt.Errorf("convert amount string to integer error, amount: %s", amount)
	}
	// 计算 amount 对应的 share 数量
	newShare := &big.Int{}
	if tokens.Cmp(big.NewInt(0)) == 0 && shares.Cmp(big.NewInt(0)) == 0 {
		newShare = value
	} else if tokens.Cmp(big.NewInt(0)) == 1 {
		// 计算 shares 的数量， new_shares = shares * amount / tokens
		percentage := decimal.NewFromBigInt(value, 0).Div(decimal.NewFromBigInt(tokens, 0))
		newShare = percentage.Mul(decimal.NewFromBigInt(shares, 0)).BigInt()
	} else if tokens.Cmp(big.NewInt(0)) == -1 {
		return nil, fmt.Errorf("validator's token amount is less than 0, token amount: %s", tokens.String())
	}
	return newShare, nil
}

// 获取或创建 delegation
func getOrCreateDelegation(context protocol.TxSimContext, delegatorAddress, validatorAddress string) (*delegation, error) {
	bz, err := context.Get(commonPb.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String(), []byte(toDelegationKey(delegatorAddress, validatorAddress)))
	if err != nil {
		return nil, err
	}
	d := &delegation{}
	if len(bz) > 0 {
		err = Unmarshal(bz, d)
		if err != nil {
			return nil, err
		}
	} else {
		d = newDelegation(delegatorAddress, validatorAddress, big.NewInt(0))
	}
	return d, nil
}

func (d *delegation) updateShares(shares *big.Int) bool {
	total := &big.Int{}
	total.Add(shares, d.Shares)
	if total.Cmp(big.NewInt(0)) == -1 {
		return false
	}
	d.Shares = total
	return true
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

// 获得验证人数据
func getValidator(context protocol.TxSimContext, validatorAddress string) (*validator, error) {
	// 获取验证人数据
	bz, err := context.Get(commonPb.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String(), []byte(toValidatorKey(validatorAddress)))
	if err != nil {
		return nil, err
	}

	v := &validator{}
	if len(bz) > 0 {
		err = Unmarshal(bz, v)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, nil
	}

	return v, nil
}

// 获得最少抵押数量的基础配置
func getMinSelfDelegation(context protocol.TxSimContext) (*big.Int, error) {
	bz, err := context.Get(commonPb.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String(), []byte(keyMinSelfDelegation))
	if err != nil {
		return nil, err
	}
	v, err := stringToBigInt(string(bz))
	if err != nil {
		return nil, err
	}
	return v, nil
}

func getEpoch(context protocol.TxSimContext, epochID string) (*epoch, error) {
	i, err := strconv.Atoi(epochID)
	if err != nil {
		return nil, err
	}
	bz, err := context.Get(commonPb.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String(), []byte(toEpochKey(i)))
	if err != nil {
		return nil, err
	}
	e := &epoch{}
	err = Unmarshal(bz, e)
	if err != nil {
		return nil, err
	}
	return e, nil
}

func checkParamBytesType(bz []byte, o interface{}) bool {
	err := Unmarshal(bz, o)
	if err != nil {
		return false
	}
	return true
}

func save(context protocol.TxSimContext, key string, o interface{}) error {
	bz, err := Marshal(o)
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