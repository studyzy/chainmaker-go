package rawsqlprovider

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var confSqldbTxTest = &SqlDbConfig{
	SqlDbType: "sqlite",
	Dsn:       filepath.Join(os.TempDir(), fmt.Sprintf("%d_sql_test_db", time.Now().UnixNano())+":memory:"),
}
var sqlTxTestOp = &NewSqlDBOptions{
	Config:    confSqldbTxTest,
	Logger:    log,
	Encryptor: nil,
	ChainId:   "test-chain1",
	DbName:    "dbName1",
}

func TestSqlDBTx_Save(t *testing.T) {
	dbHandle := NewSqlDBHandle(sqlTxTestOp)
	//defer dbHandle.Close()
	test := &Test{
		TestColumn: 1,
	}
	point := &SavePoint{
		BlockHeight: 20,
	}
	err := dbHandle.CreateTableIfNotExist(test)
	assert.Nil(t, err)
	err = dbHandle.CreateTableIfNotExist(point)
	assert.Nil(t, err)
	txName1 := "1234567890"

	dbType, err := ParseSqlDbType("sqlite")
	assert.Nil(t, err)
	tx, err := dbHandle.db.Begin()
	assert.Nil(t, err)

	sqltx := NewSqlDBTx(txName1, dbType, tx, dbHandle.log)

	code, err := sqltx.Save(test)
	assert.Equal(t, int64(1), code)
	assert.Nil(t, err)

	code, err = sqltx.Save(&BlockInfo{})
	assert.Equal(t, int64(0), code)
	assert.NotNil(t, err)

	code, err = sqltx.Save(point)
	assert.Equal(t, int64(1), code)
	assert.Nil(t, err)

	code, err = sqltx.Save(point)
	assert.Equal(t, int64(1), code)
	assert.Nil(t, err)

	err = sqltx.Commit()
	assert.Nil(t, err)

	dbHandle.Close()
	code, err = sqltx.Save(test)
	assert.Equal(t, int64(0), code)
	assert.NotNil(t, err)
}

func TestNewSqlDBTx(t *testing.T) {
}

func TestSqlDBTx_BeginDbSavePoint(t *testing.T) {

}

func TestSqlDBTx_ChangeContextDb(t *testing.T) {
	dbHandle := NewSqlDBHandle(sqlTxTestOp)
	defer dbHandle.Close()
	txName1 := "1234567890"

	dbType, err := ParseSqlDbType("sqlite")
	assert.Nil(t, err)
	tx, err := dbHandle.db.Begin()
	assert.Nil(t, err)
	sqltx := NewSqlDBTx(txName1, dbType, tx, dbHandle.log)

	err = sqltx.ChangeContextDb("")
	assert.Nil(t, err)

	err = sqltx.ChangeContextDb("test2")
	assert.Nil(t, err)

	sqltx.dbType, _ = ParseSqlDbType("mysql")
	err = sqltx.ChangeContextDb("test2")
	assert.NotNil(t, err)
}

func TestSqlDBTx_QueryMulti(t *testing.T) {
	dbHandle := NewSqlDBHandle(sqlTxTestOp)
	defer dbHandle.Close()
	txName1 := "qwert"
	txName2 := "qwert!#"

	point1 := &SavePoint{
		BlockHeight: 40,
	}

	point2 := &SavePoint{
		BlockHeight: 41,
	}

	point3 := &SavePoint{
		BlockHeight: 42,
	}

	dbType, err := ParseSqlDbType("sqlite")
	assert.Nil(t, err)
	tx, err := dbHandle.db.Begin()
	assert.Nil(t, err)
	sqltx := NewSqlDBTx(txName1, dbType, tx, dbHandle.log)

	sql, value := point1.GetInsertSql()
	_, err = sqltx.ExecSql(sql, value...)
	assert.Nil(t, err)
	sql, value = point2.GetInsertSql()
	_, err = sqltx.ExecSql(sql, value...)
	assert.Nil(t, err)
	sql, value = point3.GetInsertSql()
	_, err = sqltx.ExecSql(sql, value...)
	assert.Nil(t, err)

	err = sqltx.Commit()

	tx, err = dbHandle.db.Begin()
	assert.Nil(t, err)
	sqltx = NewSqlDBTx(txName2, dbType, tx, dbHandle.log)
	res, err := sqltx.QueryMulti(fmt.Sprintf("SELECT block_height FROM %s WHERE block_height>=? AND block_height<=?", point1.GetTableName()), 40, 42)
	assert.Nil(t, err)
	count := 0
	for res.Next() {
		count++
	}
	assert.Equal(t, 3, count)
	assert.Nil(t, err)
	err = sqltx.Commit()
	assert.Nil(t, err)
}

func TestSqlDBTx_QuerySingle(t *testing.T) {
	dbHandle := NewSqlDBHandle(sqlTxTestOp)
	defer dbHandle.Close()
	txName1 := "1234567890123"

	point := &SavePoint{
		BlockHeight: 10,
	}

	err := dbHandle.CreateTableIfNotExist(point)
	assert.Nil(t, err)

	dbType, err := ParseSqlDbType("sqlite")
	assert.Nil(t, err)
	tx, err := dbHandle.db.Begin()
	assert.Nil(t, err)
	sqltx := NewSqlDBTx(txName1, dbType, tx, dbHandle.log)

	sql, value := point.GetInsertSql()
	_, err = sqltx.ExecSql(sql, value...)
	assert.Nil(t, err)

	res, err := sqltx.QuerySingle(fmt.Sprintf("SELECT block_height FROM %s WHERE block_height = ?", point.GetTableName()), 10)
	assert.Nil(t, err)
	resMap, err := res.Data()
	assert.Nil(t, err)
	assert.NotNil(t, resMap["block_height"])

	err = sqltx.Commit()
	assert.Nil(t, err)
	dbHandle.Close()

	res, err = sqltx.QuerySingle(fmt.Sprintf("SELECT block_height FROM %s WHERE block_height = ?", point.GetTableName()), 10)
	assert.Nil(t, res)
	assert.NotNil(t, err)
}

func TestSqlDBTx_Rollback(t *testing.T) {
}

func TestSqlDBTx_RollbackDbSavePoint(t *testing.T) {
	dbHandle := NewSqlDBHandle(sqlTxTestOp)
	defer dbHandle.Close()
	txName1 := "1234567890qwe"

	dbType, err := ParseSqlDbType("sqlite")
	assert.Nil(t, err)
	tx, err := dbHandle.db.Begin()
	assert.Nil(t, err)
	sqltx := NewSqlDBTx(txName1, dbType, tx, dbHandle.log)

	err = sqltx.BeginDbSavePoint("test")
	assert.Nil(t, err)
	err = sqltx.RollbackDbSavePoint("test")
	assert.Nil(t, err)
	err = sqltx.Commit()
	assert.Nil(t, err)
}

func Test_getSavePointName(t *testing.T) {
	name := getSavePointName("test")
	assert.Equal(t, "SP_test", name)
}
