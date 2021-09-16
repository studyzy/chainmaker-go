/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"crypto/sha256"
	"encoding/json"
	_ "flag"
	"fmt"
	"io/ioutil"

	"chainmaker.org/chainmaker/utils/v2"
	"github.com/gogo/protobuf/proto"

	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"
	"github.com/mr-tron/base58/base58"
	"github.com/spf13/cobra"
)

var (
	userAddr string
	amount   string
)

const (
	userAddrName        = "user_addr"
	userAddrComments    = "address of the user"
	amountName          = "amount"
	amountValueComments = "amount of the value, the type is string"
)

func ERC20Cert2Address() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "erc20Cert2Address",
		Short: "cert to address",
		RunE: func(_ *cobra.Command, _ []string) error {
			return calAddressFromCert()
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&certPath, "cert_path", "", "path of cert that will calculate address")

	return cmd
}

func calAddressFromCert() error {
	if len(certPath) == 0 {
		panic("cert path is null")
	}

	certContent, err := ioutil.ReadFile(certPath)
	if err != nil {
		panic(fmt.Errorf("read cert content failed, reason: %s", err))
	}
	cert, err := utils.ParseCert(certContent)
	if err != nil {
		panic(fmt.Errorf("parse cert failed, reason: %s", err))
	}
	pubkey, err := cert.PublicKey.Bytes()
	if err != nil {
		panic(fmt.Errorf("get pubkey failed from cert, reason: %s", err))
	}
	hash := sha256.Sum256(pubkey)
	addr := base58.Encode(hash[:])
	fmt.Printf("address: %s from cert: %s\n", addr, certPath)

	result := &Result{
		Code:                  commonPb.TxStatusCode_SUCCESS,
		Message:               "success",
		ContractQueryResult:   addr,
		ContractResultMessage: "success",
	}
	fmt.Println(result.ToJsonString())
	return nil
}

func ERC20Mint() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "erc20Mint",
		Short: "mint feature of the erc20",
		RunE: func(_ *cobra.Command, _ []string) error {
			return mint()
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&userAddr, userAddrName, "", userAddrComments)
	flags.StringVar(&amount, amountName, "", amountValueComments)

	return cmd
}

func mint() error {
	if err := checkBase58Addr(userAddr); err != nil {
		return err
	}
	params := []*commonPb.KeyValuePair{
		{
			Key:   "to",
			Value: []byte(userAddr),
		},
		{
			Key:   "value",
			Value: []byte(amount),
		},
	}
	resp, txId, err := configUpdateRequest(sk3, client, &InvokerMsg{
		txId: "", chainId: chainId,
		txType:       commonPb.TxType_INVOKE_CONTRACT,
		contractName: syscontract.SystemContract_DPOS_ERC20.String(),
		method:       syscontract.DPoSERC20Function_MINT.String(),
		pairs:        params,
	})
	return processRespWithTxId(resp, txId, err)
}

func ERC20Transfer() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "erc20Transfer",
		Short: "transfer feature of the erc20",
		RunE: func(_ *cobra.Command, _ []string) error {
			return transfer()
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&userAddr, userAddrName, "", userAddrComments)
	flags.StringVar(&amount, amountName, "", amountValueComments)
	return cmd
}

func transfer() error {
	if err := checkBase58Addr(userAddr); err != nil {
		return err
	}
	params := []*commonPb.KeyValuePair{
		{
			Key:   "to",
			Value: []byte(userAddr),
		},
		{
			Key:   "value",
			Value: []byte(amount),
		},
	}
	resp, txId, err := configUpdateRequest(sk3, client, &InvokerMsg{
		txId: "", chainId: chainId,
		txType:       commonPb.TxType_INVOKE_CONTRACT,
		contractName: syscontract.SystemContract_DPOS_ERC20.String(),
		method:       syscontract.DPoSERC20Function_TRANSFER.String(),
		pairs:        params,
	})
	return processRespWithTxId(resp, txId, err)
}

func processRespWithTxId(resp *commonPb.TxResponse, txId string, err error) error {
	if err != nil {
		return err
	}
	result := &Result{
		Code:    resp.Code,
		Message: resp.Message,
		TxId:    txId,
	}
	bytes, err := json.Marshal(result)
	if err != nil {
		return err
	}

	fmt.Println(string(bytes))
	return nil
}

func ERC20BalanceOf() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "erc20BalanceOf",
		Short: "balance of the userAddr in erc20 contract",
		RunE: func(_ *cobra.Command, _ []string) error {
			return balanceOf()
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&userAddr, userAddrName, "", userAddrComments)
	return cmd
}

func balanceOf() error {
	if err := checkBase58Addr(userAddr); err != nil {
		return err
	}
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   "owner",
			Value: []byte(userAddr),
		},
	}
	payloadBytes, err := constructQueryPayload(chainId, syscontract.SystemContract_DPOS_ERC20.String(), syscontract.DPoSERC20Function_GET_BALANCEOF.String(), pairs)
	if err != nil {
		log.Fatalf("create payload failed, err: %s", err)
	}
	resp, err := proposalRequest(sk3, client, payloadBytes)
	if err != nil {
		return err
	}
	return processResult(resp, nil)
}

func checkBase58Addr(addr string) error {
	_, err := base58.Decode(addr)
	return err
}

func ERC20Owner() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "erc20Owner",
		Short: "owner of erc20",
		Long:  "the owner of erc20 contract",
		RunE: func(_ *cobra.Command, _ []string) error {
			return owner()
		},
	}
	return cmd
}

func owner() error {
	pairs := make([]*commonPb.KeyValuePair, 0)
	payloadBytes, err := constructQueryPayload(chainId, syscontract.SystemContract_DPOS_ERC20.String(), syscontract.DPoSERC20Function_GET_OWNER.String(), pairs)
	if err != nil {
		log.Fatalf("create payload failed, err: %s", err)
	}
	resp, err := proposalRequest(sk3, client, payloadBytes)
	if err != nil {
		return err
	}
	return processResult(resp, nil)
}

func ERC20Decimals() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "erc20Decimals",
		Short: "decimals of erc20",
		Long:  "the decimals of erc20 contract",
		RunE: func(_ *cobra.Command, _ []string) error {
			return decimals()
		},
	}
	return cmd
}

func decimals() error {
	pairs := make([]*commonPb.KeyValuePair, 0)
	payloadBytes, err := constructQueryPayload(chainId, syscontract.SystemContract_DPOS_ERC20.String(), syscontract.DPoSERC20Function_GET_DECIMALS.String(), pairs)
	if err != nil {
		log.Fatalf("create payload failed, err: %s", err)
	}
	resp, err := proposalRequest(sk3, client, payloadBytes)
	if err != nil {
		return err
	}
	return processResult(resp, nil)
}

func ERC20Total() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "erc20Total",
		Short: "total supply of erc20",
		Long:  " get total supply of tokens",
		RunE: func(_ *cobra.Command, _ []string) error {
			return total()
		},
	}
	return cmd
}

func total() error {
	pairs := make([]*commonPb.KeyValuePair, 0)
	payloadBytes, err := constructQueryPayload(chainId, syscontract.SystemContract_DPOS_ERC20.String(), syscontract.DPoSERC20Function_GET_TOTAL_SUPPLY.String(), pairs)
	if err != nil {
		log.Fatalf("create payload failed, err: %s", err)
	}
	resp, err := proposalRequest(sk3, client, payloadBytes)
	if err != nil {
		return err
	}
	return processResult(resp, nil)
}

func processResult(resp *commonPb.TxResponse, m proto.Message) error {
	if m != nil {
		err := proto.Unmarshal(resp.ContractResult.Result, m)
		if err != nil {
			return err
		}
	}
	var queryResult string
	if m != nil {
		bz, err := json.Marshal(m)
		if err != nil {
			return err
		}
		queryResult = string(bz)
	} else {
		queryResult = string(resp.ContractResult.Result)
	}
	result := &Result{
		Code:                  resp.Code,
		Message:               resp.Message,
		TxId:                  txId,
		ContractQueryResult:   queryResult,
		ContractResultMessage: resp.ContractResult.Message,
	}
	fmt.Println(result.ToJsonString())
	return nil
}
