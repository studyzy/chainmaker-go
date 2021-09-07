/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package blocksqldb

import (
	"strings"
	"testing"

	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"

	"chainmaker.org/chainmaker/pb-go/v2/config"
	"chainmaker.org/chainmaker/pb-go/v2/consensus"
	"chainmaker.org/chainmaker/utils/v2"
	"github.com/stretchr/testify/assert"
)

var (
	createTableMysqlSql = `CREATE TABLE block_infos (chain_id varchar(128),block_height bigint,pre_block_hash varbinary(128),
block_hash varbinary(128),
pre_conf_height bigint DEFAULT 0,
block_version int,
dag_hash varbinary(128),
rw_set_root varbinary(128),
tx_root varbinary(128),
block_timestamp bigint DEFAULT 0,
proposer_org_id varchar(128),
proposer_member_info blob,
proposer_member_type int,
proposer_sa int,
consensus_args blob,
tx_count bigint DEFAULT 0,
signature blob,
block_type int,
dag blob,
tx_ids longtext,
additional_data longblob,
PRIMARY KEY (block_height),
INDEX idx_hash (block_hash)) 
default character set utf8`
	createTableSqliteSql = `CREATE TABLE block_infos (
    chain_id text,block_height integer,pre_block_hash blob,block_hash blob,
    pre_conf_height integer DEFAULT 0,block_version integer,dag_hash blob,
    rw_set_root blob,tx_root blob,block_timestamp integer DEFAULT 0,
proposer_org_id varchar(128),
proposer_member_info blob,
proposer_member_type integer,
proposer_sa integer,
    consensus_args blob,tx_count integer DEFAULT 0,signature blob,block_type integer,dag blob,
    tx_ids longtext,additional_data longblob,PRIMARY KEY (block_height)
)`
)

func TestNewBlockInfo(t *testing.T) {
	chainConfig := &config.ChainConfig{ChainId: "chain1", Crypto: &config.CryptoConfig{Hash: "SM3"}, Consensus: &config.ConsensusConfig{Type: consensus.ConsensusType_SOLO}}
	genesis, _, _ := utils.CreateGenesis(chainConfig)
	binfo, err := NewBlockInfo(genesis)
	assert.Nil(t, err)
	t.Logf("%v", binfo)

	block := &commonPb.Block{
		Header: nil,
	}

	res, err := NewBlockInfo(block)
	assert.Nil(t, res)
	assert.Equal(t, errNullPoint, err)
}

func TestBlockInfo_GetCreateTableSql(t *testing.T) {
	defer func() {
		err := recover()
		assert.Equal(t, strings.Contains(err.(string), "Unsupported db type:test"), true)
	}()
	blockInfo := &BlockInfo{}

	mysqlSql := blockInfo.GetCreateTableSql("mysql")
	assert.Equal(t, createTableMysqlSql, mysqlSql)

	sqliteSql := blockInfo.GetCreateTableSql("sqlite")
	assert.Equal(t, createTableSqliteSql, sqliteSql)

	//dbType error should panic
	_ = blockInfo.GetCreateTableSql("test")
}

func TestBlockInfo_GetUpdateSql(t *testing.T) {
	blockInfo := &BlockInfo{
		ChainId:     "chain1",
		BlockHeight: 1,
	}

	sql, values := blockInfo.GetUpdateSql()
	assert.Equal(t, sql, "UPDATE block_infos set chain_id=? WHERE block_height=?")
	assert.Equal(t, values[0].(string), "chain1")
	assert.Equal(t, values[1].(uint64), uint64(1))
}

func TestConvertHeader2BlockInfo(t *testing.T) {
	blockInfo := ConvertHeader2BlockInfo(block1.Header)
	assert.Equal(t, blockInfo.ChainId, block1.Header.ChainId)
	assert.Equal(t, blockInfo.BlockHeight, block1.Header.BlockHeight)
}

func TestBlockInfo_GetTxList(t *testing.T) {
	blockInfo, err := NewBlockInfo(block1)
	assert.Nil(t, err)
	txList, err := blockInfo.GetTxList()
	assert.Equal(t, len(txList), 10)
}
