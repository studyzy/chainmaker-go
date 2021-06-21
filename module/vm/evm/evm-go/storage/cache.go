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
	"chainmaker.org/chainmaker/common/evmutils"
	"chainmaker.org/chainmaker-go/evm/evm-go/environment"
)

type Cache map[string]*evmutils.Int
type CacheUnderAddress map[string]Cache

func (c CacheUnderAddress) Get(address string, key string) *evmutils.Int {
	if c[address] == nil {
		return nil
	} else {
		return c[address][key]
	}
}

func (c CacheUnderAddress) Set(address string, key string, v *evmutils.Int) {
	if c[address] == nil {
		c[address] = Cache{}
	}

	c[address][key] = v
}

type balance struct {
	Address *evmutils.Int
	Balance *evmutils.Int
}

type BalanceCache map[string]*balance

type Log struct {
	Topics  [][]byte
	Data    []byte
	Context environment.Context
}

type LogCache map[string][]Log

type ResultCache struct {
	OriginalData CacheUnderAddress
	CachedData   CacheUnderAddress

	Balance   BalanceCache
	Logs      LogCache
	Destructs Cache
}

type CodeCache map[string][]byte

type readOnlyCache struct {
	Code      CodeCache
	CodeSize  Cache
	CodeHash  Cache
	BlockHash Cache
}

func MergeResultCache(src *ResultCache, to *ResultCache) {
	for k, v := range src.OriginalData {
		to.OriginalData[k] = v
	}

	for k, v := range src.CachedData {
		to.CachedData[k] = v
	}

	for k, v := range src.Balance {
		if to.Balance[k] != nil {
			to.Balance[k].Balance.Add(v.Balance)
		} else {
			to.Balance[k] = v
		}
	}

	for k, v := range src.Logs {
		to.Logs[k] = append(to.Logs[k], v...)
	}

	for k, v := range src.Destructs {
		to.Destructs[k] = v
	}
}
