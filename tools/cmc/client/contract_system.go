/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"encoding/json"
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/spf13/cobra"

	"chainmaker.org/chainmaker-go/tools/cmc/util"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"
	sdk "chainmaker.org/chainmaker/sdk-go/v2"
	sdkutils "chainmaker.org/chainmaker/sdk-go/v2/utils"
)

const DEFAULT_TIMEOUT = 5000 // ms

func systemContractCMD() *cobra.Command {
	systemContractCmd := &cobra.Command{
		Use:   "system",
		Short: "system contract command",
		Long:  "system contract command",
	}

	systemContractCmd.AddCommand(getChainInfoCMD())
	systemContractCmd.AddCommand(getBlockByHeightCMD())
	systemContractCmd.AddCommand(getTxByTxIdCMD())

	// DPoS-erc20 contract
	systemContractCmd.AddCommand(erc20Mint())
	systemContractCmd.AddCommand(erc20Transfer())
	systemContractCmd.AddCommand(erc20BalanceOf())
	systemContractCmd.AddCommand(erc20Owner())
	systemContractCmd.AddCommand(erc20Decimals())
	systemContractCmd.AddCommand(erc20Total())
	//
	////DPoS.Stake
	systemContractCmd.AddCommand(stakeGetAllCandidates())
	systemContractCmd.AddCommand(stakeGetValidatorByAddress())
	systemContractCmd.AddCommand(stakeDelegate())
	systemContractCmd.AddCommand(stakeGetDelegationsByAddress())
	systemContractCmd.AddCommand(stakeGetUserDelegationByValidator())
	systemContractCmd.AddCommand(stakeUnDelegate())
	systemContractCmd.AddCommand(stakeReadEpochByID())
	systemContractCmd.AddCommand(stakeReadLatestEpoch())
	systemContractCmd.AddCommand(stakeSetNodeID())
	systemContractCmd.AddCommand(stakeGetNodeID())
	systemContractCmd.AddCommand(stakeReadMinSelfDelegation())
	systemContractCmd.AddCommand(stakeReadEpochValidatorNumber())
	systemContractCmd.AddCommand(stakeReadEpochBlockNumber())
	systemContractCmd.AddCommand(stakeReadSystemContractAddr())
	systemContractCmd.AddCommand(stakeReadCompleteUnBoundingEpochNumber())

	// system contract manage
	systemContractCmd.AddCommand(systemContractManageCMD())

	// system contract multi sign
	systemContractCmd.AddCommand(systemContractMultiSignCMD())

	// DPoS-stake contract
	return systemContractCmd
}

func getChainInfoCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "getchaininfo",
		Short: "get chain info",
		Long:  "get chain info",
		RunE: func(_ *cobra.Command, _ []string) error {
			return getChainInfo()
		},
	}

	attachFlags(cmd, []string{
		flagUserSignKeyFilePath, flagUserSignCrtFilePath,
		flagConcurrency, flagTotalCountPerGoroutine, flagSdkConfPath, flagOrgId, flagChainId,
		flagParams, flagTimeout, flagUserTlsCrtFilePath, flagUserTlsKeyFilePath, flagEnableCertHash,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)

	return cmd
}

func getBlockByHeightCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "block",
		Short: "get block by height",
		Long:  "get block by height",
		RunE: func(_ *cobra.Command, _ []string) error {
			return getBlockByHeight()
		},
	}

	attachFlags(cmd, []string{
		flagUserSignKeyFilePath, flagUserSignCrtFilePath,
		flagSdkConfPath, flagOrgId, flagChainId, flagBlockHeight, flagWithRWSet,
		flagUserTlsCrtFilePath, flagUserTlsKeyFilePath,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagBlockHeight)
	cmd.MarkFlagRequired(flagWithRWSet)

	return cmd
}

func getTxByTxIdCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tx",
		Short: "get tx by tx id",
		Long:  "get tx by tx id",
		RunE: func(_ *cobra.Command, _ []string) error {
			return getTxByTxId()
		},
	}

	attachFlags(cmd, []string{
		flagUserSignKeyFilePath, flagUserSignCrtFilePath,
		flagSdkConfPath, flagOrgId, flagChainId, flagTxId,
		flagUserTlsCrtFilePath, flagUserTlsKeyFilePath,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagTxId)

	return cmd
}

func getChainInfo() error {
	var (
		err error
	)

	client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
		userSignCrtFilePath, userSignKeyFilePath)
	if err != nil {
		return err
	}
	defer client.Stop()
	pairs := make(map[string]string)
	if params != "" {
		err := json.Unmarshal([]byte(params), &pairs)
		if err != nil {
			return err
		}
	}

	resp, err := client.GetChainInfo()
	if err != nil {
		return fmt.Errorf("get chain info failed, %s", err.Error())
	}

	fmt.Printf("get chain info resp: %+v\n", resp)

	return nil
}

func getBlockByHeight() error {
	var (
		err error
	)

	client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
		userSignCrtFilePath, userSignKeyFilePath)
	if err != nil {
		return err
	}
	defer client.Stop()
	pairs := make(map[string]string)
	if params != "" {
		err := json.Unmarshal([]byte(params), &pairs)
		if err != nil {
			return err
		}
	}

	resp, err := client.GetBlockByHeight(blockHeight, withRWSet)
	if err != nil {
		return fmt.Errorf("get block by height failed, %s", err.Error())
	}

	fmt.Printf("get block by height resp: %+v\n", resp)

	return nil
}

func getTxByTxId() error {
	var (
		err error
	)

	client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
		userSignCrtFilePath, userSignKeyFilePath)
	if err != nil {
		return err
	}
	defer client.Stop()
	pairs := make(map[string]string)
	if params != "" {
		err := json.Unmarshal([]byte(params), &pairs)
		if err != nil {
			return err
		}
	}

	resp, err := client.GetTxByTxId(txId)
	if err != nil {
		return fmt.Errorf("get block by height failed, %s", err.Error())
	}

	fmt.Printf("get block by height resp: %+v\n", resp)

	return nil
}

func erc20Mint() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mint",
		Short: "mint feature of the erc20",
		RunE: func(_ *cobra.Command, _ []string) error {
			var (
				err error
			)
			client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
				userSignCrtFilePath, userSignKeyFilePath)
			if err != nil {
				return err
			}
			defer client.Stop()
			pairs := make(map[string]string)
			if params != "" {
				err := json.Unmarshal([]byte(params), &pairs)
				if err != nil {
					return err
				}
			}
			txId = sdkutils.GetRandTxId()
			resp, err := mint(client, address, amount, txId, DEFAULT_TIMEOUT, syncResult)
			if err != nil {
				return fmt.Errorf("mint failed, %s", err.Error())
			}

			fmt.Printf("resp: %+v\n", resp)

			return nil
		},
	}

	attachFlags(cmd, []string{
		flagAddress, flagAmount,
		flagSdkConfPath,
		flagOrgId, flagChainId,
		flagUserTlsCrtFilePath, flagUserTlsKeyFilePath,
		flagUserSignCrtFilePath, flagUserSignKeyFilePath,
		flagSyncResult,
	})

	cmd.MarkFlagRequired(flagAddress)
	cmd.MarkFlagRequired(flagAmount)

	return cmd
}

func erc20Transfer() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transfer",
		Short: "transfer feature of the erc20",
		RunE: func(_ *cobra.Command, _ []string) error {
			var (
				err error
			)

			client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
				userSignCrtFilePath, userSignKeyFilePath)
			if err != nil {
				return err
			}
			defer client.Stop()
			pairs := make(map[string]string)
			if params != "" {
				err := json.Unmarshal([]byte(params), &pairs)
				if err != nil {
					return err
				}
			}
			txId = sdkutils.GetRandTxId()
			resp, err := transfer(client, address, amount, txId, DEFAULT_TIMEOUT, false)
			if err != nil {
				return fmt.Errorf("transfer failed, %s", err.Error())
			}

			fmt.Printf("resp: %+v\n", resp)

			return nil
		},
	}

	attachFlags(cmd, []string{
		flagAddress, flagAmount,
		flagSdkConfPath,
		flagOrgId, flagChainId,
		flagUserTlsCrtFilePath, flagUserTlsKeyFilePath, flagUserSignCrtFilePath, flagUserSignKeyFilePath,
	})

	cmd.MarkFlagRequired(flagAddress)
	cmd.MarkFlagRequired(flagAmount)

	return cmd
}

func erc20BalanceOf() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "balance-of",
		Short: "balance-of feature of the erc20",
		RunE: func(_ *cobra.Command, _ []string) error {
			var (
				err error
			)

			client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
				userSignCrtFilePath, userSignKeyFilePath)
			if err != nil {
				return err
			}
			defer client.Stop()
			pairs := make(map[string]string)
			if params != "" {
				err := json.Unmarshal([]byte(params), &pairs)
				if err != nil {
					return err
				}
			}

			resp, err := balanceOf(client, address, DEFAULT_TIMEOUT)
			if err != nil {
				return fmt.Errorf("balance of failed, %s", err.Error())
			}

			fmt.Printf("resp: %+v\n", resp)

			return nil
		},
	}

	attachFlags(cmd, []string{
		flagAddress,
		flagSdkConfPath,
		flagOrgId, flagChainId,
		flagUserTlsCrtFilePath, flagUserTlsKeyFilePath, flagUserSignCrtFilePath, flagUserSignKeyFilePath,
	})

	cmd.MarkFlagRequired(flagAddress)

	return cmd
}

func erc20Owner() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "owner",
		Short: "owner feature of the erc20",
		RunE: func(_ *cobra.Command, _ []string) error {
			var (
				err error
			)

			client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
				userSignCrtFilePath, userSignKeyFilePath)
			if err != nil {
				return err
			}
			defer client.Stop()
			pairs := make(map[string]string)
			if params != "" {
				err := json.Unmarshal([]byte(params), &pairs)
				if err != nil {
					return err
				}
			}

			resp, err := owner(client, DEFAULT_TIMEOUT)
			if err != nil {
				return fmt.Errorf("owner failed, %s", err.Error())
			}

			fmt.Printf("resp: %+v\n", resp)

			return nil
		},
	}

	attachFlags(cmd, []string{
		flagSdkConfPath,
		flagOrgId, flagChainId,
		flagUserTlsCrtFilePath, flagUserTlsKeyFilePath, flagUserSignCrtFilePath, flagUserSignKeyFilePath,
	})

	return cmd
}

func erc20Decimals() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "decimals",
		Short: "decimals feature of the erc20",
		RunE: func(_ *cobra.Command, _ []string) error {
			var (
				err error
			)

			client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
				userSignCrtFilePath, userSignKeyFilePath)
			if err != nil {
				return err
			}
			defer client.Stop()
			pairs := make(map[string]string)
			if params != "" {
				err := json.Unmarshal([]byte(params), &pairs)
				if err != nil {
					return err
				}
			}

			resp, err := decimals(client, DEFAULT_TIMEOUT)
			if err != nil {
				return fmt.Errorf("decimals failed, %s", err.Error())
			}

			fmt.Printf("resp: %+v\n", resp)

			return nil
		},
	}

	attachFlags(cmd, []string{
		flagSdkConfPath,
		flagOrgId, flagChainId,
		flagUserTlsCrtFilePath, flagUserTlsKeyFilePath, flagUserSignCrtFilePath, flagUserSignKeyFilePath,
	})

	return cmd
}

func erc20Total() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "total",
		Short: "total feature of the erc20",
		RunE: func(_ *cobra.Command, _ []string) error {
			var (
				err error
			)

			client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
				userSignCrtFilePath, userSignKeyFilePath)
			if err != nil {
				return err
			}
			defer client.Stop()
			pairs := make(map[string]string)
			if params != "" {
				err := json.Unmarshal([]byte(params), &pairs)
				if err != nil {
					return err
				}
			}

			resp, err := total(client, DEFAULT_TIMEOUT)
			if err != nil {
				return fmt.Errorf("total failed, %s", err.Error())
			}

			fmt.Printf("resp: %+v\n", resp)

			return nil
		},
	}

	attachFlags(cmd, []string{
		flagSdkConfPath,
		flagOrgId, flagChainId,
		flagUserTlsCrtFilePath, flagUserTlsKeyFilePath, flagUserSignCrtFilePath, flagUserSignKeyFilePath,
	})

	return cmd
}

func stakeGetAllCandidates() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "all-candidates",
		Short: "all-candidates feature of the stake",
		RunE: func(_ *cobra.Command, _ []string) error {
			var (
				err error
			)

			client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
				userSignCrtFilePath, userSignKeyFilePath)
			if err != nil {
				return err
			}
			defer client.Stop()
			pairs := make(map[string]string)
			if params != "" {
				err := json.Unmarshal([]byte(params), &pairs)
				if err != nil {
					return err
				}
			}

			resp, err := getAllCandidates(client, DEFAULT_TIMEOUT)
			if err != nil {
				return fmt.Errorf("all-candidates failed, %s", err.Error())
			}

			fmt.Printf("resp: %+v\n", resp)

			return nil
		},
	}

	attachFlags(cmd, []string{
		flagSdkConfPath,
		flagOrgId, flagChainId,
		flagUserTlsCrtFilePath, flagUserTlsKeyFilePath, flagUserSignCrtFilePath, flagUserSignKeyFilePath,
	})

	return cmd
}

func stakeGetValidatorByAddress() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-validator",
		Short: "get-validator feature of the stake",
		RunE: func(_ *cobra.Command, _ []string) error {
			var (
				err error
			)

			client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
				userSignCrtFilePath, userSignKeyFilePath)
			if err != nil {
				return err
			}
			defer client.Stop()
			pairs := make(map[string]string)
			if params != "" {
				err := json.Unmarshal([]byte(params), &pairs)
				if err != nil {
					return err
				}
			}

			resp, err := getValidatorByAddress(client, address, DEFAULT_TIMEOUT)
			if err != nil {
				return fmt.Errorf("get-validator failed, %s", err.Error())
			}
			val := &syscontract.Validator{}
			if err := proto.Unmarshal(resp.ContractResult.Result, val); err != nil {
				fmt.Println("unmarshal validatorInfo failed")
				return nil
			}
			resp.ContractResult.Result = []byte(val.String())
			fmt.Printf("resp: %+v\n", resp)
			return nil
		},
	}

	attachFlags(cmd, []string{
		flagAddress,
		flagSdkConfPath,
		flagOrgId, flagChainId,
		flagUserTlsCrtFilePath, flagUserTlsKeyFilePath, flagUserSignCrtFilePath, flagUserSignKeyFilePath,
	})

	cmd.MarkFlagRequired(flagAddress)

	return cmd
}

func stakeDelegate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delegate",
		Short: "delegate feature of the stake",
		RunE: func(_ *cobra.Command, _ []string) error {
			client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
				userSignCrtFilePath, userSignKeyFilePath)
			if err != nil {
				return err
			}
			defer client.Stop()
			pairs := make(map[string]string)
			if params != "" {
				if err := json.Unmarshal([]byte(params), &pairs); err != nil {
					return err
				}
			}
			txId = sdkutils.GetRandTxId()
			resp, err := delegate(client, address, amount, txId, DEFAULT_TIMEOUT, syncResult)
			if err != nil {
				return fmt.Errorf("delegate failed, %s", err.Error())
			}
			if resp.ContractResult == nil {
				fmt.Printf("resp: %+v\n", resp)
				return nil
			}
			info := &syscontract.Delegation{}
			if err := proto.Unmarshal(resp.ContractResult.Result, info); err != nil {
				return fmt.Errorf("unmarshal delegate info failed, %v", err)
			}
			resp.ContractResult.Result = []byte(info.String())
			fmt.Printf("resp: %+v\n", resp)
			return nil
		},
	}

	attachFlags(cmd, []string{
		flagAddress, flagAmount,
		flagSdkConfPath,
		flagOrgId, flagChainId,
		flagUserTlsCrtFilePath, flagUserTlsKeyFilePath,
		flagUserSignCrtFilePath, flagUserSignKeyFilePath,
		flagSyncResult,
	})

	cmd.MarkFlagRequired(flagAddress)
	cmd.MarkFlagRequired(flagAmount)

	return cmd
}

func stakeGetDelegationsByAddress() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-delegations-by-address",
		Short: "get-delegations-by-address feature of the stake",
		RunE: func(_ *cobra.Command, _ []string) error {
			var (
				err error
			)

			client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
				userSignCrtFilePath, userSignKeyFilePath)
			if err != nil {
				return err
			}
			defer client.Stop()
			pairs := make(map[string]string)
			if params != "" {
				err := json.Unmarshal([]byte(params), &pairs)
				if err != nil {
					return err
				}
			}

			resp, err := getDelegationsByAddress(client, address, DEFAULT_TIMEOUT)
			if err != nil {
				return fmt.Errorf("get-delegations-by-address failed, %s", err.Error())
			}
			delegateInfo := &syscontract.DelegationInfo{}
			if err := proto.Unmarshal(resp.ContractResult.Result, delegateInfo); err != nil {
				fmt.Println("unmarshal delegateInfo failed: ", err)
				return nil
			}
			resp.ContractResult.Result = []byte(delegateInfo.String())
			fmt.Printf("resp: %+v\n", resp)
			return nil
		},
	}

	attachFlags(cmd, []string{
		flagAddress,
		flagSdkConfPath,
		flagOrgId, flagChainId,
		flagUserTlsCrtFilePath, flagUserTlsKeyFilePath, flagUserSignCrtFilePath, flagUserSignKeyFilePath,
	})

	cmd.MarkFlagRequired(flagAddress)

	return cmd
}

func stakeGetUserDelegationByValidator() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-user-delegation-by-validator",
		Short: "get-delegations-by-address feature of the stake",
		RunE: func(_ *cobra.Command, _ []string) error {
			var (
				err error
			)

			client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
				userSignCrtFilePath, userSignKeyFilePath)
			if err != nil {
				return err
			}
			defer client.Stop()
			pairs := make(map[string]string)
			if params != "" {
				err := json.Unmarshal([]byte(params), &pairs)
				if err != nil {
					return err
				}
			}

			resp, err := getUserDelegationByValidator(client, delegator, validator, DEFAULT_TIMEOUT)
			if err != nil {
				return fmt.Errorf("get-user-delegation-by-validator failed, %s", err.Error())
			}
			info := &syscontract.Delegation{}
			if err := proto.Unmarshal(resp.ContractResult.Result, info); err != nil {
				return fmt.Errorf("unmarshal delegate info failed, %v", err)
			}
			resp.ContractResult.Result = []byte(info.String())
			fmt.Printf("resp: %+v\n", resp)
			return nil
		},
	}

	attachFlags(cmd, []string{
		flagDelegator, flagValidator,
		flagSdkConfPath,
		flagOrgId, flagChainId,
		flagUserTlsCrtFilePath, flagUserTlsKeyFilePath, flagUserSignCrtFilePath, flagUserSignKeyFilePath,
	})

	cmd.MarkFlagRequired(flagDelegator)
	cmd.MarkFlagRequired(flagValidator)

	return cmd
}

func stakeUnDelegate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "undelegate",
		Short: "undelegate feature of the stake",
		RunE: func(_ *cobra.Command, _ []string) error {
			var (
				err error
			)

			client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
				userSignCrtFilePath, userSignKeyFilePath)
			if err != nil {
				return err
			}
			defer client.Stop()
			pairs := make(map[string]string)
			if params != "" {
				err := json.Unmarshal([]byte(params), &pairs)
				if err != nil {
					return err
				}
			}
			txId = sdkutils.GetRandTxId()
			resp, err := unDelegate(client, address, amount, txId, DEFAULT_TIMEOUT, syncResult)
			if err != nil {
				return fmt.Errorf("undelegate failed, %s", err.Error())
			}
			if resp.ContractResult == nil {
				fmt.Printf("resp: %+v\n", resp)
				return nil
			}
			info := &syscontract.UnbondingDelegation{}
			if err := proto.Unmarshal(resp.ContractResult.Result, info); err != nil {
				return fmt.Errorf("unmarshal UnbondingDelegation info failed, %v", err)
			}
			resp.ContractResult.Result = []byte(info.String())
			fmt.Printf("resp: %+v\n", resp)

			return nil
		},
	}

	attachFlags(cmd, []string{
		flagAddress, flagAmount,
		flagSdkConfPath,
		flagOrgId, flagChainId,
		flagUserTlsCrtFilePath, flagUserTlsKeyFilePath,
		flagUserSignCrtFilePath, flagUserSignKeyFilePath,
		flagSyncResult,
	})

	cmd.MarkFlagRequired(flagAddress)
	cmd.MarkFlagRequired(flagAmount)

	return cmd
}

func stakeReadEpochByID() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "read-epoch-by-id",
		Short: "read-epoch-by-id feature of the stake",
		RunE: func(_ *cobra.Command, _ []string) error {
			var (
				err error
			)

			client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
				userSignCrtFilePath, userSignKeyFilePath)
			if err != nil {
				return err
			}
			defer client.Stop()
			pairs := make(map[string]string)
			if params != "" {
				err := json.Unmarshal([]byte(params), &pairs)
				if err != nil {
					return err
				}
			}

			resp, err := readEpochByID(client, epochID, DEFAULT_TIMEOUT)
			if err != nil {
				return fmt.Errorf("read-epoch-by-id failed, %s", err.Error())
			}
			info := &syscontract.Epoch{}
			if err := proto.Unmarshal(resp.ContractResult.Result, info); err != nil {
				return fmt.Errorf("unmarshal epoch err: %v", err)
			}
			resp.ContractResult.Result = []byte(info.String())
			fmt.Printf("resp: %+v\n", resp)

			return nil
		},
	}

	attachFlags(cmd, []string{
		flagEpochID,
		flagSdkConfPath,
		flagOrgId, flagChainId,
		flagUserTlsCrtFilePath, flagUserTlsKeyFilePath, flagUserSignCrtFilePath, flagUserSignKeyFilePath,
	})

	cmd.MarkFlagRequired(flagEpochID)

	return cmd
}

func stakeReadLatestEpoch() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "read-latest-epoch",
		Short: "read-latest-epoch feature of the stake",
		RunE: func(_ *cobra.Command, _ []string) error {
			var (
				err error
			)

			client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
				userSignCrtFilePath, userSignKeyFilePath)
			if err != nil {
				return err
			}
			defer client.Stop()
			pairs := make(map[string]string)
			if params != "" {
				err := json.Unmarshal([]byte(params), &pairs)
				if err != nil {
					return err
				}
			}

			resp, err := readLatestEpoch(client, DEFAULT_TIMEOUT)
			if err != nil {
				return fmt.Errorf("read-latest-epoch failed, %s", err.Error())
			}
			info := &syscontract.Epoch{}
			if err := proto.Unmarshal(resp.ContractResult.Result, info); err != nil {
				return fmt.Errorf("unmarshal epoch err: %v", err)
			}
			resp.ContractResult.Result = []byte(info.String())
			fmt.Printf("resp: %+v\n", resp)

			return nil
		},
	}

	attachFlags(cmd, []string{
		flagSdkConfPath,
		flagOrgId, flagChainId,
		flagUserTlsCrtFilePath, flagUserTlsKeyFilePath, flagUserSignCrtFilePath, flagUserSignKeyFilePath,
	})

	return cmd
}

func stakeSetNodeID() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-node-id",
		Short: "set-node-id feature of the stake",
		RunE: func(_ *cobra.Command, _ []string) error {
			var (
				err error
			)

			client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
				userSignCrtFilePath, userSignKeyFilePath)
			if err != nil {
				return err
			}
			defer client.Stop()
			pairs := make(map[string]string)
			if params != "" {
				err := json.Unmarshal([]byte(params), &pairs)
				if err != nil {
					return err
				}
			}

			resp, err := setNodeID(client, nodeId, DEFAULT_TIMEOUT, syncResult)
			if err != nil {
				return fmt.Errorf("set-node-id failed, %s", err.Error())
			}

			fmt.Printf("resp: %+v\n", resp)

			return nil
		},
	}

	attachFlags(cmd, []string{
		flagNodeId,
		flagSdkConfPath,
		flagOrgId, flagChainId,
		flagUserTlsCrtFilePath, flagUserTlsKeyFilePath,
		flagUserSignCrtFilePath, flagUserSignKeyFilePath,
		flagSyncResult,
	})

	cmd.MarkFlagRequired(flagNodeId)

	return cmd
}

func stakeGetNodeID() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-node-id",
		Short: "get-node-id feature of the stake",
		RunE: func(_ *cobra.Command, _ []string) error {
			var (
				err error
			)

			client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
				userSignCrtFilePath, userSignKeyFilePath)
			if err != nil {
				return err
			}
			defer client.Stop()
			pairs := make(map[string]string)
			if params != "" {
				err := json.Unmarshal([]byte(params), &pairs)
				if err != nil {
					return err
				}
			}

			resp, err := getNodeID(client, address, DEFAULT_TIMEOUT)
			if err != nil {
				return fmt.Errorf("get-node-id failed, %s", err.Error())
			}

			fmt.Printf("resp: %+v\n", resp)

			return nil
		},
	}

	attachFlags(cmd, []string{
		flagAddress,
		flagSdkConfPath,
		flagOrgId, flagChainId,
		flagUserTlsCrtFilePath, flagUserTlsKeyFilePath, flagUserSignCrtFilePath, flagUserSignKeyFilePath,
	})

	cmd.MarkFlagRequired(flagAddress)

	return cmd
}

func stakeReadMinSelfDelegation() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "min-self-delegation",
		Short: "min-self-delegation feature of the stake",
		RunE: func(_ *cobra.Command, _ []string) error {
			var (
				err error
			)

			client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
				userSignCrtFilePath, userSignKeyFilePath)
			if err != nil {
				return err
			}
			defer client.Stop()
			pairs := make(map[string]string)
			if params != "" {
				err := json.Unmarshal([]byte(params), &pairs)
				if err != nil {
					return err
				}
			}

			resp, err := readMinSelfDelegation(client, DEFAULT_TIMEOUT)
			if err != nil {
				return fmt.Errorf("min-self-delegation failed, %s", err.Error())
			}

			fmt.Printf("resp: %+v\n", resp)

			return nil
		},
	}

	attachFlags(cmd, []string{
		flagSdkConfPath,
		flagOrgId, flagChainId,
		flagUserTlsCrtFilePath, flagUserTlsKeyFilePath, flagUserSignCrtFilePath, flagUserSignKeyFilePath,
	})

	return cmd
}

func stakeReadEpochValidatorNumber() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validator-number",
		Short: "validator-number feature of the stake",
		RunE: func(_ *cobra.Command, _ []string) error {
			var (
				err error
			)

			client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
				userSignCrtFilePath, userSignKeyFilePath)
			if err != nil {
				return err
			}
			defer client.Stop()
			pairs := make(map[string]string)
			if params != "" {
				err := json.Unmarshal([]byte(params), &pairs)
				if err != nil {
					return err
				}
			}

			resp, err := readEpochValidatorNumber(client, DEFAULT_TIMEOUT)
			if err != nil {
				return fmt.Errorf("validator-number failed, %s", err.Error())
			}

			fmt.Printf("resp: %+v\n", resp)

			return nil
		},
	}

	attachFlags(cmd, []string{
		flagSdkConfPath,
		flagOrgId, flagChainId,
		flagUserTlsCrtFilePath, flagUserTlsKeyFilePath, flagUserSignCrtFilePath, flagUserSignKeyFilePath,
	})

	return cmd
}

func stakeReadEpochBlockNumber() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "epoch-block-number",
		Short: "epoch-block-number feature of the stake",
		RunE: func(_ *cobra.Command, _ []string) error {
			var (
				err error
			)

			client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
				userSignCrtFilePath, userSignKeyFilePath)
			if err != nil {
				return err
			}
			defer client.Stop()
			pairs := make(map[string]string)
			if params != "" {
				err := json.Unmarshal([]byte(params), &pairs)
				if err != nil {
					return err
				}
			}

			resp, err := readEpochBlockNumber(client, DEFAULT_TIMEOUT)
			if err != nil {
				return fmt.Errorf("epoch-block-number failed, %s", err.Error())
			}

			fmt.Printf("resp: %+v\n", resp)

			return nil
		},
	}

	attachFlags(cmd, []string{
		flagSdkConfPath,
		flagOrgId, flagChainId,
		flagUserTlsCrtFilePath, flagUserTlsKeyFilePath, flagUserSignCrtFilePath, flagUserSignKeyFilePath,
	})

	return cmd
}

func stakeReadSystemContractAddr() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "system-address",
		Short: "system-address feature of the stake",
		RunE: func(_ *cobra.Command, _ []string) error {
			var (
				err error
			)

			client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
				userSignCrtFilePath, userSignKeyFilePath)
			if err != nil {
				return err
			}
			defer client.Stop()
			pairs := make(map[string]string)
			if params != "" {
				err := json.Unmarshal([]byte(params), &pairs)
				if err != nil {
					return err
				}
			}

			resp, err := readSystemContractAddr(client, DEFAULT_TIMEOUT)
			if err != nil {
				return fmt.Errorf("system-address failed, %s", err.Error())
			}

			fmt.Printf("resp: %+v\n", resp)

			return nil
		},
	}

	attachFlags(cmd, []string{
		flagSdkConfPath,
		flagOrgId, flagChainId,
		flagUserTlsCrtFilePath, flagUserTlsKeyFilePath, flagUserSignCrtFilePath, flagUserSignKeyFilePath,
	})

	return cmd
}

func stakeReadCompleteUnBoundingEpochNumber() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unbonding-epoch-number",
		Short: "unbonding-epoch-number feature of the stake",
		RunE: func(_ *cobra.Command, _ []string) error {
			var (
				err error
			)

			client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
				userSignCrtFilePath, userSignKeyFilePath)
			if err != nil {
				return err
			}
			defer client.Stop()
			pairs := make(map[string]string)
			if params != "" {
				err := json.Unmarshal([]byte(params), &pairs)
				if err != nil {
					return err
				}
			}

			resp, err := readCompleteUnBoundingEpochNumber(client, DEFAULT_TIMEOUT)
			if err != nil {
				return fmt.Errorf("unbonding-epoch-number failed, %s", err.Error())
			}

			fmt.Printf("resp: %+v\n", resp)

			return nil
		},
	}

	attachFlags(cmd, []string{
		flagSdkConfPath,
		flagOrgId, flagChainId,
		flagUserTlsCrtFilePath, flagUserTlsKeyFilePath, flagUserSignCrtFilePath, flagUserSignKeyFilePath,
	})

	return cmd
}

func mint(cc *sdk.ChainClient, address, amount string, txId string, timeout int64,
	withSyncResult bool) (*common.TxResponse, error) {
	params := map[string]string{
		"to":    address,
		"value": amount,
	}
	if txId == "" {
		txId = sdkutils.GetRandTxId()
	}
	resp, err := cc.InvokeSystemContract(
		syscontract.SystemContract_DPOS_ERC20.String(),
		syscontract.DPoSERC20Function_MINT.String(),
		txId,
		util.ConvertParameters(params),
		timeout,
		withSyncResult,
	)
	if err != nil {
		return nil, fmt.Errorf("%s failed, %s", common.TxType_INVOKE_CONTRACT.String(), err.Error())
	}

	return resp, nil
}

func transfer(cc *sdk.ChainClient, address, amount string, txId string, timeout int64,
	withSyncResult bool) (*common.TxResponse, error) {
	params := map[string]string{
		"to":    address,
		"value": amount,
	}
	if txId == "" {
		txId = sdkutils.GetRandTxId()
	}
	resp, err := cc.InvokeSystemContract(
		syscontract.SystemContract_DPOS_ERC20.String(),
		syscontract.DPoSERC20Function_TRANSFER.String(),
		txId,
		util.ConvertParameters(params),
		timeout,
		withSyncResult,
	)
	if err != nil {
		return nil, fmt.Errorf("%s failed, %s", common.TxType_INVOKE_CONTRACT.String(), err.Error())
	}

	return resp, nil
}

func balanceOf(cc *sdk.ChainClient, address string, timeout int64) (*common.TxResponse, error) {
	params := map[string]string{
		"owner": address,
	}
	resp, err := cc.QuerySystemContract(
		syscontract.SystemContract_DPOS_ERC20.String(),
		syscontract.DPoSERC20Function_GET_BALANCEOF.String(),
		util.ConvertParameters(params),
		timeout,
	)
	if err != nil {
		return nil, fmt.Errorf("%s failed, %s", common.TxType_QUERY_CONTRACT.String(), err.Error())
	}

	return resp, nil
}

func owner(cc *sdk.ChainClient, timeout int64) (*common.TxResponse, error) {
	resp, err := cc.QuerySystemContract(
		syscontract.SystemContract_DPOS_ERC20.String(),
		syscontract.DPoSERC20Function_GET_OWNER.String(),
		nil,
		timeout,
	)
	if err != nil {
		return nil, fmt.Errorf("%s failed, %s", common.TxType_QUERY_CONTRACT.String(), err.Error())
	}

	return resp, nil
}

func decimals(cc *sdk.ChainClient, timeout int64) (*common.TxResponse, error) {
	resp, err := cc.QuerySystemContract(
		syscontract.SystemContract_DPOS_ERC20.String(),
		syscontract.DPoSERC20Function_GET_DECIMALS.String(),
		nil,
		timeout,
	)
	if err != nil {
		return nil, fmt.Errorf("%s failed, %s", common.TxType_QUERY_CONTRACT.String(), err.Error())
	}

	return resp, nil
}

func total(cc *sdk.ChainClient, timeout int64) (*common.TxResponse, error) {
	resp, err := cc.QuerySystemContract(
		syscontract.SystemContract_DPOS_ERC20.String(),
		syscontract.DPoSERC20Function_GET_TOTAL_SUPPLY.String(),
		nil,
		timeout,
	)
	if err != nil {
		return nil, fmt.Errorf("%s failed, %s", common.TxType_QUERY_CONTRACT.String(), err.Error())
	}

	return resp, nil
}

func getAllCandidates(cc *sdk.ChainClient, timeout int64) (*common.TxResponse, error) {
	resp, err := cc.QuerySystemContract(
		syscontract.SystemContract_DPOS_STAKE.String(),
		syscontract.DPoSStakeFunction_GET_ALL_CANDIDATES.String(),
		nil,
		timeout,
	)
	if err != nil {
		return nil, fmt.Errorf("%s failed, %s", common.TxType_QUERY_CONTRACT.String(), err.Error())
	}

	return resp, nil
}

func getValidatorByAddress(cc *sdk.ChainClient, address string, timeout int64) (*common.TxResponse, error) {
	params := map[string]string{
		"address": address,
	}
	resp, err := cc.QuerySystemContract(
		syscontract.SystemContract_DPOS_STAKE.String(),
		syscontract.DPoSStakeFunction_GET_VALIDATOR_BY_ADDRESS.String(),
		util.ConvertParameters(params),
		timeout,
	)
	if err != nil {
		return nil, fmt.Errorf("%s failed, %s", common.TxType_QUERY_CONTRACT.String(), err.Error())
	}

	return resp, nil
}

func delegate(cc *sdk.ChainClient, address, amount string, txId string, timeout int64,
	withSyncResult bool) (*common.TxResponse, error) {
	params := map[string]string{
		"to":     address,
		"amount": amount,
	}
	if txId == "" {
		txId = sdkutils.GetRandTxId()
	}
	resp, err := cc.InvokeSystemContract(
		syscontract.SystemContract_DPOS_STAKE.String(),
		syscontract.DPoSStakeFunction_DELEGATE.String(),
		txId,
		util.ConvertParameters(params),
		timeout,
		withSyncResult,
	)
	if err != nil {
		return nil, fmt.Errorf("%s failed, %s", common.TxType_INVOKE_CONTRACT.String(), err.Error())
	}

	return resp, nil
}

func getDelegationsByAddress(cc *sdk.ChainClient, address string, timeout int64) (*common.TxResponse, error) {
	params := map[string]string{
		"address": address,
	}
	resp, err := cc.QuerySystemContract(
		syscontract.SystemContract_DPOS_STAKE.String(),
		syscontract.DPoSStakeFunction_GET_DELEGATIONS_BY_ADDRESS.String(),
		util.ConvertParameters(params),
		timeout,
	)
	if err != nil {
		return nil, fmt.Errorf("%s failed, %s", common.TxType_QUERY_CONTRACT.String(), err.Error())
	}

	return resp, nil
}

func getUserDelegationByValidator(cc *sdk.ChainClient, delegatorAddress, validatorAddress string,
	timeout int64) (*common.TxResponse, error) {
	params := map[string]string{
		"delegator_address": delegatorAddress,
		"validator_address": validatorAddress,
	}
	resp, err := cc.QuerySystemContract(
		syscontract.SystemContract_DPOS_STAKE.String(),
		syscontract.DPoSStakeFunction_GET_USER_DELEGATION_BY_VALIDATOR.String(),
		util.ConvertParameters(params),
		timeout,
	)
	if err != nil {
		return nil, fmt.Errorf("%s failed, %s", common.TxType_QUERY_CONTRACT.String(), err.Error())
	}

	return resp, nil
}

func unDelegate(cc *sdk.ChainClient, address, amount string, txId string, timeout int64,
	withSyncResult bool) (*common.TxResponse, error) {
	params := map[string]string{
		"from":   address,
		"amount": amount,
	}
	if txId == "" {
		txId = sdkutils.GetRandTxId()
	}
	resp, err := cc.InvokeSystemContract(
		syscontract.SystemContract_DPOS_STAKE.String(),
		syscontract.DPoSStakeFunction_UNDELEGATE.String(),
		txId,
		util.ConvertParameters(params),
		timeout,
		withSyncResult,
	)
	if err != nil {
		return nil, fmt.Errorf("%s failed, %s", common.TxType_INVOKE_CONTRACT.String(), err.Error())
	}

	return resp, nil
}

func readEpochByID(cc *sdk.ChainClient, epochID string, timeout int64) (*common.TxResponse, error) {
	params := map[string]string{
		"epoch_id": epochID,
	}
	resp, err := cc.QuerySystemContract(
		syscontract.SystemContract_DPOS_STAKE.String(),
		syscontract.DPoSStakeFunction_READ_EPOCH_BY_ID.String(),
		util.ConvertParameters(params),
		timeout,
	)
	if err != nil {
		return nil, fmt.Errorf("%s failed, %s", common.TxType_QUERY_CONTRACT.String(), err.Error())
	}

	return resp, nil
}

func readLatestEpoch(cc *sdk.ChainClient, timeout int64) (*common.TxResponse, error) {
	resp, err := cc.QuerySystemContract(
		syscontract.SystemContract_DPOS_STAKE.String(),
		syscontract.DPoSStakeFunction_READ_LATEST_EPOCH.String(),
		nil,
		timeout,
	)
	if err != nil {
		return nil, fmt.Errorf("%s failed, %s", common.TxType_QUERY_CONTRACT.String(), err.Error())
	}

	return resp, nil
}

func setNodeID(cc *sdk.ChainClient, nodeID string, timeout int64, withSyncResult bool) (*common.TxResponse, error) {
	params := map[string]string{
		"node_id": nodeID,
	}
	if txId == "" {
		txId = sdkutils.GetRandTxId()
	}
	resp, err := cc.InvokeSystemContract(
		syscontract.SystemContract_DPOS_STAKE.String(),
		syscontract.DPoSStakeFunction_SET_NODE_ID.String(),
		txId,
		util.ConvertParameters(params),
		timeout,
		withSyncResult,
	)
	if err != nil {
		return nil, fmt.Errorf("%s failed, %s", common.TxType_INVOKE_CONTRACT.String(), err.Error())
	}

	return resp, nil
}

func getNodeID(cc *sdk.ChainClient, address string, timeout int64) (*common.TxResponse, error) {
	params := map[string]string{
		"address": address,
	}
	resp, err := cc.QuerySystemContract(
		syscontract.SystemContract_DPOS_STAKE.String(),
		syscontract.DPoSStakeFunction_GET_NODE_ID.String(),
		util.ConvertParameters(params),
		timeout,
	)
	if err != nil {
		return nil, fmt.Errorf("%s failed, %s", common.TxType_QUERY_CONTRACT.String(), err.Error())
	}

	return resp, nil
}

func readMinSelfDelegation(cc *sdk.ChainClient, timeout int64) (*common.TxResponse, error) {
	resp, err := cc.QuerySystemContract(
		syscontract.SystemContract_DPOS_STAKE.String(),
		syscontract.DPoSStakeFunction_READ_MIN_SELF_DELEGATION.String(),
		nil,
		timeout,
	)
	if err != nil {
		return nil, fmt.Errorf("%s failed, %s", common.TxType_QUERY_CONTRACT.String(), err.Error())
	}

	return resp, nil
}

func readEpochValidatorNumber(cc *sdk.ChainClient, timeout int64) (*common.TxResponse, error) {
	resp, err := cc.QuerySystemContract(
		syscontract.SystemContract_DPOS_STAKE.String(),
		syscontract.DPoSStakeFunction_READ_EPOCH_VALIDATOR_NUMBER.String(),
		nil,
		timeout,
	)
	if err != nil {
		return nil, fmt.Errorf("%s failed, %s", common.TxType_QUERY_CONTRACT.String(), err.Error())
	}

	return resp, nil
}

func readEpochBlockNumber(cc *sdk.ChainClient, timeout int64) (*common.TxResponse, error) {
	resp, err := cc.QuerySystemContract(
		syscontract.SystemContract_DPOS_STAKE.String(),
		syscontract.DPoSStakeFunction_READ_EPOCH_BLOCK_NUMBER.String(),
		nil,
		timeout,
	)
	if err != nil {
		return nil, fmt.Errorf("%s failed, %s", common.TxType_QUERY_CONTRACT.String(), err.Error())
	}

	return resp, nil
}

func readSystemContractAddr(cc *sdk.ChainClient, timeout int64) (*common.TxResponse, error) {
	resp, err := cc.QuerySystemContract(
		syscontract.SystemContract_DPOS_STAKE.String(),
		syscontract.DPoSStakeFunction_READ_SYSTEM_CONTRACT_ADDR.String(),
		nil,
		timeout,
	)
	if err != nil {
		return nil, fmt.Errorf("%s failed, %s", common.TxType_QUERY_CONTRACT.String(), err.Error())
	}

	return resp, nil
}

func readCompleteUnBoundingEpochNumber(cc *sdk.ChainClient, timeout int64) (*common.TxResponse, error) {
	resp, err := cc.QuerySystemContract(
		syscontract.SystemContract_DPOS_STAKE.String(),
		syscontract.DPoSStakeFunction_READ_COMPLETE_UNBOUNDING_EPOCH_NUMBER.String(),
		nil,
		timeout,
	)
	if err != nil {
		return nil, fmt.Errorf("%s failed, %s", common.TxType_QUERY_CONTRACT.String(), err.Error())
	}

	return resp, nil
}
