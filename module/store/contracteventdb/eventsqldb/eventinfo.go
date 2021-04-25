package eventsqldb

//const (
//	CreateBlockHeightWithTopicTableDdl string = `CREATE TABLE IF NOT EXISTS block_height_topic_table_index( id bigint unsigned NOT NULL AUTO_INCREMENT,chain_id varchar(128),
//block_height bigint,topic_table_name_src varchar(1000),topic_table_name_hex varchar(64),PRIMARY KEY (id) ) ENGINE=InnoDB DEFAULT CHARSET=utf8;`
//	BlockHeightWithTopicTableName      string = `block_height_topic_table_index`
//	CreateBlockHeightIndexTableDDL     string = `CREATE TABLE IF NOT EXISTS block_height_index ( id bigint unsigned NOT NULL AUTO_INCREMENT,block_height bigint,PRIMARY KEY (id) ) ENGINE=InnoDB DEFAULT CHARSET=utf8;`
//	InitBlockHeightIndexTableDDL       string = `INSERT IGNORE INTO block_height_index (id,block_height) VALUES('1','0')`
//	BlockHeightIndexTableName          string = `block_height_index`
//)
type BlockHeightTopicTableIndex struct {
	Id                uint64 `gorm:"primaryKey;autoIncrement:true"`
	ChainId           string `gorm:"size:128"`
	BlockHeight       uint64
	TopicTableNameSrc string `gorm:"size:1000"`
	TopicTableNameHex string `gorm:"size:64"`
}
type BlockHeightIndex struct {
	Id          uint64 `gorm:"primaryKey;autoIncrement:true"`
	BlockHeight uint64
}
