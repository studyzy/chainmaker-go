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

	"chainmaker.org/chainmaker/common/v2/crypto"
	"chainmaker.org/chainmaker/common/v2/crypto/hibe"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"github.com/spf13/cobra"
)

func HibeDecryptCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hibeDecrypt",
		Short: "get hibe transaction by transaction Id, and decrypt",
		Long:  "",
		RunE: func(_ *cobra.Command, _ []string) error {
			return decryptHibeMessage()
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&hibeLocalParams, "hibe-params-file", "", "", "your hibe system params file path")
	flags.StringVarP(&localId, "hibe-local-id", "", "", "your hibe id")
	flags.StringVarP(&hibePrvKey, "hibe-prvKey-file", "", "", "your hibe prvKey file path")
	flags.StringVarP(&symKeyType, "sym-key-type", "", "aes", "symmetric key type (aes or sm4)")
	flags.StringVarP(&hibeMsg, "hibe-msg", "", "", "decrypted ciphertext")
	return cmd
}

func decryptHibeMessageExec() (string, commonPb.TxStatusCode, string) {
	var result_output string

	hibeParamsBytes, err := readHibeParamsWithFilePath(hibeLocalParams)
	if err != nil {
		return result_output, 1, fmt.Sprintf("readHibeParamsWithFilePath, %s, err: %s", hibeLocalParams, err)
	}

	localParams, ok := new(hibe.Params).Unmarshal(hibeParamsBytes)
	if !ok {
		return result_output, 1, fmt.Sprintf("hibe.Params.Unmarshal failed, please check your file, err: %v", ok)
	}

	hibePrvKeyBytes, err := readHibePrvKeysWithFilePath(hibePrvKey)
	if err != nil {
		return result_output, 1, fmt.Sprintf("readHibePrvKeysWithFilePath, %s, err: %s", hibePrvKey, err)
	}

	prvKey, ok := new(hibe.PrivateKey).Unmarshal(hibePrvKeyBytes)
	if !ok {
		return result_output, 1, fmt.Sprintf("hibe.PrivateKey.Unmarshal failed, please check your file, err: %v", ok)
	}

	hibeMsgMap := make(map[string]string)
	err = json.Unmarshal([]byte(hibeMsg), &hibeMsgMap)
	if err != nil {
		return result_output, 1, fmt.Sprintf("Unmarshal failed, please check your file, err: %s", err)
	}

	var keyType crypto.KeyType
	if symKeyType == "aes" {
		keyType = crypto.AES
	} else if symKeyType == "sm4" {
		keyType = crypto.SM4
	} else {
		return result_output, 1, fmt.Sprintf("invalid symKeyType, %s", symKeyType)
	}

	message, err := hibe.DecryptHibeMsg(localId, localParams, prvKey, hibeMsgMap, keyType)
	if err != nil {
		return result_output, 1, fmt.Sprintf("DecryptHibeMsg failure, err: %s", err)
	}

	return string(message), 0, ""
}

// Returns the serialized byte array of hibeParams
func readHibeParamsWithFilePath(hibeParamsFilePath string) ([]byte, error) {
	paramsBytes, err := ioutil.ReadFile(hibeParamsFilePath)
	if err != nil {
		return nil, fmt.Errorf("open hibe params file failed, [err:%s]", err)
	}

	return paramsBytes, nil
}

// Returns the serialized byte array of hibePrvKey
func readHibePrvKeysWithFilePath(hibePrvKeyFilePath string) ([]byte, error) {
	prvKeyBytes, err := ioutil.ReadFile(hibePrvKeyFilePath)
	if err != nil {
		return nil, fmt.Errorf("open hibe privateKey file failed, [err:%s]", err)
	}

	return prvKeyBytes, nil
}

func decryptHibeMessage() error {
	var result Result
	hibeMessage, result_code, result_err := decryptHibeMessageExec()
	result_msg := "SUCCESS"
	if result_err != "" {
		result_msg = result_err
	}

	result.Code = result_code
	result.Message = result_msg
	result.ContractResultCode = 0
	result.ContractResultMessage = "OK"
	result.HibeExecMsg = hibeMessage

	fmt.Println(result.ToJsonString())
	return nil
}
