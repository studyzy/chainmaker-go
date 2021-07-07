/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package payload

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"strings"

	"chainmaker.org/chainmaker/pb-go/consts"

	sdkPbCommon "chainmaker.org/chainmaker/pb-go/common"

	"github.com/gogo/protobuf/proto"
	"github.com/spf13/cobra"
)

var (
	createOutput string
)

func createCMD() *cobra.Command {
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create pb file command",
		Long:  "Create pb file command",
	}

	flags := createCmd.PersistentFlags()
	flags.StringVarP(&createOutput, "output", "o", "./collect.pb", "specify output file")

	createCmd.AddCommand(createConfigUpdatePayloadCMD())
	createCmd.AddCommand(createContractMgmtPayloadCMD())

	return createCmd
}

func createConfigUpdatePayloadCMD() *cobra.Command {
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Config command",
		Long:  "Config command",
		RunE: func(_ *cobra.Command, _ []string) error {
			return createConfigUpdatePayload()
		},
	}

	attachFlags(configCmd, []string{
		"chain-id", "contract-name", "method", "kv-pairs", "sequence",
	})

	return configCmd
}

func createContractMgmtPayloadCMD() *cobra.Command {
	contractCmd := &cobra.Command{
		Use:   "contract",
		Short: "Contract command",
		Long:  "Contract command",
		RunE: func(_ *cobra.Command, _ []string) error {
			return createContractMgmtPayload()
		},
	}

	attachFlags(contractCmd, []string{
		"chain-id", "contract-name", "version", "runtime", "method", "kv-pairs", "byte-code-path",
	})

	return contractCmd
}

func createConfigUpdatePayload() error {
	payload := &sdkPbCommon.Payload{
		ChainId:      chainId,
		ContractName: contractName,
		Method:       method,
		Parameters:   []*sdkPbCommon.KeyValuePair{},
		Sequence:     uint64(sequence),
	}
	kvs := strings.Split(kvPairs, ";")
	for _, kv := range kvs {
		s := strings.Split(kv, ":")
		if len(s) != 2 {
			return fmt.Errorf("Key value invalid: %s", kv)
		}
		payload.Parameters = append(payload.Parameters, &sdkPbCommon.KeyValuePair{
			Key:   s[0],
			Value: s[1],
		})
	}

	bytes, err := proto.Marshal(payload)
	if err != nil {
		return fmt.Errorf("SystemContractPayload marshal error: %s", err)
	}

	if err = ioutil.WriteFile(createOutput, bytes, 0600); err != nil {
		return fmt.Errorf("Write to file %s error: %s", createOutput, err)
	}

	return nil
}

func GenerateInstallContractPayload(contractName, version string, runtimeType sdkPbCommon.RuntimeType, bytecode []byte,
	initParameters []*sdkPbCommon.KeyValuePair) (*sdkPbCommon.TransactPayload, error) {
	var pairs []*sdkPbCommon.KeyValuePair
	pairs = append(pairs, &sdkPbCommon.KeyValuePair{
		Key:   consts.ContractManager_Install_ContractName.String(),
		Value: contractName,
	})
	pairs = append(pairs, &sdkPbCommon.KeyValuePair{
		Key:   consts.ContractManager_Install_Version.String(),
		Value: version,
	})
	pairs = append(pairs, &sdkPbCommon.KeyValuePair{
		Key:   consts.ContractManager_Install_RuntimeType.String(),
		Value: runtimeType.String(),
	})
	pairs = append(pairs, &sdkPbCommon.KeyValuePair{
		Key:   consts.ContractManager_Install_ByteCode.String(),
		Value: base64.StdEncoding.EncodeToString(bytecode),
	})
	for _, kv := range initParameters {
		pairs = append(pairs, kv)
	}
	payload := &sdkPbCommon.TransactPayload{
		ContractName: sdkPbCommon.ContractName_SYSTEM_CONTRACT_USER_CONTRACT_MANAGE.String(),
		Method:       consts.ContractManager_INIT_CONTRACT.String(),
		Parameters:   pairs,
	}
	return payload, nil
}

func createContractMgmtPayload() error {
	runtimeValue, ok := sdkPbCommon.RuntimeType_value[strings.ToUpper(runtime)]
	if !ok {
		return fmt.Errorf("Runtime invalid: %s", runtime)
	}
	codeBytes, err := ioutil.ReadFile(byteCodePath)
	if err != nil {
		return fmt.Errorf("Read from file %s error: %s", byteCodePath, err)
	}
	payload, err := GenerateInstallContractPayload(contractName, version, sdkPbCommon.RuntimeType(runtimeValue), codeBytes, []*sdkPbCommon.KeyValuePair{})
	if err != nil {
		return fmt.Errorf("Generate install contract payload error: %s", err)
	}
	kvs := strings.Split(kvPairs, ";")
	for _, kv := range kvs {
		s := strings.Split(kv, ":")
		if len(s) != 2 {
			return fmt.Errorf("Key value invalid: %s", kv)
		}
		payload.Parameters = append(payload.Parameters, &sdkPbCommon.KeyValuePair{
			Key:   s[0],
			Value: s[1],
		})
	}

	bytes, err := proto.Marshal(payload)
	if err != nil {
		return fmt.Errorf("ContractMgmtPayload marshal error: %s", err)
	}

	if err = ioutil.WriteFile(createOutput, bytes, 0600); err != nil {
		return fmt.Errorf("Write to file %s error: %s", createOutput, err)
	}

	return nil
}
