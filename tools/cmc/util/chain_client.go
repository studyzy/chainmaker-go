// Copyright (C) BABEC. All rights reserved.
// Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	sdk "chainmaker.org/chainmaker/sdk-go/v2"
)

// CreateChainClient create a chain client with sdk config file path.
// sdkConfPath must not empty. chainId, orgId, userTlsCrtPath, userTlsKeyPath, userSignCrtPath, userSignKeyPath
// will overwrite sdk config generated from sdkConfPath if they are not empty string,
// otherwise sdk config will not be overwritten.
func CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtPath, userTlsKeyPath,
	userSignCrtPath, userSignKeyPath string) (*sdk.ChainClient, error) {
	cc, err := sdk.NewChainClient(
		sdk.WithConfPath(sdkConfPath),
		sdk.WithChainClientChainId(chainId),
		sdk.WithChainClientOrgId(orgId),
		sdk.WithUserCrtFilePath(userTlsCrtPath),
		sdk.WithUserKeyFilePath(userTlsKeyPath),
		sdk.WithUserSignCrtFilePath(userSignCrtPath),
		sdk.WithUserSignKeyFilePath(userSignKeyPath),
	)
	if err != nil {
		return nil, err
	}

	// Enable certificate compression
	if cc.GetAuthType() == sdk.PermissionedWithCert {
		err = cc.EnableCertHash()
	}
	if err != nil {
		return nil, err
	}
	return cc, nil
}

func AttachAndRequiredFlags(cmd *cobra.Command, flags *pflag.FlagSet, names []string) {
	cmdFlags := cmd.Flags()
	for _, name := range names {
		if flag := flags.Lookup(name); flag != nil {
			cmdFlags.AddFlag(flag)
		}
		cmd.MarkFlagRequired(name)
	}
}

func AttachFlags(cmd *cobra.Command, flags *pflag.FlagSet, names []string) {
	cmdFlags := cmd.Flags()
	for _, name := range names {
		if flag := flags.Lookup(name); flag != nil {
			cmdFlags.AddFlag(flag)
		}
	}
}
