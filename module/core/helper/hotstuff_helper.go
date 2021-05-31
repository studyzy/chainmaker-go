/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package helper

import (
	commonpb "chainmaker.org/chainmaker/pb-go/common"
	consensusPb "chainmaker.org/chainmaker/pb-go/consensus"
	"chainmaker.org/chainmaker-go/protocol"
)

type hotStuffHelper struct {
	txPool        protocol.TxPool
	chainConf     protocol.ChainConf
	proposalCache protocol.ProposalCache
}

func NewHotStuffHelper(txPool protocol.TxPool,
	chainConf protocol.ChainConf, proposalCache protocol.ProposalCache) protocol.HotStuffHelper {
	return &hotStuffHelper{txPool: txPool, chainConf: chainConf, proposalCache: proposalCache}
}

func (hp *hotStuffHelper) DiscardAboveHeight(baseHeight int64) {
	if hp.chainConf.ChainConfig().Consensus.Type != consensusPb.ConsensusType_HOTSTUFF {
		return
	}
	delBlocks := hp.proposalCache.DiscardAboveHeight(baseHeight)
	if len(delBlocks) == 0 {
		return
	}
	txs := make([]*commonpb.Transaction, 0, 100)
	for _, blk := range delBlocks {
		txs = append(txs, blk.Txs...)
	}
	hp.txPool.RetryAndRemoveTxs(txs, nil)
}
