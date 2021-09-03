/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package bulletproofs

import (
	"encoding/base64"
	"fmt"

	"chainmaker.org/chainmaker/common/v2/crypto/bulletproofs"
	"github.com/spf13/cobra"
)

func genOpeningCMD() *cobra.Command {
	genOpeningCmd := &cobra.Command{
		Use:   "genOpening",
		Short: "Bulletproofs generate opening command",
		Long:  "Bulletproofs generate opening command",
		RunE: func(_ *cobra.Command, _ []string) error {
			return genOpening()
		},
	}

	return genOpeningCmd
}

func genOpening() error {
	opening, err := bulletproofs.PedersenRNG()
	if err != nil {
		return err
	}

	openingStr := base64.StdEncoding.EncodeToString(opening)
	fmt.Printf("opening: [%s]\n", openingStr)

	return nil
}
