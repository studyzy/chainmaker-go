/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"chainmaker.org/chainmaker-go/tools/cmc/util"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	sdk "chainmaker.org/chainmaker/sdk-go/v2"
)

const (
	addNodeOrg = iota
	removeNodeOrg
	updateNodeOrg
)

func configConsensueNodeOrgCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "consensusnodeorg",
		Short: "consensus node org management",
		Long:  "consensus node org management",
	}
	cmd.AddCommand(addConsensusNodeOrgCMD())
	cmd.AddCommand(removeConsensusNodeOrgCMD())
	cmd.AddCommand(updateConsensusNodeOrgCMD())

	return cmd
}

func addConsensusNodeOrgCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "add consensus node org cmd",
		Long:  "add consensus node org cmd",
		RunE: func(_ *cobra.Command, _ []string) error {
			return configConsensusNodeOrg(addNodeOrg)
		},
	}

	attachFlags(cmd, []string{
		flagUserSignKeyFilePath, flagUserSignCrtFilePath,
		flagSdkConfPath, flagOrgId, flagEnableCertHash, flagNodeOrgId, flagNodeIds,
		flagAdminCrtFilePaths, flagAdminKeyFilePaths, flagUserTlsCrtFilePath, flagUserTlsKeyFilePath,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagAdminCrtFilePaths)
	cmd.MarkFlagRequired(flagAdminKeyFilePaths)
	cmd.MarkFlagRequired(flagNodeOrgId)
	cmd.MarkFlagRequired(flagNodeIds)

	return cmd
}

func removeConsensusNodeOrgCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove",
		Short: "remove consensus node org cmd",
		Long:  "remove consensus node org cmd",
		RunE: func(_ *cobra.Command, _ []string) error {
			return configConsensusNodeOrg(removeNodeOrg)
		},
	}

	attachFlags(cmd, []string{
		flagUserSignKeyFilePath, flagUserSignCrtFilePath,
		flagSdkConfPath, flagOrgId, flagEnableCertHash, flagNodeOrgId,
		flagAdminCrtFilePaths, flagAdminKeyFilePaths, flagUserTlsCrtFilePath, flagUserTlsKeyFilePath,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagAdminCrtFilePaths)
	cmd.MarkFlagRequired(flagAdminKeyFilePaths)
	cmd.MarkFlagRequired(flagNodeOrgId)

	return cmd
}

func updateConsensusNodeOrgCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "update consensus node org cmd",
		Long:  "update consensus node org cmd",
		RunE: func(_ *cobra.Command, _ []string) error {
			return configConsensusNodeOrg(updateNodeOrg)
		},
	}

	attachFlags(cmd, []string{
		flagUserSignKeyFilePath, flagUserSignCrtFilePath,
		flagSdkConfPath, flagOrgId, flagEnableCertHash, flagNodeOrgId, flagNodeIds,
		flagAdminCrtFilePaths, flagAdminKeyFilePaths, flagUserTlsCrtFilePath, flagUserTlsKeyFilePath,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagAdminCrtFilePaths)
	cmd.MarkFlagRequired(flagAdminKeyFilePaths)
	cmd.MarkFlagRequired(flagNodeOrgId)
	cmd.MarkFlagRequired(flagNodeIds)

	return cmd
}

func configConsensusNodeOrg(op int) error {
	nodeIdSlice := strings.Split(nodeIds, ",")
	adminKeys := strings.Split(adminKeyFilePaths, ",")
	adminCrts := strings.Split(adminCrtFilePaths, ",")
	if len(adminKeys) == 0 || len(adminCrts) == 0 {
		return errAdminOrgIdKeyCertIsEmpty
	}
	if len(adminKeys) != len(adminCrts) {
		return fmt.Errorf(ADMIN_ORGID_KEY_CERT_LENGTH_NOT_EQUAL_FORMAT, len(adminKeys), len(adminCrts))
	}

	client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
		userSignCrtFilePath, userSignKeyFilePath)
	if err != nil {
		return err
	}
	defer client.Stop()

	var payload *common.Payload
	switch op {
	case addNodeOrg:
		payload, err = client.CreateChainConfigConsensusNodeOrgAddPayload(nodeOrgId, nodeIdSlice)
	case removeNodeOrg:
		payload, err = client.CreateChainConfigConsensusNodeOrgDeletePayload(nodeOrgId)
	case updateNodeOrg:
		payload, err = client.CreateChainConfigConsensusNodeOrgUpdatePayload(nodeOrgId, nodeIdSlice)
	default:
		err = errors.New("invalid node address operation")
	}
	if err != nil {
		return err
	}

	endorsementEntrys := make([]*common.EndorsementEntry, len(adminKeys))
	for i := range adminKeys {
		e, err := sdk.SignPayloadWithPath(adminKeys[i], adminCrts[i], payload)
		if err != nil {
			return err
		}

		endorsementEntrys[i] = e
	}

	resp, err := client.SendChainConfigUpdateRequest(payload, endorsementEntrys, timeout, syncResult)
	if err != nil {
		return err
	}
	err = util.CheckProposalRequestResp(resp, false)
	if err != nil {
		return err
	}
	fmt.Printf("consensusnodeorg response %+v\n", resp)
	return nil
}
