/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package bulletproofs

import (
	"chainmaker.org/chainmaker/common/crypto/bulletproofs"
	"encoding/base64"
	"errors"
	"fmt"

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

	ok, err := bulletproofs.Helper().NewBulletproofs().PedersenVerify(commitment, opening, uint64(valueX))
	if err != nil {
		return err
	}

	if ok {
		fmt.Printf("verify: true\n")
		return nil
	} else {
		fmt.Printf("verify: false\n")
		return nil
	}
}
