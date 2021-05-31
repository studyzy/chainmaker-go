/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package vm

import (
	"bytes"
	acPb "chainmaker.org/chainmaker/pb-go/accesscontrol"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"math/rand"
	"strconv"
	"testing"

	commonPb "chainmaker.org/chainmaker/pb-go/common"
	"chainmaker.org/chainmaker/protocol"
)

type mockMemCache struct {
	txExecSeq int32
	store     map[string][]byte
	txRwSet   *commonPb.TxRWSet
}

const implement_me string = "implement me"

func (m *mockMemCache) Select(namespace string, startKey []byte, limit []byte) (protocol.Iterator, error) {
	panic(implement_me)
}

func (m *mockMemCache) GetBlockHeight() int64 {
	panic(implement_me)
}

func (m *mockMemCache) GetTxResult() *commonPb.Result {
	panic(implement_me)
}

func (m *mockMemCache) SetTxResult(result *commonPb.Result) {
	panic(implement_me)
}

func (m *mockMemCache) GetCreator(namespace string) *acPb.SerializedMember {
	panic(implement_me)
}

func (m *mockMemCache) GetSender() *acPb.SerializedMember {
	panic(implement_me)
}

func (m *mockMemCache) GetAccessControl() (protocol.AccessControlProvider, error) {
	panic(implement_me)
}

func getFinalKey(contractName string, key []byte) []byte {
	return append(append([]byte(contractName), protocol.ContractKey...), key...)
}

func (m *mockMemCache) putIntoReadSet(contractName string, key []byte, value []byte) {
	finalKey := getFinalKey(contractName, key)

	m.txRwSet.TxReads = append(m.txRwSet.TxReads, &commonPb.TxRead{
		Key:     finalKey,
		Value:   value,
		Version: nil,
	})
}

func (m *mockMemCache) putIntoWriteSet(contractName string, key []byte, value []byte) {
	finalKey := getFinalKey(contractName, key)

	m.txRwSet.TxWrites = append(m.txRwSet.TxWrites, &commonPb.TxWrite{
		Key:   finalKey,
		Value: value,
	})
}

func (m *mockMemCache) getFromReadSet(contractName string, key []byte) ([]byte, bool) {
	finalKey := getFinalKey(contractName, key)

	txRWSet := m.txRwSet
	for index, _ := range txRWSet.TxReads {
		txRead := txRWSet.TxReads[len(txRWSet.TxReads)-index-1]
		if bytes.Compare(txRead.Key, finalKey) == 0 {
			return txRead.Value, true
		}
	}
	return nil, false
}

func (m *mockMemCache) getFromWriteSet(contractName string, key []byte) ([]byte, bool) {
	finalKey := getFinalKey(contractName, key)

	txRWSet := m.txRwSet
	for index, _ := range txRWSet.TxWrites {
		txWrite := txRWSet.TxWrites[len(txRWSet.TxWrites)-index-1]
		if bytes.Compare(txWrite.Key, finalKey) == 0 {
			return txWrite.Value, true
		}
	}
	return nil, false
}

func (m mockMemCache) Get(contractName string, key []byte) ([]byte, error) {

	// Get from write set
	value, done := m.getFromWriteSet(contractName, key)
	if done {
		return value, nil
	}

	// Get from read set
	value, done = m.getFromReadSet(contractName, key)
	if done {
		return value, nil
	}

	// Get from db
	finalKey := getFinalKey(contractName, key)
	if value, ok := m.store[hex.EncodeToString(finalKey)]; ok {
		m.putIntoReadSet(contractName, key, value)
		return value, nil
	}

	return nil, nil

}

func (m mockMemCache) Put(contractName string, key []byte, value []byte) error {
	m.putIntoWriteSet(contractName, key, value)
	return nil
}

func (m mockMemCache) Del(contractName string, key []byte) error {
	m.putIntoWriteSet(contractName, key, nil)
	return nil
}

func (m mockMemCache) GetTx() *commonPb.Transaction {
	return &commonPb.Transaction{
		Header: &commonPb.TxHeader{
			ChainId:        "chain1",
			Sender:         nil,
			TxType:         0,
			TxId:           "",
			Timestamp:      0,
			ExpirationTime: 0,
		},
		RequestPayload:   nil,
		RequestSignature: nil,
		Result:           nil,
	}
}

func (m mockMemCache) GetTxRWSet() *commonPb.TxRWSet {
	return &commonPb.TxRWSet{
		TxReads:  nil,
		TxWrites: nil,
	}
}

func (m mockMemCache) GetInvoker() []byte {
	return nil
}

func (m mockMemCache) GetTxExecSeq() int32 {
	return m.txExecSeq
}

func (m mockMemCache) SetTxExecSeq(i int32) {
	m.txExecSeq = int32(i)
}

func (*mockMemCache) GetBlockchainStore() protocol.BlockchainStore {
	panic(implement_me)
}

func TestRand(t *testing.T) {
	{
		txId := []byte("00000000000000000000000000000001")
		sha256 := sha256.New()
		sha256.Write(txId)
		sha := sha256.Sum(nil)
		seed := binary.BigEndian.Uint64(sha)
		println(seed)
	}

	{
		txId := []byte("00000000000000000000000000000002")
		sha256 := sha256.New()
		sha256.Write(txId)
		sha := sha256.Sum(nil)
		seed := binary.BigEndian.Uint64(sha)
		println(seed)
	}
	txId := []byte("00000000000000000000000000000002")
	sha256 := sha256.New()
	sha256.Write(txId)
	sha := sha256.Sum(nil)
	seed := binary.BigEndian.Uint64(sha)
	println(seed)
	source := rand.NewSource(int64(seed))
	rr := rand.New(source)

	for i := 0; i < 5; i++ {
		kId := rr.Intn(10)
		key := "K" + strconv.Itoa(kId)
		println(key)
	}
}
