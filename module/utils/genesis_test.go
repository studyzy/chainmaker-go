/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package utils

import (
	commonPb "chainmaker.org/chainmaker/pb-go/common"
	"crypto/sha256"
	"testing"

	"github.com/mr-tron/base58/base58"
	"github.com/stretchr/testify/require"
)

func TestERC20Config_load(t *testing.T) {
	/**
	- key: erc20.total
	  value: 1000000
	- key: erc20.owner
	  value: 5pQfwDwtyA
	- key: erc20.decimals
	  value: 18
	- key: erc20.account:<addr1>
	  value: 800000
	- key: erc20.account:<addr2>
	  value: 200000
	*/
	var (
		//stakeHash1 = sha256.Sum256([]byte("stake1"))
		//stakeAddr1 = base58.Encode(stakeHash1[:])
		//
		//stakeHash2 = sha256.Sum256([]byte("stake2"))
		//stakeAddr2 = base58.Encode(stakeHash2[:])
		//
		//stakeHash3 = sha256.Sum256([]byte("stake3"))
		//stakeAddr3 = base58.Encode(stakeHash3[:])

		contractAddr = getContractAddress()

		hash  = sha256.Sum256([]byte("owner"))
		owner = base58.Encode(hash[:])
	)

	var tests = []*commonPb.KeyValuePair{
		{
			Key:   keyERC20Total,
			Value: "1000000",
		},
		{
			Key:   keyERC20Owner,
			Value: owner,
		},
		{
			Key:   keyERC20Decimals,
			Value: "18",
		},
		{
			Key:   keyERC20Acc + owner,
			Value: "800000",
		},
		{
			Key:   keyERC20Acc + commonPb.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String(),
			Value: "200000",
		},
	}
	erc20Config, err := loadERC20Config(tests)
	require.Nil(t, err)
	require.NotNil(t, erc20Config)
	require.Equal(t, "1000000", erc20Config.total.String())
	require.Equal(t, owner, erc20Config.owner)
	require.Equal(t, "18", erc20Config.decimals.String())
	ownerToken := erc20Config.loadToken(owner)
	require.Equal(t, "800000", ownerToken.String())
	contractAddrToken := erc20Config.loadToken(contractAddr)
	require.Equal(t, "200000", contractAddrToken.String())
	err = erc20Config.legal()
	require.Nil(t, err)
	txWrites := erc20Config.toTxWrites()
	require.Equal(t, 5, len(txWrites))
}
