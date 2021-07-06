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
		flagSdkConfPath, flagOrgId, flagEnableCertHash, flagNodeOrgId, flagNodeIdOld, flagNodeId, flagAdminOrgIds,
		flagAdminCrtFilePaths, flagAdminKeyFilePaths, flagClientCrtFilePaths, flagClientKeyFilePaths,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagAdminOrgIds)
	cmd.MarkFlagRequired(flagAdminCrtFilePaths)
	cmd.MarkFlagRequired(flagAdminKeyFilePaths)
	cmd.MarkFlagRequired(flagNodeOrgId)
	cmd.MarkFlagRequired(flagNodeIdOld)
	cmd.MarkFlagRequired(flagNodeId)

	return cmd
}

func configConsensusNodeId(op int) error {
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
	case addNodeId:
		payloadBytes, err = client.CreateChainConfigConsensusNodeIdAddPayload(nodeOrgId, []string{nodeId})
	case removeNodeId:
		payloadBytes, err = client.CreateChainConfigConsensusNodeIdDeletePayload(nodeOrgId, nodeId)
	case updateNodeId:
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
	fmt.Printf("consensusnodeid response %+v\n", resp)
	return nil
}
