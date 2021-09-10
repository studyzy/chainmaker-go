/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package paillier

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"chainmaker.org/chainmaker/common/v2/crypto/paillier"
	"github.com/spf13/cobra"
)

var (
	// genKeyCmd flags
	paillierKeySavePath string
	paillierKeyFileName string
)

func PaillierCMD() *cobra.Command {
	paillierCmd := &cobra.Command{
		Use:   "paillier",
		Short: "ChainMaker paillier command",
		Long:  "ChainMaker paillier command",
	}

	paillierCmd.AddCommand(genKeyCMD())
	return paillierCmd
}

func genKeyCMD() *cobra.Command {
	genKeyCmd := &cobra.Command{
		Use:   "genKey",
		Short: "generates paillier private public key",
		Long:  "generates paillier private public key",
		RunE: func(_ *cobra.Command, _ []string) error {
			return genKey()
		},
	}

	flags := genKeyCmd.Flags()
	flags.StringVarP(&paillierKeySavePath, "path", "", "", "the result storage file path, and the file name is the id")
	flags.StringVarP(&paillierKeyFileName, "name", "", "", "")

	return genKeyCmd
}

func genKey() error {
	prvFilePath := filepath.Join(paillierKeySavePath, fmt.Sprintf("%s.prvKey", paillierKeyFileName))
	pubFilePath := filepath.Join(paillierKeySavePath, fmt.Sprintf("%s.pubKey", paillierKeyFileName))

	_, err := pathExists(prvFilePath)
	if err != nil {
		return err
	}
	exist, err := pathExists(pubFilePath)
	if exist {
		return fmt.Errorf("file [ %s ] already exist", pubFilePath)
	}

	if err != nil {
		return err
	}
	prvKey, err := paillier.GenKey()
	if err != nil {
		return err
	}

	if exist {
		return fmt.Errorf("file [ %s ] already exist", prvFilePath)
	}
	pubKey, err := prvKey.GetPubKey()
	if err != nil {
		return err
	}

	pubKeyBytes, err := pubKey.Marshal()
	if err != nil {
		return err
	}

	prvKeyBytes, err := prvKey.Marshal()
	if err != nil {
		return err
	}

	if err = os.MkdirAll(paillierKeySavePath, os.ModePerm); err != nil {
		return fmt.Errorf("mk pailier dir failed, %s", err.Error())
	}

	if err = ioutil.WriteFile(prvFilePath,
		prvKeyBytes, 0600); err != nil {
		return fmt.Errorf("save paillier to file [%s] failed, %s", prvFilePath, err.Error())
	}
	fmt.Printf("[paillier Private Key] storage file path: %s\n", prvFilePath)

	if err = ioutil.WriteFile(pubFilePath,
		pubKeyBytes, 0600); err != nil {
		return fmt.Errorf("save paillier to file [%s] failed, %s", pubFilePath, err.Error())
	}
	fmt.Printf("[paillier Public Key] storage file path: %s\n", pubFilePath)
	return nil
}

// pathExists is used to determine whether a file or folder exists
func pathExists(path string) (bool, error) {
	if path == "" {
		return false, errors.New("invalid parameter, the file path cannot be empty")
	}
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
