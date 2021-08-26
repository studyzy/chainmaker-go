/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package utils

import (
	"crypto/sha256"
	"testing"

	"chainmaker.org/chainmaker/pb-go/config"
	"chainmaker.org/chainmaker/pb-go/consensus"
	"chainmaker.org/chainmaker/pb-go/syscontract"
	"github.com/stretchr/testify/assert"

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

	var tests = []*config.ConfigKeyValue{
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
			Key:   keyERC20Acc + syscontract.SystemContract_DPOS_STAKE.String(),
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
func TestGenConfigTxRWSet(t *testing.T) {
	chainConfig := &config.ChainConfig{ChainId: "chain1", Consensus: &config.ConsensusConfig{Type: consensus.ConsensusType_SOLO}}
	rwset, err := genConfigTxRWSet(chainConfig)
	assert.Nil(t, err)
	for _, write := range rwset.TxWrites {
		t.Logf("[%s]\t%s\t%x", write.ContractName, write.Key, write.Value)
	}
}
func TestCreateGenesis(t *testing.T) {
	chainConfig := &config.ChainConfig{ChainId: "chain1", Crypto: &config.CryptoConfig{Hash: "SM3"}, Consensus: &config.ConsensusConfig{Type: consensus.ConsensusType_SOLO}}
	genesis, _, err := CreateGenesis(chainConfig)
	t.Log(genesis)
	assert.Nil(t, err)
	assert.True(t, IsConfBlock(genesis))

}
