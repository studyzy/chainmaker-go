/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package chainedbft

import (
	"fmt"
	"sort"
	"testing"

	"chainmaker.org/chainmaker/pb-go/v2/common"
)

func TestConsensusChainedBftImpl_ProcessProposedBlock(t *testing.T) {
	blks := make([]*common.Block, 0, 5)
	for i := 0; i < cap(blks); i++ {
		blks = append(blks, &common.Block{Header: &common.BlockHeader{BlockHeight: uint64(100 - i)}})
	}
	fmt.Println(blks)

	sort.Slice(blks, func(i, j int) bool {
		return blks[i].Header.BlockHeight < blks[j].Header.BlockHeight
	})
	fmt.Println(blks)
}
