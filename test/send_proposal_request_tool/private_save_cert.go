/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"errors"
	"fmt"
	"time"

	"chainmaker.org/chainmaker/utils/v2"

	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"

	"github.com/Rican7/retry"
	"github.com/Rican7/retry/backoff"
	"github.com/Rican7/retry/strategy"

	"github.com/gogo/protobuf/proto"
	"github.com/spf13/cobra"
)

const (
	defaultSequence = 0
	// 轮训交易结果最大次数
	retryCnt        = 10
	errStringFormat = "%s failed, %s"
	retryInterval   = 500 // 获取可用客户端连接对象重试时间间隔，单位：ms
)

var (
	enclaveCert    string
	enclaveId      string
	withSyncResult bool
)

func SaveCertCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "saveCert",
		Short: "save cert to blockchain",
		Long:  "save cert to blockchain",
		RunE: func(_ *cobra.Command, _ []string) error {
			return saveCert()
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&enclaveCert, "enclave_cert", "l", "", "enclave cert")
	flags.StringVarP(&enclaveId, "enclave_id", "m", "", "enclave id")
	flags.BoolVarP(&withSyncResult, "with_sync_result", "w", false, "with sync result")

	return cmd
}

func saveCert() error {

	// 构造Payload
	pairs := paramsMap2KVPairs(map[string]string{
		"enclave_cert": enclaveCert,
		"enclave_id":   enclaveId,
	})

	payloadBytes, err := constructSystemContractPayload(
		chainId,
		syscontract.SystemContract_PRIVATE_COMPUTE.String(),
		syscontract.PrivateComputeFunction_SAVE_CA_CERT.String(),
		pairs,
		defaultSequence,
	)
	if err != nil {
		return fmt.Errorf("construct save cert payload failed, %s", err.Error())
	}

	resp, err = proposalRequest(sk3, client, payloadBytes)
	if err != nil {
		return fmt.Errorf(errStringFormat, commonPb.TxType_INVOKE_CONTRACT.String(), err.Error())
	}

	if resp.Code == commonPb.TxStatusCode_SUCCESS {
		if !withSyncResult {
			resp.ContractResult = &commonPb.ContractResult{
				Code:    0,
				Message: "OK",
				Result:  []byte(txId),
			}
		} else {
			contractResult, err := getSyncResult(txId)
			if err != nil {
				return fmt.Errorf("get sync result failed, %s", err.Error())
			}

			if contractResult.Code != 0 {
				resp.Code = commonPb.TxStatusCode_CONTRACT_FAIL
				resp.Message = contractResult.Message
			}

			resp.ContractResult = contractResult
		}
	}

	if err = checkProposalRequestResp(resp, true); err != nil {
		return fmt.Errorf(errStringFormat, commonPb.TxType_INVOKE_CONTRACT.String(), err.Error())
	}

	resultStruct := &Result{
		Code:    resp.Code,
		Message: resp.Message,
	}

	if resp.ContractResult != nil {
		resultStruct.TxId = string(resp.ContractResult.Result)
		resultStruct.ContractResultCode = resp.ContractResult.Code
		resultStruct.ContractResultMessage = resp.ContractResult.Message
		resultStruct.ContractQueryResult = string(resp.ContractResult.Result)
	} else {
		fmt.Println("resp.ContractResult is nil ")
	}

	fmt.Println(resultStruct.ToJsonString())

	return nil

}

func constructSystemContractPayload(chainId, contractName, method string, pairs []*commonPb.KeyValuePair, sequence uint64) (*commonPb.Payload, error) {

	payload := &commonPb.Payload{
		ChainId:      chainId,
		ContractName: contractName,
		Method:       method,
		Parameters:   pairs,
		Sequence:     sequence,
		TxType:       commonPb.TxType_INVOKE_CONTRACT,
		TxId:         utils.GetRandTxId(),
		Timestamp:    time.Now().Unix(),
	}

	return payload, nil
}

func paramsMap2KVPairs(params map[string]string) (kvPairs []*commonPb.KeyValuePair) {
	for key, val := range params {
		kvPair := &commonPb.KeyValuePair{
			Key:   key,
			Value: []byte(val),
		}

		kvPairs = append(kvPairs, kvPair)
	}

	return
}

func checkProposalRequestResp(resp *commonPb.TxResponse, needContractResult bool) error {
	if resp.Code != commonPb.TxStatusCode_SUCCESS {
		return errors.New(resp.Message)
	}

	if needContractResult && resp.ContractResult == nil {
		return fmt.Errorf("contract result is nil")
	}

	if resp.ContractResult != nil && resp.ContractResult.Code != 0 {
		return errors.New(resp.ContractResult.Message)
	}

	return nil
}

func getSyncResult(txId string) (*commonPb.ContractResult, error) {
	var (
		txInfo *commonPb.TransactionInfo
		err    error
	)

	err = retry.Retry(func(uint) error {
		txInfo, err = GetTxByTxId(txId)
		if err != nil {
			return err
		}

		return nil
	},
		strategy.Limit(retryCnt),
		strategy.Backoff(backoff.Fibonacci(retryInterval*time.Millisecond)),
	)

	if err != nil {
		return nil, fmt.Errorf("get tx by txId [%s] failed, %s", txId, err.Error())
	}
	if txInfo == nil || txInfo.Transaction == nil || txInfo.Transaction.Result == nil {
		return nil, fmt.Errorf("get result by txId [%s] failed, %+v", txId, txInfo)
	}
	return txInfo.Transaction.Result.ContractResult, nil
}

func GetTxByTxId(txId string) (*commonPb.TransactionInfo, error) {

	payloadBytes, err := constructQueryPayload(chainId,
		syscontract.SystemContract_CHAIN_QUERY.String(),
		syscontract.ChainQueryFunction_GET_TX_BY_TX_ID.String(),
		[]*commonPb.KeyValuePair{
			{
				Key:   "txId",
				Value: []byte(txId),
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("GetTxByTxId marshal query payload failed, %s", err.Error())
	}

	resp, err = proposalRequest(sk3, client, payloadBytes)
	if err != nil {
		return nil, fmt.Errorf(errStringFormat, commonPb.TxType_QUERY_CONTRACT.String(), err.Error())
	}

	if err = checkProposalRequestResp(resp, true); err != nil {
		return nil, fmt.Errorf(errStringFormat, commonPb.TxType_QUERY_CONTRACT.String(), err.Error())
	}

	transactionInfo := &commonPb.TransactionInfo{}
	if err = proto.Unmarshal(resp.ContractResult.Result, transactionInfo); err != nil {
		return nil, fmt.Errorf("unmarshal transaction info payload failed, %s", err.Error())
	}

	return transactionInfo, nil
}
