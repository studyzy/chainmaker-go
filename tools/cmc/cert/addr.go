/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cert

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"io/ioutil"

	sdk "chainmaker.org/chainmaker-sdk-go"

	bcx509 "chainmaker.org/chainmaker-go/common/crypto/x509"
	"chainmaker.org/chainmaker-go/common/evmutils"
	"github.com/mr-tron/base58"
	"github.com/spf13/cobra"
)

func addrCMD() *cobra.Command {
	addrCmd := &cobra.Command{
		Use:   "addr",
		Short: "get addr from cert",
		Long:  "get addr from cert",
		RunE: func(_ *cobra.Command, _ []string) error {
			return getAddr()
		},
	}

	attachFlags(addrCmd, []string{
		flagCertPath,
	})

	return addrCmd
}

func getAddr() error {

	certBytes, err := ioutil.ReadFile(certPath)
	if err != nil {
		return fmt.Errorf("read cert file [%s] failed, %s", certPath, err)
	}

	block, _ := pem.Decode(certBytes)
	cert, err := bcx509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Errorf("parseCertificate cert failed, %s", err)
	}

	ski := hex.EncodeToString(cert.SubjectKeyId)
	addrInt, err := evmutils.MakeAddressFromHex(ski)
	if err != nil {
		return fmt.Errorf("make address from cert SKI failed, %s", err)
	}

	fmt.Printf("ski:       %s\n", ski)
	fmt.Printf("addr(Int): %s\n", addrInt.String())
	fmt.Printf("addr:      0x%x\n", addrInt.AsStringKey())
	return nil
}

func certToUserAddrInStake() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "userAddr",
		Short: "get user addr feature of the DPoS from cert",
		RunE: func(_ *cobra.Command, _ []string) error {
			if len(certPath) == 0 {
				return fmt.Errorf("cert path is null")
			}
			certContent, err := ioutil.ReadFile(certPath)
			if err != nil {
				return fmt.Errorf("read cert content failed, reason: %s", err)
			}
			cert, err := sdk.ParseCert(certContent)
			if err != nil {
				return fmt.Errorf("parse cert failed, reason: %s", err)
			}
			pubkey, err := cert.PublicKey.Bytes()
			if err != nil {
				return fmt.Errorf("get pubkey failed from cert, reason: %s", err)
			}
			hash := sha256.Sum256(pubkey)
			addr := base58.Encode(hash[:])
			fmt.Printf("address: %s \n\nfrom cert: %s\n", addr, certPath)

			return nil
		},
	}

	attachFlags(cmd, []string{
		flagCertPath,
	})

	cmd.MarkFlagRequired(certPath)
	return cmd
}
