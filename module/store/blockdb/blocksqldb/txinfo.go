/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package blocksqldb

import (
	"encoding/json"

	"chainmaker.org/chainmaker-go/localconf"
	acPb "chainmaker.org/chainmaker/pb-go/accesscontrol"
	commonPb "chainmaker.org/chainmaker/pb-go/common"
)

// TxInfo defines mysql orm model, used to create mysql table 'tx_infos'
type TxInfo struct {
	ChainId        string `gorm:"size:128"`
	TxType         int32  `gorm:"default:0"`
	TxId           string `gorm:"primaryKey;size:128"`
	Timestamp      int64  `gorm:"default:0"`
	ExpirationTime int64  `gorm:"default:0"`

	ContractName string `gorm:"size:128"`
	// invoke method
	Method string `gorm:"size:128"`
	// invoke parameters in k-v format
	Parameters []byte `gorm:"type:longblob"` //json
	//sequence number 交易顺序号，以Sender为主键，0表示无顺序要求，>0则必须连续递增。
	Sequence uint64 `gorm:"default:0"`
	// gas price+gas limit; fee; timeout seconds;
	Limit []byte `gorm:"type:blob;size:65535"`

	SenderOrgId      string `gorm:"size:128"`
	SenderMemberInfo []byte `gorm:"type:blob;size:65535"`
	SenderMemberType int    `gorm:"default:0"`
	SenderSA         uint32 `gorm:"default:0"`
	SenderSignature  []byte `gorm:"type:blob;size:65535"`
	Endorsers        string `gorm:"type:longtext"` //json

	TxStatusCode       int32
	ContractResultCode uint32
	ResultData         []byte `gorm:"type:longblob"`
	ResultMessage      string `gorm:"size:2000"`
	GasUsed            uint64
	ContractEvents     string `gorm:"type:longtext"` //json
	RwSetHash          []byte `gorm:"size:128"`
	Message            string `gorm:"size:2000"`

	BlockHeight uint64 `gorm:"index:idx_height_offset"`
	BlockHash   []byte `gorm:"size:128"`
	Offset      uint32 `gorm:"index:idx_height_offset"`
}

func (t *TxInfo) ScanObject(scan func(dest ...interface{}) error) error {
	return scan(&t.ChainId, &t.TxType, &t.TxId, &t.Timestamp, &t.ExpirationTime,
		&t.ContractName, &t.Method, &t.Parameters, &t.Sequence, &t.Limit,
		&t.SenderOrgId, &t.SenderMemberInfo, &t.SenderMemberType, &t.SenderSA, &t.SenderSignature, &t.Endorsers,
		&t.TxStatusCode, &t.ContractResultCode, &t.ResultData, &t.ResultMessage, &t.GasUsed,
		&t.ContractEvents, &t.RwSetHash, &t.Message,
		&t.BlockHeight, &t.BlockHash, &t.Offset)
}
func (t *TxInfo) GetCreateTableSql(dbType string) string {
	if dbType == localconf.SqlDbConfig_SqlDbType_MySQL {
		return `CREATE TABLE tx_infos (chain_id varchar(128),tx_type int,tx_id varchar(128),timestamp bigint DEFAULT 0,
expiration_time bigint DEFAULT 0,
contract_name varchar(128),method varchar(128),parameters longblob,sequence bigint,limits blob,
sender_org_id varchar(128),sender_member_info blob,sender_member_type int,sender_sa int,sender_signature blob,
endorsers longtext,
tx_status_code int,contract_result_code int,result_data longblob,result_message varchar(2000),
gas_used bigint,contract_events longtext,rw_set_hash varbinary(128),message varchar(2000),
block_height bigint unsigned,block_hash varbinary(128),offset int unsigned,
PRIMARY KEY (tx_id),
INDEX idx_height_offset (block_height,offset)) default character set utf8`
	}
	if dbType == localconf.SqlDbConfig_SqlDbType_Sqlite {
		return `CREATE TABLE tx_infos (chain_id text,tx_type int,tx_id text,timestamp integer DEFAULT 0,
expiration_time integer DEFAULT 0,
contract_name text,method text,parameters longblob,sequence bigint,limits blob,
sender_org_id text,sender_member_info blob,sender_member_type integer,sender_sa integer,sender_signature blob,
endorsers longtext,
tx_status_code integer,contract_result_code integer,result_data longblob,result_message text,
gas_used integer,contract_events longtext,rw_set_hash blob,message text,
block_height integer ,block_hash blob,offset integer,
PRIMARY KEY (tx_id))`
	}
	panic("Unsupported db type:" + dbType)
}
func (t *TxInfo) GetTableName() string {
	return "tx_infos"
}
func (t *TxInfo) GetInsertSql() (string, []interface{}) {
	return "INSERT INTO tx_infos values(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)", []interface{}{
		t.ChainId, t.TxType, t.TxId, t.Timestamp, t.ExpirationTime,
		t.ContractName, t.Method, t.Parameters, t.Sequence, t.Limit,
		t.SenderOrgId, t.SenderMemberInfo, t.SenderMemberType, t.SenderSA, t.SenderSignature, t.Endorsers,
		t.TxStatusCode, t.ContractResultCode, t.ResultData, t.ResultMessage, t.GasUsed,
		t.ContractEvents, t.RwSetHash, t.Message, t.BlockHeight, t.BlockHash, t.Offset}
}
func (t *TxInfo) GetUpdateSql() (string, []interface{}) {
	return "UPDATE tx_infos SET chain_id=? WHERE tx_id=?", []interface{}{t.ChainId, t.TxId}
}
func (b *TxInfo) GetCountSql() (string, []interface{}) {
	return "SELECT count(*) FROM tx_infos WHERE tx_id=?", []interface{}{b.TxId}
}

// NewTxInfo construct new `TxInfo`
func NewTxInfo(tx *commonPb.Transaction, blockHeight uint64, blockHash []byte, offset uint32) (*TxInfo, error) {
	par, _ := json.Marshal(tx.Payload.Parameters)
	var endorsers []byte
	if len(tx.Endorsers) > 0 {
		endorsers, _ = json.Marshal(tx.Endorsers)
	}
	var events []byte
	if len(tx.Result.ContractResult.ContractEvent) > 0 {
		events, _ = json.Marshal(tx.Result.ContractResult.ContractEvent)
	}
	txInfo := &TxInfo{
		ChainId:            tx.Payload.ChainId,
		TxType:             int32(tx.Payload.TxType),
		TxId:               tx.Payload.TxId,
		Timestamp:          tx.Payload.Timestamp,
		ExpirationTime:     tx.Payload.ExpirationTime,
		ContractName:       tx.Payload.ContractName,
		Method:             tx.Payload.Method,
		Parameters:         par,
		Sequence:           tx.Payload.Sequence,
		Limit:              tx.Payload.Limit,
		SenderOrgId:        getSender(tx).Signer.OrgId,
		SenderMemberInfo:   getSender(tx).Signer.MemberInfo,
		SenderMemberType:   int(getSender(tx).Signer.MemberType),
		SenderSignature:    getSender(tx).Signature,
		Endorsers:          string(endorsers),
		TxStatusCode:       int32(tx.Result.Code),
		ContractResultCode: tx.Result.ContractResult.Code,
		ResultData:         tx.Result.ContractResult.Result,
		ResultMessage:      tx.Result.ContractResult.Message,
		GasUsed:            tx.Result.ContractResult.GasUsed,
		ContractEvents:     string(events),
		RwSetHash:          tx.Result.RwSetHash,
		Message:            tx.Result.Message,

		BlockHeight: blockHeight,
		BlockHash:   blockHash,
		Offset:      offset,
	}

	return txInfo, nil
}
func getSender(tx *commonPb.Transaction) *commonPb.EndorsementEntry {
	if tx.Sender == nil {
		return &commonPb.EndorsementEntry{
			Signer: &acPb.Member{
				OrgId:      "",
				MemberType: 0,
				MemberInfo: nil,
			},
			Signature: nil,
		}
	}
	return tx.Sender
}
func getParameters(par []byte) []*commonPb.KeyValuePair {
	var pairs []*commonPb.KeyValuePair
	json.Unmarshal(par, &pairs)
	return pairs
}
func getEndorsers(endorsers string) []*commonPb.EndorsementEntry {
	var pairs []*commonPb.EndorsementEntry
	json.Unmarshal([]byte(endorsers), &pairs)
	return pairs
}
func getContractEvents(events string) []*commonPb.ContractEvent {
	var pairs []*commonPb.ContractEvent
	json.Unmarshal([]byte(events), &pairs)
	return pairs
}

// GetTx transfer TxInfo to commonPb.Transaction
func (t *TxInfo) GetTx() (*commonPb.Transaction, error) {
	tx := &commonPb.Transaction{
		Payload: &commonPb.Payload{
			ChainId:        t.ChainId,
			TxType:         commonPb.TxType(t.TxType),
			TxId:           t.TxId,
			Timestamp:      t.Timestamp,
			ExpirationTime: t.ExpirationTime,
			ContractName:   t.ContractName,
			Method:         t.Method,
			Parameters:     getParameters(t.Parameters),
			Sequence:       t.Sequence,
			Limit:          t.Limit,
		},
		Sender: &commonPb.EndorsementEntry{
			Signer: &acPb.Member{
				OrgId:      t.SenderOrgId,
				MemberInfo: t.SenderMemberInfo,
				MemberType: acPb.MemberType(t.SenderMemberType),
				//SignatureAlgorithm: t.SenderSA,
			},
			Signature: t.SenderSignature,
		},
		Endorsers: getEndorsers(t.Endorsers),
		Result: &commonPb.Result{
			Code:      commonPb.TxStatusCode(t.TxStatusCode),
			RwSetHash: t.RwSetHash,
			Message:   t.Message,
			ContractResult: &commonPb.ContractResult{
				Code:          t.ContractResultCode,
				Result:        t.ResultData,
				Message:       t.ResultMessage,
				GasUsed:       t.GasUsed,
				ContractEvent: getContractEvents(t.ContractEvents),
			},
		},
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
