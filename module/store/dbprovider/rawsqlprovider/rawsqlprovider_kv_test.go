package rawsqlprovider

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"chainmaker.org/chainmaker-go/store/types"
	"github.com/stretchr/testify/assert"
)

var confKvTest = &SqlDbConfig{
	SqlDbType: "sqlite",
	Dsn:       filepath.Join(os.TempDir(), fmt.Sprintf("%d_unit_test_db", time.Now().UnixNano())+":memory:"),
}

var (
	key1         = []byte("key1")
	value1       = []byte("value1")
	key2         = []byte("key2")
	value2       = []byte("value2")
	key3         = []byte("key3")
	value3       = []byte("value3")
	key4         = []byte("key4")
	value4       = []byte("value4")
	key5         = []byte("key5")
	value5       = []byte("value5")
	key6         = []byte("key6")
	keyFilter1   = []byte("keyFilter1")
	valueFilter1 = []byte("valueFilter1")
	keyFilter2   = []byte("keyFilter2")
	valueFilter2 = []byte("valueFilter2")
	keyFilter3   = []byte("keyFilter3")
	valueFilter3 = []byte("valueFilter3")
)
var op = &NewSqlDBOptions{
	Config:    confKvTest,
	Logger:    log,
	Encryptor: nil,
	ChainId:   "test-chain1",
	DbName:    "dbName1",
}

func TestSqlDBHandle_NewIteratorWithPrefix(t *testing.T) {
	dbHandle := NewSqlDBHandle(op)
	defer dbHandle.Close()

	err := dbHandle.CreateTableIfNotExist(&KeyValue{})
	assert.Nil(t, err)

	err = dbHandle.Put(keyFilter1, valueFilter1)
	assert.Nil(t, err)
	err = dbHandle.Put(keyFilter2, valueFilter2)
	assert.Nil(t, err)
	err = dbHandle.Put(keyFilter3, valueFilter3)
	assert.Nil(t, err)

	res, err := dbHandle.NewIteratorWithPrefix([]byte("keyFilter%"))
	assert.Nil(t, err)
	key := res.Key()
	assert.Equal(t, true, strings.Contains(string(keyFilter1)+string(keyFilter2)+string(keyFilter3), string(key)))
	value := res.Value()
	assert.Equal(t, true, strings.Contains(string(valueFilter1)+string(valueFilter2)+string(valueFilter3), string(value)))
	isFirst := res.First()
	assert.True(t, isFirst)
	count := 1
	for res.Next() {
		count++
	}
	assert.Equal(t, 3, count)
	fmt.Println(res)

	isFirst = res.First()
	assert.False(t, isFirst)

	err = res.Error()
	assert.Nil(t, err)
	res.Release()
	assert.False(t, res.Next())
}

func TestSqlDBHandle_NewIteratorWithRange(t *testing.T) {
	dbHandle := NewSqlDBHandle(op)
	//defer dbHandle.Close()

	err := dbHandle.CreateTableIfNotExist(&KeyValue{})
	assert.Nil(t, err)

	err = dbHandle.Put(key3, value3)
	assert.Nil(t, err)
	err = dbHandle.Put(key4, value4)
	assert.Nil(t, err)
	err = dbHandle.Put(key5, value5)
	assert.Nil(t, err)

	res, err := dbHandle.NewIteratorWithRange(key3, key6)
	assert.Nil(t, err)
	count := 1
	for res.Next() {
		count++
	}
	assert.Equal(t, 3, count)
	fmt.Println(res)

	res, err = dbHandle.NewIteratorWithRange([]byte(""), []byte(""))
	assert.Nil(t, res)
	assert.NotNil(t, err)

	dbHandle.Close()
	res, err = dbHandle.NewIteratorWithRange(key3, key6)
	assert.Nil(t, res)
	assert.Nil(t, err)
}

func TestSqlDBHandle_Put(t *testing.T) {
	dbHandle := NewSqlDBHandle(op)
	//defer dbHandle.Close()

	err := dbHandle.CreateTableIfNotExist(&KeyValue{})
	assert.Nil(t, err)

	err = dbHandle.Put(key1, value1)
	assert.Nil(t, err)

	res, err := dbHandle.Get(key1)
	assert.Nil(t, err)
	assert.Equal(t, string(value1), string(res))

	has, err := dbHandle.Has(key1)
	assert.True(t, has)
	assert.Nil(t, err)

	err = dbHandle.Delete(key1)
	assert.Nil(t, err)

	err = dbHandle.Delete(key1)
	assert.NotNil(t, err)

	res, err = dbHandle.Get(key2)
	assert.Nil(t, err)
	assert.Nil(t, res)

	dbHandle.Close()
	res, err = dbHandle.Get(key2)
	assert.NotNil(t, err)

	has, err = dbHandle.Has(key2)
	assert.False(t, has)
	assert.NotNil(t, err)

	err = dbHandle.Delete(key1)
	assert.NotNil(t, err)
}

func TestSqlDBHandle_WriteBatch(t *testing.T) {
	dbHandle := NewSqlDBHandle(op)
	//defer dbHandle.Close()

	batch := types.NewUpdateBatch()

	batch.Put(key1, value1)
	batch.Put(key2, value2)
	batch.Put(key2, value1)
	err := dbHandle.WriteBatch(batch, true)
	assert.Nil(t, err)

	dbHandle.Close()

	err = dbHandle.WriteBatch(batch, true)
	assert.NotNil(t, err)
}

func Test_deleteInTx(t *testing.T) {
	dbHandle := NewSqlDBHandle(op)
	//defer dbHandle.Close()

	tx, err := dbHandle.BeginDbTransaction("1234567890")
	assert.Nil(t, err)

	err = dbHandle.Put(key1, value1)
	assert.Nil(t, err)
	err = deleteInTx(tx, key1)
	assert.Nil(t, err)

	err = deleteInTx(tx, key1)
	assert.NotNil(t, err)

	dbHandle.Close()

	err = deleteInTx(tx, key1)
	assert.NotNil(t, err)
}
