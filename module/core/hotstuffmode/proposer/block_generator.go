/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package proposer

import (
	commonpb "chainmaker.org/chainmaker-go/pb/protogo/common"
)

func (bp *BlockProposerImpl) generateNewBlock(proposingHeight int64, preHash []byte, txBatch []*commonpb.Transaction) (*commonpb.Block, []int64, error) {
	return bp.blockBuilder.GenerateNewBlock(proposingHeight, preHash, txBatch)
}
