/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"
	"github.com/spf13/cobra"
)

var (
	nodeID        = ""
	epochID       = ""
	delegatorAddr = ""
	validatorAddr = ""
)

func StakeDelegate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delegation",
		Short: "delegation feature of the stake",
		Long:  "delegate tokens to the designated address to participate in the consensus",
		RunE: func(_ *cobra.Command, _ []string) error {
			return delegation()
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&userAddr, userAddrName, "", userAddrComments)
	flags.StringVar(&amount, amountName, "", amountValueComments)

	return cmd
}

func delegation() error {
	if err := checkBase58Addr(userAddr); err != nil {
		return err
	}
	params := []*commonPb.KeyValuePair{
		{
			Key:   "to",
			Value: []byte(userAddr),
		},
		{
			Key:   "amount",
			Value: []byte(amount),
		},
	}
	resp, txId, err := configUpdateRequest(sk3, client, &InvokerMsg{
		txId: "", chainId: chainId,
		txType:       commonPb.TxType_INVOKE_CONTRACT,
		contractName: syscontract.SystemContract_DPOS_STAKE.String(),
		method:       syscontract.DPoSStakeFunction_DELEGATE.String(),
		pairs:        params,
	})
	return processRespWithTxId(resp, txId, err)
}

func StakeUnDelegate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "undelegation",
		Short: "undelegation feature of the stake",
		Long:  "To redeem the delegate tokens at a designated address",
		RunE: func(_ *cobra.Command, _ []string) error {
			return undelegation()
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&userAddr, userAddrName, "", userAddrComments)
	flags.StringVar(&amount, amountName, "", amountValueComments)

	return cmd
}

func undelegation() error {
	if err := checkBase58Addr(userAddr); err != nil {
		return err
	}
	params := []*commonPb.KeyValuePair{
		{
			Key:   "from",
			Value: []byte(userAddr),
		},
		{
			Key:   "amount",
			Value: []byte(amount),
		},
	}
	resp, txId, err := configUpdateRequest(sk3, client, &InvokerMsg{
		txId: "", chainId: chainId,
		txType:       commonPb.TxType_INVOKE_CONTRACT,
		contractName: syscontract.SystemContract_DPOS_STAKE.String(),
		method:       syscontract.DPoSStakeFunction_UNDELEGATE.String(),
		pairs:        params,
	})
	return processRespWithTxId(resp, txId, err)
}

func StakeSetNodeID() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "setNodeID",
		Short: "setNodeID feature of the stake",
		Long:  "Set the user's nodeID",
		RunE: func(_ *cobra.Command, _ []string) error {
			return setNodeID()
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&nodeID, "nodeID", "", "id of the node")
	return cmd
}

func setNodeID() error {
	params := []*commonPb.KeyValuePair{
		{
			Key:   "node_id",
			Value: []byte(nodeID),
		},
	}
	resp, txId, err := configUpdateRequest(sk3, client, &InvokerMsg{
		chainId:      chainId,
		txType:       commonPb.TxType_INVOKE_CONTRACT,
		contractName: syscontract.SystemContract_DPOS_STAKE.String(),
		method:       syscontract.DPoSStakeFunction_SET_NODE_ID.String(),
		pairs:        params,
	})
	return processRespWithTxId(resp, txId, err)
}

func StakeGetAllCandidates() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "getAllCandidates",
		Short: "getAllCandidates feature of the stake",
		Long:  "Get a list of all candidates",
		RunE: func(_ *cobra.Command, _ []string) error {
			return getAllCandidates()
		},
	}
	return cmd
}

func getAllCandidates() error {
	pairs := make([]*commonPb.KeyValuePair, 0)
	payloadBytes, err := constructQueryPayload(chainId, syscontract.SystemContract_DPOS_STAKE.String(), syscontract.DPoSStakeFunction_GET_ALL_CANDIDATES.String(), pairs)
	if err != nil {
		log.Fatalf("create payload failed, err: %s", err)
	}
	resp, err := proposalRequest(sk3, client, payloadBytes)
	if err != nil {
		return err
	}
	return processResult(resp, &syscontract.ValidatorVector{})
}

func StakeGetNodeID() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "getNodeID",
		Short: "getNodeID feature of the stake",
		Long:  "Get the user's NodeID",
		RunE: func(_ *cobra.Command, _ []string) error {
			return getNodeID()
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&userAddr, userAddrName, "", userAddrComments)
	return cmd
}

func getNodeID() error {
	if err := checkBase58Addr(userAddr); err != nil {
		return err
	}
	pairs := make([]*commonPb.KeyValuePair, 0)
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "address",
		Value: []byte(userAddr),
	})
	payloadBytes, err := constructQueryPayload(chainId, syscontract.SystemContract_DPOS_STAKE.String(), syscontract.DPoSStakeFunction_GET_NODE_ID.String(), pairs)
	if err != nil {
		log.Fatalf("create payload failed, err: %s", err)
	}
	resp, err := proposalRequest(sk3, client, payloadBytes)
	if err != nil {
		return err
	}
	return processResult(resp, nil)
}

func StakeGetEpochByID() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "getEpochByID",
		Short: "getEpochByID feature of the stake",
		Long:  "Gets the content of the specified epoch by id",
		RunE: func(_ *cobra.Command, _ []string) error {
			return getEpochByID()
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&epochID, "epochID", "", "id of the epoch")
	return cmd
}

func getEpochByID() error {
	pairs := make([]*commonPb.KeyValuePair, 0)
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "epoch_id",
		Value: []byte(epochID),
	})
	payloadBytes, err := constructQueryPayload(chainId, syscontract.SystemContract_DPOS_STAKE.String(), syscontract.DPoSStakeFunction_READ_EPOCH_BY_ID.String(), pairs)
	if err != nil {
		log.Fatalf("create payload failed, err: %s", err)
	}
	resp, err := proposalRequest(sk3, client, payloadBytes)
	if err != nil {
		return err
	}
	return processResult(resp, &syscontract.Epoch{})
}

func StakeGetLatestEpoch() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "getLatestEpoch",
		Short: "getLatestEpoch feature of the stake",
		Long:  "Get the latest epoch content",
		RunE: func(_ *cobra.Command, _ []string) error {
			return getLatestEpoch()
		},
	}
	return cmd
}

func getLatestEpoch() error {
	pairs := make([]*commonPb.KeyValuePair, 0)
	payloadBytes, err := constructQueryPayload(chainId, syscontract.SystemContract_DPOS_STAKE.String(), syscontract.DPoSStakeFunction_READ_LATEST_EPOCH.String(), pairs)
	if err != nil {
		log.Fatalf("create payload failed, err: %s", err)
	}
	resp, err := proposalRequest(sk3, client, payloadBytes)
	if err != nil {
		return err
	}
	return processResult(resp, &syscontract.Epoch{})
}

func StakeGetSystemAddr() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "getSystemAddr",
		Short: "getSystemAddr feature of the stake",
		Long:  "Get the address of stake system contract",
		RunE: func(_ *cobra.Command, _ []string) error {
			return getSystemAddr()
		},
	}
	return cmd
}

func getSystemAddr() error {
	pairs := make([]*commonPb.KeyValuePair, 0)
	payloadBytes, err := constructQueryPayload(chainId, syscontract.SystemContract_DPOS_STAKE.String(), syscontract.DPoSStakeFunction_READ_SYSTEM_CONTRACT_ADDR.String(), pairs)
	if err != nil {
		log.Fatalf("create payload failed, err: %s", err)
	}
	resp, err := proposalRequest(sk3, client, payloadBytes)
	if err != nil {
		return err
	}
	return processResult(resp, nil)
}

func StakeGetDelegationsByAddress() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "getDelegationsByAddress",
		Short: "getDelegationsByAddress feature of the stake",
		Long:  "Get all delegate records for the specified address",
		RunE: func(_ *cobra.Command, _ []string) error {
			return getDelegationsByAddress()
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&userAddr, userAddrName, "", userAddrComments)
	return cmd
}

func getDelegationsByAddress() error {
	pairs := make([]*commonPb.KeyValuePair, 0)
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "address",
		Value: []byte(userAddr),
	})
	payloadBytes, err := constructQueryPayload(chainId, syscontract.SystemContract_DPOS_STAKE.String(), syscontract.DPoSStakeFunction_GET_DELEGATIONS_BY_ADDRESS.String(), pairs)
	if err != nil {
		log.Fatalf("create payload failed, err: %s", err)
	}
	resp, err := proposalRequest(sk3, client, payloadBytes)
	if err != nil {
		return err
	}
	return processResult(resp, &syscontract.DelegationInfo{})
}

func StakeGetDelegationByValidator() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "getDelegationByValidator",
		Short: "getDelegationByValidator feature of the stake",
		Long:  "Gets the delegate information for the specified address on the specified user",
		RunE: func(_ *cobra.Command, _ []string) error {
			return getDelegationByValidator()
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&delegatorAddr, "delegatorAddr", "", "address of the delegator")
	flags.StringVar(&validatorAddr, "validatorAddr", "", "address of the validator")

	return cmd
}

func getDelegationByValidator() error {
	if err := checkBase58Addr(delegatorAddr); err != nil {
		return err
	}
	if err := checkBase58Addr(validatorAddr); err != nil {
		return err
	}
	pairs := make([]*commonPb.KeyValuePair, 0)
	pairs = append(pairs,
		&commonPb.KeyValuePair{
			Key:   "delegator_address",
			Value: []byte(delegatorAddr),
		},
		&commonPb.KeyValuePair{
			Key:   "validator_address",
			Value: []byte(validatorAddr),
		},
	)
	payloadBytes, err := constructQueryPayload(chainId, syscontract.SystemContract_DPOS_STAKE.String(), syscontract.DPoSStakeFunction_GET_USER_DELEGATION_BY_VALIDATOR.String(), pairs)
	if err != nil {
		log.Fatalf("create payload failed, err: %s", err)
	}
	resp, err := proposalRequest(sk3, client, payloadBytes)
	if err != nil {
		return err
	}
	return processResult(resp, &syscontract.Delegation{})
}

func StakeGetMinSelfDelegation() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "getMinSelfDelegation",
		Short: "getMinSelfDelegation feature of the stake",
		Long:  "Get the minimum amount of delegates for a node to become a verifier candidate",
		RunE: func(_ *cobra.Command, _ []string) error {
			return getMinSelfDelegation()
		},
	}
	return cmd
}

func getMinSelfDelegation() error {
	pairs := make([]*commonPb.KeyValuePair, 0)
	payloadBytes, err := constructQueryPayload(chainId, syscontract.SystemContract_DPOS_STAKE.String(), syscontract.DPoSStakeFunction_READ_MIN_SELF_DELEGATION.String(), pairs)
	if err != nil {
		log.Fatalf("create payload failed, err: %s", err)
	}
	resp, err := proposalRequest(sk3, client, payloadBytes)
	if err != nil {
		return err
	}
	return processResult(resp, nil)
}

func StakeGetEpochValidatorNumber() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "getEpochValidatorNumber",
		Short: "getEpochValidatorNumber feature of the stake",
		Long:  "Get the number of validators for the epoch",
		RunE: func(_ *cobra.Command, _ []string) error {
			return getEpochValidatorNumber()
		},
	}
	return cmd
}

func getEpochValidatorNumber() error {
	pairs := make([]*commonPb.KeyValuePair, 0)
	payloadBytes, err := constructQueryPayload(chainId, syscontract.SystemContract_DPOS_STAKE.String(), syscontract.DPoSStakeFunction_READ_EPOCH_VALIDATOR_NUMBER.String(), pairs)
	if err != nil {
		log.Fatalf("create payload failed, err: %s", err)
	}
	resp, err := proposalRequest(sk3, client, payloadBytes)
	if err != nil {
		return err
	}
	return processResult(resp, nil)
}

func StakeGetEpochBlockNumber() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "getEpochBlockNumber",
		Short: "getEpochBlockNumber feature of the stake",
		Long:  "Get the number of blocks in the generation",
		RunE: func(_ *cobra.Command, _ []string) error {
			return getEpochBlockNumber()
		},
	}
	return cmd
}

func getEpochBlockNumber() error {
	pairs := make([]*commonPb.KeyValuePair, 0)
	payloadBytes, err := constructQueryPayload(chainId, syscontract.SystemContract_DPOS_STAKE.String(), syscontract.DPoSStakeFunction_READ_EPOCH_BLOCK_NUMBER.String(), pairs)
	if err != nil {
		log.Fatalf("create payload failed, err: %s", err)
	}
	resp, err := proposalRequest(sk3, client, payloadBytes)
	if err != nil {
		return err
	}
	return processResult(resp, nil)
}

func StakeGetUnbondingEpochNumber() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "getUnbondingEpochNumber",
		Short: "getUnbondingEpochNumber feature of the stake",
		Long:  "Get the number of unbonding epoch number",
		RunE: func(_ *cobra.Command, _ []string) error {
			return getUnbondingEpochNumber()
		},
	}
	return cmd
}

func getUnbondingEpochNumber() error {
	pairs := make([]*commonPb.KeyValuePair, 0)
	payloadBytes, err := constructQueryPayload(chainId, syscontract.SystemContract_DPOS_STAKE.String(), syscontract.DPoSStakeFunction_READ_COMPLETE_UNBOUNDING_EPOCH_NUMBER.String(), pairs)
	if err != nil {
		log.Fatalf("create payload failed, err: %s", err)
	}
	resp, err := proposalRequest(sk3, client, payloadBytes)
	if err != nil {
		return err
	}
	return processResult(resp, nil)
}
