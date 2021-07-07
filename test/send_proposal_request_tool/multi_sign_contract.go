/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

// description: chainmaker-go
//
// @author: xwc1125
// @date: 2020/12/24
package main

import (
	commonPb "chainmaker.org/chainmaker/pb-go/common"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"


	"chainmaker.org/chainmaker-go/utils"
	"github.com/gogo/protobuf/proto"
	"github.com/spf13/cobra"
)

var (
	txType        string
	deadlineBlock int
	payload       string
	voteStatus    bool // true 赞成，false 不赞成
	//adminOrgId    string // 签名者的组织
	//adminKeyPath  string // 签名者的key
	//adminCrtPath  string // 签名者的cert

	multiTxId   string
	payloadHash string
)

var (
	payloadHashStr = "payload_hash"
)

func MultiSignReqCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "multiSignReq",
		Short: "Multi sign req",
		Long:  "Multi sign req（need the admin）",
		RunE: func(_ *cobra.Command, _ []string) error {
			return multiSignReq()
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&txType, "tx_type", "", "the payload tx_type")
	flags.IntVar(&deadlineBlock, "deadline_block", 0, "the deadline block,default is 0. 0 for unlimited")
	flags.StringVar(&payload, "payload", "", "transfer the payloadBytes to hex.")
	//flags.StringVar(&adminOrgId, "admin_org_id", "", "the admin orgId")
	//flags.StringVar(&adminKeyPath, "admin_key_path", "", "the admin keyPath")
	//flags.StringVar(&adminCrtPath, "admin_crt_path", "", "the admin certPath")
	return cmd
}

func MultiSignVoteCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "multiSignVote",
		Short: "Multi sign vote",
		Long:  "Multi sign vote",
		RunE: func(_ *cobra.Command, _ []string) error {
			return multiSignVote()
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&multiTxId, "multi_tx_id", "", "the multi sign req tx_id")
	flags.StringVar(&payloadHash, payloadHashStr, "", "the multi sign req payload_hash")
	flags.BoolVar(&voteStatus, "vote_status", false, "vote or no")
	//flags.StringVar(&adminOrgId, "admin_org_id", "", "the admin orgId")
	//flags.StringVar(&adminKeyPath, "admin_key_path", "", "the admin keyPath")
	//flags.StringVar(&adminCrtPath, "admin_crt_path", "", "the admin certPath")
	return cmd
}

func MultiSignQueryCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "multiSignQuery",
		Short: "Multi sign query",
		Long:  "Multi sign query",
		RunE: func(_ *cobra.Command, _ []string) error {
			return multiSignQuery()
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&multiTxId, "multi_tx_id", "", "the multi_sign req tx_id")
	flags.StringVar(&payloadHash, payloadHashStr, "", "the multi_sign req payload_hash")
	return cmd
}

func multiSignReq() error {
	// 构造Payload
	txId := utils.GetRandTxId()
	pairs := make([]*commonPb.KeyValuePair, 0)
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "tx_type", // 多签内的交易类型
		Value: []byte(txType),
	})
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "deadline_block", // 过期的区块高度
		Value: []byte(strconv.Itoa(deadlineBlock)),
	})
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "payload",
		Value: []byte(payload),
	})

	multiPayloadBytes, err := hex.DecodeString(payload)
	if err != nil {
		fmt.Printf("hex.DecodeString err: %v\n", err)
		return err
	}

	payloadHash, err := utils.GetCertificateIdFromDER(multiPayloadBytes, "SHA256")

	if txType == "UPDATE_CHAIN_CONFIG" {
		contractPayload := &commonPb.Payload{}
		err := proto.Unmarshal(multiPayloadBytes, contractPayload)
		if err != nil {
			return err
		}
		fmt.Println(contractPayload)
	}

	//endorsementEntry, err := aclSignOne(multiPayloadBytes, adminOrgId, adminKeyPath, adminCrtPath)
	endorsementEntry, err := aclSignOne(multiPayloadBytes, orgId, userKeyPath, userCrtPath)
	if err != nil {
		fmt.Printf("signWith err: %v\n", err)
		return err
	}

	voteInfo := &commonPb.MultiSignVoteInfo{
		Vote:        commonPb.VoteStatus_AGREE,
		Endorsement: endorsementEntry,
	}

	voteInfoBytes, err := proto.Marshal(voteInfo)
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "vote_info",
		Value: voteInfoBytes,
	})

	payload := &commonPb.Payload{
		ChainId:      chainId,
		ContractName: commonPb.ContractName_SYSTEM_CONTRACT_MULTI_SIGN.String(),
		Method:       commonPb.MultiSignFunction_REQ.String(),
		Parameters:   pairs,
		Sequence:     seq,
	}
	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		return err
	}
	resp, err := proposalRequest(sk3, client, commonPb.TxType_INVOKE_CONTRACT,
		chainId, txId, payloadBytes)
	if err != nil {
		return err
	}

	result := &Result{
		Code:        resp.Code,
		Message:     resp.Message,
		TxId:        txId,
		PayloadHash: hex.EncodeToString(payloadHash),
	}
	bytes, err := json.Marshal(result)
	if err != nil {
		return err
	}
	fmt.Println(string(bytes))

	return nil
}

func multiSignVote() error {
	// 构造Payload
	txId := utils.GetRandTxId()
	pairs := make([]*commonPb.KeyValuePair, 0)
	if multiTxId != "" {
		pairs = append(pairs, &commonPb.KeyValuePair{
			Key:   "tx_id",
			Value: []byte(multiTxId),
		})
	}
	if payloadHash != "" {
		pairs = append(pairs, &commonPb.KeyValuePair{
			Key:   payloadHashStr,
			Value: []byte(payloadHash),
		})
	}
	_, multiSignInfo, err := getMultiSign()
	if err != nil {
		return err
	}

	var voteInfo *commonPb.MultiSignVoteInfo
	if voteStatus {
		//endorsementEntry, err := aclSignOne(multiSignInfo.PayloadBytes, adminOrgId, adminKeyPath, adminCrtPath)
		endorsementEntry, err := aclSignOne(multiSignInfo.PayloadBytes, orgId, userKeyPath, userCrtPath)
		if err != nil {
			fmt.Printf("signWith err: %v\n", err)
			return err
		}
		voteInfo = &commonPb.MultiSignVoteInfo{
			Vote:        commonPb.VoteStatus_AGREE,
			Endorsement: endorsementEntry,
		}
	} else {
		voteInfo = &commonPb.MultiSignVoteInfo{
			Vote: commonPb.VoteStatus_DISAGREE,
		}
	}

	voteInfoBytes, err := proto.Marshal(voteInfo)
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "vote_info",
		Value: voteInfoBytes,
	})

	payload := &commonPb.Payload{
		ChainId:      chainId,
		ContractName: commonPb.ContractName_SYSTEM_CONTRACT_MULTI_SIGN.String(),
		Method:       commonPb.MultiSignFunction_VOTE.String(),
		Parameters:   pairs,
		Sequence:     seq,
	}
	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		return err
	}
	resp, err := proposalRequest(sk3, client, commonPb.TxType_INVOKE_CONTRACT,
		chainId, txId, payloadBytes)
	if err != nil {
		return err
	}

	result := &Result{
		Code:        resp.Code,
		Message:     resp.Message,
		TxId:        txId,
		PayloadHash: payloadHash,
	}
	bytes, err := json.Marshal(result)
	if err != nil {
		return err
	}
	fmt.Println(string(bytes))

	return nil
}

func multiSignQuery() error {
	resp, multiSignInfo, err := getMultiSign()
	if err != nil {
		return err
	}

	payloadHash, err := utils.GetCertificateIdFromDER(multiSignInfo.PayloadBytes, "SHA256")

	result := &Result{
		Code:          resp.Code,
		Message:       resp.Message,
		TxId:          txId,
		PayloadHash:   hex.EncodeToString(payloadHash),
		MultiSignInfo: multiSignInfo,
	}
	bytes, err := json.Marshal(result)
	if err != nil {
		return err
	}
	fmt.Println(string(bytes))

	return nil
}

func getMultiSign() (*commonPb.TxResponse, *commonPb.MultiSignInfo, error) {
	pairs := make([]*commonPb.KeyValuePair, 0)
	if multiTxId != "" {
		pairs = append(pairs, &commonPb.KeyValuePair{
			Key:   "tx_id",
			Value: []byte(multiTxId),
		})
	}
	if payloadHash != "" {
		pairs = append(pairs, &commonPb.KeyValuePair{
			Key:   payloadHashStr,
			Value: []byte(payloadHash),
		})
	}
	if len(pairs) == 0 {
		return nil, nil, errors.New("params is emtpy")
	}
	payloadBytes, err := constructPayload(commonPb.ContractName_SYSTEM_CONTRACT_MULTI_SIGN.String(), commonPb.MultiSignFunction_QUERY.String(), pairs)
	if err != nil {
		return nil, nil, err
	}

	resp, err := proposalRequest(sk3, client, commonPb.TxType_QUERY_CONTRACT,
		chainId, txId, payloadBytes)
	if err != nil {
		return nil, nil, err
	}
	if resp.Code == commonPb.TxStatusCode_SUCCESS && resp.ContractResult.Code == 0 {
		multiSignInfo := new(commonPb.MultiSignInfo)
		result := resp.ContractResult.Result
		err = proto.Unmarshal(result, multiSignInfo)
		if err != nil {
			return resp, nil, err
		}
		fmt.Println("multiSignInfo", multiSignInfo)
		return resp, multiSignInfo, nil
	}
	return resp, nil, nil
}
