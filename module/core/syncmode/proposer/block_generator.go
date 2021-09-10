/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package proposer

import (
	commonpb "chainmaker.org/chainmaker/pb-go/v2/common"
)

func (bp *BlockProposerImpl) generateNewBlock(proposingHeight uint64, preHash []byte,
	txBatch []*commonpb.Transaction) (*commonpb.Block, []int64, error) {
	return bp.blockBuilder.GenerateNewBlock(proposingHeight, preHash, txBatch)
}
