/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"

	"time"

	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"

	"chainmaker.org/chainmaker/utils/v2"

	"github.com/spf13/cobra"
)

var (
	txType        string
	deadlineBlock int
	payload       string
	voteStatus    bool // true 赞成，false 不赞成

	multiTxId   string
	payloadHash string

	sysContractName string
	sysMethod       string
	txId            string
	height          uint64
	hash            string
	withRWSets      bool
	useTLS          bool
	runTime         int32
	reqPairsString  string
	pairsString     string
	pairsFile       string
	method          string
	version         string
	requestTimeout  int
	reqTimestamp    int64
	memberNum       int
	timestamp       int64
)

var (
	payloadHashStr = "payload_hash"
)

func MultiSignReqCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "multiSignReq",
		Short: "Multi sign req",
		Long:  "Multi sign req",
		RunE: func(_ *cobra.Command, _ []string) error {
			return multiSignReq()
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&sysContractName, "sys-contract-name", "s", "", "specify syscontractName")
	flags.StringVarP(&sysMethod, "sys-method", "m", "", "specify sysMethod")
	flags.StringVarP(&pairsString, "pairs", "", "", "specify pairs")
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
	flags.StringVar(&multiTxId, "multi-tx-id", "", "the multi sign req tx_id")
	flags.StringVarP(&sysContractName, "sys-contract-name", "s", "", "specify syscontractName")
	flags.StringVarP(&sysMethod, "sys-method", "m", "", "specify sysMethod")
	flags.StringVarP(&reqPairsString, "req-pairs", "", "", "specify reqpairs")
	flags.Int64VarP(&reqTimestamp, "req-timestamp", "", 0, "specify reqtimestamp")
	flags.IntVarP(&memberNum, "member-num", "", 2, "specify reqtimestamp")

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
	flags.StringVar(&multiTxId, "multi-tx-id", "", "the multi_sign req tx_id")
	return cmd
}

type ParamMultiSign struct {
	Key    string
	Value  string
	IsFile bool
}

func multiSignReq() error {
	// 构造Payload
	txId := utils.GetRandTxId()
	fmt.Println("req txid:%s", txId)
	//log.Infof("req txid:%s",txId)
	timestamp = time.Now().Unix()
	fmt.Println("req timestamp:%d", timestamp)
	//sk, _ := GetUserSK(1)
	var pairs []*commonPb.KeyValuePair
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   syscontract.MultiReq_SYS_CONTRACT_NAME.String(),
		Value: []byte(sysContractName),
	})
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   syscontract.MultiReq_SYS_METHOD.String(),
		Value: []byte(sysMethod),
	})
	var pms []*ParamMultiSign
	json.Unmarshal([]byte(pairsString), &pms)
	for _, pm := range pms {
		if pm.IsFile {
			byteCode, err := ioutil.ReadFile(pm.Value)
			if err != nil {
				panic(err)
			}
			pairs = append(pairs, &commonPb.KeyValuePair{
				Key:   pm.Key,
				Value: byteCode,
			})

		} else {
			pairs = append(pairs, &commonPb.KeyValuePair{
				Key:   pm.Key,
				Value: []byte(pm.Value),
			})
		}

	}

	payload := &commonPb.Payload{
		TxType:       commonPb.TxType_INVOKE_CONTRACT,
		ContractName: syscontract.SystemContract_MULTI_SIGN.String(),
		Method:       syscontract.MultiSignFunction_REQ.String(),
		Parameters:   pairs,
		TxId:         txId,
		ChainId:      CHAIN1,
		Timestamp:    timestamp,
	}
	endorsement, err := acSign(payload)
	if err != nil {
		return err
	}
	resp, err := proposalRequestWithMultiSign(sk3, client, payload, endorsement)
	if err != nil {
		return err
	}

	result := &Result{
		Code:    resp.Code,
		Message: resp.Message,
		TxId:    txId,
	}
	fmt.Println(result.ToJsonString())

	return nil
}

func multiSignVote() error {
	// 构造Payload
	txId := utils.GetRandTxId()
	timestamp = time.Now().Unix()
	var reqpairs []*commonPb.KeyValuePair
	reqpairs = append(reqpairs, &commonPb.KeyValuePair{
		Key:   syscontract.MultiReq_SYS_CONTRACT_NAME.String(),
		Value: []byte(sysContractName),
	})
	reqpairs = append(reqpairs, &commonPb.KeyValuePair{
		Key:   syscontract.MultiReq_SYS_METHOD.String(),
		Value: []byte(sysMethod),
	})
	var pms []*ParamMultiSign
	json.Unmarshal([]byte(reqPairsString), &pms)
	for _, pm := range pms {
		if pm.IsFile {
			byteCode, err := ioutil.ReadFile(pm.Value)
			if err != nil {
				panic(err)
			}
			reqpairs = append(reqpairs, &commonPb.KeyValuePair{
				Key:   pm.Key,
				Value: byteCode,
			})

		} else {
			reqpairs = append(reqpairs, &commonPb.KeyValuePair{
				Key:   pm.Key,
				Value: []byte(pm.Value),
			})
		}

	}

	payload1 := &commonPb.Payload{
		TxType:       commonPb.TxType_INVOKE_CONTRACT,
		ContractName: syscontract.SystemContract_MULTI_SIGN.String(),
		Method:       syscontract.MultiSignFunction_REQ.String(),
		Parameters:   reqpairs,
		TxId:         multiTxId,
		ChainId:      CHAIN1,
		Timestamp:    reqTimestamp,
	}
	ee, err := acSign(payload1)
	if err != nil {
		panic(err)
	}
	//构造多签投票信息
	msvi := &syscontract.MultiSignVoteInfo{
		Vote:        syscontract.VoteStatus_AGREE,
		Endorsement: ee[0],
	}
	msviByte, _ := msvi.Marshal()
	fmt.Printf("msviByte:%s", msviByte)

	var pairs []*commonPb.KeyValuePair
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   syscontract.MultiVote_VOTE_INFO.String(),
		Value: msviByte,
	})
	if multiTxId != "" {
		pairs = append(pairs, &commonPb.KeyValuePair{
			Key:   syscontract.MultiVote_TX_ID.String(),
			Value: []byte(multiTxId),
		})
	}

	payload := &commonPb.Payload{
		ChainId:      chainId,
		ContractName: syscontract.SystemContract_MULTI_SIGN.String(),
		Method:       syscontract.MultiSignFunction_VOTE.String(),
		Parameters:   pairs,
		TxType:       commonPb.TxType_INVOKE_CONTRACT,
		TxId:         utils.GetRandTxId(),
		Timestamp:    timestamp,
	}

	resp, err := proposalRequest(sk3, client, payload)
	if err != nil {
		return err
	}

	result := &Result{
		Code:    resp.Code,
		Message: resp.Message,
		TxId:    txId,
	}
	fmt.Println(result.ToJsonString())

	return nil
}

func multiSignQuery() error {
	pairs := make([]*commonPb.KeyValuePair, 0)
	if multiTxId != "" {
		pairs = append(pairs, &commonPb.KeyValuePair{
			Key:   syscontract.MultiVote_TX_ID.String(),
			Value: []byte(multiTxId),
		})
	}

	if len(pairs) == 0 {
		return errors.New("params is emtpy")
	}
	payload := &commonPb.Payload{
		TxType:       commonPb.TxType_INVOKE_CONTRACT,
		ContractName: syscontract.SystemContract_MULTI_SIGN.String(),
		Method:       syscontract.MultiSignFunction_QUERY.String(),
		Parameters:   pairs,
		ChainId:      CHAIN1,
	}
	resp, err := proposalRequest(sk3, client, payload)
	if err != nil {
		return err
	}

	result := &Result{
		Code:    resp.Code,
		Message: resp.Message,
		TxId:    txId,
	}
	fmt.Println(result.ToJsonString())

	return nil
}
