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
		flagSdkConfPath, flagOrgId, flagEnableCertHash, flagNodeOrgId, flagNodeIds, flagAdminOrgIds,
		flagAdminCrtFilePaths, flagAdminKeyFilePaths, flagUserTlsCrtFilePath, flagUserTlsKeyFilePath,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagAdminOrgIds)
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
		flagSdkConfPath, flagOrgId, flagEnableCertHash, flagNodeOrgId, flagNodeId, flagAdminOrgIds,
		flagAdminCrtFilePaths, flagAdminKeyFilePaths, flagUserTlsCrtFilePath, flagUserTlsKeyFilePath,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagAdminOrgIds)
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
		flagSdkConfPath, flagOrgId, flagEnableCertHash, flagNodeOrgId, flagNodeIdOld, flagNodeIds, flagAdminOrgIds,
		flagAdminCrtFilePaths, flagAdminKeyFilePaths, flagUserTlsCrtFilePath, flagUserTlsKeyFilePath,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagAdminOrgIds)
	cmd.MarkFlagRequired(flagAdminCrtFilePaths)
	cmd.MarkFlagRequired(flagAdminKeyFilePaths)
	cmd.MarkFlagRequired(flagNodeOrgId)
	cmd.MarkFlagRequired(flagNodeIds)

	return cmd
}

func configConsensusNodeOrg(op int) error {
	nodeIdSlice := strings.Split(nodeIds, ",")
	adminOrgIdSlice := strings.Split(adminOrgIds, ",")
	adminKeys := strings.Split(adminKeyFilePaths, ",")
	adminCrts := strings.Split(adminCrtFilePaths, ",")
	if len(adminKeys) == 0 || len(adminCrts) == 0 || len(adminOrgIdSlice) == 0 {
		return ErrAdminOrgIdKeyCertIsEmpty
	}
	if len(adminKeys) != len(adminCrts) || len(adminOrgIdSlice) != len(adminCrts) {
		return fmt.Errorf(ADMIN_ORGID_KEY_CERT_LENGTH_NOT_EQUAL_FORMAT, len(adminOrgIdSlice), len(adminKeys), len(adminCrts))
	}

	client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath, userSignCrtFilePath, userSignKeyFilePath)
	if err != nil && !strings.Contains(err.Error(), "user cert havenot on chain yet, and try again") {
		return fmt.Errorf(CREATE_USER_CLIENT_FAILED_FORMAT, err)
	}
	defer client.Stop()

	var payloadBytes []byte
	switch op {
	case addNodeOrg:
		payloadBytes, err = client.CreateChainConfigConsensusNodeOrgAddPayload(nodeOrgId, nodeIdSlice)
	case removeNodeOrg:
		payloadBytes, err = client.CreateChainConfigConsensusNodeOrgDeletePayload(nodeOrgId)
	case updateNodeOrg:
		payloadBytes, err = client.CreateChainConfigConsensusNodeOrgUpdatePayload(nodeOrgId, nodeIdSlice)
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

	resp, err := client.SendChainConfigUpdateRequest(mergedSignedPayloadBytes, -1, true)
	if err != nil {
		return err
	}
	err = util.CheckProposalRequestResp(resp, true)
	if err != nil {
		return err
	}
	fmt.Printf("consensusnodeorg response %+v\n", resp)
	return nil
}
