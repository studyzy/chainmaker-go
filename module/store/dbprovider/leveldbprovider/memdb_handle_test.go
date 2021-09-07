package leveldbprovider

import (
	"strings"
	"testing"

	"chainmaker.org/chainmaker-go/store/types"
	"github.com/stretchr/testify/assert"
)

func TestMemdbHandle_Close(t *testing.T) {
	dbHandle := NewMemdbHandle()
	err := dbHandle.Close()
	assert.Nil(t, err)
}

func TestMemdbHandle_CompactRange(t *testing.T) {
	dbHandle := NewMemdbHandle()
	defer dbHandle.Close()
	err := dbHandle.CompactRange([]byte(""), []byte(""))
	assert.Nil(t, err)
}

func TestMemdbHandle_NewIteratorWithPrefix(t *testing.T) {
	dbHandle := NewMemdbHandle()
	defer dbHandle.Close()

	batch := types.NewUpdateBatch()

	batch.Put([]byte("key1"), []byte("value1"))
	batch.Put([]byte("key2"), []byte("value2"))
	batch.Put([]byte("key3"), []byte("value3"))
	batch.Put([]byte("key4"), []byte("value4"))
	batch.Put([]byte("keyx"), []byte("value5"))

	err := dbHandle.WriteBatch(batch, true)
	assert.Equal(t, nil, err)

	iter, err := dbHandle.NewIteratorWithPrefix([]byte("key"))
	assert.Nil(t, err)
	defer iter.Release()
	var count int
	for iter.Next() {
		count++
	}
	assert.Equal(t, 5, count)

	_, err = dbHandle.NewIteratorWithPrefix([]byte(""))
	assert.NotNil(t, err)
	assert.Equal(t, strings.Contains(err.Error(), "iterator prefix should not be empty key"), true)
}

func TestMemdbHandle_NewIteratorWithRange(t *testing.T) {
	dbHandle := NewMemdbHandle()
	defer dbHandle.Close()

	batch := types.NewUpdateBatch()
	key1 := []byte("key1")
	value1 := []byte("value1")
	key2 := []byte("key2")
	value2 := []byte("value2")
	batch.Put(key1, value1)
	batch.Put(key2, value2)
	err := dbHandle.WriteBatch(batch, true)
	assert.Nil(t, err)

	iter, err := dbHandle.NewIteratorWithRange(key1, []byte("key3"))
	assert.Nil(t, err)
	defer iter.Release()
	var count int
	for iter.Next() {
		count++
	}
	assert.Equal(t, 2, count)

	_, err = dbHandle.NewIteratorWithRange([]byte(""), []byte(""))
	assert.NotNil(t, err)
	assert.Equal(t, strings.Contains(err.Error(), "iterator range should not start"), true)
}

func TestMemdbHandle_Put(t *testing.T) {
	dbHandle := NewMemdbHandle()
	defer dbHandle.Close()
	key1 := []byte("key1")
	value1 := []byte("value1")
	err := dbHandle.Put(key1, value1)
	assert.Nil(t, err)

	value, err := dbHandle.Get(key1)
	assert.Nil(t, err)
	assert.Equal(t, value1, value)

	exist, err := dbHandle.Has(key1)
	assert.True(t, exist)

	err = dbHandle.Delete(key1)
	assert.Nil(t, err)

	exist, err = dbHandle.Has(key1)
	assert.False(t, exist)
}
