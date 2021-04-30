/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"fmt"
	"github.com/spf13/cobra"
)

const (
	addNode = iota
	removeNode
	updateNode
)

func configConsensueNodeCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "consensusnode",
		Short: "consensus node management",
		Long:  "consensus node management",
	}
	cmd.AddCommand(addConsensusNodeCMD())
	cmd.AddCommand(removeConsensusNodeCMD())
	cmd.AddCommand(updateConsensusNodeCMD())

	return cmd
}

func addConsensusNodeCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "add consensus node cmd",
		Long:  "add consensus node cmd",
		RunE: func(_ *cobra.Command, _ []string) error {
			return configConsensusNode(addNode)
		},
	}

	attachFlags(cmd, []string{
		flagSdkConfPath, flagOrgId, flagEnableCertHash, flagNodeOrgId, flagNodeId,
		flagAdminCrtFilePaths, flagAdminKeyFilePaths, flagClientCrtFilePaths, flagClientKeyFilePaths,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagAdminCrtFilePaths)
	cmd.MarkFlagRequired(flagAdminKeyFilePaths)
	cmd.MarkFlagRequired(flagNodeOrgId)
	cmd.MarkFlagRequired(flagNodeId)

	return cmd
}

func removeConsensusNodeCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove",
		Short: "remove consensus node cmd",
		Long:  "remove consensus node cmd",
		RunE: func(_ *cobra.Command, _ []string) error {
			return configConsensusNode(removeNode)
		},
	}

	attachFlags(cmd, []string{
		flagSdkConfPath, flagOrgId, flagEnableCertHash, flagNodeOrgId, flagNodeId,
		flagAdminCrtFilePaths, flagAdminKeyFilePaths, flagClientCrtFilePaths, flagClientKeyFilePaths,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagAdminCrtFilePaths)
	cmd.MarkFlagRequired(flagAdminKeyFilePaths)
	cmd.MarkFlagRequired(flagNodeOrgId)
	cmd.MarkFlagRequired(flagNodeId)

	return cmd
}

func updateConsensusNodeCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "update consensus node cmd",
		Long:  "update consensus node cmd",
		RunE: func(_ *cobra.Command, _ []string) error {
			return configConsensusNode(updateTrustRoot)
		},
	}

	attachFlags(cmd, []string{
		flagSdkConfPath, flagOrgId, flagEnableCertHash, flagNodeOrgId, flagNodeIdOld, flagNodeId,
		flagAdminCrtFilePaths, flagAdminKeyFilePaths, flagClientCrtFilePaths, flagClientKeyFilePaths,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagAdminCrtFilePaths)
	cmd.MarkFlagRequired(flagAdminKeyFilePaths)
	cmd.MarkFlagRequired(flagNodeOrgId)
	cmd.MarkFlagRequired(flagNodeIdOld)
	cmd.MarkFlagRequired(flagNodeId)

	return cmd
}

func configConsensusNode(op int) error {
	client, err := createClientWithConfig()
	if err != nil {
		return fmt.Errorf("create user client failed, %s", err.Error())
	}

	adminClient, err := createAdminWithConfig(adminKeyFilePaths, adminCrtFilePaths)
	if err != nil {
		return fmt.Errorf("create admin client failed, %s", err.Error())
	}
	defer adminClient.Stop()

	var payloadBytes []byte
	switch op {
	case addNode:
		payloadBytes, err = client.CreateChainConfigConsensusNodeIdAddPayload(nodeOrgId, []string{nodeId})
	case removeNode:
		payloadBytes, err = client.CreateChainConfigConsensusNodeIdDeletePayload(nodeOrgId, nodeId)
	case updateNode:
		payloadBytes, err = client.CreateChainConfigConsensusNodeIdUpdatePayload(nodeOrgId, nodeIdOld, nodeId)
	default:
		err = fmt.Errorf("invalid node addres operation")
	}
	if err != nil {
		return err
	}
	signedPayload, err := adminClient.SignChainConfigPayload(payloadBytes)
	if err != nil {
		return err
	}
	mergeSignedPayloadBytes, err := client.MergeChainConfigSignedPayload([][]byte{signedPayload})
	if err != nil {
		return err
	}
	resp, err := client.SendChainConfigUpdateRequest(mergeSignedPayloadBytes)
	if err != nil {
		return err
	}
	fmt.Printf("add or remove request response %+v\n", resp)

	return nil
}
