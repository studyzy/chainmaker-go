/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package native

import (
	"chainmaker.org/chainmaker-go/common/json"
	"chainmaker.org/chainmaker-go/logger"
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-sdk-go/pb/protogo/common"
	"crypto/sha256"
	"fmt"
	"github.com/gorilla/context"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/mr-tron/base58"
	"github.com/shopspring/decimal"
	"math/big"
	"reflect"
	"strconv"
)

type BondStatus string

const (
	Bonded 		BondStatus = "Bonded"
	Unbonding 	BondStatus = "Unbonding"
	Unbonded 	BondStatus = "Unbonded"
)

const (
	validatorPrefix 			= "validator"
	delegationPrefix			= "delegation"
	epochPrefix					= "epoch"
	unbondPrefix				= "unbond"
	keyAllValidatorAddress		= "allValidatorAddress"
	keyMinSelfDelegation		= "minSelfDelegation"
	keyUnbondingDelegationQueue	= "unbondingDelegationQueue"
)

type validator struct {
	ValidatorAddress			string		// 验证人地址，由公钥派生: base58.Encode(sha256(pubkey))
	Jailed						bool		// 活性惩罚后是否被移除验证人集合的标记
	Status						BondStatus	// 验证人状态包含 Bonded / Unbonding / Unbonded
	Tokens						*big.Int	// 抵押的 token 数量
	DelegatorShares 			*big.Int	// 抵押物的股权总计
	UnbondingEpochID			int			// 发起解除质押物交易的 Epoch
	UnbondingCompletionEpochID  int			// 解除质押 Epoch
	SelfDelegation       		*big.Int	// 自抵押 token 数
}

func newValidator(validatorAddress string) *validator {
	return &validator{
		ValidatorAddress: validatorAddress,
		Jailed: false,
		Status: Unbonded,
		Tokens: big.NewInt(0),
		DelegatorShares: big.NewInt(0),
		UnbondingEpochID: -1,
		UnbondingCompletionEpochID: -1,
		SelfDelegation: big.NewInt(0),
	}
}

// Key: validatorPrefix + ValidatorAddress
func toValidatorKey(ValidatorAddress string) string {
	return validatorPrefix + ValidatorAddress
}

type delegation struct {
	DelegatorAddress	string		//抵押人的ID
	ValidatorAddress	string		//验证人的ID
	Shares				*big.Int	//抵押股权
}

func newDelegation(delegatorAddress, validatorAddress string, shares *big.Int) *delegation {
	return &delegation{
		DelegatorAddress: delegatorAddress,
		ValidatorAddress: validatorAddress,
		Shares: shares,
	}
}

// Key: delegationPrefix + DelegatorAddress + ValidatorAddress
func toDelegationKey(DelegatorAddress, ValidatorAddress string) string  {
	return delegationPrefix + DelegatorAddress + ValidatorAddress
}

type epoch struct {
	EpochID			int			// 自增ID
	ProposerVector	[]string	// 负责出块的 ValidatorID 数组
}

// Key：epochPrefix + EPOCHID
func toEpochKey(epochID int) string {
	return epochPrefix + strconv.Itoa(epochID)
}

type unbondingDelegation struct {
	DelegatorAddress	string						// 抵押人ID
	ValidatorAddress	string						// 验证人ID
	Entries				[]unbondingDelegationEntry 	// Unbond 记录
}

type unbondingDelegationEntry struct {
	CreationEpochID 	int    	// 创建 Epoch 高度
	UnbondedEpochID 	int     // 退出 Epoch 高度
	CompletionEpochID 	int		// 完成Epoch高度
	InitialBalance 		int		// 解抵押初始金额
	Balance        		int		// 解抵押后余额
}

// Key：unbondPrefix + DelegatorID + ValidatorID
func toUnbondingDelegationKey(DelegatorID, ValidatorID string) string {
	return unbondPrefix + DelegatorID + ValidatorID
}

type unbondingDelegationQueue []unbondingDelegation	 // 顺序执行 Unbond 操作，FIFO

//// Key：UnbondingDelegationQueue
//func toUnbondingDelegationQueueKey() string {
//	return keyUnbondingDelegationQueue
//}

type validatorAddressVector []string // 验证人数组

// Marshal / Unmarshal
func Marshal(o interface{}) ([]byte, error){
	return json.Marshal(o)
}

func Unmarshal(bz []byte, o interface{}) error {
	return json.Unmarshal(bz, o)
}

// main implement here
//
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
	queryMethodMap[commonPb.DposStakeContractFunction_READ_EPOCH.String()] = DPosStakeRuntime.ReadEpoch
	queryMethodMap[commonPb.DposStakeContractFunction_UPDATE_EPOCH.String()] = DPosStakeRuntime.UpdateEpoch

	return queryMethodMap
}

type DPosStakeRuntime struct {
	log *logger.CMLogger
}

// * GetAllValidator() []ValidatorAddress		// 返回所有满足最低抵押条件验证人
func (s *DPosStakeRuntime) GetAllValidator(context protocol.TxSimContext, params map[string]string) ([]byte, error) {
	// 获取验证人数据
	vc, err := getAllValidatorAddress(context)
	if err != nil {
		s.log.Error("get validator address error")
		return nil, err
	}

	// 过滤
	var collection []string
	minSelfDelegation, err := getMinSelfDelegation(context)
	if err != nil {
		s.log.Error("get min self delegation error: ", err.Error())
		return nil, err
	}

	for _, v := range vc {
		validator, err := getValidator(context, v)
		if err != nil {
			s.log.Errorf("get validator [%s] err: ", v, err.Error())
			continue
		}
		if validator == nil {
			s.log.Errorf("validator [%s] is nil", v)
			continue
		}
		if validator.Jailed == true || validator.SelfDelegation.Cmp(minSelfDelegation) == -1 || validator.Status != Bonded {
			continue
		}
		collection = append(collection, validator.ValidatorAddress)
	}

	// 序列化
	bz, err := Marshal(collection)
	if err != nil {
		s.log.Errorf("marshal validator collection error: ", err.Error())
		return nil, err
	}
	return bz, nil
}

// * Delegation(to string, amount int) bool		// 创建抵押，更新验证人，如果MsgSender是给自己，即给自己抵押，则创建验证人
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
	from, err := getSenderAddress(context)
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
		return nil, err
	}
	err = v.updateTokens(amount)
	if err != nil {
		return nil, err
	}
	if from == to {
		err = v.updateSelfDelegate(amount)
		if err != nil {
			return nil, err
		}
		if v.Status == Unbonded && v.SelfDelegation.Cmp(minSelfDelegation) == 1 {
			v.updateStatus(Bonded)
		}
	}

	// 跨合约转账
	contractVersion, err := context.Get(
		commonPb.ContractName_SYSTEM_CONTRACT_DPOS_ERC20.String(),
		[]byte(protocol.ContractVersion + commonPb.ContractName_SYSTEM_CONTRACT_DPOS_ERC20.String())
	)
	if err != nil {
		return nil, err
	}
	contractId := &commonPb.ContractId{
		commonPb.ContractName_SYSTEM_CONTRACT_DPOS_ERC20.String(),
		string(contractVersion),
		commonPb.RuntimeType_GASM,
	}

	context.CallContract(
		contractId,
		"transfer",
		[]byte{},
		map[string]string{
			"to": base58.Encode([]byte{sha256.Sum256([]byte("Stake"))}),
			"value": amount,
		},
		100000000,
		commonPb.TxType_INVOKE_SYSTEM_CONTRACT,
	)

	// 写入存储


	// return Delegate info
	return Marshal(d)
}

// * Undelegation(from string, amount int) bool	// 解除抵押，更新验证人
func (s *DPosStakeRuntime) Undelegation(context protocol.TxSimContext, params map[string]string) ([]byte, error) {

}

// * ReadEpoch() []ValidatorAddress				// 读取世代数据
func (s *DPosStakeRuntime) ReadEpoch(context protocol.TxSimContext, params map[string]string) ([]byte, error) {
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
func (s *DPosStakeRuntime) UpdateEpoch(context protocol.TxSimContext, params map[string]string) ([]byte, error) {
	epochIDStr := params["epoch_id"]
	proposerVector := params["proposer_vector"]

	// check params
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
func getOrCreateValidator(context protocol.TxSimContext, delegatorAddress, validatorAddress string) (*validator, error) {
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
func getAllValidatorAddress(context protocol.TxSimContext) ([]string, error) {
	// 获取所有验证人数据
	bz, err := context.Get(commonPb.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String(), []byte(keyAllValidatorAddress))
	if err != nil {
		return nil, err
	}
	// 反序列化
	vc := validatorAddressVector{}
	err = Unmarshal(bz, vc)
	if err != nil {
		return nil, err
	}
	return vc, nil
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
	v := &big.Int{}
	err = Unmarshal(bz, v)
	if err != nil {
		return nil, err
	}
	return v, nil
}

// 获取消息发送人的公钥，转换成地址 TODO
func getSenderAddress(context protocol.TxSimContext) (string, error) {
	return "", nil
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