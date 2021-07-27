/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package blocksqldb

import (
	"testing"

	"chainmaker.org/chainmaker-go/utils"
	"chainmaker.org/chainmaker/pb-go/config"
	"chainmaker.org/chainmaker/pb-go/consensus"
	"github.com/stretchr/testify/assert"
)

func TestNewBlockInfo(t *testing.T) {
	chainConfig := &config.ChainConfig{ChainId: "chain1", Crypto: &config.CryptoConfig{Hash: "SM3"}, Consensus: &config.ConsensusConfig{Type: consensus.ConsensusType_SOLO}}
	genesis, _, _ := utils.CreateGenesis(chainConfig)
	binfo, err := NewBlockInfo(genesis)
	assert.Nil(t, err)
	t.Logf("%v", binfo)
}
