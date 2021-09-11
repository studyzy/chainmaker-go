/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package resultsqldb

import (
	"chainmaker.org/chainmaker-go/store/conf"
	commonpb "chainmaker.org/chainmaker/pb-go/v2/common"
	"github.com/gogo/protobuf/proto"
)

// ResultInfo defines mysql orm model, used to create mysql table 'result_infos'
type ResultInfo struct {
	TxId        string `gorm:"size:128;primaryKey"`
	BlockHeight uint64
	TxIndex     uint32
	Rwset       []byte `gorm:"type:longblob"`
	Status      int    `gorm:"default:0"`
	Result      []byte `gorm:"type:blob"`
	Message     string `gorm:"type:longtext"`
}

func (b *ResultInfo) ScanObject(scan func(dest ...interface{}) error) error {
	return scan(&b.TxId, &b.BlockHeight, &b.TxIndex, &b.Rwset, &b.Status, &b.Result, &b.Message)
}
func (b *ResultInfo) GetCreateTableSql(dbType string) string {
	if dbType == conf.SqldbconfigSqldbtypeMysql {
		return `CREATE TABLE result_infos (
    tx_id varchar(128),block_height bigint,tx_index bigint,
    rwset longblob,status bigint DEFAULT 0,result blob,
    message longtext,PRIMARY KEY (tx_id)
    ) default character set utf8`
	} else if dbType == conf.SqldbconfigSqldbtypeSqlite {
		return `CREATE TABLE result_infos (
    tx_id text,block_height integer,tx_index integer,rwset longblob,
    status integer DEFAULT 0,result blob,message longtext,
    PRIMARY KEY (tx_id)
    )`
	}
	panic("Unsupported db type:" + dbType)
}
func (b *ResultInfo) GetTableName() string {
	return "result_infos"
}
func (b *ResultInfo) GetInsertSql() (string, []interface{}) {
	return "INSERT INTO result_infos values(?,?,?,?,?,?,?)",
		[]interface{}{b.TxId, b.BlockHeight, b.TxIndex, b.Rwset, b.Status, b.Result, b.Message}
}
func (b *ResultInfo) GetUpdateSql() (string, []interface{}) {
	return "UPDATE result_infos set block_height=?,tx_index=?,rwset=?,status=?,result=?,message=?" +
			" WHERE tx_id=?",
		[]interface{}{b.BlockHeight, b.TxIndex, b.Rwset, b.Status, b.Result, b.Message, b.TxId}
}
func (b *ResultInfo) GetCountSql() (string, []interface{}) {
	return "SELECT count(*) FROM result_infos WHERE tx_id=?", []interface{}{b.TxId}
}

// NewResultInfo construct a new HistoryInfo
func NewResultInfo(txid string, blockHeight uint64, txIndex uint32, result *commonpb.ContractResult,
	rw *commonpb.TxRWSet) *ResultInfo {
	rwBytes, _ := proto.Marshal(rw)

	return &ResultInfo{
		TxId:        txid,
		BlockHeight: blockHeight,
		TxIndex:     txIndex,
		Status:      int(result.Code),
		Result:      result.Result,
		Message:     result.Message,
		Rwset:       rwBytes,
	}
}
