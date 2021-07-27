/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package serialization

import (
	"sync"

	commonPb "chainmaker.org/chainmaker/pb-go/common"
	storePb "chainmaker.org/chainmaker/pb-go/store"

	"github.com/gogo/protobuf/proto"
)

// BlockWithSerializedInfo contains block,txs and corresponding serialized data
type BlockWithSerializedInfo struct {
	Block                    *commonPb.Block
	Meta                     *storePb.SerializedBlock //Block without Txs
	SerializedMeta           []byte
	Txs                      []*commonPb.Transaction
	SerializedTxs            [][]byte
	TxRWSets                 []*commonPb.TxRWSet
	SerializedTxRWSets       [][]byte
	ContractEvents           []*commonPb.ContractEvent
	SerializedContractEvents [][]byte
}

// SerializeBlock serialized a BlockWithRWSet and return serialized data
// which combined as a BlockWithSerializedInfo
func SerializeBlock(blockWithRWSet *storePb.BlockWithRWSet) ([]byte, *BlockWithSerializedInfo, error) {
	data, info, err := serializeBlockParallel(blockWithRWSet)
	//if err!=nil{
	//	data,info,err=serializeBlockSequence(blockWithRWSet)
	//}
	return data, info, err
}

//并行序列化
func serializeBlockParallel(blockWithRWSet *storePb.BlockWithRWSet) ([]byte, *BlockWithSerializedInfo, error) {
	buf := proto.NewBuffer(nil)
	block := blockWithRWSet.Block
	txRWSets := blockWithRWSet.TxRWSets
	events := blockWithRWSet.ContractEvents
	info := &BlockWithSerializedInfo{}
	info.Block = block
	meta := &storePb.SerializedBlock{
		Header:         block.Header,
		Dag:            block.Dag,
		TxIds:          make([]string, 0, len(block.Txs)),
		AdditionalData: block.AdditionalData,
	}
	for _, tx := range block.Txs {
		meta.TxIds = append(meta.TxIds, tx.Payload.TxId)
		info.Txs = append(info.Txs, tx)
	}

	info.TxRWSets = append(info.TxRWSets, txRWSets...)
	info.Meta = meta
	info.ContractEvents = events

	if err := info.serializeMeta(buf); err != nil {
		return nil, nil, err
	}

	if err := info.serializeTxs(buf); err != nil {
		return nil, nil, err
	}

	if err := info.serializeTxRWSets(buf); err != nil {
		return nil, nil, err
	}

	if err := info.serializeEventTopicTable(buf); err != nil {
		return nil, nil, err
	}

	return buf.Bytes(), info, nil
}

// DeserializeBlock returns a deserialized block for given serialized bytes
func DeserializeBlock(serializedBlock []byte) (*storePb.BlockWithRWSet, error) {
	b, err := deserializeBlockParallel(serializedBlock)
	if err != nil {
		b, err = deserializeBlockSequence(serializedBlock)
	}
	return b, err
}

//并行反序列化
func deserializeBlockParallel(serializedBlock []byte) (*storePb.BlockWithRWSet, error) {

	info := &BlockWithSerializedInfo{}
	buf := proto.NewBuffer(serializedBlock)
	var err error
	if info.Meta, err = info.deserializeMeta(buf); err != nil {
		return nil, err
	}
	if info.Txs, err = info.deserializeTxs(buf); err != nil {
		return nil, err
	}
	if info.TxRWSets, err = info.deserializeRWSets(buf); err != nil {
		return nil, err
	}
	if info.ContractEvents, err = info.deserializeEventTopicTable(buf); err != nil {
		return nil, err
	}
	block := &commonPb.Block{
		Header:         info.Meta.Header,
		Dag:            info.Meta.Dag,
		Txs:            info.Txs,
		AdditionalData: info.Meta.AdditionalData,
	}
	blockWithRWSet := &storePb.BlockWithRWSet{
		Block:          block,
		TxRWSets:       info.TxRWSets,
		ContractEvents: info.ContractEvents,
	}
	return blockWithRWSet, nil
}
func deserializeBlockSequence(serializedBlock []byte) (*storePb.BlockWithRWSet, error) {
	blockWithRWSet := &storePb.BlockWithRWSet{}
	err := blockWithRWSet.Unmarshal(serializedBlock)
	if err != nil {
		return nil, err
	}
	//info := &BlockWithSerializedInfo{}
	//info.Block = blockWithRWSet.Block
	//info.TxRWSets = blockWithRWSet.TxRWSets
	return blockWithRWSet, nil
}

func (b *BlockWithSerializedInfo) serializeMeta(buf *proto.Buffer) error {
	metaBytes, err := proto.Marshal(b.Meta)
	if err != nil {
		return err
	}
	if err := buf.EncodeRawBytes(metaBytes); err != nil {
		return err
	}
	b.SerializedMeta = metaBytes
	return nil
}
func (b *BlockWithSerializedInfo) serializeEventTopicTable(buf *proto.Buffer) error {
	if err := buf.EncodeVarint(uint64(len(b.ContractEvents))); err != nil {
		return err
	}
	serializedEventList := make([][]byte, len(b.ContractEvents))
	batchSize := 1000
	taskNum := len(b.ContractEvents)/batchSize + 1
	errsChan := make(chan error, taskNum)
	wg := sync.WaitGroup{}
	wg.Add(taskNum)
	for taskId := 0; taskId < taskNum; taskId++ {
		startIndex := taskId * batchSize
		endIndex := (taskId + 1) * batchSize
		if endIndex > len(b.ContractEvents) {
			endIndex = len(b.ContractEvents)
		}
		go func(start int, end int) {
			defer wg.Done()
			for offset, et := range b.ContractEvents[start:end] {
				txBytes, err := proto.Marshal(et)
				if err != nil {
					errsChan <- err
				}
				serializedEventList[start+offset] = txBytes
			}
		}(startIndex, endIndex)
	}
	wg.Wait()
	if len(errsChan) > 0 {
		return <-errsChan
	}
	for _, txBytes := range serializedEventList {
		b.SerializedContractEvents = append(b.SerializedContractEvents, txBytes)
		if err := buf.EncodeRawBytes(txBytes); err != nil {
			return err
		}
	}
	return nil
}

func (b *BlockWithSerializedInfo) serializeTxs(buf *proto.Buffer) error {
	if err := buf.EncodeVarint(uint64(len(b.Txs))); err != nil {
		return err
	}

	serializedTxList := make([][]byte, len(b.Txs))
	batchSize := 1000
	taskNum := len(b.Txs)/batchSize + 1
	errsChan := make(chan error, taskNum)
	wg := sync.WaitGroup{}
	wg.Add(taskNum)
	for taskId := 0; taskId < taskNum; taskId++ {
		startIndex := taskId * batchSize
		endIndex := (taskId + 1) * batchSize
		if endIndex > len(b.Txs) {
			endIndex = len(b.Txs)
		}
		go func(start int, end int) {
			defer wg.Done()
			for offset, tx := range b.Txs[start:end] {
				txBytes, err := proto.Marshal(tx)
				if err != nil {
					errsChan <- err
				}
				serializedTxList[start+offset] = txBytes
			}
		}(startIndex, endIndex)
	}
	wg.Wait()
	if len(errsChan) > 0 {
		return <-errsChan
	}
	for _, txBytes := range serializedTxList {
		b.SerializedTxs = append(b.SerializedTxs, txBytes)
		if err := buf.EncodeRawBytes(txBytes); err != nil {
			return err
		}
	}
	return nil
}

func (b *BlockWithSerializedInfo) serializeTxRWSets(buf *proto.Buffer) error {
	if err := buf.EncodeVarint(uint64(len(b.TxRWSets))); err != nil {
		return err
	}

	serializedTxRWSets := make([][]byte, len(b.TxRWSets))
	batchSize := 1000
	taskNum := len(b.TxRWSets)/batchSize + 1
	errsChan := make(chan error, taskNum)
	wg := sync.WaitGroup{}
	wg.Add(taskNum)
	for taskId := 0; taskId < taskNum; taskId++ {
		startIndex := taskId * batchSize
		endIndex := (taskId + 1) * batchSize
		if endIndex > len(b.TxRWSets) {
			endIndex = len(b.TxRWSets)
		}
		go func(start int, end int) {
			defer wg.Done()
			for offset, txRWSet := range b.TxRWSets[start:end] {
				txRWSetBytes, err := proto.Marshal(txRWSet)
				if err != nil {
					errsChan <- err
				}
				serializedTxRWSets[start+offset] = txRWSetBytes
			}
		}(startIndex, endIndex)
	}
	wg.Wait()
	if len(errsChan) > 0 {
		return <-errsChan
	}
	for _, rwSetBytes := range serializedTxRWSets {
		b.SerializedTxRWSets = append(b.SerializedTxRWSets, rwSetBytes)
		if err := buf.EncodeRawBytes(rwSetBytes); err != nil {
			return err
		}
	}
	return nil
}

func (b *BlockWithSerializedInfo) deserializeMeta(buf *proto.Buffer) (*storePb.SerializedBlock, error) {
	meta := &storePb.SerializedBlock{}
	serializedMeta, err := buf.DecodeRawBytes(false)
	if err != nil {
		return nil, err
	}
	err = proto.Unmarshal(serializedMeta, meta)
	if err != nil {
		return nil, err
	}
	return meta, nil
}

func (b *BlockWithSerializedInfo) deserializeTxs(buf *proto.Buffer) ([]*commonPb.Transaction, error) {
	var txs []*commonPb.Transaction
	txNum, err := buf.DecodeVarint()
	if err != nil {
		return nil, err
	}
	for i := uint64(0); i < txNum; i++ {
		txBytes, err := buf.DecodeRawBytes(false)
		if err != nil {
			return nil, err
		}
		tx := &commonPb.Transaction{}
		if err = proto.Unmarshal(txBytes, tx); err != nil {
			return nil, err
		}
		txs = append(txs, tx)
	}
	return txs, nil
}

func (b *BlockWithSerializedInfo) deserializeRWSets(buf *proto.Buffer) ([]*commonPb.TxRWSet, error) {
	var txRWSets []*commonPb.TxRWSet
	num, err := buf.DecodeVarint()
	if err != nil {
		return nil, err
	}
	for i := uint64(0); i < num; i++ {
		rwSetBytes, err := buf.DecodeRawBytes(false)
		if err != nil {
			return nil, err
		}
		rwSet := &commonPb.TxRWSet{}
		if err = proto.Unmarshal(rwSetBytes, rwSet); err != nil {
			return nil, err
		}
		txRWSets = append(txRWSets, rwSet)
	}
	return txRWSets, nil
}
func (b *BlockWithSerializedInfo) deserializeEventTopicTable(buf *proto.Buffer) ([]*commonPb.ContractEvent, error) {
	var ets []*commonPb.ContractEvent
	txNum, err := buf.DecodeVarint()
	if err != nil {
		return nil, err
	}
	for i := uint64(0); i < txNum; i++ {
		txBytes, err := buf.DecodeRawBytes(false)
		if err != nil {
			return nil, err
		}
		et := &commonPb.ContractEvent{}
		if err = proto.Unmarshal(txBytes, et); err != nil {
			return nil, err
		}
		ets = append(ets, et)
	}
	return ets, nil
}
