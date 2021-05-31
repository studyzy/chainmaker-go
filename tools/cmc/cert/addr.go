/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cert

import (
	bcx509 "chainmaker.org/chainmaker/common/crypto/x509"
	"chainmaker.org/chainmaker/common/evmutils"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"github.com/spf13/cobra"
	"io/ioutil"
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

