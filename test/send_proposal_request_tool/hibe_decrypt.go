package main

import (
	"chainmaker.org/chainmaker-go/common/crypto"
	localhibe "chainmaker.org/chainmaker-go/common/crypto/hibe"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/samkumar/hibe"
	"github.com/spf13/cobra"
	"io/ioutil"
)

func HibeDecrypt() *cobra.Command {
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

func decryptHibeMessage() error {
	var result Result

	hibeParamsBytes, err := readHibeParamsWithFilePath(hibeLocalParams)
	if err != nil {
		return err
	}

	localParams, ok := new(hibe.Params).Unmarshal(hibeParamsBytes)
	if !ok {
		return errors.New("hibe.Params.Unmarshal failed, please check your file")
	}

	hibePrvKeyBytes, err := readHibePrvKeysWithFilePath(hibePrvKey)
	if err != nil {
		return err
	}

	prvKey, ok := new(hibe.PrivateKey).Unmarshal(hibePrvKeyBytes)
	if !ok {
		return errors.New("hibe.PrivateKey.Unmarshal failed, please check your file")
	}

	hibeMsgMap := make(map[string]string)
	err = json.Unmarshal([]byte(hibeMsg), &hibeMsgMap)
	if err != nil {
		return err
	}

	var keyType crypto.KeyType
	if symKeyType == "aes" {
		keyType = crypto.AES
	} else if symKeyType == "sm4" {
		keyType = crypto.SM4

	} else {
		return fmt.Errorf("invalid symKeyType, %s", symKeyType)
	}

	message, err := localhibe.DecryptHibeMsg(localId, localParams, prvKey, hibeMsgMap, keyType)
	if err != nil {
		result.Code = 1
		result.Message = err.Error()
		return err
	}

	result.Code = 0
	result.Message = "SUCCESS"
	result.ContractResultCode = 0
	result.ContractResultMessage = "OK"
	result.HibeExecMsg = string(message)

	bytes, err := json.Marshal(result)
	if err != nil {
		return err
	}
	fmt.Println(string(bytes))
	return nil
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
