/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package helper

import (
	"chainmaker.org/chainmaker-go/logger"
	commonpb "chainmaker.org/chainmaker-go/pb/protogo/common"
	consensusPb "chainmaker.org/chainmaker-go/pb/protogo/consensus"
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
	log := logger.GetLogger("hotstuff helper ...")
	if hp.chainConf.ChainConfig().Consensus.Type != consensusPb.ConsensusType_HOTSTUFF {
		return
	}
	log.Debugf(" hotStuffHelper 11111")
	delBlocks := hp.proposalCache.DiscardAboveHeight(baseHeight)
	log.Debugf(" hotStuffHelper 2222")
	if len(delBlocks) == 0 {
		return
	}
	log.Debugf(" hotStuffHelper 3333")
	txs := make([]*commonpb.Transaction, 0, 100)
	for _, blk := range delBlocks {
		txs = append(txs, blk.Txs...)
	}
	hp.txPool.RetryAndRemoveTxs(txs, nil)
	log.Debugf(" hotStuffHelper 4444")
}
