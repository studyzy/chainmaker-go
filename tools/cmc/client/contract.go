/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import "github.com/spf13/cobra"

func contractCMD() *cobra.Command {
	contractCmd := &cobra.Command{
		Use:   "contract",
		Short: "contract command",
		Long:  "contract command",
	}

	contractCmd.AddCommand(userContractCMD())
	contractCmd.AddCommand(systemContractCMD())

	return contractCmd
}
