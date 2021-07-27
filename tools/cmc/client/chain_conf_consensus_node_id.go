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
	"chainmaker.org/chainmaker/pb-go/common"
	sdk "chainmaker.org/chainmaker/sdk-go"
)

const (
	addNodeId = iota
	removeNodeId
	updateNodeId
)

func configConsensueNodeIdCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "consensusnodeid",
		Short: "consensus node id management",
		Long:  "consensus node id management",
	}
	cmd.AddCommand(addConsensusNodeIdCMD())
	cmd.AddCommand(removeConsensusNodeIdCMD())
	cmd.AddCommand(updateConsensusNodeIdCMD())

	return cmd
}

func addConsensusNodeIdCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "add consensus node id cmd",
		Long:  "add consensus node id cmd",
		RunE: func(_ *cobra.Command, _ []string) error {
			return configConsensusNodeId(addNodeId)
		},
	}

	attachFlags(cmd, []string{
		flagUserSignKeyFilePath, flagUserSignCrtFilePath,
		flagSdkConfPath, flagOrgId, flagEnableCertHash, flagNodeOrgId, flagNodeId,
		flagAdminCrtFilePaths, flagAdminKeyFilePaths, flagUserTlsCrtFilePath, flagUserTlsKeyFilePath,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagAdminCrtFilePaths)
	cmd.MarkFlagRequired(flagAdminKeyFilePaths)
	cmd.MarkFlagRequired(flagNodeOrgId)
	cmd.MarkFlagRequired(flagNodeId)

	return cmd
}

func removeConsensusNodeIdCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove",
		Short: "remove consensus node id cmd",
		Long:  "remove consensus node id cmd",
		RunE: func(_ *cobra.Command, _ []string) error {
			return configConsensusNodeId(removeNodeId)
		},
	}

	attachFlags(cmd, []string{
		flagUserSignKeyFilePath, flagUserSignCrtFilePath,
		flagSdkConfPath, flagOrgId, flagEnableCertHash, flagNodeOrgId, flagNodeId,
		flagAdminCrtFilePaths, flagAdminKeyFilePaths, flagUserTlsCrtFilePath, flagUserTlsKeyFilePath,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagAdminCrtFilePaths)
	cmd.MarkFlagRequired(flagAdminKeyFilePaths)
	cmd.MarkFlagRequired(flagNodeOrgId)
	cmd.MarkFlagRequired(flagNodeId)

	return cmd
}

func updateConsensusNodeIdCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "update consensus node id cmd",
		Long:  "update consensus node id cmd",
		RunE: func(_ *cobra.Command, _ []string) error {
			return configConsensusNodeId(updateNodeId)
		},
	}

	attachFlags(cmd, []string{
		flagUserSignKeyFilePath, flagUserSignCrtFilePath,
		flagSdkConfPath, flagOrgId, flagEnableCertHash, flagNodeOrgId, flagNodeIdOld, flagNodeId,
		flagAdminCrtFilePaths, flagAdminKeyFilePaths, flagUserTlsCrtFilePath, flagUserTlsKeyFilePath,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagAdminCrtFilePaths)
	cmd.MarkFlagRequired(flagAdminKeyFilePaths)
	cmd.MarkFlagRequired(flagNodeOrgId)
	cmd.MarkFlagRequired(flagNodeIdOld)
	cmd.MarkFlagRequired(flagNodeId)

	return cmd
}

func configConsensusNodeId(op int) error {
	adminKeys := strings.Split(adminKeyFilePaths, ",")
	adminCrts := strings.Split(adminCrtFilePaths, ",")
	if len(adminKeys) == 0 || len(adminCrts) == 0 {
		return ErrAdminOrgIdKeyCertIsEmpty
	}
	if len(adminKeys) != len(adminCrts) {
		return fmt.Errorf(ADMIN_ORGID_KEY_CERT_LENGTH_NOT_EQUAL_FORMAT, len(adminKeys), len(adminCrts))
	}

	client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath, userSignCrtFilePath, userSignKeyFilePath)
	if err != nil {
		return err
	}
	defer client.Stop()

	var payload *common.Payload
	switch op {
	case addNodeId:
		payload, err = client.CreateChainConfigConsensusNodeIdAddPayload(nodeOrgId, []string{nodeId})
	case removeNodeId:
		payload, err = client.CreateChainConfigConsensusNodeIdDeletePayload(nodeOrgId, nodeId)
	case updateNodeId:
		payload, err = client.CreateChainConfigConsensusNodeIdUpdatePayload(nodeOrgId, nodeIdOld, nodeId)
	default:
		err = errors.New("invalid node addres operation")
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

	resp, err := client.SendChainConfigUpdateRequest(payload, endorsementEntrys, -1, syncResult)
	if err != nil {
		return err
	}
	err = util.CheckProposalRequestResp(resp, true)
	if err != nil {
		return err
	}
	fmt.Printf("consensusnodeid response %+v\n", resp)
	return nil
}
