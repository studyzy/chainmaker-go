package utils

import (
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
)

const TopicTableColumnDdl = `id bigint unsigned NOT NULL AUTO_INCREMENT,chain_id varchar(128),block_height bigint,tx_id varchar(64),topic varchar(255),contract_name varchar(1000),contract_version varchar(128),data1 text(65535),data2 text(65535),data3 text(65535),data4 text(65535),data5 text(65535),data6 text(65535),data7 text(65535),data8 text(65535),data9 text(65535),data10 text(65535),data11 text(65535),data12 text(65535),data13 text(65535),data14 text(65535),data15 text(65535),data16 text(65535),`

func GenerateSaveContractEventDdl(t *commonPb.ContractEvent) string {
	var saveDdl string
	var eventDataDdl string
	var columnDdl string
	tableName := t.ChainId + "_" + t.ContractName + "_" + t.Topic
	topicTableNameHash := sha256.Sum256([]byte(tableName))
	topicTableNameHex := "event" + hex.EncodeToString(topicTableNameHash[:32])[5:]

	columnDdl += `chain_id,block_height,topic,tx_id,contract_name,contract_version,`
	for index, _ := range t.EventData {
		columnDdl += "data" + strconv.Itoa(index+1) + ","
	}

	eventDataDdl += `'` + t.ChainId + `',` + `'` + strconv.FormatInt(t.BlockHeight, 10) + `',` +
		`'` + t.Topic + `',`+ `'` + t.TxId + `',`+ `'` + t.ContractName + `',` + `'` + t.ContractVersion + `',`

	for _, data := range t.EventData {
		eventDataDdl += `'` + data + `'` + `,`
	}
	columnDdl = columnDdl[:len(columnDdl)-1]
	eventDataDdl = eventDataDdl[:len(eventDataDdl)-1]
	saveDdl += "INSERT INTO " + topicTableNameHex + " (" + columnDdl + ") " + "VALUES (" + eventDataDdl + " );"
	return saveDdl
}
func GenerateSaveBlockHeightWithTopicDdl(t *commonPb.ContractEvent) string {
	var saveDdl string
	var DataDdl string
	var columnDdl string

	tableName := `blockheight_topictable_index`
	topicTableNameSrc := t.ChainId + "_" + t.ContractName + "_" + t.Topic
	topicTableNameHash := sha256.Sum256([]byte(topicTableNameSrc))
	topicTableNameHex := "event" + hex.EncodeToString(topicTableNameHash[:32])[5:]
	columnDdl += `chain_id,block_height,topic_table_name_src,topic_table_name_hex`

	DataDdl += `'` + t.ChainId + `',` + `'` + strconv.FormatInt(t.BlockHeight, 10) + `',` + `'` + topicTableNameSrc + `',` + `'` + topicTableNameHex + `',`
	DataDdl = DataDdl[:len(DataDdl)-1]
	saveDdl += "INSERT INTO " + tableName + " (" + columnDdl + ") " + "VALUES (" + DataDdl + " );"
	return saveDdl
}
func GenerateSaveBlockHeightIndexDdl(blockHeight int64) string {
	var saveDdl string
	var DataDdl string
	var columnDdl string
	tableName := `blokcheight_index`
	columnDdl += `block_height`
	DataDdl += `'` + strconv.FormatInt(blockHeight, 10) + `'`
	saveDdl += "INSERT INTO " + tableName + " (" + columnDdl + ") " + "VALUES (" + DataDdl + " );"
	return saveDdl
}
func GenerateCreateTopicTableDdl(t *commonPb.ContractEvent) string {
	var createTopicTableSql string
	tableName := t.ChainId + "_" + t.ContractName + "_" + t.Topic
	topicTableNameHash := sha256.Sum256([]byte(tableName))
	topicTableNameHex := "event" + hex.EncodeToString(topicTableNameHash[:32])[5:]
	createTopicTableSql = "CREATE TABLE IF NOT EXISTS" + " " + topicTableNameHex + " ( " + TopicTableColumnDdl + "PRIMARY KEY (id)" + " ) ENGINE=InnoDB DEFAULT CHARSET=utf8;"
	return createTopicTableSql
}
