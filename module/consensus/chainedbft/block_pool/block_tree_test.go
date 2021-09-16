/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package blockpool

import (
	"testing"

	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/utils/v2"
)

func TestBlockTree_InsertBlock(t *testing.T) {
	rootBlk := common.Block{Header: &common.BlockHeader{BlockHash: []byte(utils.GetRandTxId())}}
	tree := NewBlockTree(&rootBlk, 10)
	//tree.InsertBlock()
	_ = tree
}
