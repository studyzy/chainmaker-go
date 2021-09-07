package rawsqlprovider

type SavePoint struct {
	BlockHeight uint64 `gorm:"primarykey"`
}

func (b *SavePoint) GetCreateTableSql(dbType string) string {
	if dbType == sqliteStr {
		return "CREATE TABLE `save_points` (`block_height` bigint unsigned AUTO_INCREMENT,PRIMARY KEY (`block_height`))"
	} else if dbType == sqliteStr {
		return "CREATE TABLE `save_points` (`block_height` integer,PRIMARY KEY (`block_height`))"
	}
	panic("Unsupported db type:" + dbType)
}
func (b *SavePoint) GetTableName() string {
	return "save_points"
}
func (b *SavePoint) GetInsertSql() (string, []interface{}) {
	return "INSERT INTO save_points values(?)", []interface{}{b.BlockHeight}
}
func (b *SavePoint) GetUpdateSql() (string, []interface{}) {
	return "UPDATE save_points set block_height=?", []interface{}{b.BlockHeight}
}
func (b *SavePoint) GetCountSql() (string, []interface{}) {
	return "SELECT count(*) FROM save_points", []interface{}{}
}

type Test struct {
	TestColumn uint64 `gorm:"primarykey"`
}

func (t *Test) GetCreateTableSql(dbType string) string {
	if dbType == "mysql" {
		return "CREATE TABLE `test_table` (`block_height` bigint unsigned AUTO_INCREMENT,PRIMARY KEY (`block_height`))"
	} else if dbType == "sqlite" {
		return "CREATE TABLE `test_table` (`block_height` integer,PRIMARY KEY (`block_height`))"
	}
	panic("Unsupported db type:" + dbType)
}
func (t *Test) GetTableName() string {
	return "test_table"
}
func (t *Test) GetSaveSql() (string, []interface{}) {
	return "INSERT INTO test_table values(?)", []interface{}{t.TestColumn}
}

type BlockInfo struct {
}
