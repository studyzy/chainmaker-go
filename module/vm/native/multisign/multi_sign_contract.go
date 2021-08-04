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
	"strconv"
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
	syscontractName := parameters["sysContractName"]
	sysMethod := parameters["sysMethod"]
	if utils.IsAnyBlank(syscontractName, sysMethod) {
		err = fmt.Errorf("params contractName,sysMethod cannot be empty")
		return nil, err
	}
	// todo verify
	delete(parameters, "syscontractNameContractName")
	delete(parameters, "sysMethod")

	kvParam := make([]*commonPb.KeyValuePair, 0)
	for k, v := range parameters {
		kvParam = append(kvParam, &commonPb.KeyValuePair{
			Key:   k,
			Value: v,
		})
	}

	payload := txSimContext.GetTx().Payload
	multiSignInfo := &syscontract.MultiSignInfo{
		Payload:      payload,
		ContractName: string(syscontractName),
		Method:       string(sysMethod),
		Parameters:   kvParam,
		Status:       syscontract.MultiSignStatus_PROCESSING,
		VoteInfos:    nil,
	}
	bytes, _ := multiSignInfo.Marshal()
	//key := utils.GetContractDbKey(string(syscontractName))
	txSimContext.Put("multi_sign_contract", []byte(payload.TxId), bytes) // MultiSignInfo
	//txSimContext.Put("multi_sign_contract", key, bytes)// MultiSignInfo
	r.log.Infof(" multisigncontract test1")

	return result, nil
}

func (r *MultiSignRuntime) voteContract(txSimContext protocol.TxSimContext, parameters map[string][]byte) (result []byte, err error) {
	// 1、检查参数
	//multiSignInfo := new(commonPb.MultiSignInfo)
	//status:=multiSignInfo.Status
	//voteInfo:=multiSignInfo.VoteInfos
	//voteInfoBytes:=parameters["vote_info"]
	r.log.Infof(" multisigncontract VOTE test")
	p := &commonPb.Payload{}
	reqPayload := parameters["payload"]
	proto.Unmarshal(reqPayload, p)

	m := &syscontract.MultiSignInfo{}
	mbyte, _ := txSimContext.Get("multi_sign_contract", []byte(p.TxId)) // MultiSignInfo
	proto.Unmarshal(mbyte, m)

	voteState := parameters["voteState"]
	Signature := parameters["Signature"]
	if utils.IsAnyBlank(voteState, Signature) {
		err = fmt.Errorf("params voteState,Signature cannot be empty")
		return nil, err
	}
	votestate, _ := strconv.Atoi(string(voteState))
	m.VoteInfos = append(m.VoteInfos, &syscontract.MultiSignVoteInfo{
		Vote: syscontract.VoteStatus(votestate),
		Endorsement: &commonPb.EndorsementEntry{
			Signature: Signature,
			Signer:    txSimContext.GetTx().Sender.Signer,
		}})
	b, _ := m.Marshal()
	txSimContext.Put("multi_sign_contract", []byte(p.TxId), b) // MultiSignInfo

	es := make([]*commonPb.EndorsementEntry, 0)
	for _, info := range m.VoteInfos {
		es = append(es, info.Endorsement)
	}

	//name :=string(parameters["CallContractName"])
	//version:=string(parameters["contractVersion"])
	var payloadHash []byte
	if utils.IsAnyBlank(p.TxId) {
		err = fmt.Errorf("params txIdStr cannot be empty")
		return nil, err
	}
	//}else {
	//	payloadHash, _ = txSimContext.Get(syscontract.SystemContract_MULTI_SIGN.String(), []byte(p.TxId))
	//}

	//if payloadHash == nil || len(payloadHash) == 0 {
	//	err = fmt.Errorf("the params of payload_hash is not exist")
	//	r.log.Error(err.Error(), "err", err)
	//	return nil, err
	//}

	ac, err := txSimContext.GetAccessControl()
	if err != nil {
		r.log.Errorw("txSimContext.GetAccessControl is err", "err", err)
		return nil, err
	}

	{
		principal, err := ac.CreatePrincipal(m.Method, es, reqPayload) //?
		if err != nil {
			r.log.Error(err)
			return nil, err
		}
		endorsement, err := ac.GetValidEndorsements(principal)
		if err != nil {
			r.log.Error(err)
			return nil, err
		}
		if endorsement == nil || len(endorsement) == 0 {
			err = fmt.Errorf("the vote is err")
			r.log.Errorw(err.Error(), "err", err)
			return nil, err
		}
	}

	// 2、获取历史投票记录
	// 3、判断是否继续可以对该多签交易投票
	// 4、根据传入参数的状态修改多签结果
	// verify the vote permission

	// append EndorsementEntry
	endorsements := make([]*commonPb.EndorsementEntry, 0) //?
	//hashType := ac.GetHashAlg()

	//voteSignerId := string(voteInfo.Endorsement.Signer.MemberInfo)

	for _, v := range m.VoteInfos {
		//if v.Endorsement != nil && v.Endorsement.Signer != nil {
		// vSingerId := string(v.Endorsement.Signer.MemberInfo)
		// if v.Endorsement.Signer.IsFullCert {
		//    vSingerId, err = utils.GetCertificateIdHex(v.Endorsement.Signer.MemberInfo, hashType)
		//    if err != nil {
		//       r.log.Errorw("get certHash is err", "err", err)
		//       return nil, err
		//    }
		// }
		// if voteInfo.Endorsement.Signer.IsFullCert {
		//    voteSignerId, err = utils.GetCertificateIdHex(voteInfo.Endorsement.Signer.MemberInfo, hashType)
		//    if err != nil {
		//       r.log.Errorw("get certHash is err", "err", err)
		//       return nil, err
		//    }
		// }
		// if strings.Compare(vSingerId, voteSignerId) == 0 {  //判断用户是否已投过票
		//    err = fmt.Errorf("the sender voted")
		//    r.log.Errorw(err.Error())
		//    return nil, err
		// }
		//}
		if v != nil && v.Vote == syscontract.VoteStatus_ARGUE {
			// agree
			endorsements = append(endorsements, v.Endorsement)
		}
	}

	// 5、根据结果调用accessControl校验是否认证成功
	var (
		contractResultBytes []byte
		contractErr         error
	)
	// 确认多签交易是否成功
	{
		//principal, err := ac.CreatePrincipal(resourceName, endorsements, multiSignInfo.PayloadBytes)
		//if err != nil {
		// err = fmt.Errorf("newPolicy is err")
		// r.log.Error(err.Error(), "err", err)
		// return nil, err
		//}
		//v, _ := ac.VerifyPrincipal(principal)
		//if err != nil {
		// r.log.Debugw("ac.VerifyPolicy", "err", err)
		//}

		voteflag := false
		if len(m.VoteInfos) > 2 {
			voteflag = true
		}

		contract := &commonPb.Contract{
			Name:        m.ContractName,
			RuntimeType: commonPb.RuntimeType_NATIVE,
			Status:      commonPb.ContractStatus_NORMAL,
		}

		if voteflag {
			// 6、调用真实系统合约完成该交易
			m.Status = syscontract.MultiSignStatus_ADOPTED
			initParam := make(map[string][]byte)
			for i := range p.Parameters {
				initParam[p.Parameters[i].Key] = p.Parameters[i].Value
			}
			bytecode := initParam[syscontract.InitContract_CONTRACT_BYTECODE.String()]
			//contractResult, statusCode := txSimContext.CallContract(contractId, methodName, byteCode, parameter, gasUsed, payloadInfo.txType)
			contractResult, statusCode := txSimContext.CallContract(contract, m.Method, bytecode, initParam, 0, commonPb.TxType_INVOKE_CONTRACT)
			if statusCode == commonPb.TxStatusCode_SUCCESS {
				// call success
				contractResultBytes = contractResult.Result
			} else {
				// call failture
				contractErr = errors.New(contractResult.Message)
			}
		}
	}

	// 7、记录成功

	multiSingInfoBytes, err := proto.Marshal(m)
	if err != nil {
		r.log.Error(err)
		return nil, err
	}
	err = txSimContext.Put(syscontract.SystemContract_CONTRACT_MANAGE.String(), payloadHash, multiSingInfoBytes)
	if err != nil {
		r.log.Error(err)
		return nil, err
	}
	if contractResultBytes == nil && contractErr == nil {
		return []byte("vote success"), nil
	}
	return contractResultBytes, contractErr
	//return result, err
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
