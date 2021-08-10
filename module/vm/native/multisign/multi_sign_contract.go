/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package multisign

import (
	"bytes"
	"chainmaker.org/chainmaker-go/utils"
	"chainmaker.org/chainmaker-go/vm/native/common"
	commonPb "chainmaker.org/chainmaker/pb-go/common"
	"chainmaker.org/chainmaker/pb-go/syscontract"
	"chainmaker.org/chainmaker/protocol"
	"crypto/md5"
	"errors"
	"fmt"
	"github.com/gogo/protobuf/proto"
)

var (
	contractName              = syscontract.SystemContract_MULTI_SIGN.String()
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
	methodMap[syscontract.MultiSignFunction_REQ.String()] = runtime.Req
	methodMap[syscontract.MultiSignFunction_VOTE.String()] = runtime.Vote
	methodMap[syscontract.MultiSignFunction_QUERY.String()] = runtime.Query
	return methodMap
}

type MultiSignRuntime struct {
	log protocol.Logger
}

// Req request to multi sign
func (r *MultiSignRuntime) Req(txSimContext protocol.TxSimContext, parameters map[string][]byte) (result []byte, err error) {
	// 1、校验并获取参数
	sysContractName := parameters[syscontract.MultiReq_SYS_CONTRACT_NAME.String()]
	sysMethod := parameters[syscontract.MultiReq_SYS_METHOD.String()]
	if utils.IsAnyBlank(sysContractName, sysMethod) {
		err = fmt.Errorf("multi req params verify fail. sysContractName/sysMethod cannot be empty")
		return nil, err
	}

	tx := txSimContext.GetTx()
	r.log.Infof("multi sign req start. ContractName[%s] Method[%s] ParamsLen[%d]", sysContractName, sysMethod, len(tx.Payload.Parameters))
	{
		// md5 payload log
		bytes, _ := tx.Payload.Marshal()
		m5 := fmt.Sprintf("%x", md5.Sum(bytes))
		r.log.Debugf("multi req payload md5 is %s", m5)
	}
	multiSignInfo := &syscontract.MultiSignInfo{
		Payload:      tx.Payload,
		ContractName: string(sysContractName),
		Method:       string(sysMethod),
		Status:       syscontract.MultiSignStatus_PROCESSING,
		VoteInfos:    nil,
	}

	// multi sign record
	for _, endorser := range tx.Endorsers {
		multiSignInfo.VoteInfos = append(multiSignInfo.VoteInfos, &syscontract.MultiSignVoteInfo{
			Vote:        syscontract.VoteStatus_AGREE,
			Endorsement: endorser,
		})
	}

	bytes, _ := multiSignInfo.Marshal()
	txSimContext.Put(contractName, []byte(tx.Payload.TxId), bytes)

	r.log.Infof("multi sign req end")
	return []byte(tx.Payload.TxId), nil
}

func (r *MultiSignRuntime) Vote(txSimContext protocol.TxSimContext, parameters map[string][]byte) (result []byte, err error) {
	// 1、检查参数
	// 2、获取历史投票记录
	// 3、判断是否继续可以对该多签交易投票
	// 4、根据传入参数的状态修改多签结果
	// 5、根据结果调用accessControl校验是否认证成功

	voteInfo := &syscontract.MultiSignVoteInfo{}

	voteInfoBytes := parameters[syscontract.MultiVote_VOTE_INFO.String()]
	txId := parameters[syscontract.MultiVote_TX_ID.String()]
	if utils.IsAnyBlank(voteInfoBytes, txId) {
		err = fmt.Errorf("multi vote params verify fail. voteInfo/txId cannot be empty")
		r.log.Warn(err)
		return nil, err
	}
	r.log.Infof("multi sign vote start. MultiVote_TX_ID[%s]", txId)
	err = proto.Unmarshal(voteInfoBytes, voteInfo)
	if err != nil {
		r.log.Warn(err)
		return nil, err
	}

	multiSignInfoBytes, err := txSimContext.Get(contractName, txId)
	if err != nil {
		r.log.Warn(err)
		return nil, err
	}
	if len(multiSignInfoBytes) == 0 {
		return nil, fmt.Errorf("not found tx id from db %s", txId)
	}
	multiSignInfo := &syscontract.MultiSignInfo{}
	proto.Unmarshal(multiSignInfoBytes, multiSignInfo)

	// 校验：该多签是否已完成投票
	if multiSignInfo.Status != syscontract.MultiSignStatus_PROCESSING {
		err = fmt.Errorf("the multi sign[%s] has been completed", txId)
		r.log.Warn(err)
		return nil, err
	}

	// 校验：该用户是否已投票
	for _, info := range multiSignInfo.VoteInfos {
		if bytes.Equal(voteInfo.Endorsement.Signer.MemberInfo, info.Endorsement.Signer.MemberInfo) {
			err = fmt.Errorf("the signer is voted")
			r.log.Warn(err)
			return nil, err
		}
	}

	resourceName := multiSignInfo.ContractName + "-" + multiSignInfo.Method
	ac, err := txSimContext.GetAccessControl()
	if err != nil {
		r.log.Warn(err)
		return nil, err
	}
	mPayloadByte, _ := multiSignInfo.Payload.Marshal()
	// 校验当前签名
	{
		principal, err := ac.CreatePrincipal(resourceName, []*commonPb.EndorsementEntry{voteInfo.Endorsement}, mPayloadByte)
		if err != nil {
			r.log.Warn(err)
			return nil, err
		}
		endorsement, err := ac.GetValidEndorsements(principal)
		if err != nil {
			r.log.Warn(err)
			return nil, err
		}
		if endorsement == nil || len(endorsement) == 0 {
			err = fmt.Errorf("the multi sign vote signature is err, error:%s", err)
			r.log.Error(err)
			return nil, err
		}
		multiSignInfo.VoteInfos = append(multiSignInfo.VoteInfos, voteInfo)
	}

	var (
		contractResultBytes []byte
		contractErr         error
	)
	// 校验多签签名
	{
		endorsers := make([]*commonPb.EndorsementEntry, 0)
		for _, info := range multiSignInfo.VoteInfos {
			endorsers = append(endorsers, info.Endorsement)
		}
		principal, err := ac.CreatePrincipal(resourceName, endorsers, mPayloadByte)
		if err != nil {
			r.log.Warn(err)
			return nil, err
		}
		endorsement, err := ac.GetValidEndorsements(principal) //problem
		if err != nil {
			r.log.Warn(err)
			return nil, err
		}
		if len(endorsement) == 0 {
			err = fmt.Errorf("the multi vote is err, error: %s", err.Error())
			r.log.Warn(err)
			return nil, err
		}
		multiSignVerify, err := ac.VerifyPrincipal(principal)
		if err != nil {
			r.log.Warn("multi sign vote verify fail.")
			r.log.Warn(err)
		}

		if multiSignVerify {
			r.log.Info("multi sign vote verify success.")
			contract := &commonPb.Contract{
				Name:        multiSignInfo.ContractName,
				RuntimeType: commonPb.RuntimeType_NATIVE, // multi sign only support native contract
				Status:      commonPb.ContractStatus_NORMAL,
				Creator:     nil,
			}

			initParam := make(map[string][]byte)
			for _, parameter := range multiSignInfo.Payload.Parameters {
				// is sysContractName or sysMethod continue
				if parameter.Key == syscontract.MultiReq_SYS_CONTRACT_NAME.String() || parameter.Key == syscontract.MultiReq_SYS_METHOD.String() {
					continue
				}
				initParam[parameter.Key] = parameter.Value
			}
			byteCode := initParam[syscontract.InitContract_CONTRACT_BYTECODE.String()]
			contractResult, statusCode := txSimContext.CallContract(contract, multiSignInfo.Method, byteCode, initParam, 0, commonPb.TxType_INVOKE_CONTRACT)
			if statusCode == commonPb.TxStatusCode_SUCCESS {
				contractResultBytes = contractResult.Result
				multiSignInfo.Status = syscontract.MultiSignStatus_ADOPTED
				r.log.Infof("multi sign vote[%s] finished, result: %s", txId, contractResultBytes)
			} else {
				contractErr = errors.New(contractResult.Message)
				multiSignInfo.Status = syscontract.MultiSignStatus_FAILED
				r.log.Warnf("multi sign vote[%s] failed, msg: %s", txId, contractErr)
			}
		}
	}

	// 7、记录状态
	multiSignInfoBytes, err = multiSignInfo.Marshal()
	if err != nil {
		r.log.Error(err)
		return nil, err
	}
	err = txSimContext.Put(contractName, txId, multiSignInfoBytes)
	if err != nil {
		r.log.Error(err)
		return nil, err
	}
	// return must not nil
	if len(contractResultBytes) == 0 {
		contractResultBytes = []byte("vote success")
	}
	r.log.Infof("multi sign vote[%s] end", txId)
	return contractResultBytes, contractErr
}

func (r *MultiSignRuntime) Query(txSimContext protocol.TxSimContext, parameters map[string][]byte) (result []byte, err error) {
	//func (r *MultiSignRuntime) Query(txSimContext protocol.TxSimContext) (result []byte,err error) {
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
