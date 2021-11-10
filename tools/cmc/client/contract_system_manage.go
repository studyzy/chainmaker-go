package client

import (
	"fmt"
	"strings"

	"chainmaker.org/chainmaker-go/tools/cmc/util"
	"chainmaker.org/chainmaker/common/v2/crypto"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/protocol/v2"
	sdk "chainmaker.org/chainmaker/sdk-go/v2"
	sdkutils "chainmaker.org/chainmaker/sdk-go/v2/utils"
	"github.com/hokaccha/go-prettyjson"
	"github.com/spf13/cobra"
)

func systemContractManageCMD() *cobra.Command {
	systemContractMultiSignCmd := &cobra.Command{
		Use:   "manage",
		Short: "system contract manage command",
		Long:  "system contract manage command",
	}

	systemContractMultiSignCmd.AddCommand(contractAccessGrantCMD())
	systemContractMultiSignCmd.AddCommand(contractAccessRevokeCMD())
	systemContractMultiSignCmd.AddCommand(contractAccessQueryCMD())

	return systemContractMultiSignCmd
}

func contractAccessGrantCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "access-grant",
		Short: "contract access grant",
		Long:  "contract access grant",
		RunE: func(_ *cobra.Command, _ []string) error {
			return grantOrRevokeContractAccess(1)
		},
	}

	attachFlags(cmd, []string{
		flagUserSignKeyFilePath, flagUserSignCrtFilePath,
		flagConcurrency, flagTotalCountPerGoroutine, flagSdkConfPath, flagOrgId, flagChainId,
		flagTimeout, flagUserTlsCrtFilePath, flagUserTlsKeyFilePath, flagEnableCertHash,
		flagAdminKeyFilePaths, flagAdminCrtFilePaths, flagGrantContractList, flagAdminOrgIds,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagGrantContractList)

	return cmd
}

func contractAccessRevokeCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "access-revoke",
		Short: "contract access revoke",
		Long:  "contract access revoke",
		RunE: func(_ *cobra.Command, _ []string) error {
			return grantOrRevokeContractAccess(2)
		},
	}

	attachFlags(cmd, []string{
		flagUserSignKeyFilePath, flagUserSignCrtFilePath,
		flagConcurrency, flagTotalCountPerGoroutine, flagSdkConfPath, flagOrgId, flagChainId,
		flagTimeout, flagUserTlsCrtFilePath, flagUserTlsKeyFilePath, flagEnableCertHash,
		flagAdminKeyFilePaths, flagAdminCrtFilePaths, flagRevokeContractList, flagAdminOrgIds,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagRevokeContractList)

	return cmd
}

func contractAccessQueryCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "access-query",
		Short: "contract access query",
		Long:  "contract access query",
		RunE: func(_ *cobra.Command, _ []string) error {
			return queryContractAccess()
		},
	}

	attachFlags(cmd, []string{
		flagUserSignKeyFilePath, flagUserSignCrtFilePath,
		flagConcurrency, flagTotalCountPerGoroutine, flagSdkConfPath, flagOrgId, flagChainId,
		flagTimeout, flagUserTlsCrtFilePath, flagUserTlsKeyFilePath,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)

	return cmd
}

func grantOrRevokeContractAccess(which int) error {
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

	var (
		payload        *common.Payload
		whichOperation string
	)

	switch which {
	case 1:
		payload, err = client.CreateNativeContractAccessGrantPayload(grantContractList)
		whichOperation = "access grant"
	case 2:
		payload, err = client.CreateNativeContractAccessRevokePayload(revokeContractList)
		whichOperation = "access revoke"
	default:
		err = fmt.Errorf("wrong which param")
	}
	if err != nil {
		return fmt.Errorf("create contract manage %s payload failed, %s", whichOperation, err.Error())
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

	// 发送创建合约请求
	resp, err := client.SendContractManageRequest(payload, endorsementEntrys, timeout, syncResult)
	if err != nil {
		return fmt.Errorf(SEND_CONTRACT_MANAGE_REQUEST_FAILED_FORMAT, err.Error())
	}

	err = util.CheckProposalRequestResp(resp, false)
	if err != nil {
		return fmt.Errorf(CHECK_PROPOSAL_RESPONSE_FAILED_FORMAT, err.Error())
	}

	fmt.Printf("%s contract resp: %+v\n", whichOperation, resp)

	return nil
}

func queryContractAccess() error {
	client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
		userSignCrtFilePath, userSignKeyFilePath)
	if err != nil {
		return err
	}
	defer client.Stop()

	disabledNativeContractList, err := client.GetDisabledNativeContractList()
	if err != nil {
		return fmt.Errorf("get disabled native contract list failed, %s", err)
	}

	output, err := prettyjson.Marshal(disabledNativeContractList)
	if err != nil {
		return err
	}

	fmt.Println(string(output))
	return nil
}
