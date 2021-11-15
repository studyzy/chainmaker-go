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

	"chainmaker.org/chainmaker-go/tools/cmc/util"
	"chainmaker.org/chainmaker/common/v2/crypto"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/protocol/v2"
	sdk "chainmaker.org/chainmaker/sdk-go/v2"
	sdkutils "chainmaker.org/chainmaker/sdk-go/v2/utils"
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
		flagUserSignKeyFilePath, flagUserSignCrtFilePath,
		flagSdkConfPath, flagOrgId, flagEnableCertHash, flagNodeOrgId, flagNodeId,
		flagAdminCrtFilePaths, flagAdminKeyFilePaths, flagAdminOrgIds, flagUserTlsCrtFilePath, flagUserTlsKeyFilePath,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
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
		flagAdminCrtFilePaths, flagAdminKeyFilePaths, flagAdminOrgIds, flagUserTlsCrtFilePath, flagUserTlsKeyFilePath,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
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
		flagAdminCrtFilePaths, flagAdminKeyFilePaths, flagAdminOrgIds, flagUserTlsCrtFilePath, flagUserTlsKeyFilePath,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagNodeOrgId)
	cmd.MarkFlagRequired(flagNodeIdOld)
	cmd.MarkFlagRequired(flagNodeId)

	return cmd
}

func configConsensusNodeId(op int) error {
	var adminKeys []string
	var adminCrts []string
	var adminOrgs []string

	client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
		userSignCrtFilePath, userSignKeyFilePath)
	if err != nil {
		return err
	}
	defer client.Stop()

	if sdk.AuthTypeToStringMap[client.GetAuthType()] == protocol.PermissionedWithCert {
		if adminKeyFilePaths != "" {
			adminKeys = strings.Split(adminKeyFilePaths, ",")
		}
		if adminCrtFilePaths != "" {
			adminCrts = strings.Split(adminCrtFilePaths, ",")
		}
		if len(adminKeys) != len(adminCrts) {
			return fmt.Errorf(ADMIN_ORGID_KEY_CERT_LENGTH_NOT_EQUAL_FORMAT, len(adminKeys), len(adminCrts))
		}
	} else if sdk.AuthTypeToStringMap[client.GetAuthType()] == protocol.PermissionedWithKey {
		if adminKeyFilePaths != "" {
			adminKeys = strings.Split(adminKeyFilePaths, ",")
		}
		if adminOrgIds != "" {
			adminOrgs = strings.Split(adminOrgIds, ",")
		}
		if len(adminKeys) != len(adminOrgs) {
			return fmt.Errorf(ADMIN_ORGID_KEY_LENGTH_NOT_EQUAL_FORMAT, len(adminKeys), len(adminOrgs))
		}
	}

	var payload *common.Payload
	switch op {
	case addNodeId:
		payload, err = client.CreateChainConfigConsensusNodeIdAddPayload(nodeOrgId, []string{nodeId})
	case removeNodeId:
		payload, err = client.CreateChainConfigConsensusNodeIdDeletePayload(nodeOrgId, nodeId)
	case updateNodeId:
		payload, err = client.CreateChainConfigConsensusNodeIdUpdatePayload(nodeOrgId, nodeIdOld, nodeId)
	default:
		err = errors.New("invalid node address operation")
	}
	if err != nil {
		return err
	}

	endorsementEntrys := make([]*common.EndorsementEntry, len(adminKeys))
	for i := range adminKeys {
		if sdk.AuthTypeToStringMap[client.GetAuthType()] == protocol.PermissionedWithCert {
			e, err := sdkutils.MakeEndorserWithPath(adminKeys[i], adminCrts[i], payload)
			if err != nil {
				return err
			}

			endorsementEntrys[i] = e
		} else if sdk.AuthTypeToStringMap[client.GetAuthType()] == protocol.PermissionedWithKey {
			e, err := sdkutils.MakePkEndorserWithPath(
				adminKeys[i],
				crypto.HashAlgoMap[client.GetHashType()],
				adminOrgs[i],
				payload,
			)
			if err != nil {
				return err
			}

			endorsementEntrys[i] = e
		}
	}

	resp, err := client.SendChainConfigUpdateRequest(payload, endorsementEntrys, -1, syncResult)
	if err != nil {
		return err
	}
	err = util.CheckProposalRequestResp(resp, false)
	if err != nil {
		return err
	}
	fmt.Printf("consensusnodeid response %+v\n", resp)
	return nil
}
