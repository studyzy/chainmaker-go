/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package bulletproofs

import (
	"encoding/base64"
	"errors"
	"fmt"

	"chainmaker.org/chainmaker/common/v2/crypto/bulletproofs"

	"github.com/spf13/cobra"
)

func pedersenVerifyCMD() *cobra.Command {
	pedersenVerifyCmd := &cobra.Command{
		Use:   "pedersenVerify",
		Short: "Bulletproofs pedersenVerify command",
		Long:  "Bulletproofs pedersenVerify command",
		RunE: func(_ *cobra.Command, _ []string) error {
			return pedersenVerify()
		},
	}

	flags := pedersenVerifyCmd.Flags()
	flags.StringVarP(&commitmentXStr, "commitment", "", "", "commitment")
	flags.StringVarP(&openingXStr, "opening", "", "", "opening")
	flags.Int64VarP(&valueX, "value", "", -1, "value")

	return pedersenVerifyCmd
}

func pedersenVerify() error {
	if valueX == -1 {
		return errors.New("invalid input, please check it")
	}

	if commitmentXStr == "" || openingXStr == "" {
		return errors.New("invalid input, please check it")
	}

	commitment, err := base64.StdEncoding.DecodeString(commitmentXStr)
	if err != nil {
		return err
	}

	opening, err := base64.StdEncoding.DecodeString(openingXStr)
	if err != nil {
		return err
	}

	ok, err := bulletproofs.PedersenVerify(commitment, opening, uint64(valueX))
	if err != nil {
		return err
	}

	fmt.Printf("verify: %t\n", ok)
	return nil
}
