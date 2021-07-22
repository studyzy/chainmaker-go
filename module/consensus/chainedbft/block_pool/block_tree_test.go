/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package blockpool

import (
	"testing"

	"chainmaker.org/chainmaker-go/utils"
	"chainmaker.org/chainmaker/pb-go/common"
)

func TestBlockTree_InsertBlock(t *testing.T) {
	rootBlk := common.Block{Header: &common.BlockHeader{BlockHash: []byte(utils.GetRandTxId())}}
	tree := NewBlockTree(&rootBlk, nil, 10)
	//tree.InsertBlock()
	_ = tree
}
