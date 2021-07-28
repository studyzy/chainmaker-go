/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package multisign

import (
	"chainmaker.org/chainmaker-go/vm/native/common"
	commonPb "chainmaker.org/chainmaker/pb-go/common"
	"chainmaker.org/chainmaker/pb-go/syscontract"
	"chainmaker.org/chainmaker/protocol"
	"github.com/gogo/protobuf/proto"
)

var (
	ContractName = syscontract.SystemContract_MULTI_SIGN.String()
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
func (r *MultiSignRuntime) reqContract(txSimContext protocol.TxSimContext, parameters map[string][]byte) (result []byte,err error) {
	r.log.Infof(" multisigncontract test1:%s",parameters)
	// 1、校验并获取参数
	txid :=txSimContext.GetTx().Payload.TxId
	// method
	//contractName, ok := parameters["contractName"]
	//if !ok {
	//	err = fmt.Errorf("the params of contractName is nil")
	//	r.log.Error(err)
	//	return nil, err
	//}
	//contractName := string(parameters["contractName"])

	//payloadBytes, ok := parameters["payload"]
	//if !ok {
	//	err = fmt.Errorf("the params of payload is nil")
	//	r.log.Error(err)
	//	return nil, err
	//}

	//ac, err := txSimContext.GetAccessControl()
	//if err != nil {
	//	r.log.Errorw("txSimContext.GetAccessControl is err", "err", err)
	//	return nil, err
	//}
	//txId := txSimContext.GetTx().Header.TxId

	// 2、组装结构体

	// payload transfer to hash
	//hashType := ac.GetHashAlg()
	//payloadHash, err := utils.GetCertificateIdFromDER(payloadBytes, hashType)
	//bytes, err := txSimContext.Get(syscontract.SystemContract_MULTI_SIGN.String(), payloadHash)
	//if err == nil && len(bytes) > 0 {
	//	r.log.Errorw("payload is exist", "payloadHash", hex.EncodeToString(payloadHash))
	//	return nil, err
	//}

	//multiSignInfo := &commonPb.MultiSignInfo{
	//	//TxId:          txId,
	//	//Name :  contractName,
	//	//TxType:        txType,
	//	//PayloadBytes:  payloadBytes,
	//	//DeadlineBlock: deadlineBlock,
	//	//VoteInfos:
	//	Status: commonPb.MultiSignStatus_PROCESSING,
	//}
	//cdata, _ := multiSignInfo.Marshal()
	//key := utils.GetContractDbKey(contractName)
	//txSimContext.Put(ContractName,key, cdata)
	//r.log.Infof(" multisigncontract test[name:%s ]", multiSignInfo.Name)

	//multiSingInfoBytes, err := proto.Marshal(multiSignInfo)
	//if err != nil {
	//	r.log.Error(err)
	//	return nil, err
	//}


	//3、保存

	// payloadHash==>multiSingInfoBytes
	//err = txSimContext.Put(syscontract.SystemContract_MULTI_SIGN.String(), payloadHash, multiSingInfoBytes)
	//if err != nil {
	//	return nil, err
	//}
	// txId==>payloadHash
	//err = txSimContext.Put(syscontract.SystemContract_MULTI_SIGN.String(), []byte(txId), payloadHash)
	//if err != nil {
	//	return nil, err
	//}

	//r.log.Infow("multiSign info", "payloadHash", hex.EncodeToString(payloadHash), "txId", txId)
	resp := &commonPb.MultiSignResp{
		//TxId:        txId,
		//PayloadHash: payloadStr,
	}
	result, err = proto.Marshal(resp)
	if err != nil {
		return nil, err
	}
	return result, nil
}


func (r *MultiSignRuntime) voteContract(txSimContext protocol.TxSimContext, parameters map[string][]byte) (result []byte,err error) {
	// 1、检查参数

	//txIdStr, ok := params["tx_id"]
	//var payloadHash []byte
	//if ok {
	//	payloadHash, _ = txSimContext.Get(commonPb.ContractName_SYSTEM_CONTRACT_MULTI_SIGN.String(), []byte(txIdStr)) //?
	//}
	//if payloadHash == nil || len(payloadHash) == 0 {
	//	payloadHashStr, ok := params["payload_hash"]
	//	if !ok {
	//		err = fmt.Errorf("the params of payload_hash is nil")
	//		r.log.Error(err)
	//		return nil, err
	//	}
	//	payloadHash, err = hex.DecodeString(payloadHashStr)
	//	if err != nil {
	//		err = fmt.Errorf("the params of payload_hash is err. payload_hash= %s", payloadHashStr)
	//		r.log.Error(err.Error(), "err", err)
	//		return nil, err
	//	}
	//}
	//if payloadHash == nil || len(payloadHash) == 0 {
	//	err = fmt.Errorf("the params of payload_hash is not exist")
	//	r.log.Error(err.Error(), "err", err)
	//	return nil, err
	//}
	//
	//voteInfoStr, ok := params["vote_info"]
	//if !ok {
	//	err = fmt.Errorf("the params of vote_info is nil")
	//	r.log.Error(err)
	//	return nil, err
	//}

	// 2、获取历史投票记录


	// 3、判断是否继续可以对该多签交易投票


	// 4、根据传入参数的状态修改多签结果

	//multiSignInfo := new(commonPb.MultiSignInfo)
	//endorsements := make([]*commonPb.EndorsementEntry, 0) //?
	//hashType := ac.GetHashAlg()
	//
	//voteSignerId := string(voteInfo.Endorsement.Signer.MemberInfo)
	//
	//for _, v := range multiSignInfo.VoteInfos {
	//	if v.Endorsement != nil && v.Endorsement.Signer != nil {
	//		vSingerId := string(v.Endorsement.Signer.MemberInfo)
	//		if v.Endorsement.Signer.IsFullCert {
	//			vSingerId, err = utils.GetCertificateIdHex(v.Endorsement.Signer.MemberInfo, hashType)
	//			if err != nil {
	//				r.log.Errorw("get certHash is err", "err", err)
	//				return nil, err
	//			}
	//		}
	//		if voteInfo.Endorsement.Signer.IsFullCert {
	//			voteSignerId, err = utils.GetCertificateIdHex(voteInfo.Endorsement.Signer.MemberInfo, hashType)
	//			if err != nil {
	//				r.log.Errorw("get certHash is err", "err", err)
	//				return nil, err
	//			}
	//		}
	//		if strings.Compare(vSingerId, voteSignerId) == 0 {  //判断用户是否已投过票
	//			err = fmt.Errorf("the sender voted")
	//			r.log.Errorw(err.Error())
	//			return nil, err
	//		}
	//	}
	//	if v != nil && v.Vote == commonPb.VoteStatus_AGREE {
	//		// agree
	//		endorsements = append(endorsements, v.Endorsement)
	//	}
	//}
	//
	//multiSignInfo.VoteInfos = append(multiSignInfo.VoteInfos, voteInfo)
	//if voteInfo.Vote == commonPb.VoteStatus_AGREE {
	//	endorsements = append(endorsements, voteInfo.Endorsement)
	//}
	// 5、根据结果调用accessControl校验是否认证成功

	//principal, err := ac.CreatePrincipal(resourceName, endorsements, multiSignInfo.PayloadBytes)
	//if err != nil {
	//	err = fmt.Errorf("newPolicy is err")
	//	r.log.Error(err.Error(), "err", err)
	//	return nil, err
	//}
	//v, _ := ac.VerifyPrincipal(principal)
	//if err != nil {
	//	r.log.Debugw("ac.VerifyPolicy", "err", err)
	//}

	// 6、调用真实系统合约完成该交易

	//if v {
	//	// call other Contract
	//	multiSignInfo.Status = commonPb.MultiSignStatus_ADOPTED
	//	contractResult, statusCode := txSimContext.CallContract(contractId, methodName, byteCode, parameter, gasUsed, payloadInfo.txType)
	//	if statusCode == commonPb.TxStatusCode_SUCCESS {
	//		// call success
	//		contractResultBytes = contractResult.Result
	//	} else {
	//		// call failture
	//		contractErr = errors.New(contractResult.Message)
	//	}
	//}

	// 7、记录成功

	//multiSingInfoBytes, err := proto.Marshal(multiSignInfo)
	//if err != nil {
	//	r.log.Error(err)
	//	return nil, err
	//}
	//err = txSimContext.Put(commonPb.ContractName_SYSTEM_CONTRACT_MULTI_SIGN.String(), payloadHash, multiSingInfoBytes)
	//if err != nil {
	//	r.log.Error(err)
	//	return nil, err
	//}
	//if contractResultBytes == nil && contractErr == nil {
	//	return []byte("vote success"), nil
	//}

	result, statusCode := context.CallContract(contract, protocol.ContractInitMethod, byteCode, initParameters, 0, commonPb.TxType_INVOKE_CONTRACT)


	return nil, err
}

func (r *MultiSignRuntime) queryContract(txSimContext protocol.TxSimContext, parameters map[string][]byte) (result []byte,err error) {
	// 1、校验并获取参数

	//txIdStr, ok := params["tx_id"]
	//var payloadHash []byte
	//if ok {
	//	payloadHash, _ = txSimContext.Get(commonPb.ContractName_SYSTEM_CONTRACT_MULTI_SIGN.String(), []byte(txIdStr))
	//}
	//if payloadHash == nil || len(payloadHash) == 0 {
	//	payloadHashStr, ok := params["payload_hash"]
	//	if !ok {
	//		err = fmt.Errorf("the params of payload_hash is nil")
	//		r.log.Error(err)
	//		return nil, err
	//	}
	//	payloadHash, err = hex.DecodeString(payloadHashStr)
	//	if err != nil {
	//		err = fmt.Errorf("the params of payload_hash is err. payload_hash= %s", payloadHashStr)
	//		r.log.Error(err.Error(), "err", err)
	//		return nil, err
	//	}
	//}
	//if payloadHash == nil || len(payloadHash) == 0 {
	//	err = fmt.Errorf("the params of payload_hash is not exist")
	//	r.log.Error(err.Error(), "err", err)
	//	return nil, err
	//}

	// 2、返回结果

	//multiSignInfoBytes, err := txSimContext.Get(commonPb.ContractName_SYSTEM_CONTRACT_MULTI_SIGN.String(), payloadHash)
	//if err != nil {
	//	r.log.Error(err)
	//	return nil, err
	//}
	//
	//return multiSignInfoBytes, nil

	return nil, err
}

