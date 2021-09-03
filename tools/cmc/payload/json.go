/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package payload

import (
	"fmt"
	"io/ioutil"

	sdkPbCommon "chainmaker.org/chainmaker/pb-go/v2/common"

	"github.com/gogo/protobuf/proto"
	"github.com/hokaccha/go-prettyjson"
	"github.com/spf13/cobra"
)

var (
	jsonInput string
)

func jsonCMD() *cobra.Command {
	jsonCmd := &cobra.Command{
		Use:   "tojson",
		Short: "Parse to json command",
		Long:  "Parse to json command",
	}

	flags := jsonCmd.PersistentFlags()
	flags.StringVarP(&jsonInput, "input", "i", "./collect.pb", "specify input file")

	jsonCmd.AddCommand(printConfigUpdatePayloadCMD())
	jsonCmd.AddCommand(printContractMgmtPayloadCMD())

	return jsonCmd
}

func printConfigUpdatePayloadCMD() *cobra.Command {
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Config command",
		Long:  "Config command",
		RunE: func(_ *cobra.Command, _ []string) error {
			return printConfigUpdatePayload()
		},
	}
	return configCmd
}

func printContractMgmtPayloadCMD() *cobra.Command {
	configCmd := &cobra.Command{
		Use:   "contract",
		Short: "Contract command",
		Long:  "Contract command",
		RunE: func(_ *cobra.Command, _ []string) error {
			return printContractMgmtPayload()
		},
	}
	return configCmd
}

func printConfigUpdatePayload() error {
	raw, err := ioutil.ReadFile(jsonInput)
	if err != nil {
		return fmt.Errorf("load file %s error: %s", jsonInput, err)
	}

	payload := &sdkPbCommon.Payload{}
	if err := proto.Unmarshal(raw, payload); err != nil {
		return fmt.Errorf("SystemContractPayload unmarshal error: %s", err)
	}

	result, err := prettyjson.Marshal(payload)
	if err != nil {
		return fmt.Errorf("SystemContractPayload marshal error: %s", err)
	}
	fmt.Println(string(result))

	return nil
}

func printContractMgmtPayload() error {
	raw, err := ioutil.ReadFile(jsonInput)
	if err != nil {
		return fmt.Errorf("load file %s error: %s", jsonInput, err)
	}

	payload := &sdkPbCommon.Payload{}
	if err := proto.Unmarshal(raw, payload); err != nil {
		return fmt.Errorf("ContractMgmtPayload unmarshal error: %s", err)
	}

	result, err := prettyjson.Marshal(payload)
	if err != nil {
		return fmt.Errorf("ContractMgmtPayload marshal error: %s", err)
	}
	fmt.Println(string(result))

	return nil
}
