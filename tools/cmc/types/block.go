package types

import (
	"chainmaker.org/chainmaker/pb-go/common"
	"chainmaker.org/chainmaker/pb-go/store"
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
