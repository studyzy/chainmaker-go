/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Rican7/retry"
	"github.com/Rican7/retry/backoff"
	"github.com/Rican7/retry/strategy"
	"time"

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
		commonPb.ContractName_SYSTEM_CONTRACT_PRIVATE_COMPUTE.String(),
		commonPb.PrivateComputeContractFunction_SAVE_CERT.String(),
		pairs,
		defaultSequence,
	)
	if err != nil {
		return fmt.Errorf("construct save cert payload failed, %s", err.Error())
	}

	resp, err = proposalRequest(sk3, client, commonPb.TxType_INVOKE_SYSTEM_CONTRACT, chainId, "", payloadBytes)
	if err != nil {
		return fmt.Errorf(errStringFormat, commonPb.TxType_INVOKE_SYSTEM_CONTRACT.String(), err.Error())
	}

	if resp.Code == commonPb.TxStatusCode_SUCCESS {
		if !withSyncResult {
			resp.ContractResult = &commonPb.ContractResult{
				Code:    commonPb.ContractResultCode_OK,
				Message: commonPb.ContractResultCode_OK.String(),
				Result:  []byte(txId),
			}
		} else {
			contractResult, err := getSyncResult(txId)
			if err != nil {
				return fmt.Errorf("get sync result failed, %s", err.Error())
			}

			if contractResult.Code != commonPb.ContractResultCode_OK {
				resp.Code = commonPb.TxStatusCode_CONTRACT_FAIL
				resp.Message = contractResult.Message
			}

			resp.ContractResult = contractResult
		}
	}

	if err = checkProposalRequestResp(resp, true); err != nil {
		return fmt.Errorf(errStringFormat, commonPb.TxType_INVOKE_SYSTEM_CONTRACT.String(), err.Error())
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

	bytes, err := json.Marshal(resultStruct)
	if err != nil {
		return err
	}
	fmt.Println(string(bytes))

	return nil

}

func constructSystemContractPayload(chainId, contractName, method string, pairs []*commonPb.KeyValuePair, sequence uint64) ([]byte, error) {

	payload := &commonPb.SystemContractPayload{
		ChainId:      chainId,
		ContractName: contractName,
		Method:       method,
		Parameters:   pairs,
		Sequence:     sequence,
	}

	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return payloadBytes, nil
}

func paramsMap2KVPairs(params map[string]string) (kvPairs []*commonPb.KeyValuePair) {
	for key, val := range params {
		kvPair := &commonPb.KeyValuePair{
			Key:   key,
			Value: val,
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

	if resp.ContractResult != nil && resp.ContractResult.Code != commonPb.ContractResultCode_OK {
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

	payloadBytes, err := constructQueryPayload(
		commonPb.ContractName_SYSTEM_CONTRACT_QUERY.String(),
		commonPb.QueryFunction_GET_TX_BY_TX_ID.String(),
		[]*commonPb.KeyValuePair{
			{
				Key:   "txId",
				Value: txId,
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("GetTxByTxId marshal query payload failed, %s", err.Error())
	}

	resp, err = proposalRequest(sk3, client, commonPb.TxType_QUERY_SYSTEM_CONTRACT, chainId, txId, payloadBytes)
	if err != nil {
		return nil, fmt.Errorf(errStringFormat, commonPb.TxType_QUERY_SYSTEM_CONTRACT.String(), err.Error())
	}

	if err = checkProposalRequestResp(resp, true); err != nil {
		return nil, fmt.Errorf(errStringFormat, commonPb.TxType_QUERY_SYSTEM_CONTRACT.String(), err.Error())
	}

	transactionInfo := &commonPb.TransactionInfo{}
	if err = proto.Unmarshal(resp.ContractResult.Result, transactionInfo); err != nil {
		return nil, fmt.Errorf("unmarshal transaction info payload failed, %s", err.Error())
	}

	return transactionInfo, nil
}

func constructQueryPayload(contractName, method string, pairs []*commonPb.KeyValuePair) ([]byte, error) {
	payload := &commonPb.QueryPayload{
		ContractName: contractName,
		Method:       method,
		Parameters:   pairs,
	}

	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return payloadBytes, nil
}