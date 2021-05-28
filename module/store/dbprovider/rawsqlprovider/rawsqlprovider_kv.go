/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package rawsqlprovider

import (
	"errors"
	"fmt"
	"time"

	"chainmaker.org/chainmaker-go/protocol"
)

type KeyValue struct {
	ObjectKey   []byte `gorm:"size:128;primaryKey;default:''"`
	ObjectValue []byte `gorm:"type:longblob"`
}

func (kv *KeyValue) GetInsertSql() string {
	return "TODO"
}
func (kv *KeyValue) GetUpdateSql() string {
	return "TODO"
}

// Get returns the value for the given key, or returns nil if none exists
func (p *SqlDBHandle) Get(key []byte) ([]byte, error) {
	sql := "select object_value from key_values where object_key=?"
	result, err := p.QuerySingle(sql, key)
	if err != nil {
		return nil, err
	}
	if result.IsEmpty() {
		p.log.Infof("cannot query value by key=%s", string(key))
		return nil, nil
	}
	var v []byte
	err = result.ScanColumns(&v)
	if err != nil {
		return nil, err
	}
	return v, nil
}

// Put saves the key-values
func (p *SqlDBHandle) Put(key []byte, value []byte) error {
	kv := &KeyValue{
		ObjectKey:   key,
		ObjectValue: value,
	}
	_, err := p.Save(kv)
	return err
}

// Has return true if the given key exist, or return false if none exists
func (p *SqlDBHandle) Has(key []byte) (bool, error) {
	sql := "select count(*) from key_values where object_key=?"
	result, err := p.QuerySingle(sql, key)
	if err != nil {
		return false, err
	}
	var count int
	err = result.ScanColumns(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

const DELETE_SQL = "delete from key_values where object_key=?"

// Delete deletes the given key
func (p *SqlDBHandle) Delete(key []byte) error {
	count, err := p.ExecSql(DELETE_SQL, key)
	if err != nil {
		return err
	}
	if count == 0 {
		return errors.New("no row exist")
	}
	return nil
}
func deleteInTx(tx protocol.SqlDBTransaction, key []byte) error {
	count, err := tx.ExecSql(DELETE_SQL, key)
	if err != nil {
		return err
	}
	if count == 0 {
		return errors.New("no row exist")
	}
	return nil
}

// WriteBatch writes a batch in an atomic operation
func (p *SqlDBHandle) WriteBatch(batch protocol.StoreBatcher, sync bool) error {
	txName := fmt.Sprintf("%d", time.Now().UnixNano())
	tx, err := p.BeginDbTransaction(txName)
	if err != nil {
		return err
	}
	for k, v := range batch.KVs() {
		key := []byte(k)
		if v == nil {
			err := deleteInTx(tx, key)
			if err != nil {
				if err2 := p.RollbackDbTransaction(txName); err2 != nil {
					p.log.Errorf("rollback db transaction[%s] get an error:%s", txName, err2)
				}
				return err
			}
		} else {
			kv := &KeyValue{key, v}
			_, err := tx.Save(kv)
			if err != nil {
				if err2 := p.RollbackDbTransaction(txName); err2 != nil {
					p.log.Errorf("rollback db transaction[%s] get an error:%s", txName, err2)
				}
				return err
			}
		}
	}
	return p.CommitDbTransaction(txName)
}

// NewIteratorWithRange returns an iterator that contains all the key-values between given key ranges
// start is included in the results and limit is excluded.
func (p *SqlDBHandle) NewIteratorWithRange(start []byte, limit []byte) protocol.Iterator {
	sql := "select * from key_values where object_key between ? and ?"
	rows, err := p.QueryMulti(sql, start, limit)
	if err != nil {
		return nil
	}
	defer rows.Close()
	result := &kvIterator{}
	for rows.Next() {
		var kv KeyValue
		_ = rows.ScanColumns(&kv.ObjectKey, &kv.ObjectValue)
		result.append(&kv)
	}
	return result

}

// NewIteratorWithPrefix returns an iterator that contains all the key-values with given prefix
func (p *SqlDBHandle) NewIteratorWithPrefix(prefix []byte) protocol.Iterator {
	sql := "select * from key_values where object_key like ?%"
	rows, err := p.QueryMulti(sql, prefix)
	if err != nil {
		return nil
	}
	defer rows.Close()
	result := &kvIterator{}
	for rows.Next() {
		var kv KeyValue
		_ = rows.ScanColumns(&kv.ObjectKey, &kv.ObjectValue)
		result.append(&kv)
	}
	return result
}

type kvIterator struct {
	keyValues []*KeyValue
	idx       int
	count     int
}

func (kvi *kvIterator) append(kv *KeyValue) {
	kvi.keyValues = append(kvi.keyValues, kv)
	kvi.count++
}
func (kvi *kvIterator) Next() bool {
	kvi.idx++
	return kvi.idx < kvi.count
}
func (kvi *kvIterator) First() bool {
	return kvi.idx == 0
}
func (kvi *kvIterator) Error() error {
	return nil
}
func (kvi *kvIterator) Key() []byte {
	return kvi.keyValues[kvi.idx].ObjectKey
}
func (kvi *kvIterator) Value() []byte {
	return kvi.keyValues[kvi.idx].ObjectValue
}
func (kvi *kvIterator) Release() {
	kvi.idx = 0
	kvi.count = 0
	kvi.keyValues = make([]*KeyValue, 0)
}
