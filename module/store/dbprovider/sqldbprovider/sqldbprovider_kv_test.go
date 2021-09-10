package sqldbprovider

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

var kvConf = &SqlDbConfig{
	Dsn:        filepath.Join(os.TempDir(), fmt.Sprintf("%d_unit_test_db", time.Now().UnixNano())+":memory:"),
	SqlDbType:  "sqlite",
	SqlLogMode: "info",
}

var (
	key1      = []byte("key1")
	value1    = []byte("value1")
	key2      = []byte("key2")
	value2    = []byte("value2")
	key3      = []byte("key3")
	value3    = []byte("value3")
	keyFilter = []byte("key")
)

func initKvProvider() *SqlDBHandle {
	p := NewSqlDBHandle("chain1", kvConf, log)
	return p
}

func initTable(p *SqlDBHandle, t *testing.T) {
	err := p.CreateTableIfNotExist(&KeyValue{})
	assert.Nil(t, err)
}

func initKv(p *SqlDBHandle, t *testing.T) {
	err := p.Put(key1, value1)
	assert.Nil(t, err)
	err = p.Put(key2, value2)
	assert.Nil(t, err)
	err = p.Put(key3, value3)
	assert.Nil(t, err)
}

func deleteKv(p *SqlDBHandle, t *testing.T) {
	err := p.Delete(key1)
	assert.Nil(t, err)
	err = p.Delete(key2)
	assert.Nil(t, err)
	err = p.Delete(key3)
	assert.Nil(t, err)
}

func TestSqlDBHandle_Has(t *testing.T) {
	p := initKvProvider()
	initTable(p, t)

	err := p.Put(key1, value1)
	assert.Nil(t, err)

	has, err := p.Has(key1)
	assert.True(t, has)
	assert.Nil(t, err)

	err = p.Delete(key1)
	assert.Nil(t, err)

	err = p.Delete(key1)
	assert.Equal(t, strings.Contains(err.Error(), "no row exist"), true)

	has, err = p.Has(key1)
	assert.False(t, has)
	assert.Nil(t, err)

	p.Close()

	err = p.Delete(key1)
	assert.Equal(t, strings.Contains(err.Error(), "database is closed"), true)

	has, err = p.Has(key1)
	assert.False(t, has)
	assert.Equal(t, strings.Contains(err.Error(), "database is closed"), true)
}

func TestSqlDBHandle_NewIteratorWithPrefix(t *testing.T) {
	p := initKvProvider()
	defer p.Close()
	initTable(p, t)
	initKv(p, t)

	res, err := p.NewIteratorWithPrefix(keyFilter)
	assert.Nil(t, err)
	count := 1
	for res.Next() {
		count++
	}
	assert.Equal(t, count, 3)
	res.Release()

	deleteKv(p, t)

	res, err = p.NewIteratorWithPrefix([]byte(""))
	assert.Nil(t, res)
	assert.Equal(t, strings.Contains(err.Error(), "iterator prefix should not be empty key"), true)

	p.Close()
	res, err = p.NewIteratorWithPrefix(keyFilter)
	assert.Nil(t, err)
	assert.Nil(t, res)
}

func TestSqlDBHandle_NewIteratorWithRange(t *testing.T) {
	p := initKvProvider()
	//defer p.Close()
	initTable(p, t)
	initKv(p, t)

	res, err := p.NewIteratorWithRange(key1, key3)
	assert.Nil(t, err)
	count := 1
	for res.Next() {
		count++
	}
	assert.Equal(t, count, 2)
	res.Release()

	deleteKv(p, t)

	res, err = p.NewIteratorWithRange([]byte(""), key3)
	assert.Nil(t, res)
	assert.Equal(t, strings.Contains(err.Error(), fmt.Sprintf("iterator range should not start() or limit(%s) with empty key", string(key3))), true)

	res, err = p.NewIteratorWithRange([]byte(""), []byte(""))
	assert.Nil(t, res)
	assert.Equal(t, strings.Contains(err.Error(), "iterator range should not start() or limit() with empty key"), true)

	res, err = p.NewIteratorWithRange(key1, []byte(""))
	assert.Nil(t, res)
	assert.Equal(t, strings.Contains(err.Error(), fmt.Sprintf("iterator range should not start(%s) or limit() with empty key", string(key1))), true)

	p.Close()
	res, err = p.NewIteratorWithRange(key1, key3)
	assert.Nil(t, err)
	assert.Nil(t, res)
}

func TestSqlDBHandle_Put(t *testing.T) {
	p := initKvProvider()
	initTable(p, t)

	err := p.Put(key1, value1)
	assert.Nil(t, err)
	res, err := p.Get(key1)
	assert.Nil(t, err)
	assert.Equal(t, res, value1)

	res, err = p.Get(key2)
	assert.Nil(t, err)
	assert.Nil(t, res)

	err = p.Delete(key1)
	assert.Nil(t, err)

	p.Close()
	res, err = p.Get(key1)
	assert.Nil(t, res)
	assert.Equal(t, strings.Contains(err.Error(), "database is closed"), true)
}

func TestSqlDBHandle_WriteBatch(t *testing.T) {
	p := initKvProvider()
	defer p.Close()
	batch := types.NewUpdateBatch()

	batch.Put(key1, value1)
	batch.Put(key2, value2)
	batch.Put(key3, value3)
	err := p.WriteBatch(batch, false)
	assert.Nil(t, err)

	deleteKv(p, t)
}

func Test_deleteInTx(t *testing.T) {
	p := initKvProvider()
	//defer p.Close()
	initData(p)
	txName := "Block1"
	tx, _ := p.BeginDbTransaction(txName)
	_, err := tx.Save(&KeyValue{key1, value1})
	assert.Nil(t, err)

	err = deleteInTx(tx, key1)
	assert.Nil(t, err)

	err = deleteInTx(tx, key1)
	assert.Equal(t, strings.Contains(err.Error(), "no row exist"), true)

	err = p.CommitDbTransaction(txName)
	assert.Nil(t, err)

	has, err := p.Has(value1)
	assert.False(t, has)
	assert.Nil(t, err)
}

func Test_kvIterator_Error(t *testing.T) {
	kv := &kvIterator{}
	err := kv.Error()
	assert.Nil(t, err)
}

func Test_kvIterator_First(t *testing.T) {
	p := initKvProvider()
	defer p.Close()
	initTable(p, t)
	initKv(p, t)

	res, err := p.NewIteratorWithPrefix(keyFilter)
	assert.Nil(t, err)
	isFirst := res.First()
	assert.True(t, isFirst)

	isFirst = res.Next()

	isFirst = res.First()
	assert.False(t, isFirst)

	key := res.Key()
	value := res.Value()

	assert.Equal(t, strings.Contains(string(key1)+string(key2)+string(key3), string(key)), true)
	assert.Equal(t, strings.Contains(string(value1)+string(value2)+string(value3), string(value)), true)

	res.Release()
}

func Test_kvIterator_Key(t *testing.T) {
}

func Test_kvIterator_Next(t *testing.T) {
}

func Test_kvIterator_Release(t *testing.T) {
}

func Test_kvIterator_Value(t *testing.T) {
}

func Test_kvIterator_append(t *testing.T) {
}
