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
		flagSdkConfPath, flagOrgId, flagEnableCertHash, flagNodeOrgId, flagNodeId, flagAdminOrgIds,
		flagAdminCrtFilePaths, flagAdminKeyFilePaths, flagClientCrtFilePaths, flagClientKeyFilePaths,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagAdminOrgIds)
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
	adminOrgIdSlice := strings.Split(adminOrgIds, ",")
	adminKeys := strings.Split(adminKeyFilePaths, ",")
	adminCrts := strings.Split(adminCrtFilePaths, ",")
	if len(adminKeys) == 0 || len(adminCrts) == 0 || len(adminOrgIdSlice) == 0 || len(adminKeys) != len(adminCrts) || len(adminOrgIdSlice) != len(adminCrts) {
		return fmt.Errorf(ADMIN_KEY_AND_CERT_NOT_ENOUGH_FORMAT, len(adminKeys), len(adminCrts))
	}

	client, err := createClientWithConfig()
	if err != nil && !strings.Contains(err.Error(), "user cert havenot on chain yet, and try again") {
		return fmt.Errorf(CREATE_USER_CLIENT_FAILED_FORMAT, err)
	}
	defer client.Stop()

	var payloadBytes []byte
	switch op {
	case addNode:
		payloadBytes, err = client.CreateChainConfigConsensusNodeIdAddPayload(nodeOrgId, []string{nodeId})
	case removeNode:
		payloadBytes, err = client.CreateChainConfigConsensusNodeIdDeletePayload(nodeOrgId, nodeId)
	case updateNode:
		payloadBytes, err = client.CreateChainConfigConsensusNodeIdUpdatePayload(nodeOrgId, nodeIdOld, nodeId)
	default:
		err = errors.New("invalid node addres operation")
	}
	if err != nil {
		return err
	}

	signedPayloads := make([][]byte, len(adminKeys))
	for i := range adminKeys {
		_, privKey, err := dealUserKey(adminKeys[i])
		if err != nil {
			return err
		}
		crtBytes, crt, err := dealUserCrt(adminCrts[i])
		if err != nil {
			return err
		}

		signedPayload, err := signChainConfigPayload(payloadBytes, crtBytes, privKey, crt, adminOrgIdSlice[i])
		if err != nil {
			return err
		}
		signedPayloads[i] = signedPayload
	}

	mergedSignedPayloadBytes, err := client.MergeChainConfigSignedPayload(signedPayloads)
	if err != nil {
		return err
	}

	resp, err := client.SendChainConfigUpdateRequest(mergedSignedPayloadBytes)
	if err != nil {
		return err
	}
	err = checkProposalRequestResp(resp, true)
	if err != nil {
		return err
	}
	fmt.Printf("consensusnode response %+v\n", resp)
	return nil
}
