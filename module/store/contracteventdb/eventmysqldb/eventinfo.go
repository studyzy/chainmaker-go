package eventmysqldb

const (
	CreateBlockHeightWithTopicTableDdl string = `CREATE TABLE IF NOT EXISTS blockheight_topictable_index( id bigint unsigned NOT NULL AUTO_INCREMENT,chain_id varchar(128),block_height bigint,topic_table_name_src varchar(1000),topic_table_name_hex varchar(64),PRIMARY KEY (id) ) ENGINE=InnoDB DEFAULT CHARSET=utf8;`
	BlockHeightWithTopicTableName      string = `blockheight_topictable_index`
	CreateBlockHeightIndexTableDDL     string = `CREATE TABLE IF NOT EXISTS blokcheight_index ( id bigint unsigned NOT NULL AUTO_INCREMENT,block_height bigint,PRIMARY KEY (id) ) ENGINE=InnoDB DEFAULT CHARSET=utf8;`
	BlockHeightIndexTableName          string = `blokcheight_index`
)
