/*
 * Copyright 2020 The SealEVM Authors
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *  http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

package storage

import (
	"encoding/hex"

	"chainmaker.org/chainmaker-go/evm/evm-go/environment"
	"chainmaker.org/chainmaker-go/evm/evm-go/utils"
	"chainmaker.org/chainmaker/common/v2/evmutils"
)

type Storage struct {
	ResultCache     ResultCache
	ExternalStorage IExternalStorage
	readOnlyCache   readOnlyCache
}

func New(extStorage IExternalStorage) *Storage {
	s := &Storage{
		ResultCache: ResultCache{
			OriginalData: CacheUnderAddress{},
			CachedData:   CacheUnderAddress{},
			Balance:      BalanceCache{},
			Logs:         LogCache{},
			Destructs:    Cache{},
		},
		ExternalStorage: extStorage,
		readOnlyCache: readOnlyCache{
			Code:      CodeCache{},
			CodeSize:  Cache{},
			CodeHash:  Cache{},
			BlockHash: Cache{},
		},
	}

	return s
}

func (s *Storage) SLoad(n *evmutils.Int, k *evmutils.Int) (*evmutils.Int, error) {
	//fmt.Println("SLoad", n.String(), "k", k.String())
	if s.ResultCache.OriginalData == nil || s.ResultCache.CachedData == nil || s.ExternalStorage == nil {
		return nil, utils.ErrStorageNotInitialized
	}

	nsStr := hex.EncodeToString(n.Bytes())
	keyStr := hex.EncodeToString(k.Bytes())
	//nsStr := n.String()
	//keyStr := k.String()

	var err error = nil
	i := s.ResultCache.CachedData.Get(nsStr, keyStr)
	if i == nil {
		i, err = s.ExternalStorage.Load(nsStr, keyStr)
		if err != nil {
			return nil, utils.NoSuchDataInTheStorage(err)
		}

		s.ResultCache.OriginalData.Set(nsStr, keyStr, i)
		s.ResultCache.CachedData.Set(nsStr, keyStr, i)
	}

	return i, nil
}

func (s *Storage) SStore(n *evmutils.Int, k *evmutils.Int, v *evmutils.Int) {
	nsStr := hex.EncodeToString(n.Bytes())
	keyStr := hex.EncodeToString(k.Bytes())
	s.ResultCache.CachedData.Set(nsStr, keyStr, v)
	//fmt.Println("SStore", n.String(), "k", k.String(), "v", v.String())
}

func (s *Storage) BalanceModify(address *evmutils.Int, value *evmutils.Int, neg bool) {
	//kString := address.String()
	kString := hex.EncodeToString(address.Bytes())

	b, exist := s.ResultCache.Balance[kString]
	if !exist {
		b = &balance{
			Address: evmutils.FromBigInt(address.Int),
			Balance: evmutils.New(0),
		}

		s.ResultCache.Balance[kString] = b
	}

	if neg {
		b.Balance.Int.Sub(b.Balance.Int, value.Int)
	} else {
		b.Balance.Int.Add(b.Balance.Int, value.Int)
	}
}

func (s *Storage) Log(address *evmutils.Int, topics [][]byte, data []byte, context environment.Context) {
	//kString := address.String()
	kString := hex.EncodeToString(address.Bytes())

	var theLog = Log{
		Topics:  topics,
		Data:    data,
		Context: context,
	}
	l := s.ResultCache.Logs[kString]
	s.ResultCache.Logs[kString] = append(l, theLog)

	return
}

func (s *Storage) Destruct(address *evmutils.Int) {
	//s.ResultCache.Destructs[address.String()] = address
	s.ResultCache.Destructs[hex.EncodeToString(address.Bytes())] = address
}

type commonGetterFunc func(*evmutils.Int) (*evmutils.Int, error)

func (s *Storage) commonGetter(key *evmutils.Int, cache Cache, getterFunc commonGetterFunc) (*evmutils.Int, error) {
	//keyStr := key.String()
	keyStr := hex.EncodeToString(key.Bytes())
	if b, exists := cache[keyStr]; exists {
		return evmutils.FromBigInt(b.Int), nil
	}

	b, err := getterFunc(key)
	if err == nil {
		cache[keyStr] = b
	}

	return b, err
}

func (s *Storage) Balance(address *evmutils.Int) (*evmutils.Int, error) {
	return s.ExternalStorage.GetBalance(address)
}
func (s *Storage) SetCode(address *evmutils.Int, code []byte) {
	//keyStr := address.String()
	keyStr := hex.EncodeToString(address.Bytes())
	s.readOnlyCache.Code[keyStr] = code
}
func (s *Storage) GetCode(address *evmutils.Int) ([]byte, error) {
	//keyStr := address.String()
	keyStr := hex.EncodeToString(address.Bytes())
	if b, exists := s.readOnlyCache.Code[keyStr]; exists {
		return b, nil
	}

	b, err := s.ExternalStorage.GetCode(address)
	if err == nil {
		s.readOnlyCache.Code[keyStr] = b
	}

	return b, err
}
func (s *Storage) SetCodeSize(address *evmutils.Int, size *evmutils.Int) {
	//keyStr := address.String()
	keyStr := hex.EncodeToString(address.Bytes())
	s.readOnlyCache.CodeSize[keyStr] = size
}
func (s *Storage) GetCodeSize(address *evmutils.Int) (*evmutils.Int, error) {
	//keyStr := address.String()
	keyStr := hex.EncodeToString(address.Bytes())
	if size, exists := s.readOnlyCache.CodeSize[keyStr]; exists {
		return size, nil
	}

	size, err := s.ExternalStorage.GetCodeSize(address)
	if err == nil {
		s.readOnlyCache.CodeSize[keyStr] = size
	}

	return size, err
}
func (s *Storage) SetCodeHash(address *evmutils.Int, codeHash *evmutils.Int) {
	//keyStr := address.String()
	keyStr := hex.EncodeToString(address.Bytes())
	s.readOnlyCache.CodeHash[keyStr] = codeHash
}
func (s *Storage) GetCodeHash(address *evmutils.Int) (*evmutils.Int, error) {
	//keyStr := address.String()
	keyStr := hex.EncodeToString(address.Bytes())
	if hash, exists := s.readOnlyCache.CodeHash[keyStr]; exists {
		return hash, nil
	}

	hash, err := s.ExternalStorage.GetCodeHash(address)
	if err == nil {
		s.readOnlyCache.CodeHash[keyStr] = hash
	}

	return hash, err
}

func (s *Storage) GetBlockHash(block *evmutils.Int) (*evmutils.Int, error) {
	//keyStr := block.String()
	keyStr := hex.EncodeToString(block.Bytes())
	if hash, exists := s.readOnlyCache.BlockHash[keyStr]; exists {
		return hash, nil
	}

	hash, err := s.ExternalStorage.GetBlockHash(block)
	if err == nil {
		s.readOnlyCache.BlockHash[keyStr] = hash
	}

	return hash, err
}
func (s *Storage) CreateAddress(caller *evmutils.Int, tx environment.Transaction) *evmutils.Int {
	return s.ExternalStorage.CreateAddress(caller, tx)
}

func (s *Storage) CreateFixedAddress(caller *evmutils.Int, salt *evmutils.Int, tx environment.Transaction) *evmutils.Int {
	return s.ExternalStorage.CreateFixedAddress(caller, salt, tx)
}
