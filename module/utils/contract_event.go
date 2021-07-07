/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package utils

import (
	commonPb "chainmaker.org/chainmaker/pb-go/common"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
)

const TopicTableColumnDdl = `id bigint unsigned NOT NULL AUTO_INCREMENT,chain_id varchar(128),block_height bigint,tx_id varchar(64),event_index int,topic varchar(255),contract_name varchar(255),contract_version varchar(128),data1 text(65535),data2 text(65535),data3 text(65535),data4 text(65535),data5 text(65535),data6 text(65535),data7 text(65535),data8 text(65535),data9 text(65535),data10 text(65535),data11 text(65535),data12 text(65535),data13 text(65535),data14 text(65535),data15 text(65535),data16 text(65535),`
const TopicTableUniqueKey = `UNIQUE KEY unique_index(chain_id,block_height,tx_id,event_index)`
const TopicTableIndex = `INDEX index_chain_id (chain_id ASC),INDEX index_block_height (block_height ASC),INDEX index_tx_id (tx_id ASC),INDEX index_event_index (event_index ASC),INDEX index_topic (topic ASC),INDEX index_contract_name (contract_name ASC),INDEX index_contract_version (contract_version ASC)`

func GenerateSaveContractEventDdl(t *commonPb.ContractEvent, chainId string, blockHeight uint64, event_index int) string {
	var saveDdl string
	var eventDataDdl string
	var columnDdl string

	tableName := fmt.Sprintf("%s_%s_%s", chainId, t.ContractName, t.Topic)
	topicTableNameHash := sha256.Sum256([]byte(tableName))
	topicTableNameHex := fmt.Sprintf("event%s", hex.EncodeToString(topicTableNameHash[:20])[5:])
	columnDdl += fmt.Sprintf(`chain_id,block_height,topic,tx_id,event_index,contract_name,contract_version,`)

	for index, _ := range t.EventData {
		columnDdl += fmt.Sprintf("data%s,", strconv.Itoa(index+1))
	}
	eventDataDdl += fmt.Sprintf("'%s', '%d','%s','%s','%s','%s','%s',", chainId, blockHeight, t.Topic, t.TxId, strconv.Itoa(event_index), t.ContractName, t.ContractVersion)

	for _, data := range t.EventData {
		eventDataDdl += fmt.Sprintf("'%s',", data)
	}
	columnDdl = columnDdl[:len(columnDdl)-1]
	eventDataDdl = eventDataDdl[:len(eventDataDdl)-1]
	saveDdl += fmt.Sprintf("INSERT IGNORE INTO  %s (%s) VALUES (%s);", topicTableNameHex, columnDdl, eventDataDdl)

	return saveDdl
}
func GenerateSaveBlockHeightWithTopicDdl(t *commonPb.ContractEvent, chainId string, blockHeight uint64) string {
	var saveDdl string
	var DataDdl string
	var columnDdl string

	tableName := fmt.Sprintf("block_height_topic_table_index")
	topicTableNameSrc := fmt.Sprintf("%s_%s_%s", chainId, t.ContractName, t.Topic)
	topicTableNameHash := sha256.Sum256([]byte(topicTableNameSrc))
	topicTableNameHex := fmt.Sprintf("event%s", hex.EncodeToString(topicTableNameHash[:20])[5:])
	columnDdl += fmt.Sprintf("chain_id,block_height,topic_table_name_src,topic_table_name_hex")
	DataDdl += fmt.Sprintf("'%s','%d','%s','%s'", chainId, blockHeight, topicTableNameSrc, topicTableNameHex)
	saveDdl += fmt.Sprintf("INSERT IGNORE INTO  %s (%s) VALUES (%s);", tableName, columnDdl, DataDdl)
	return saveDdl
}
func GenerateUpdateBlockHeightIndexDdl(blockHeight uint64) string {
	var saveDdl string
	var DataDdl string
	var columnDdl string
	tableName := `block_height_index`
	columnDdl += `block_height`
	DataDdl += fmt.Sprintf("'%d'", blockHeight)
	saveDdl += fmt.Sprintf("UPDATE %s SET %s = %s ;", tableName, columnDdl, DataDdl)
	return saveDdl
}
func GenerateCreateTopicTableDdl(t *commonPb.ContractEvent, chainId string) string {
	var createTopicTableSql string
	tableName := fmt.Sprintf("%s_%s_%s", chainId, t.ContractName, t.Topic)
	topicTableNameHash := sha256.Sum256([]byte(tableName))
	topicTableNameHex := fmt.Sprintf("event%s", hex.EncodeToString(topicTableNameHash[:20])[5:])
	createTopicTableSql = fmt.Sprintf("CREATE TABLE IF NOT EXISTS  %s (%s PRIMARY KEY (id),%s,%s ) ENGINE=InnoDB DEFAULT CHARSET=utf8;", topicTableNameHex, TopicTableColumnDdl, TopicTableUniqueKey, TopicTableIndex)
	return createTopicTableSql
}
