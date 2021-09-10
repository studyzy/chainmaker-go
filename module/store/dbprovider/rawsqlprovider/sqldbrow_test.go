package rawsqlprovider

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSqlDBRow(t *testing.T) {
	dbHandle := NewSqlDBHandle(&NewSqlDBOptions{
		Config:    confProvideTest,
		Logger:    log,
		Encryptor: nil,
		ChainId:   "test-chain1",
		DbName:    "dbName1",
	})
	defer dbHandle.Close()

	point := &SavePoint{
		BlockHeight: 30,
	}

	sql, value := point.GetInsertSql()
	_, err := dbHandle.ExecSql(sql, value...)
	assert.Nil(t, err)

	res, err := dbHandle.QuerySingle(fmt.Sprintf("SELECT block_height FROM %s WHERE block_height = ?", point.GetTableName()), 30)
	assert.Nil(t, err)

	var height uint64 = 30
	err = res.ScanColumns(&height)
	assert.Nil(t, err)

	isempty := res.IsEmpty()
	assert.False(t, isempty)

	err = res.ScanColumns("test")
	assert.NotNil(t, err)

	empty := &emptyRow{}
	err = empty.ScanColumns()
	assert.Nil(t, err)
	isempty = empty.IsEmpty()
	assert.True(t, isempty)
}

func TestNewSqlDBRows(t *testing.T) {
	dbHandle := NewSqlDBHandle(&NewSqlDBOptions{
		Config:    confProvideTest,
		Logger:    log,
		Encryptor: nil,
		ChainId:   "test-chain1",
		DbName:    "dbName1",
	})
	//defer dbHandle.Close()

	point1 := &SavePoint{
		BlockHeight: 31,
	}
	point2 := &SavePoint{
		BlockHeight: 32,
	}
	point3 := &SavePoint{
		BlockHeight: 33,
	}

	sql, value := point1.GetInsertSql()
	_, err := dbHandle.ExecSql(sql, value...)
	assert.Nil(t, err)

	sql, value = point2.GetInsertSql()
	_, err = dbHandle.ExecSql(sql, value...)
	assert.Nil(t, err)

	sql, value = point3.GetInsertSql()
	_, err = dbHandle.ExecSql(sql, value...)
	assert.Nil(t, err)

	res, err := dbHandle.QueryMulti(fmt.Sprintf("SELECT block_height FROM %s WHERE block_height>=? AND block_height<=?", point1.GetTableName()), 31, 39)
	assert.Nil(t, err)
	res.Next()

	var height uint64 = 31
	err = res.ScanColumns(&height)
	assert.Nil(t, err)

	kv, err := res.Data()
	assert.Nil(t, err)
	fmt.Println(kv)

	err = res.Close()
	assert.Nil(t, err)

	kv, err = res.Data()
	assert.NotNil(t, err)
}
