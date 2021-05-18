/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package serialization

import (
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	storePb "chainmaker.org/chainmaker-go/pb/protogo/store"
)

// BlockWithSerializedInfo contains block,txs and corresponding serialized data
type BlockWithSerializedInfo struct {
	Block              *commonPb.Block
	TxRWSets           []*commonPb.TxRWSet
	meta               *storePb.SerializedBlock //Block without Txs
	serializedMeta     []byte
	serializedTxs      [][]byte
	serializedTxRWSets [][]byte
}

// SerializeBlock serialized a BlockWithRWSet and return serialized data
// which combined as a BlockWithSerializedInfo
func SerializeBlock(blockWithRWSet *storePb.BlockWithRWSet) ([]byte, *BlockWithSerializedInfo, error) {
	info := &BlockWithSerializedInfo{}
	info.Block = blockWithRWSet.Block
	info.TxRWSets = blockWithRWSet.TxRWSets
	data, err := blockWithRWSet.Marshal()
	return data, info, err
}

// DeserializeBlock returns a deserialized block for given serialized bytes
func DeserializeBlock(serializedBlock []byte) (*BlockWithSerializedInfo, error) {
	blockWithRWSet := &storePb.BlockWithRWSet{}
	err := blockWithRWSet.Unmarshal(serializedBlock)
	if err != nil {
		return nil, err
	}
	info := &BlockWithSerializedInfo{}
	info.Block = blockWithRWSet.Block
	info.TxRWSets = blockWithRWSet.TxRWSets
	return info, nil
}
func (b *BlockWithSerializedInfo) GetSerializedBlock() *storePb.SerializedBlock {
	if b.meta != nil {
		return b.meta
	}
	block := b.Block
	meta := &storePb.SerializedBlock{
		Header:         block.Header,
		Dag:            block.Dag,
		TxIds:          make([]string, 0, len(block.Txs)),
		AdditionalData: block.AdditionalData,
	}
	for _, tx := range block.Txs {
		meta.TxIds = append(meta.TxIds, tx.Header.TxId)
	}
	b.meta = meta
	return meta
}
func (b *BlockWithSerializedInfo) GetSerializedMeta() []byte {
	if len(b.serializedMeta) > 0 {
		return b.serializedMeta
	}
	b.meta = b.GetSerializedBlock()
	b.serializedMeta, _ = b.meta.Marshal()
	return b.serializedMeta
}
func (b *BlockWithSerializedInfo) GetSerializedTxs() [][]byte {
	if len(b.serializedTxs) > 0 {
		return b.serializedTxs
	}
	b.serializedTxs = [][]byte{}
	for _, tx := range b.Block.Txs {
		txData, _ := tx.Marshal()
		b.serializedTxs = append(b.serializedTxs, txData)
	}
	return b.serializedTxs
}
func (b *BlockWithSerializedInfo) GetSerializedTxRWSets() [][]byte {
	if len(b.serializedTxRWSets) > 0 {
		return b.serializedTxRWSets
	}
	b.serializedTxRWSets = [][]byte{}
	for _, rwset := range b.TxRWSets {
		txData, _ := rwset.Marshal()
		b.serializedTxRWSets = append(b.serializedTxRWSets, txData)
	}
	return b.serializedTxRWSets
}
