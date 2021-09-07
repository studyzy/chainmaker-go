/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"fmt"
	"io/ioutil"

	"chainmaker.org/chainmaker/utils/v2"
	"github.com/spf13/cobra"
)

func GetShortCertBase64() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "getShortCertBase64",
		Short: "Short Cert",
		Long:  "Get short certificate, the params(hash-algorithm, cert-path)",
		RunE: func(_ *cobra.Command, _ []string) error {
			return getShortCertBase64()
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&hashAlgo, "hash-algorithm", "SHA256", "hash algorithm set in chain configuration")
	flags.StringVar(&certPath, "cert-path", "", "path for the target certificate")

	return cmd
}

func getShortCertBase64() error {
	cert, err := ioutil.ReadFile(certPath)
	if err != nil {
		return err
	}

	certId, err := utils.GetCertificateIdHex(cert, hashAlgo)
	if err != nil {
		return err
	}

	//certBase64 := base64.StdEncoding.EncodeToString(certId)

	result := &Result{
		ShortCert: certId,
	}
	fmt.Println(result.ToJsonString())

	return nil
}
