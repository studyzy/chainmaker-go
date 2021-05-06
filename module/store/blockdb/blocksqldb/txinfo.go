/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package blocksqldb

import (
	acPb "chainmaker.org/chainmaker-go/pb/protogo/accesscontrol"
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"github.com/gogo/protobuf/proto"
)

// TxInfo defines mysql orm model, used to create mysql table 'tx_infos'
type TxInfo struct {
	TxId             string `gorm:"primaryKey;size:128"`
	ChainId          string `gorm:"size:128"`
	Sender           []byte `gorm:"type:blob;size:65535"`
	TxType           int32
	BlockHeight      uint64 `gorm:"index:idx_height_offset"`
	BlockHash        []byte `gorm:"size:128"`
	Offset           uint32 `gorm:"index:idx_height_offset"`
	Timestamp        int64  `gorm:"default:0"`
	ExpirationTime   int64  `gorm:"default:0"`
	RequestPayload   []byte `gorm:"type:longblob"`
	RequestSignature []byte `gorm:"type:blob;size:65535"`
	Code             int32
	ContractResult   []byte `gorm:"type:longblob"`
	RwSetHash        []byte `gorm:"size:128"`
}

// NewTxInfo construct new `TxInfo`
func NewTxInfo(tx *commonPb.Transaction, blockHeight uint64, blockHash []byte, offset uint32) (*TxInfo, error) {
	txInfo := &TxInfo{
		ChainId:          tx.Header.ChainId,
		TxId:             tx.Header.TxId,
		TxType:           int32(tx.Header.TxType),
		BlockHeight:      blockHeight,
		BlockHash:        blockHash,
		Offset:           offset,
		Timestamp:        tx.Header.Timestamp,
		ExpirationTime:   tx.Header.ExpirationTime,
		RequestPayload:   tx.RequestPayload,
		RequestSignature: tx.RequestSignature,
		Code:             int32(tx.Result.Code),
		RwSetHash:        tx.Result.RwSetHash,
	}
	if tx.Header.Sender != nil {
		senderBytes, err := proto.Marshal(tx.Header.Sender)
		if err != nil {
			return nil, err
		}
		txInfo.Sender = senderBytes
	}

	if tx.Result != nil && tx.Result.ContractResult != nil {
		contractResultBytes, err := proto.Marshal(tx.Result.ContractResult)
		if err != nil {
			return nil, err
		}
		txInfo.ContractResult = contractResultBytes
	}

	return txInfo, nil
}

// GetTx transfer TxInfo to commonPb.Transaction
func (t *TxInfo) GetTx() (*commonPb.Transaction, error) {
	tx := &commonPb.Transaction{
		Header: &commonPb.TxHeader{
			ChainId:        t.ChainId,
			TxType:         commonPb.TxType(t.TxType),
			TxId:           t.TxId,
			Timestamp:      t.Timestamp,
			ExpirationTime: t.ExpirationTime,
		},
		RequestPayload:   t.RequestPayload,
		RequestSignature: t.RequestSignature,
		Result: &commonPb.Result{
			Code:      commonPb.TxStatusCode(t.Code),
			RwSetHash: t.RwSetHash,
		},
	}
	var sender acPb.SerializedMember
	err := proto.Unmarshal(t.Sender, &sender)
	if err != nil {
		return nil, err
	}
	tx.Header.Sender = &sender

	if t.ContractResult != nil {
		var contractResult commonPb.ContractResult
		err = proto.Unmarshal(t.ContractResult, &contractResult)
		if err != nil {
			return nil, err
		}
		tx.Result.ContractResult = &contractResult
	}
	return tx, nil
}
func (t *TxInfo) GetTxInfo() (*commonPb.TransactionInfo, error) {
	txInfo := &commonPb.TransactionInfo{
		BlockHeight: t.BlockHeight,
		BlockHash:   t.BlockHash,
		TxIndex:     t.Offset,
	}
	var err error
	txInfo.Transaction, err = t.GetTx()
	return txInfo, err
}
