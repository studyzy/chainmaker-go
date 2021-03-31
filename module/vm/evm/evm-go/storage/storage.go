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
	"chainmaker.org/chainmaker-go/evm/evm-go/environment"
	"chainmaker.org/chainmaker-go/evm/evm-go/utils"
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

func (s *Storage) SLoad(n *utils.Int, k *utils.Int) (*utils.Int, error) {
	//fmt.Println("SLoad", n.String(), "k", k.String())
	if s.ResultCache.OriginalData == nil || s.ResultCache.CachedData == nil || s.ExternalStorage == nil {
		return nil, utils.ErrStorageNotInitialized
	}

	nsStr := n.String()
	keyStr := k.String()

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

func (s *Storage) SStore(n *utils.Int, k *utils.Int, v *utils.Int) {
	s.ResultCache.CachedData.Set(n.String(), k.String(), v)
	//fmt.Println("SStore", n.String(), "k", k.String(), "v", v.String())
}

func (s *Storage) BalanceModify(address *utils.Int, value *utils.Int, neg bool) {
	kString := address.String()

	b, exist := s.ResultCache.Balance[kString]
	if !exist {
		b = &balance{
			Address: utils.FromBigInt(address.Int),
			Balance: utils.New(0),
		}

		s.ResultCache.Balance[kString] = b
	}

	if neg {
		b.Balance.Int.Sub(b.Balance.Int, value.Int)
	} else {
		b.Balance.Int.Add(b.Balance.Int, value.Int)
	}
}

func (s *Storage) Log(address *utils.Int, topics [][]byte, data []byte, context environment.Context) {
	kString := address.String()

	var theLog = Log{
		Topics:  topics,
		Data:    data,
		Context: context,
	}
	l := s.ResultCache.Logs[kString]
	s.ResultCache.Logs[kString] = append(l, theLog)

	return
}

func (s *Storage) Destruct(address *utils.Int) {
	s.ResultCache.Destructs[address.String()] = address
}

type commonGetterFunc func(*utils.Int) (*utils.Int, error)

func (s *Storage) commonGetter(key *utils.Int, cache Cache, getterFunc commonGetterFunc) (*utils.Int, error) {
	keyStr := key.String()
	if b, exists := cache[keyStr]; exists {
		return utils.FromBigInt(b.Int), nil
	}

	b, err := getterFunc(key)
	if err == nil {
		cache[keyStr] = b
	}

	return b, err
}

func (s *Storage) Balance(address *utils.Int) (*utils.Int, error) {
	return s.ExternalStorage.GetBalance(address)
}
func (s *Storage) SetCode(address *utils.Int, code []byte) {
	keyStr := address.String()
	s.readOnlyCache.Code[keyStr] = code
}
func (s *Storage) GetCode(address *utils.Int) ([]byte, error) {
	keyStr := address.String()
	if b, exists := s.readOnlyCache.Code[keyStr]; exists {
		return b, nil
	}

	b, err := s.ExternalStorage.GetCode(address)
	if err == nil {
		s.readOnlyCache.Code[keyStr] = b
	}

	return b, err
}
func (s *Storage) SetCodeSize(address *utils.Int, size *utils.Int) {
	keyStr := address.String()
	s.readOnlyCache.CodeSize[keyStr] = size
}
func (s *Storage) GetCodeSize(address *utils.Int) (*utils.Int, error) {
	keyStr := address.String()
	if size, exists := s.readOnlyCache.CodeSize[keyStr]; exists {
		return size, nil
	}

	size, err := s.ExternalStorage.GetCodeSize(address)
	if err == nil {
		s.readOnlyCache.CodeSize[keyStr] = size
	}

	return size, err
}
func (s *Storage) SetCodeHash(address *utils.Int, codeHash *utils.Int) {
	keyStr := address.String()
	s.readOnlyCache.CodeHash[keyStr] = codeHash
}
func (s *Storage) GetCodeHash(address *utils.Int) (*utils.Int, error) {
	keyStr := address.String()
	if hash, exists := s.readOnlyCache.CodeHash[keyStr]; exists {
		return hash, nil
	}

	hash, err := s.ExternalStorage.GetCodeHash(address)
	if err == nil {
		s.readOnlyCache.CodeHash[keyStr] = hash
	}

	return hash, err
}

func (s *Storage) GetBlockHash(block *utils.Int) (*utils.Int, error) {
	keyStr := block.String()
	if hash, exists := s.readOnlyCache.BlockHash[keyStr]; exists {
		return hash, nil
	}

	hash, err := s.ExternalStorage.GetBlockHash(block)
	if err == nil {
		s.readOnlyCache.BlockHash[keyStr] = hash
	}

	return hash, err
}
func (s *Storage) CreateAddress(caller *utils.Int, tx environment.Transaction) *utils.Int {
	return s.ExternalStorage.CreateAddress(caller, tx)
}

func (s *Storage) CreateFixedAddress(caller *utils.Int, salt *utils.Int, tx environment.Transaction) *utils.Int {
	return s.ExternalStorage.CreateFixedAddress(caller, salt, tx)
}
