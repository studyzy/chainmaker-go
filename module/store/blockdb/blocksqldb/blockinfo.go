/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package blocksqldb

import (
	"chainmaker.org/chainmaker/common/json"
	"chainmaker.org/chainmaker-go/localconf"
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	storePb "chainmaker.org/chainmaker-go/pb/protogo/store"
	"github.com/gogo/protobuf/proto"
)

// BlockInfo defines mysql orm model, used to create mysql table 'block_infos'
type BlockInfo struct {
	ChainId        string `gorm:"size:128"`
	BlockHeight    int64  `gorm:"primaryKey;autoIncrement:false"`
	PreBlockHash   []byte `gorm:"size:128"`
	BlockHash      []byte `gorm:"size:128;index:idx_hash"`
	PreConfHeight  int64  `gorm:"default:0"`
	BlockVersion   []byte `gorm:"size:128"`
	DagHash        []byte `gorm:"size:128"`
	RwSetRoot      []byte `gorm:"size:128"`
	TxRoot         []byte `gorm:"size:128"`
	BlockTimestamp int64  `gorm:"default:0"`
	Proposer       []byte `gorm:"type:blob;size:65535"`
	ConsensusArgs  []byte `gorm:"type:blob"`
	TxCount        int64  `gorm:"default:0"`
	Signature      []byte `gorm:"type:blob;size:65535"`
	Dag            []byte `gorm:"type:blob"`
	TxIds          string `gorm:"type:longtext"`
	AdditionalData []byte `gorm:"type:longblob"`
}

func (b *BlockInfo) ScanObject(scan func(dest ...interface{}) error) error {
	return scan(&b.ChainId, &b.BlockHeight, &b.PreBlockHash, &b.BlockHash, &b.PreConfHeight, &b.BlockVersion,
		&b.DagHash, &b.RwSetRoot, &b.TxRoot, &b.BlockTimestamp, &b.Proposer, &b.ConsensusArgs, &b.TxCount,
		&b.Signature, &b.Dag, &b.TxIds, &b.AdditionalData)
}
func (b *BlockInfo) GetCreateTableSql(dbType string) string {
	if dbType == localconf.SqlDbConfig_SqlDbType_MySQL {
		return `CREATE TABLE block_infos (chain_id varchar(128),block_height bigint,pre_block_hash varbinary(128),
block_hash varbinary(128),
pre_conf_height bigint DEFAULT 0,
block_version varbinary(128),
dag_hash varbinary(128),
rw_set_root varbinary(128),
tx_root varbinary(128),
block_timestamp bigint DEFAULT 0,
proposer blob,
consensus_args blob,
tx_count bigint DEFAULT 0,
signature blob,dag blob,tx_ids longtext,
additional_data longblob,
PRIMARY KEY (block_height),
INDEX idx_hash (block_hash)) 
default character set utf8`
	} else if dbType == localconf.SqlDbConfig_SqlDbType_Sqlite {
		return `CREATE TABLE block_infos (
    chain_id text,block_height integer,pre_block_hash blob,block_hash blob,
    pre_conf_height integer DEFAULT 0,block_version blob,dag_hash blob,
    rw_set_root blob,tx_root blob,block_timestamp integer DEFAULT 0,proposer blob,
    consensus_args blob,tx_count integer DEFAULT 0,signature blob,dag blob,
    tx_ids longtext,additional_data longblob,PRIMARY KEY (block_height)
)`
	}
	panic("Unsupported db type:" + dbType)
}
func (b *BlockInfo) GetTableName() string {
	return "block_infos"
}
func (b *BlockInfo) GetInsertSql() (string, []interface{}) {
	return "INSERT INTO block_infos values(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)",
		[]interface{}{b.ChainId, b.BlockHeight, b.PreBlockHash, b.BlockHash, b.PreConfHeight, b.BlockVersion,
			b.DagHash, b.RwSetRoot, b.TxRoot, b.BlockTimestamp, b.Proposer, b.ConsensusArgs, b.TxCount,
			b.Signature, b.Dag, b.TxIds, b.AdditionalData}
}
func (b *BlockInfo) GetUpdateSql() (string, []interface{}) {
	return "UPDATE block_infos set chain_id=?" +
		" WHERE block_height=?", []interface{}{b.ChainId, b.BlockHeight}
}
func NewBlockInfo(block *commonPb.Block) (*BlockInfo, error) {
	blockInfo := &BlockInfo{
		ChainId:        block.Header.ChainId,
		BlockHeight:    block.Header.BlockHeight,
		PreBlockHash:   block.Header.PreBlockHash,
		BlockHash:      block.Header.BlockHash,
		PreConfHeight:  block.Header.PreConfHeight,
		BlockVersion:   block.Header.BlockVersion,
		DagHash:        block.Header.DagHash,
		RwSetRoot:      block.Header.RwSetRoot,
		TxRoot:         block.Header.TxRoot,
		BlockTimestamp: block.Header.BlockTimestamp,
		Proposer:       block.Header.Proposer,
		ConsensusArgs:  block.Header.ConsensusArgs,
		TxCount:        block.Header.TxCount,
		Signature:      block.Header.Signature,
	}
	if block.Dag != nil {
		dagBytes, err := proto.Marshal(block.Dag)
		if err != nil {
			return nil, err
		}
		blockInfo.Dag = dagBytes
	}
	if block.AdditionalData != nil {
		additionalDataBytes, err := proto.Marshal(block.AdditionalData)
		if err != nil {
			return nil, err
		}
		blockInfo.AdditionalData = additionalDataBytes
	}

	var txList []string
	for _, tx := range block.Txs {
		txList = append(txList, tx.Header.TxId)
	}
	txListBytes, err := json.Marshal(txList)
	if err != nil {
		return nil, err
	}
	blockInfo.TxIds = string(txListBytes)

	return blockInfo, nil
}

func ConvertHeader2BlockInfo(header *commonPb.BlockHeader) *BlockInfo {
	blockInfo := &BlockInfo{
		ChainId:        header.ChainId,
		BlockHeight:    header.BlockHeight,
		PreBlockHash:   header.PreBlockHash,
		BlockHash:      header.BlockHash,
		PreConfHeight:  header.PreConfHeight,
		BlockVersion:   header.BlockVersion,
		DagHash:        header.DagHash,
		RwSetRoot:      header.RwSetRoot,
		TxRoot:         header.TxRoot,
		BlockTimestamp: header.BlockTimestamp,
		Proposer:       header.Proposer,
		ConsensusArgs:  header.ConsensusArgs,
		TxCount:        header.TxCount,
		Signature:      header.Signature,
	}

	return blockInfo
}

// GetTxList returns the txId list , or return nil if an error occurred
func (b *BlockInfo) GetTxList() ([]string, error) {
	var txList []string
	err := json.Unmarshal([]byte(b.TxIds), &txList)
	if err != nil {
		return nil, err
	}
	return txList, nil
}

// GetBlock transfer the BlockInfo to commonPb.Block
func (b *BlockInfo) GetBlock() (*commonPb.Block, error) {
	block := &commonPb.Block{
		Header: &commonPb.BlockHeader{
			ChainId:        b.ChainId,
			BlockHeight:    b.BlockHeight,
			PreBlockHash:   b.PreBlockHash,
			BlockHash:      b.BlockHash,
			PreConfHeight:  b.PreConfHeight,
			BlockVersion:   b.BlockVersion,
			DagHash:        b.DagHash,
			RwSetRoot:      b.RwSetRoot,
			TxRoot:         b.TxRoot,
			BlockTimestamp: b.BlockTimestamp,
			Proposer:       b.Proposer,
			ConsensusArgs:  b.ConsensusArgs,
			TxCount:        b.TxCount,
			Signature:      b.Signature,
		},
	}
	if b.Dag != nil {
		var dag commonPb.DAG
		err := proto.Unmarshal(b.Dag, &dag)
		if err != nil {
			return nil, err
		}
		block.Dag = &dag
	}

	if b.AdditionalData != nil {
		var additionalData commonPb.AdditionalData
		err := proto.Unmarshal(b.AdditionalData, &additionalData)
		if err != nil {
			return nil, err
		}
		block.AdditionalData = &additionalData
	}

	return block, nil
}

// GetFilteredBlock returns a filtered block given it's block height, or return nil if none exists.
func (b *BlockInfo) GetFilteredBlock() (*storePb.SerializedBlock, error) {
	block, err := b.GetBlock()
	if err != nil {
		return nil, err
	}
	var txList []string
	err = json.Unmarshal([]byte(b.TxIds), &txList)
	if err != nil {
		return nil, err
	}
	filteredBlock := &storePb.SerializedBlock{
		Header:         block.Header,
		Dag:            block.Dag,
		TxIds:          txList,
		AdditionalData: block.AdditionalData,
	}

	return filteredBlock, nil
}
