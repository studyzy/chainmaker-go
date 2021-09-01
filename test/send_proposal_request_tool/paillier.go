/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"

	"chainmaker.org/chainmaker/common/v2/crypto/paillier"
	"github.com/spf13/cobra"
)

const (
	pubKeyFileNameSuffix = ".pubKey"
	prvKeyFileNameSuffix = ".prvKey"
)

var (
	pubKeyStr string
	prvKeyStr string
	//methodStr     string
	plaintext     string
	ciphertextStr string
)

func PaillierCMD() *cobra.Command {
	paillierCmd := &cobra.Command{
		Use:   "paillier",
		Short: "ChainMaker paillier command",
		Long:  "ChainMaker paillier command",
	}

	paillierCmd.AddCommand(keyGenCMD())
	paillierCmd.AddCommand(encryptCMD())
	paillierCmd.AddCommand(decryptCMD())
	return paillierCmd
}

func keyGenCMD() *cobra.Command {
	keyGenCmd := &cobra.Command{
		Use:   "genKey",
		Short: "Generate paillier's private, public Keys",
		Long:  "Generate paillier's private, public Keys",
		RunE: func(_ *cobra.Command, _ []string) error {
			return generatePrvPubKeys()
		},
	}

	return keyGenCmd
}

func encryptCMD() *cobra.Command {
	keyGenCmd := &cobra.Command{
		Use:   "encrypt",
		Short: "converts the provided plaintext to ciphertext, using the provided public key.",
		Long:  "converts the provided plaintext to ciphertext, using the provided public key.",
		RunE: func(_ *cobra.Command, _ []string) error {
			return paillierEncrypt()
		},
	}

	flags := keyGenCmd.Flags()
	flags.StringVarP(&plaintext, "pt", "", "", "Plaintext")
	flags.StringVarP(&pubKeyStr, "pubkey", "", "", "Public key")

	return keyGenCmd
}

func decryptCMD() *cobra.Command {
	keyGenCmd := &cobra.Command{
		Use:   "decrypt",
		Short: "decrypt the supplied ciphertext into plaintext using the private key provided",
		Long:  "decrypt the supplied ciphertext into plaintext using the private key provided",
		RunE: func(_ *cobra.Command, _ []string) error {
			return paillierDecrypt()
		},
	}

	flags := keyGenCmd.Flags()
	flags.StringVarP(&ciphertextStr, "ct", "", "", "Ciphertext")
	flags.StringVarP(&prvKeyStr, "prvkey", "", "", "Private key")

	return keyGenCmd
}

func generatePrvPubKeys() error {
	prvKey, err := paillier.GenKey()
	if err != nil {
		return err
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

	fmt.Printf("paillier pubKey: [%s]\n", pubKeyBytes)
	fmt.Printf("paillier prvKey: \n%s", prvKeyBytes)
	return nil
}

func paillierEncrypt() error {
	pubKey := new(paillier.PubKey)
	err := pubKey.Unmarshal([]byte(pubKeyStr))
	if err != nil {
		return err
	}
	pt, ok := new(big.Int).SetString(plaintext, 10)
	if !ok {
		return errors.New("invalid plaintext, please check it")
	}
	result, err := pubKey.Encrypt(pt)
	if err != nil {
		return err
	}

	resultBytes, err := result.Marshal()
	base64Result := base64.StdEncoding.EncodeToString(resultBytes)
	fmt.Printf("encrypt [%s] to: [%s]\n", plaintext, base64Result)

	return nil
}

func paillierDecrypt() error {
	prvKey := new(paillier.PrvKey)

	if err := prvKey.Unmarshal([]byte(prvKeyStr)); err != nil {
		return err
	}

	ct := new(paillier.Ciphertext)
	base64Decode, err := base64.StdEncoding.DecodeString(ciphertextStr)
	if err != nil {
		return err
	}
	err = ct.Unmarshal(base64Decode)
	if err != nil {
		return err
	}

	result, err := prvKey.Decrypt(ct)
	if err != nil {
		return err
	}

	fmt.Printf("decrypt [%s] to: [%s]\n", ciphertextStr, result.String())

	return nil
}
