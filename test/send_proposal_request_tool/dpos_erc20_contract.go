package main

import (
	"encoding/json"
	"fmt"

	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
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
			Value: userAddr,
		},
		{
			Key:   "value",
			Value: amount,
		},
	}
	resp, txId, err := configUpdateRequest(sk3, client, &InvokerMsg{
		txId: "", chainId: chainId,
		txType:       commonPb.TxType_INVOKE_SYSTEM_CONTRACT,
		contractName: commonPb.ContractName_SYSTEM_CONTRACT_DPOS_ERC20.String(),
		method:       commonPb.DPoSERC20ContractFunction_MINT.String(),
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
			Value: userAddr,
		},
		{
			Key:   "value",
			Value: amount,
		},
	}
	resp, txId, err := configUpdateRequest(sk3, client, &InvokerMsg{
		txId: "", chainId: chainId,
		txType:       commonPb.TxType_INVOKE_SYSTEM_CONTRACT,
		contractName: commonPb.ContractName_SYSTEM_CONTRACT_DPOS_ERC20.String(),
		method:       commonPb.DPoSERC20ContractFunction_TRANSFER.String(),
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
			Value: userAddr,
		},
	}
	payloadBytes, err := constructPayload(commonPb.ContractName_SYSTEM_CONTRACT_DPOS_ERC20.String(), commonPb.DPoSERC20ContractFunction_GET_BALANCEOF.String(), pairs)
	if err != nil {
		log.Fatalf("create payload failed, err: %s", err)
	}
	resp, err := proposalRequest(sk3, client, commonPb.TxType_QUERY_SYSTEM_CONTRACT,
		chainId, "", payloadBytes)
	return processResult(resp, err)
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
	payloadBytes, err := constructPayload(commonPb.ContractName_SYSTEM_CONTRACT_DPOS_ERC20.String(), commonPb.DPoSERC20ContractFunction_GET_OWNER.String(), pairs)
	if err != nil {
		log.Fatalf("create payload failed, err: %s", err)
	}
	resp, err := proposalRequest(sk3, client, commonPb.TxType_QUERY_SYSTEM_CONTRACT,
		chainId, "", payloadBytes)
	return processResult(resp, err)
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
	payloadBytes, err := constructPayload(commonPb.ContractName_SYSTEM_CONTRACT_DPOS_ERC20.String(), commonPb.DPoSERC20ContractFunction_GET_DECIMALS.String(), pairs)
	if err != nil {
		log.Fatalf("create payload failed, err: %s", err)
	}
	resp, err := proposalRequest(sk3, client, commonPb.TxType_QUERY_SYSTEM_CONTRACT,
		chainId, "", payloadBytes)
	return processResult(resp, err)
}

func processResult(resp *commonPb.TxResponse, err error) error {
	if err != nil {
		return err
	}
	result := &Result{
		Code:                  resp.Code,
		Message:               resp.Message,
		TxId:                  txId,
		ContractQueryResult:   string(resp.ContractResult.Result),
		ContractResultMessage: resp.ContractResult.Message,
	}
	bytes, err := json.Marshal(result)
	if err != nil {
		return err
	}
	fmt.Println(string(bytes))
	return nil
}
