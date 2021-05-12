package eventsqldb

const (
	CreateBlockHeightWithTopicTableDdl string = `CREATE TABLE IF NOT EXISTS block_height_topic_table_index( id bigint unsigned NOT NULL AUTO_INCREMENT,chain_id varchar(128),block_height bigint,topic_table_name_src varchar(1000),topic_table_name_hex varchar(64),PRIMARY KEY (id) );`
	BlockHeightWithTopicTableName      string = `block_height_topic_table_index`
	CreateBlockHeightIndexTableDDL     string = `CREATE TABLE IF NOT EXISTS block_height_index ( id bigint unsigned NOT NULL AUTO_INCREMENT,block_height bigint,PRIMARY KEY (id) );`
	InitBlockHeightIndexTableDDL       string = `INSERT IGNORE INTO block_height_index (id,block_height) VALUES('1','0')`
	BlockHeightIndexTableName          string = `block_height_index`
)
