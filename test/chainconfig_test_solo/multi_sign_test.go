/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package native_test

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"testing"

	apiPb "chainmaker.org/chainmaker/pb-go/api"
	commonPb "chainmaker.org/chainmaker/pb-go/common"

	native "chainmaker.org/chainmaker-go/test/chainconfig_test"
	"chainmaker.org/chainmaker-go/utils"
	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/require"
)

// 多签请求
func TestMultiSignReq(t *testing.T) {
	txId := utils.GetRandTxId()
	fmt.Printf("\n============ send Tx [%s] ============\n", txId)

	// 构造Payload
	var pairs []*commonPb.KeyValuePair
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "tx_type", // 多签内的交易类型
		Value: commonPb.TxType_UPDATE_CHAIN_CONFIG.String(),
	})
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "deadline_block", // 过期的区块高度
		Value: strconv.Itoa(10),
	})

	// 需要多签的部分
	{
		payloadBytes, _ := getPayloadInfo()
		pairs = append(pairs, &commonPb.KeyValuePair{
			Key:   "payload",
			Value: hex.EncodeToString(payloadBytes),
		})

		endorsementEntry, err := native.AclSignOne(payloadBytes, 1)
		require.NoError(t, err)

		voteInfo := &commonPb.MultiSignVoteInfo{
			Vote:        commonPb.VoteStatus_AGREE,
			Endorsement: endorsementEntry,
		}

		voteInfoBytes, err := proto.Marshal(voteInfo)

		pairs = append(pairs, &commonPb.KeyValuePair{
			Key:   "vote_info",
			Value: hex.EncodeToString(voteInfoBytes),
		})
	}

	// 直接请求
	sk, member := native.GetUserSK(1)
	resp, err := native.UpdateSysRequest(sk, member, &native.InvokeContractMsg{TxType: commonPb.TxType_INVOKE_SYSTEM_CONTRACT, ChainId: CHAIN1,
		TxId: txId, ContractName: commonPb.ContractName_SYSTEM_CONTRACT_MULTI_SIGN.String(), MethodName: commonPb.MultiSignFunction_REQ.String(), Pairs: pairs})
	processResults(resp, err)
}

func getPayloadInfo() ([]byte, []byte) {
	var payloadPairs []*commonPb.KeyValuePair
	payloadPairs = append(payloadPairs, &commonPb.KeyValuePair{
		Key:   "tx_scheduler_timeout",
		Value: "15",
	})
	payloadPairs = append(payloadPairs, &commonPb.KeyValuePair{
		Key:   "tx_scheduler_validate_timeout",
		Value: "15",
	})
	chainConfig := getChainConfig()
	if chainConfig == nil {
		panic("chainConfig is empty")
	}
	payload := &commonPb.SystemContractPayload{
		ChainId:      CHAIN1,
		ContractName: commonPb.ContractName_SYSTEM_CONTRACT_CHAIN_CONFIG.String(),
		Method:       commonPb.ConfigFunction_CORE_UPDATE.String(),
		Parameters:   payloadPairs,
		Sequence:     chainConfig.Sequence,
		Endorsement:  nil,
	}

	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		panic(err)
	}
	payloadHash, err := utils.GetCertificateIdFromDER(payloadBytes, "SHA256")
	fmt.Println("payloadBytes", hex.EncodeToString(payloadBytes))
	fmt.Println("payloadHash", hex.EncodeToString(payloadHash))
	return payloadBytes, payloadHash
}

// 多签投票
func TestMultiSignVote(t *testing.T) {
	signerIndex := 2
	vote := commonPb.VoteStatus_AGREE //赞成
	//vote := commonPb.VoteStatus_DISAGREE //不赞成

	txId := utils.GetRandTxId()
	fmt.Printf("\n============ send Tx [%s] ============\n", txId)

	// 构造Payload
	var pairs []*commonPb.KeyValuePair

	// 需要多签的部分
	{
		payloadBytes, payloadHash := getPayloadInfo()
		// tx_id或payload_hash，如果有tx_id，会优先选择tx_id
		pairs = append(pairs, &commonPb.KeyValuePair{
			Key:   "tx_id",
			Value: "ecfca86332444da087b3f12927076a4be5a42fb5552f4600885ea27537193aeb",
		})
		pairs = append(pairs, &commonPb.KeyValuePair{
			Key:   "payload_hash",
			Value: hex.EncodeToString(payloadHash),
		})

		// ===================================
		native.DefaultOrgId = fmt.Sprintf(native.OrgIdFormat, strconv.Itoa(signerIndex))
		native.DefaultUserKeyPath = fmt.Sprintf(native.UserKeyPathFmt, strconv.Itoa(signerIndex))
		native.DefaultUserCrtPath = fmt.Sprintf(native.UserCrtPathFmt, strconv.Itoa(signerIndex))
		// ===================================

		var voteInfo *commonPb.MultiSignVoteInfo
		if vote == commonPb.VoteStatus_AGREE {
			endorsementEntry, err := native.AclSignOne(payloadBytes, signerIndex)
			if err != nil {
				fmt.Printf("AclSignOne err: %v\n", err)
				return
			}
			voteInfo = &commonPb.MultiSignVoteInfo{
				Vote:        commonPb.VoteStatus_AGREE,
				Endorsement: endorsementEntry,
			}
		} else {
			// 不同意时，不需要用户签名
			voteInfo = &commonPb.MultiSignVoteInfo{
				Vote: commonPb.VoteStatus_DISAGREE,
			}
		}

		voteInfoBytes, err := proto.Marshal(voteInfo)
		require.NoError(t, err)
		pairs = append(pairs, &commonPb.KeyValuePair{
			Key:   "vote_info",
			Value: hex.EncodeToString(voteInfoBytes),
		})
	}

	sk, member := native.GetUserSK(signerIndex)
	resp, err := native.UpdateSysRequest(sk, member, &native.InvokeContractMsg{TxType: commonPb.TxType_INVOKE_SYSTEM_CONTRACT, ChainId: CHAIN1,
		TxId: txId, ContractName: commonPb.ContractName_SYSTEM_CONTRACT_MULTI_SIGN.String(), MethodName: commonPb.MultiSignFunction_VOTE.String(), Pairs: pairs})
	processResults(resp, err)
}

// 多签查询
func TestMultiSignQuery(t *testing.T) {
	conn, err := native.InitGRPCConnect(isTls)
	require.NoError(t, err)
	client := apiPb.NewRpcNodeClient(conn)

	fmt.Println("============ get chain config by blockHeight============")
	// 构造Payload
	var pairs []*commonPb.KeyValuePair
	// tx_id或payload_hash，如果有tx_id，会优先选择tx_id
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "tx_id",
		Value: "7e066f74197e4436942f79362fe88040daf63ce52c884807b686d5f7d0fe85d1",
	})
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "payload_hash",
		Value: "be70f29b4597154acfcb1f4f208f764211ad6fcf1cbd5a894f3edd5d9c4d8c19",
	})

	sk, member := native.GetUserSK(1)
	resp, err := native.QueryRequest(sk, member, &client, &native.InvokeContractMsg{TxType: commonPb.TxType_QUERY_SYSTEM_CONTRACT, ChainId: CHAIN1,
		ContractName: commonPb.ContractName_SYSTEM_CONTRACT_MULTI_SIGN.String(), MethodName: commonPb.MultiSignFunction_QUERY.String(), Pairs: pairs})
	processResults(resp, err)
}
