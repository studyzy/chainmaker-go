/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package blocksqldb

import (
	"chainmaker.org/chainmaker-go/localconf"
	acPb "chainmaker.org/chainmaker/pb-go/accesscontrol"
	commonPb "chainmaker.org/chainmaker/pb-go/common"
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

func (t *TxInfo) ScanObject(scan func(dest ...interface{}) error) error {
	return scan(&t.TxId, &t.ChainId, &t.Sender, &t.TxType, &t.BlockHeight, &t.BlockHash, &t.Offset, &t.Timestamp,
		&t.ExpirationTime, &t.RequestPayload, &t.RequestSignature, &t.Code, &t.ContractResult, &t.RwSetHash)
}
func (t *TxInfo) GetCreateTableSql(dbType string) string {
	if dbType == localconf.SqlDbConfig_SqlDbType_MySQL {
		return `CREATE TABLE tx_infos (tx_id varchar(128),chain_id varchar(128),sender blob,tx_type int,
block_height bigint unsigned,block_hash varbinary(128),offset int unsigned,timestamp bigint DEFAULT 0,
expiration_time bigint DEFAULT 0,request_payload longblob,request_signature blob,code int,
contract_result longblob,rw_set_hash varbinary(128),PRIMARY KEY (tx_id),
INDEX idx_height_offset (block_height,offset)) default character set utf8`
	}
	if dbType == localconf.SqlDbConfig_SqlDbType_Sqlite {
		return `CREATE TABLE tx_infos (
tx_id text,chain_id text,sender blob,tx_type integer,block_height integer,block_hash blob,offset integer,
timestamp integer DEFAULT 0,expiration_time integer DEFAULT 0,request_payload longblob,request_signature blob,
code integer,contract_result longblob,rw_set_hash blob,
PRIMARY KEY (tx_id)
)`
	}
	panic("Unsupported db type:" + dbType)
}
func (t *TxInfo) GetTableName() string {
	return "tx_infos"
}
func (t *TxInfo) GetInsertSql() (string, []interface{}) {
	return "INSERT INTO tx_infos values(?,?,?,?,?,?,?,?,?,?,?,?,?,?)", []interface{}{t.TxId, t.ChainId, t.Sender, t.TxType,
		t.BlockHeight, t.BlockHash, t.Offset, t.Timestamp, t.ExpirationTime, t.RequestPayload, t.RequestSignature,
		t.Code, t.ContractResult, t.RwSetHash}
}
func (t *TxInfo) GetUpdateSql() (string, []interface{}) {
	return "UPDATE tx_infos SET chain_id=? WHERE tx_id=?", []interface{}{t.ChainId, t.TxId}
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
