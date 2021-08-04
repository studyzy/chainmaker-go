/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package multisign

import (
	"chainmaker.org/chainmaker-go/utils"
	"chainmaker.org/chainmaker-go/vm/native/common"
	commonPb "chainmaker.org/chainmaker/pb-go/common"
	"chainmaker.org/chainmaker/pb-go/syscontract"
	"chainmaker.org/chainmaker/protocol"
	"errors"
	"fmt"
	"github.com/gogo/protobuf/proto"
)

var (
	ContractName              = syscontract.SystemContract_MULTI_SIGN.String()
	KEY_SystemContractPayload = "SystemContractPayload"
)

type MultiSignContract struct {
	methods map[string]common.ContractFunc
	log     protocol.Logger
}

func NewMultiSignContract(log protocol.Logger) *MultiSignContract {
	return &MultiSignContract{
		log:     log,
		methods: InitMultiContractMethods(log),
	}
}
func (c *MultiSignContract) GetMethod(methodName string) common.ContractFunc {
	return c.methods[methodName]
}

func InitMultiContractMethods(log protocol.Logger) map[string]common.ContractFunc {
	methodMap := make(map[string]common.ContractFunc, 64)
	runtime := &MultiSignRuntime{log: log}
	methodMap[syscontract.MultiSignFunction_REQ.String()] = runtime.reqContract
	methodMap[syscontract.MultiSignFunction_VOTE.String()] = runtime.voteContract
	methodMap[syscontract.MultiSignFunction_QUERY.String()] = runtime.queryContract
	return methodMap

}

type MultiSignRuntime struct {
	log protocol.Logger
}

// Req request to multi sign
func (r *MultiSignRuntime) reqContract(txSimContext protocol.TxSimContext, parameters map[string][]byte) (result []byte, err error) {
	// 1、校验并获取参数
	sysContractName := parameters["sysContractName"]
	sysMethod := parameters["sysMethod"]
	if utils.IsAnyBlank(sysContractName, sysMethod) {
		err = fmt.Errorf("params contractName,sysMethod cannot be empty")
		return nil, err
	}

	tx := txSimContext.GetTx()
	//tx.Sender
	multiSignInfo := &syscontract.MultiSignInfo{
		Payload:      tx.Payload,
		ContractName: string(sysContractName),
		Method:       string(sysMethod),
		Status:       syscontract.MultiSignStatus_PROCESSING,
		VoteInfos:    nil,
	}

	for _, endorser := range tx.Endorsers {
		multiSignInfo.VoteInfos = append(multiSignInfo.VoteInfos, &syscontract.MultiSignVoteInfo{
			Vote:        syscontract.VoteStatus(1),
			Endorsement: endorser,
		})
	}

	bytes, _ := multiSignInfo.Marshal()
	txSimContext.Put("multi_sign_contract", []byte(tx.Payload.TxId), bytes) // MultiSignInfo
	r.log.Infof(" multi_sign_contract put %s %d ", tx.Payload.TxId, len(bytes))

	return result, nil
}

func (r *MultiSignRuntime) voteContract(txSimContext protocol.TxSimContext, parameters map[string][]byte) (result []byte, err error) {
	// 1、检查参数
	// 2、获取历史投票记录
	// 3、判断是否继续可以对该多签交易投票
	// 4、根据传入参数的状态修改多签结果
	// 5、根据结果调用accessControl校验是否认证成功
	r.log.Infof(" voteContract VOTE test")
	multiPayload := parameters["multiPayload"]
	reqVoteState := parameters["voteState"]
	signature := parameters["signature"]
	if utils.IsAnyBlank(multiPayload, reqVoteState, signature) {
		err = fmt.Errorf("params multiPayload,voteState,signature cannot be empty")
		return nil, err
	}

	oldPayload := &commonPb.Payload{}
	multiSignInfo := &syscontract.MultiSignInfo{}
	voteState := &syscontract.MultiSignVoteInfo{}

	proto.Unmarshal(multiPayload, oldPayload)
	proto.Unmarshal(reqVoteState, voteState)
	multiSignInfoDB, _ := txSimContext.Get("multi_sign_contract", []byte(oldPayload.TxId)) // MultiSignInfo
	proto.Unmarshal(multiSignInfoDB, multiSignInfo)

	multiSignInfo.VoteInfos = append(multiSignInfo.VoteInfos, voteState)
	r.log.Infof("multi vote[%s] count=%d state=%d(0:agree,1:reject)", oldPayload.TxId, multiSignInfo.VoteInfos, voteState.Vote)
	multiSignInfoByte, _ := multiSignInfo.Marshal()
	txSimContext.Put("multi_sign_contract", []byte(oldPayload.TxId), multiSignInfoByte) // MultiSignInfo

	endorsers := make([]*commonPb.EndorsementEntry, 0)
	for _, info := range multiSignInfo.VoteInfos {
		endorsers = append(endorsers, info.Endorsement)
	}

	// verify access control
	{
		ac, err := txSimContext.GetAccessControl()
		if err != nil {
			r.log.Errorw("txSimContext.GetAccessControl is err", "err", err)
			return nil, err
		}
		resourceId := multiSignInfo.ContractName + "-" + multiSignInfo.Method
		principal, err := ac.CreatePrincipal(resourceId, endorsers, multiPayload)
		if err != nil {
			r.log.Error(err)
			return nil, err
		}
		endorsement, err := ac.GetValidEndorsements(principal)
		if err != nil {
			r.log.Error(err)
			return nil, err
		}
		if len(endorsement) == 0 {
			err = fmt.Errorf("the vote is err")
			r.log.Errorw(err.Error(), "err", err)
			return nil, err
		}
		if flag, err := ac.VerifyPrincipal(principal); err != nil {
			return nil, err
		} else if !flag {
			return nil, nil
		}
	}

	var (
		contractResultBytes []byte
		contractErr         error
	)
	{

		voteFlag := false
		if len(multiSignInfo.VoteInfos) > 2 {
			voteFlag = true
			r.log.Infof("the number of vote is %d", len(multiSignInfo.VoteInfos))
		}

		contract := &commonPb.Contract{
			Name:        multiSignInfo.ContractName,
			RuntimeType: commonPb.RuntimeType_NATIVE, // multi sign only support native contract
			Status:      commonPb.ContractStatus_NORMAL,
			Creator:     nil,
		}

		if voteFlag {
			// 6、调用真实系统合约完成该交易
			initParam := make(map[string][]byte)
			for i := range oldPayload.Parameters {
				// is sysContractName or sysMethod jump
				if oldPayload.Parameters[i].Key == "sysContractName" || oldPayload.Parameters[i].Key == "sysMethod" {
					continue
				}
				initParam[oldPayload.Parameters[i].Key] = oldPayload.Parameters[i].Value
			}
			byteCode := initParam[syscontract.InitContract_CONTRACT_BYTECODE.String()]
			//contractResult, statusCode := txSimContext.CallContract(contractId, methodName, byteCode, parameter, gasUsed, payloadInfo.txType)
			contractResult, statusCode := txSimContext.CallContract(contract, multiSignInfo.Method, byteCode, initParam, 0, commonPb.TxType_INVOKE_CONTRACT)
			if statusCode == commonPb.TxStatusCode_SUCCESS {
				// call success
				contractResultBytes = contractResult.Result
				multiSignInfo.Status = syscontract.MultiSignStatus_ADOPTED
				r.log.Infof("CallContract success")
			} else {
				// call failture
				contractErr = errors.New(contractResult.Message)
				multiSignInfo.Status = syscontract.MultiSignStatus_FAILED
			}
		}
	}

	// 7、记录成功
	multiSingInfoBytes, err := proto.Marshal(multiSignInfo)
	if err != nil {
		r.log.Error(err)
		return nil, err
	}
	err = txSimContext.Put("multi_sign_contract", []byte(oldPayload.TxId), multiSingInfoBytes)
	if err != nil {
		r.log.Error(err)
		return nil, err
	}
	// return must not nil
	if len(contractResultBytes) == 0 {
		contractResultBytes = []byte("vote success")
	}
	return contractResultBytes, contractErr
}

func (r *MultiSignRuntime) queryContract(txSimContext protocol.TxSimContext, parameters map[string][]byte) (result []byte, err error) {
	//func (r *MultiSignRuntime) queryContract(txSimContext protocol.TxSimContext) (result []byte,err error) {
	// 1、校验并获取参数
	txId := txSimContext.GetTx().Payload.TxId
	var payloadHash []byte
	if utils.IsAnyBlank(txId) {
		err = fmt.Errorf("params txIdStr cannot be empty")
		return nil, err
	} else {
		payloadHash, _ = txSimContext.Get(syscontract.SystemContract_MULTI_SIGN.String(), []byte(txId))
	}

	if payloadHash == nil || len(payloadHash) == 0 {
		err = fmt.Errorf("the params of payload_hash is not exist")
		r.log.Error(err.Error(), "err", err)
		return nil, err
	}

	// 2、返回结果

	multiSignInfoBytes, err := txSimContext.Get(syscontract.SystemContract_MULTI_SIGN.String(), payloadHash)
	if err != nil {
		r.log.Error(err)
		return nil, err
	}

	return multiSignInfoBytes, nil
}
