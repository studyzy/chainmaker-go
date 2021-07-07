/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package native

import (
	"chainmaker.org/chainmaker/pb-go/accesscontrol"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"chainmaker.org/chainmaker-go/logger"
	"chainmaker.org/chainmaker-go/utils"
	commonPb "chainmaker.org/chainmaker/pb-go/common"
	"chainmaker.org/chainmaker/protocol"
	"github.com/mr-tron/base58/base58"
)

const (
	//DecBase = 10

	paramNameOwner = "owner"
	paramNameFrom  = "from"
	paramNameTo    = "to"
	paramNameValue = "value"

	// Balance:map[Account]Value
	KeyBalanceFormat = "B/%s"
	// map[sender][to]Value
	KeyApproveFormat = "A/%s/%s"
	// The Key of total Supply
	KeyTotalSupply = "TS"
	// The Key of Decimals，value：uint32
	KeyDecimals = "DEC"
	// The Key of owner for contract，value：string
	KeyOwner = "OWN"
)

var (
	dposErc20ContractName = commonPb.ContractName_SYSTEM_CONTRACT_DPOS_ERC20.String()
)

type DPoSERC20Contract struct {
	methods map[string]ContractFunc
	log     *logger.CMLogger
}

func newDPoSERC20Contract(log *logger.CMLogger) *DPoSERC20Contract {
	return &DPoSERC20Contract{
		log:     log,
		methods: registerDPoSERC20ContractMethods(log),
	}
}

func (c *DPoSERC20Contract) getMethod(methodName string) ContractFunc {
	return c.methods[methodName]
}

func registerDPoSERC20ContractMethods(log *logger.CMLogger) map[string]ContractFunc {
	methodMap := make(map[string]ContractFunc, 64)
	// [DPoS]
	dposRuntime := NewDPoSRuntime(log)
	methodMap[commonPb.DPoSERC20ContractFunction_GET_BALANCEOF.String()] = dposRuntime.BalanceOf
	methodMap[commonPb.DPoSERC20ContractFunction_TRANSFER.String()] = dposRuntime.Transfer
	//methodMap[commonPb.DPoSERC20ContractFunction_TRANSFER_FROM.String()] = dposRuntime.TransferFrom
	//methodMap[commonPb.DPoSERC20ContractFunction_GET_ALLOWANCE.String()] = dposRuntime.Allowance
	//methodMap[commonPb.DPoSERC20ContractFunction_APPROVE.String()] = dposRuntime.Approve
	methodMap[commonPb.DPoSERC20ContractFunction_MINT.String()] = dposRuntime.Mint
	//methodMap[commonPb.DPoSERC20ContractFunction_BURN.String()] = dposRuntime.Burn
	//methodMap[commonPb.DPoSERC20ContractFunction_TRANSFER_OWNERSHIP.String()] = dposRuntime.TransferOwnership
	methodMap[commonPb.DPoSERC20ContractFunction_GET_OWNER.String()] = dposRuntime.Owner
	methodMap[commonPb.DPoSERC20ContractFunction_GET_DECIMALS.String()] = dposRuntime.Decimals
	methodMap[commonPb.DPoSERC20ContractFunction_GET_TOTAL_SUPPLY.String()] = dposRuntime.Total
	return methodMap
}

// [DPoS]
type DPoSRuntime struct {
	log *logger.CMLogger
}

// NewDPoSRuntime
func NewDPoSRuntime(log *logger.CMLogger) *DPoSRuntime {
	return &DPoSRuntime{log: log}
}

// BalanceOf return balance(token) of owner
// params["owner"]:${owner}
// return balance of ${owner}
func (r *DPoSRuntime) BalanceOf(txSimContext protocol.TxSimContext, params map[string]string) (result []byte, err error) {
	if params == nil {
		return nil, ErrParamsEmpty
	}
	if owner, ok := params[paramNameOwner]; ok {
		if owner == "" {
			r.log.Errorf("contract[%s] param [%s] is nil", dposErc20ContractName, paramNameOwner)
			return nil, ErrParams
		}
		bigInteger, err := balanceOf(txSimContext, owner)
		if err != nil {
			r.log.Errorf("load balance of owner[%s] error: %s", owner, err.Error())
			return nil, err
		}
		if bigInteger == nil {
			bigInteger = utils.NewZeroBigInteger()
		}
		return []byte(bigInteger.String()), nil
	}
	return nil, fmt.Errorf("can not find param, contract[%s] param[%s]", dposErc20ContractName, paramNameOwner)
}

// Transfer
// It is equal with transfer(to string, value uint64) for ETH
// params["to"]:${to}
// params["value"]:${value}
// return token value of ${sender} after transfer
func (r *DPoSRuntime) Transfer(txSimContext protocol.TxSimContext, params map[string]string) (result []byte, err error) {
	if params == nil {
		return nil, ErrParamsEmpty
	}
	if to, ok := params[paramNameTo]; ok {
		if value, ok := params[paramNameValue]; ok {
			val, err := loadAndCheckValue(value)
			if err != nil {
				// 转账的值不正确
				r.log.Errorf("contract[%s] param [%s] is illegal", dposErc20ContractName, paramNameValue)
				return nil, err
			}
			// 获取当前用户
			from, err := loadSenderAddress(txSimContext)
			if err != nil {
				r.log.Errorf("contract[%s] load sender address failed, %s", dposErc20ContractName, err.Error())
				return nil, err
			}
			result, err := transfer(txSimContext, from, to, val)
			if err != nil {
				r.log.Errorf("transfer from [%s] to [%s] failed: %s", from, to, err.Error())
			}
			return result, err
		}
		return nil, fmt.Errorf("can not find param, contract[%s] param[%s]", dposErc20ContractName, paramNameValue)
	}
	return nil, fmt.Errorf("can not find param, contract[%s] param[%s]", dposErc20ContractName, paramNameTo)
}

// TransferFrom 交易的发送者，从from的账号中转移指定数目的token给to账户
// 该操作需要approve，即from已经提前允许当前用户允许转移操作
// params["from"]:${from}
// params["to"]:${to}
// params["value"]:${value}
// return token value of ${from} after transfer
func (r *DPoSRuntime) TransferFrom(txSimContext protocol.TxSimContext, params map[string]string) (result []byte, err error) {
	if params == nil {
		return nil, ErrParamsEmpty
	}
	if to, ok := params[paramNameTo]; ok {
		if from, ok := params[paramNameFrom]; ok {
			if value, ok := params[paramNameValue]; ok {
				val, err := loadAndCheckValue(value)
				if err != nil {
					// 转账的值不正确
					r.log.Errorf("contract[%s] param [%s] is illegal", dposErc20ContractName, paramNameValue)
					return nil, err
				}
				// 获取当前用户
				sender, err := loadSenderAddress(txSimContext)
				if err != nil {
					r.log.Errorf("contract[%s] load sender address failed, %s", dposErc20ContractName, err.Error())
					return nil, err
				}
				// 检查当前用户是否获得授权
				approveVal, err := approveValue(txSimContext, from, sender)
				if err != nil {
					r.log.Errorf("load approve from[%s] for[%s] failed, %s", from, sender, err.Error())
					return nil, err
				}
				if approveVal == nil {
					r.log.Errorf("load approve from[%s] for[%s] is zero", from, sender)
					return nil, fmt.Errorf("load approve from[%s] for[%s] is zero", from, sender)
				}
				// 判断授权是否满足
				if approveVal.Cmp(val) < 0 {
					r.log.Errorf("address approve is not enough, contract[%s] sender[%s] from[%s] approve[%s] value[%s]",
						dposErc20ContractName, sender, from, approveVal.String(), val.String())
					return nil, fmt.Errorf("address approve is not enough, contract[%s] sender[%s] from[%s] approve[%s] value[%s]",
						dposErc20ContractName, sender, from, approveVal.String(), val.String())
				}
				// 将授权值重置，然后进行转账操作
				newApproveVal := utils.Sub(approveVal, val)
				err = setApproveValue(txSimContext, from, sender, newApproveVal)
				if err != nil {
					return nil, err
				}
				return transfer(txSimContext, from, to, val)
			}
			return nil, fmt.Errorf("can not find param, contract[%s] param[%s]", dposErc20ContractName, paramNameValue)
		}
		return nil, fmt.Errorf("can not find param, contract[%s] param[%s]", dposErc20ContractName, paramNameFrom)
	}
	return nil, fmt.Errorf("can not find param, contract[%s] param[%s]", dposErc20ContractName, paramNameTo)
}

// Approve
// It is equal with approve(spender string, value uint64) error; for ETH
// params["to"]:${to}
// params["value"]:${value}
// return token value for ${sender} to ${to}
func (r *DPoSRuntime) Approve(txSimContext protocol.TxSimContext, params map[string]string) (result []byte, err error) {
	if params == nil {
		return nil, ErrParamsEmpty
	}
	if approveTo, ok := params[paramNameTo]; ok {
		// 判断value是否合法
		if value, ok := params[paramNameValue]; ok {
			val, err := loadAndCheckValue(value)
			if err != nil {
				// 授权的值不正确
				r.log.Errorf("contract[%s] param [%s] is illegal", dposErc20ContractName, paramNameValue)
				return nil, err
			}
			// 获取当前用户
			from, err := loadSenderAddress(txSimContext)
			if err != nil {
				r.log.Errorf("contract[%s] load sender address failed, %s", dposErc20ContractName, err.Error())
				return nil, err
			}
			err = setApproveValue(txSimContext, from, approveTo, val)
			if err != nil {
				return nil, err
			}
			return []byte(val.String()), nil
		}
		return nil, fmt.Errorf("can not find param, contract[%s] param[%s]", dposErc20ContractName, paramNameValue)
	}
	return nil, fmt.Errorf("can not find param, contract[%s] param[%s]", dposErc20ContractName, paramNameTo)
}

// Mint 增发
// params["to"]:${to}
// params["value"]:${value}
// return newest token of ${to} after mint
func (r *DPoSRuntime) Mint(txSimContext protocol.TxSimContext, params map[string]string) (result []byte, err error) {
	if params == nil {
		return nil, ErrParamsEmpty
	}
	if mintTo, ok := params[paramNameTo]; ok {
		if value, ok := params[paramNameValue]; ok {
			val, err := loadAndCheckValue(value)
			if err != nil {
				// 增发的值不正确
				r.log.Errorf("contract[%s] param [%s] is illegal", dposErc20ContractName, paramNameValue)
				return nil, err
			}
			// 获取当前用户
			from, err := loadSenderAddress(txSimContext)
			if err != nil {
				r.log.Errorf("contract[%s] load sender address failed, %s", dposErc20ContractName, err.Error())
				return nil, err
			}
			// 当前用户必须是Owner
			owner, err := owner(txSimContext)
			if err != nil {
				r.log.Errorf("load contract[%s] owner failed, err: %s", dposErc20ContractName, err.Error())
				return nil, fmt.Errorf("load contract[%s] owner failed, err: %s", dposErc20ContractName, err.Error())
			}
			if !strings.EqualFold(string(owner), from) {
				r.log.Errorf("contract[%s]'s owner is not sender, owner[%s] sender[%s]",
					dposErc20ContractName, string(owner), from)
				return nil, fmt.Errorf("contract[%s]'s owner is not sender, owner[%s] sender[%s]",
					dposErc20ContractName, string(owner), from)
			}
			// 增发即增加具体值和总量
			// 获取总量
			totalSupply, err := totalSupply(txSimContext)
			if err != nil {
				r.log.Errorf("load contract[%s] total supply failed, err: %s", dposErc20ContractName, err.Error())
				return nil, fmt.Errorf("load contract[%s] total supply failed, err: %s", dposErc20ContractName, err.Error())
			}
			if totalSupply == nil {
				// 默认设置为0
				totalSupply = utils.NewZeroBigInteger()
			}
			// 获取增发用户原始的值
			toBalance, err := balanceOf(txSimContext, mintTo)
			if err != nil {
				r.log.Errorf("load to address balance error, contract[%s] address[%s]", dposErc20ContractName, mintTo)
				return nil, fmt.Errorf("load to address balance error, contract[%s] address[%s]", dposErc20ContractName, mintTo)
			}
			// 重设总量
			newTotalSupply := utils.Sum(totalSupply, val)
			// 增发给具体用户
			newToBalance := utils.Sum(toBalance, val)
			// 写入到数据库
			err = txSimContext.Put(dposErc20ContractName, []byte(totalSupplyKey()), []byte(newTotalSupply.String()))
			if err != nil {
				return nil, fmt.Errorf("txSimContext put failed, err: %s", err.Error())
			}
			err = txSimContext.Put(dposErc20ContractName, []byte(BalanceKey(mintTo)), []byte(newToBalance.String()))
			if err != nil {
				return nil, fmt.Errorf("txSimContext put failed, err: %s", err.Error())
			}
			// 返回增发后该账户的值
			return []byte(newToBalance.String()), nil
		}
		return nil, fmt.Errorf("can not find param, contract[%s] param[%s]", dposErc20ContractName, paramNameValue)
	}
	return nil, fmt.Errorf("can not find param, contract[%s] param[%s]", dposErc20ContractName, paramNameTo)
}

// Burn 交易的发送者燃烧一定数量的代币，最多可燃烧殆尽
// params["value"]:${value}
// return balance of sender after burn
func (r *DPoSRuntime) Burn(txSimContext protocol.TxSimContext, params map[string]string) (result []byte, err error) {
	if params == nil {
		return nil, ErrParamsEmpty
	}
	if value, ok := params[paramNameValue]; ok {
		val, err := loadAndCheckValue(value)
		if err != nil {
			// 燃烧的值不正确
			r.log.Errorf("contract[%s] param [%s] is illegal", dposErc20ContractName, paramNameValue)
			return nil, err
		}
		// 获取当前用户
		from, err := loadSenderAddress(txSimContext)
		if err != nil {
			r.log.Errorf("contract[%s] load sender address failed, %s", dposErc20ContractName, err.Error())
			return nil, err
		}
		// 获取当前用户的token数量
		beforeFromBalance, err := balanceOf(txSimContext, from)
		if err != nil {
			r.log.Errorf("load address balance error, contract[%s] address[%s]", dposErc20ContractName, from)
			return nil, fmt.Errorf("load address balance error, contract[%s] address[%s]", dposErc20ContractName, from)
		}
		if beforeFromBalance == nil {
			r.log.Errorf("load address balance error which is zero, contract[%s] address[%s]", dposErc20ContractName, from)
			return nil, fmt.Errorf("load address balance error which is zero, contract[%s] address[%s]", dposErc20ContractName, from)
		}
		// 处理总量
		beforeTotalSupply, err := totalSupply(txSimContext)
		if err != nil {
			r.log.Errorf("load contract[%s] total supply failed, err: %s", dposErc20ContractName, err.Error())
			return nil, fmt.Errorf("load contract[%s] total supply failed, err: %s", dposErc20ContractName, err.Error())
		}
		// 检查是否会燃烧殆尽
		var afterTotalSupply, afterFromBalance *utils.BigInteger
		// 检查总量是否够燃烧
		if beforeTotalSupply.Cmp(val) < 0 {
			r.log.Errorf("total supply is not enough for burn, before[%s] burn-value[%s]", beforeTotalSupply.String(), val.String())
			return nil, fmt.Errorf("total supply is not enough for burn, before[%s] burn-value[%s]", beforeTotalSupply.String(), val.String())
		}
		// 计算燃烧后的值
		afterTotalSupply = utils.Sub(beforeTotalSupply, val)
		// 检查当前账号总量是否够燃烧
		if beforeFromBalance.Cmp(val) < 0 {
			r.log.Errorf("address[%s] is not enough for burn, before[%s] burn-value[%s]", from, beforeTotalSupply.String(), val.String())
			return nil, fmt.Errorf("address[%s] is not enough for burn, before[%s] burn-value[%s]", from, beforeTotalSupply.String(), val.String())
		}
		afterFromBalance = utils.Sub(beforeFromBalance, val)
		// 重置总量和当前账号的值
		// 写入到数据库
		err = txSimContext.Put(dposErc20ContractName, []byte(totalSupplyKey()), []byte(afterTotalSupply.String()))
		if err != nil {
			return nil, fmt.Errorf("txSimContext put failed, err: %s", err.Error())
		}
		err = txSimContext.Put(dposErc20ContractName, []byte(BalanceKey(from)), []byte(afterFromBalance.String()))
		if err != nil {
			return nil, fmt.Errorf("txSimContext put failed, err: %s", err.Error())
		}
		// 返回燃烧后的值
		return []byte(afterFromBalance.String()), nil
	}
	return nil, fmt.Errorf("can not find param, contract[%s] param[%s]", dposErc20ContractName, paramNameValue)
}

// TransferOwnership 转移拥有者给其他账户
// params["to"]:${to}
// return new owner
func (r *DPoSRuntime) TransferOwnership(txSimContext protocol.TxSimContext, params map[string]string) (result []byte, err error) {
	if params == nil {
		return nil, ErrParamsEmpty
	}
	if to, ok := params[paramNameTo]; ok {
		// 获取当前用户
		from, err := loadSenderAddress(txSimContext)
		if err != nil {
			r.log.Errorf("contract[%s] load sender address failed, %s", dposErc20ContractName, err.Error())
			return nil, err
		}
		// 当前用户必须是Owner
		owner, err := owner(txSimContext)
		if err != nil {
			r.log.Errorf("load contract[%s] owner failed, err: %s", dposErc20ContractName, err.Error())
			return nil, fmt.Errorf("load contract[%s] owner failed, err: %s", dposErc20ContractName, err.Error())
		}
		if !strings.EqualFold(string(owner), from) {
			r.log.Errorf("contract[%s]'s owner is not sender, owner[%s] sender[%s]",
				dposErc20ContractName, string(owner), from)
			return nil, fmt.Errorf("contract[%s]'s owner is not sender, owner[%s] sender[%s]",
				dposErc20ContractName, string(owner), from)
		}
		// 将owner设置魏to
		// 返回新的owner
		return setOwner(txSimContext, to)
	}
	return nil, fmt.Errorf("can not find param, contract[%s] param[%s]", dposErc20ContractName, paramNameTo)
}

// Allowance
// params["from"]:${from}
// params["to"]:${to}
// return value of approve
func (r *DPoSRuntime) Allowance(txSimContext protocol.TxSimContext, params map[string]string) (result []byte, err error) {
	if params == nil {
		return nil, ErrParamsEmpty
	}
	if to, ok := params[paramNameTo]; ok {
		// 检查是否有from
		var fromAddress string
		if from, ok := params[paramNameFrom]; ok {
			fromAddress = from
		} else {
			// 获取当前用户
			sender, err := loadSenderAddress(txSimContext)
			if err != nil {
				r.log.Errorf("contract[%s] load sender address failed, %s", dposErc20ContractName, err.Error())
				return nil, err
			}
			fromAddress = sender
		}
		return allowance(txSimContext, fromAddress, to)
	}
	return nil, fmt.Errorf("can not find param, contract[%s] param[%s]", dposErc20ContractName, paramNameTo)
}

// Owner
// return owner of DPoS
func (r *DPoSRuntime) Owner(txSimContext protocol.TxSimContext, params map[string]string) (result []byte, err error) {
	return owner(txSimContext)
}

// Total
// return total supply of tokens
func (r *DPoSRuntime) Total(txSimContext protocol.TxSimContext, params map[string]string) (result []byte, err error) {
	total, err := totalSupply(txSimContext)
	if err != nil {
		return nil, err
	}
	return []byte(total.String()), nil
}

// Decimals
// return decimals of DPoS
func (r *DPoSRuntime) Decimals(txSimContext protocol.TxSimContext, params map[string]string) (result []byte, err error) {
	decimalsBytes, err := txSimContext.Get(dposErc20ContractName, []byte(KeyDecimals))
	if err != nil {
		r.log.Errorf("load [%s] from cache failed, %s", KeyDecimals, err.Error())
		return nil, err
	}
	return decimalsBytes, nil
}

// setOwner set owner of contract
func (r *DPoSRuntime) setOwner(txSimContext protocol.TxSimContext, owner string) error {
	return txSimContext.Put(dposErc20ContractName, []byte(KeyOwner), []byte(owner))
}

// setDecimals set decimals of contract
func (r DPoSRuntime) setDecimals(txSimContext protocol.TxSimContext, decimals string) error {
	return txSimContext.Put(dposErc20ContractName, []byte(KeyDecimals), []byte(decimals))
}

func BalanceKey(account string) string {
	return fmt.Sprintf(KeyBalanceFormat, account)
}

func approveKey(account, to string) string {
	return fmt.Sprintf(KeyApproveFormat, account, to)
}

func totalSupplyKey() string {
	return KeyTotalSupply
}

func setOwner(txSimContext protocol.TxSimContext, owner string) ([]byte, error) {
	err := txSimContext.Put(dposErc20ContractName, []byte(KeyOwner), []byte(owner))
	if err != nil {
		return nil, err
	}
	return []byte(owner), nil
}

func owner(txSimContext protocol.TxSimContext) ([]byte, error) {
	owner, err := txSimContext.Get(dposErc20ContractName, []byte(KeyOwner))
	if err != nil {
		return nil, err
	}
	return owner, err
}

func totalSupply(txSimContext protocol.TxSimContext) (*utils.BigInteger, error) {
	totalSupply, err := txSimContext.Get(dposErc20ContractName, []byte(totalSupplyKey()))
	if err != nil {
		return nil, err
	}
	return utils.NewBigInteger(string(totalSupply)), nil
}

func approveValue(txSimContext protocol.TxSimContext, from, to string) (*utils.BigInteger, error) {
	approveKey := approveKey(from, to)
	valueBytes, err := txSimContext.Get(dposErc20ContractName, []byte(approveKey))
	if err != nil {
		return nil, err
	}
	return utils.NewBigInteger(string(valueBytes)), nil
}

func allowance(txSimContext protocol.TxSimContext, from, to string) ([]byte, error) {
	approveKey := approveKey(from, to)
	return txSimContext.Get(dposErc20ContractName, []byte(approveKey))
}

func setApproveValue(txSimContext protocol.TxSimContext, from, to string, val *utils.BigInteger) error {
	approveKey := approveKey(from, to)
	// 无需关注之前的结果，直接更新当前信息即可
	err := txSimContext.Put(dposErc20ContractName, []byte(approveKey), []byte(val.String()))
	if err != nil {
		return fmt.Errorf("txSimContext put failed, err: %s", err.Error())
	}
	return nil
}

func loadSenderAddress(txSimContext protocol.TxSimContext) (string, error) {
	sender := txSimContext.GetSender()
	if sender != nil {
		// 将sender转换为用户地址
		var member []byte
		if sender.MemberType==accesscontrol.MemberType_CERT {
			// 长证书
			member = sender.MemberInfo
		} else if sender.MemberType==accesscontrol.MemberType_CERT_HASH{
			// 短证书
			memberInfoHex := hex.EncodeToString(sender.MemberInfo)
			certInfo, err := getWholeCertInfo(txSimContext, memberInfoHex)
			if err != nil {
				return "", fmt.Errorf("can not load whole cert info , contract[%s] member[%s]", dposErc20ContractName, memberInfoHex)
			}
			member = certInfo.Cert
		}else {
			return "",errors.New("invalid member type")
		}
		return parseUserAddress(member)
	}
	return "", fmt.Errorf("can not find sender from tx, contract[%s]", dposErc20ContractName)
}

// parseUserAddress
func parseUserAddress(member []byte) (string, error) {
	certificate, err := utils.ParseCert(member)
	if err != nil {
		msg := fmt.Errorf("parse cert failed, name[%s] err: %+v", dposErc20ContractName, err)
		return "", msg
	}
	pubKeyBytes, err := certificate.PublicKey.Bytes()
	if err != nil {
		msg := fmt.Errorf("load public key from cert failed, name[%s] err: %+v", dposErc20ContractName, err)
		return "", msg
	}
	// 转换为SHA-256
	addressBytes := sha256.Sum256(pubKeyBytes)
	return base58.Encode(addressBytes[:]), nil
}

func getWholeCertInfo(txSimContext protocol.TxSimContext, certHash string) (*commonPb.CertInfo, error) {
	certBytes, err := txSimContext.Get(commonPb.ContractName_SYSTEM_CONTRACT_CERT_MANAGE.String(), []byte(certHash))
	if err != nil {
		return nil, err
	}
	return &commonPb.CertInfo{
		Hash: certHash,
		Cert: certBytes,
	}, nil
}

func balanceOf(txSimContext protocol.TxSimContext, address string) (*utils.BigInteger, error) {
	balanceKey := BalanceKey(address)
	balanceBytes, err := txSimContext.Get(dposErc20ContractName, []byte(balanceKey))
	if err != nil {
		msg := fmt.Errorf("txSimContext get failed, name[%s] key[%s] err: %+v", dposErc20ContractName, balanceKey, err)
		return nil, msg
	}
	if len(balanceBytes) == 0 {
		return utils.NewZeroBigInteger(), nil
	}
	return utils.NewBigInteger(string(balanceBytes)), nil
}

func loadAndCheckValue(value string) (*utils.BigInteger, error) {
	val := utils.NewBigInteger(value)
	if val == nil {
		// 转账的值不正确
		return nil, fmt.Errorf("param is error, contract[%s] param[%s]", dposErc20ContractName, paramNameValue)
	}
	if val.Cmp(utils.NewZeroBigInteger()) <= 0 {
		// 转账的值不能<=0
		return nil, fmt.Errorf("param is illegal, contract[%s] param[%s]", dposErc20ContractName, paramNameValue)
	}
	return val, nil
}

func transfer(txSimContext protocol.TxSimContext, from, to string, val *utils.BigInteger) ([]byte, error) {
	// 获取双方的Money
	fromBalance, err := balanceOf(txSimContext, from)
	if err != nil {
		return nil, fmt.Errorf("load from address balance error, contract[%s] address[%s]", dposErc20ContractName, from)
	}
	if fromBalance == nil {
		return nil, fmt.Errorf("load from address balance error which is zero, contract[%s] address[%s]", dposErc20ContractName, from)
	}
	// 判断其值是否满足
	if fromBalance.Cmp(val) < 0 {
		// 账户剩余的钱不满足需求
		return nil, fmt.Errorf("address balance is not enough, contract[%s] address[%s] balance[%s] value[%s]",
			dposErc20ContractName, from, fromBalance.String(), val.String())
	}
	toBalance, err := balanceOf(txSimContext, to)
	if err != nil {
		return nil, fmt.Errorf("load to address balance error, contract[%s] address[%s]", dposErc20ContractName, to)
	}
	if toBalance == nil {
		toBalance = utils.NewZeroBigInteger()
	}
	// 同一账户转账
	if from == to {
		return []byte(fromBalance.String()), nil
	}
	// 不同账户间转账
	// 记录原始值
	beforeSum := utils.Sum(fromBalance, toBalance)
	// 进行转账操作
	afterFromBalance, afterToBalance := utils.Sub(fromBalance, val), utils.Sum(toBalance, val)
	// 判断操作后值是否一致
	afterSum := utils.Sum(afterFromBalance, afterToBalance)
	if beforeSum.Cmp(afterSum) != 0 {
		// 前后值存在问题
		return nil, fmt.Errorf("balance is not equal before and after operation, contract[%s] before-balance[%s] after-balance[%s]",
			dposErc20ContractName, beforeSum.String(), afterSum.String())
	}
	// 一致的情况下更新存储
	fromBalanceKey, toBalanceKey := BalanceKey(from), BalanceKey(to)
	err = txSimContext.Put(dposErc20ContractName, []byte(fromBalanceKey), []byte(afterFromBalance.String()))
	if err != nil {
		return nil, fmt.Errorf("txSimContext put failed, err: %s", err.Error())
	}
	err = txSimContext.Put(dposErc20ContractName, []byte(toBalanceKey), []byte(afterToBalance.String()))
	if err != nil {
		return nil, fmt.Errorf("txSimContext put failed, err: %s", err.Error())
	}
	return []byte(afterFromBalance.String()), nil
}
