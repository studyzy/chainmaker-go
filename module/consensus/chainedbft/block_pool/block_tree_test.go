/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package blockpool

import (
	"testing"

	"chainmaker.org/chainmaker-go/pb/protogo/common"
)

func TestBlockTree_InsertBlock(t *testing.T) {
	rootBlk := common.Block{}
	tree := NewBlockTree(&rootBlk, 10)
	//tree.InsertBlock()
	_ = tree
}
