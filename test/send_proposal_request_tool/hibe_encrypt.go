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

	"chainmaker.org/chainmaker-go/common/crypto"
	localhibe "chainmaker.org/chainmaker-go/common/crypto/hibe"
	"github.com/samkumar/hibe"
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

func constructHibeTxPayloadPairs() error {
	bytes, err := ioutil.ReadFile(hibeReceiverIdsFilePath)
	if err != nil {
		return err
	}

	receiverIdsX := strings.Split(string(bytes), "\n")
	receiverIds := strings.Split(receiverIdsX[0], "|")

	paramsList := make([]*hibe.Params, 0)
	paramsListBytes, err := ioutil.ReadFile(hibeParamsFilePath)
	if err != nil {
		return err
	}

	paramsFileList := strings.Split(string(paramsListBytes), "\n")
	fileList := strings.Split(paramsFileList[0], "|")
	for _, paramsFilePath := range fileList {
		paramsBytes, err := ioutil.ReadFile(paramsFilePath)
		if err != nil {
			return err
		}

		params, ok := new(hibe.Params).Unmarshal(paramsBytes)
		if !ok {
			return fmt.Errorf("hibe.Params unmarshal failed, err: %s", err)
		}
		paramsList = append(paramsList, params)
	}

	var keyType crypto.KeyType
	if symKeyType == "aes" {
		keyType = crypto.AES
	} else if symKeyType == "sm4" {
		keyType = crypto.SM4

	} else {
		return fmt.Errorf("invalid symKeyType, %s", symKeyType)
	}

	msg, err := localhibe.EncryptHibeMsg([]byte(hibePlaintext), receiverIds, paramsList, keyType)
	if err != nil {
		return err
	}

	hibeMsgBytes, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	result := &Result{
		Code:                  0,
		Message:               "SUCCESS",
		ContractResultCode:    0,
		ContractResultMessage: "OK",
		HibeExecMsg:           string(hibeMsgBytes),
	}

	bytes, err = json.Marshal(result)
	if err != nil {
		return err
	}
	fmt.Println(string(bytes))
	return nil
}
