/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"crypto/sha256"
	"fmt"
	"io/ioutil"

	"github.com/spf13/cobra"
)

func GenerateHashCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generateHash",
		Short: "generate hash from code",
		Long:  "generate hash from code",
		RunE: func(_ *cobra.Command, _ []string) error {
			return generateHash()
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&wasmPath, "wasm-path", "w", "../wasm/counter-go.wasm", "specify wasm path")

	return cmd
}

func generateHash() error {
	wasmBin, err := ioutil.ReadFile(wasmPath)
	if err != nil {
		return err
	}
	contractCodeHash := sha256.Sum256(wasmBin)
	fmt.Printf("wasm file code hash is:%s", contractCodeHash[:])
	return nil
}
