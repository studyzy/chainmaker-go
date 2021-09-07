/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package blocksqldb

import (
	"testing"

	"chainmaker.org/chainmaker/pb-go/v2/config"
	"chainmaker.org/chainmaker/pb-go/v2/consensus"
	"chainmaker.org/chainmaker/utils/v2"
	"github.com/stretchr/testify/assert"
)

func TestNewTxInfo(t *testing.T) {
	chainConfig := &config.ChainConfig{ChainId: "chain1", Crypto: &config.CryptoConfig{Hash: "SM3"}, Consensus: &config.ConsensusConfig{Type: consensus.ConsensusType_SOLO}}
	genesis, _, _ := utils.CreateGenesis(chainConfig)
	tx := genesis.Txs[0]
	info, err := NewTxInfo(tx, 0, []byte("hash"), 0)
	assert.Nil(t, err)
	t.Logf("%#v", info)
}
func TestInsertTxInfo(t *testing.T) {
	p := initProvider()
	chainConfig := &config.ChainConfig{ChainId: "chain1", Crypto: &config.CryptoConfig{Hash: "SM3"}, Consensus: &config.ConsensusConfig{Type: consensus.ConsensusType_SOLO}}
	genesis, _, _ := utils.CreateGenesis(chainConfig)
	tx := genesis.Txs[0]
	info, _ := NewTxInfo(tx, 0, []byte("hash"), 0)
	i, err := p.Save(info)
	assert.Nil(t, err)
	assert.Equal(t, int64(1), i)
}
