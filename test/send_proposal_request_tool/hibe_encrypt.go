/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	"chainmaker.org/chainmaker/common/v2/crypto"
	"chainmaker.org/chainmaker/common/v2/crypto/hibe"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"github.com/spf13/cobra"
)

func HibeEncryptCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hibeEncrypt",
		Short: "use hibe to encrypt a message and save it to a file",
		Long:  "",
		RunE: func(_ *cobra.Command, _ []string) error {
			return constructHibeTxPayloadPairs()
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&hibePlaintext, "hibe-msg", "", "", "This is a test message")
	flags.StringVarP(&hibeReceiverIdsFilePath, "hibe-receiverId-file", "", "", "receiverIds file path")
	flags.StringVarP(&hibeParamsFilePath, "hibe-params-path-list-file", "", "", "paramsFilePath list file path")
	flags.StringVarP(&symKeyType, "sym-key-type", "", "aes", "symmetric key type (aes or sm4)")
	return cmd
}

func constructHibeTxPayloadPairsExc() (string, commonPb.TxStatusCode, string) {
	//bytes, err := ioutil.ReadFile(hibeReceiverIdsFilePath)
	//if err != nil {
	//	return err
	//}
	//
	//receiverIdsX := strings.Split(string(bytes), "\n")
	//receiverIds := strings.Split(receiverIdsX[0], "|")

	//paramsListBytes, err := ioutil.ReadFile(hibeParamsFilePath)
	//if err != nil {
	//	return err
	//}
	//paramsFileList := strings.Split(string(paramsListBytes), "\n")
	//fileList := strings.Split(paramsFileList[0], "|")

	receiverIds := strings.Split(hibeReceiverIdsFilePath, "|")
	fileList := strings.Split(hibeParamsFilePath, "|")

	result_output := ""

	paramsList := make([]*hibe.Params, 0)
	for _, paramsFilePath := range fileList {
		paramsBytes, err := ioutil.ReadFile(paramsFilePath)
		if err != nil {
			return result_output, 1, "get param file faulure!"
		}

		params, ok := new(hibe.Params).Unmarshal(paramsBytes)
		if !ok {
			return result_output, 1, fmt.Sprintf("hibe.Params unmarshal failed, err: %s", err)
		}
		paramsList = append(paramsList, params)
	}

	var keyType crypto.KeyType
	if symKeyType == "aes" {
		keyType = crypto.AES
	} else if symKeyType == "sm4" {
		keyType = crypto.SM4

	} else {
		return result_output, 1, fmt.Sprintf("invalid symKeyType, %s", symKeyType)
	}

	msg, err := hibe.EncryptHibeMsg([]byte(hibePlaintext), receiverIds, paramsList, keyType)
	if err != nil {
		return result_output, 1, fmt.Sprintf("EncryptHibeMsg failure!, err: %s", err)
	}

	hibeMsgBytes, err := json.Marshal(msg)
	if err != nil {
		return result_output, 1, fmt.Sprintf("Marshal failure!, err: %s", err)
	}

	return string(hibeMsgBytes), 0, ""
}

func constructHibeTxPayloadPairs() error {

	hibeMsgStr, result_code, result_err := constructHibeTxPayloadPairsExc()
	result_msg := "SUCCESS"
	if result_err != "" {
		result_msg = result_err
	}
	result := &Result{
		Code:                  result_code,
		Message:               result_msg,
		ContractResultCode:    0,
		ContractResultMessage: "OK",
		HibeExecMsg:           hibeMsgStr,
	}

	fmt.Println(result.ToJsonString())
	return nil
}
