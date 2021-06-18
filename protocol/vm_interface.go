/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

// define vm parameter and interface
package protocol

import (
	"bytes"
	"chainmaker.org/chainmaker-go/pb/protogo/common"
	"fmt"
	"regexp"
)

const (
	GasLimit            = 1e10    // invoke user contract max gas
	TimeLimit           = 1 * 1e9 // 1s
	CallContractGasOnce = 1e5     // Gas consumed per cross call contract
	CallContractDepth   = 5       // cross call contract stack depth, must less than vm pool min size
	EvmGasPrice         = 1
	EvmMaxStackDepth    = 1024

	ContractSdkSignalResultSuccess = 0 // sdk call chain method success result
	ContractSdkSignalResultFail    = 1 // sdk call chain method success result

	DefaultStateLen   = 64                  // key & name for contract state length
	DefaultStateRegex = "^[a-zA-Z0-9._-]+$" // key & name for contract state regex

	DefaultVersionLen   = 64                  // key & name for contract state length
	DefaultVersionRegex = "^[a-zA-Z0-9._-]+$" // key & name for contract state regex

	ParametersKeyMaxCount    = 50 //
	ParametersValueMaxLength = 1024 * 1024

	TopicMaxLen       = 255
	EventDataMaxLen   = 65535
	EventDataMaxCount = 16

	ContractKey            = ":K:"
	ContractByteHeader     = ":H:"
	ContractByteCode       = ":B:"
	ContractVersion        = ":V:"
	ContractRuntimeType    = ":R:"
	ContractCreator        = ":C:"
	ContractFreeze         = ":F:"
	ContractRevoke         = ":RV:"
	ContractAddress        = ":A:"
	ContractStoreSeparator = "#"

	// special parameters passed to contract
	ContractCreatorOrgIdParam = "__creator_org_id__"
	ContractCreatorRoleParam  = "__creator_role__"
	ContractCreatorPkParam    = "__creator_pk__"
	ContractSenderOrgIdParam  = "__sender_org_id__"
	ContractSenderRoleParam   = "__sender_role__"
	ContractSenderPkParam     = "__sender_pk__"
	ContractBlockHeightParam  = "__block_height__"
	ContractTxIdParam         = "__tx_id__"
	ContractContextPtrParam   = "__context_ptr__"

	// user contract must implement such method
	ContractInitMethod        = "init_contract"
	ContractUpgradeMethod     = "upgrade"
	ContractAllocateMethod    = "allocate"
	ContractDeallocateMethod  = "deallocate"
	ContractRuntimeTypeMethod = "runtime_type"
	ContractEvmParamKey       = "data"
	// method name used by smart contract sdk
	// common
	ContractMethodLogMessage      = "LogMessage"
	ContractMethodSuccessResult   = "SuccessResult"
	ContractMethodErrorResult     = "ErrorResult"
	ContractMethodCallContract    = "CallContract"
	ContractMethodCallContractLen = "CallContractLen"
	// kv
	ContractMethodGetStateLen = "GetStateLen"
	ContractMethodGetState    = "GetState"
	ContractMethodPutState    = "PutState"
	ContractMethodDeleteState = "DeleteState"
	// kv iterator author:whang1234
	ContractMethodKvIterator        = "KvIterator"
	ContractMethodKvIteratorHasNext = "KvIteratorHasNext"
	ContractMethodKvIteratorNextLen = "KvIteratorNextLen"
	ContractMethodKvIteratorNext    = "KvIteratorNext"
	ContractMethodKvIteratorClose   = "KvIteratorClose"
	// sql
	ContractMethodExecuteQuery       = "ExecuteQuery"
	ContractMethodExecuteQueryOne    = "ExecuteQueryOne"
	ContractMethodExecuteQueryOneLen = "ExecuteQueryOneLen"
	ContractMethodRSNext             = "RSNext"
	ContractMethodRSNextLen          = "RSNextLen"
	ContractMethodRSHasNext          = "RSHasNext"
	ContractMethodRSClose            = "RSClose"
	ContractMethodExecuteUpdate      = "ExecuteUpdate"
	ContractMethodExecuteDdl         = "ExecuteDDL"
	ContractMethodEmitEvent          = "EmitEvent"
	// paillier
	ContractMethodGetPaillierOperationResult    = "GetPaillierOperationResult"
	ContractMethodGetPaillierOperationResultLen = "GetPaillierOperationResultLen"
	PaillierOpTypeAddCiphertext                 = "AddCiphertext"
	PaillierOpTypeAddPlaintext                  = "AddPlaintext"
	PaillierOpTypeSubCiphertext                 = "SubCiphertext"
	PaillierOpTypeSubPlaintext                  = "SubPlaintext"
	PaillierOpTypeNumMul                        = "NumMul"
	// bulletproofs
	ContractMethodGetBulletproofsResult     = "GetBulletproofsResult"
	ContractMethodGetBulletproofsResultLen  = "GetBulletproofsResultLen"
	BulletProofsOpTypePedersenAddNum        = "PedersenAddNum"
	BulletProofsOpTypePedersenAddCommitment = "PedersenAddCommitment"
	BulletProofsOpTypePedersenSubNum        = "PedersenSubNum"
	BulletProofsOpTypePedersenSubCommitment = "PedersenSubCommitment"
	BulletProofsOpTypePedersenMulNum        = "PedersenMulNum"
	BulletProofsVerify                      = "BulletProofsVerify"
)

//VmManager manage vm runtime
type VmManager interface {
	// GetAccessControl get accessControl manages policies and principles
	GetAccessControl() AccessControlProvider
	// GetChainNodesInfoProvider get ChainNodesInfoProvider provide base node info list of chain.
	GetChainNodesInfoProvider() ChainNodesInfoProvider
	// RunContract run native or user contract according ContractName in contractId, and call the specified function
	RunContract(contractId *common.ContractId, method string, byteCode []byte, parameters map[string]string,
		txContext TxSimContext, gasUsed uint64, refTxType common.TxType) (*common.ContractResult, common.TxStatusCode)
}

type ContractWacsiCommon interface {
	LogMessage() int32
	SuccessResult() int32
	ErrorResult() int32
	CallContract() int32
}

type ContractWacsiKV interface {
	ContractWacsiCommon
	GetState() int32
	PutState() int32
	DeleteState() int32
	KvIterator() int32
	KvIteratorClose() int32
	KvIteratorNext() int32
	KvIteratorHasNext() int32
}

type ContractWacsiSQL interface {
	ContractWacsiCommon
	ExecuteQuery() int32
	ExecuteQueryOne() int32
	RSHasNext() int32
	RSNext() int32
	RSClose() int32
	ExecuteUpdate() int32
	ExecuteDDL() int32
}

// GetKeyStr get state key from string
func GetKeyStr(key string, field string) []byte {
	return GetKey([]byte(key), []byte(field))
}

// GetKey get state key from byte
func GetKey(key []byte, field []byte) []byte {
	var buf bytes.Buffer
	buf.Write(key)
	if len(field) > 0 {
		buf.Write([]byte(ContractStoreSeparator))
		buf.Write(field)
	}
	return buf.Bytes()
}

// CheckKeyFieldStr verify param
func CheckKeyFieldStr(key string, field string) error {
	{
		s := key
		if len(s) > DefaultStateLen {
			return fmt.Errorf("key[%s] too long", s)
		}
		match, err := regexp.MatchString(DefaultStateRegex, s)
		if err != nil || !match {
			return fmt.Errorf("key[%s] can only consist of numbers, dot, letters and underscores", s)
		}
	}
	{
		s := field
		if len(s) == 0 {
			return nil
		}
		if len(s) > DefaultStateLen {
			return fmt.Errorf("key field[%s] too long", s)
		}
		match, err := regexp.MatchString(DefaultStateRegex, s)
		if err != nil || !match {
			return fmt.Errorf("key field[%s] can only consist of numbers, dot, letters and underscores", s)
		}
	}
	return nil
}

//CheckTopicStr
func CheckTopicStr(topic string) error {
	topicLen := len(topic)
	if topicLen == 0 {
		return fmt.Errorf("topic can not empty")
	}
	if topicLen > TopicMaxLen {
		return fmt.Errorf("topic too long,longer than %v", TopicMaxLen)
	}
	return nil

}

//CheckEventTopicTableData  verify event data
func CheckEventData(eventData []string) error {

	eventDataNum := len(eventData)
	if eventDataNum == 0 {
		return fmt.Errorf("event data can not empty")

	}
	if eventDataNum > EventDataMaxCount {
		return fmt.Errorf("too many event data")
	}
	for _, data := range eventData {
		if len(data) > EventDataMaxLen {
			return fmt.Errorf("event data too long,longer than %v", EventDataMaxLen)
		}
	}
	return nil

}
