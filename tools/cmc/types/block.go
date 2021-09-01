/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package types

import (
	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/store"
)

type BlockHeader struct {
	*common.BlockHeader
	BlockHash string `json:"block_hash,omitempty"`
}

type Block struct {
	*common.Block
	Header *BlockHeader `json:"header,omitempty"`
}

type BlockWithRWSet struct {
	*store.BlockWithRWSet
	Block *Block `json:"block,omitempty"`
}
