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
	"chainmaker.org/chainmaker-go/logger"
	"chainmaker.org/chainmaker-go/protocol"
)

var log = logger.GetLogger(logger.MODULE_VM)

type ContractStorage struct {
	ResultCache     ResultCache
	ExternalStorage IExternalStorage
	readOnlyCache   readOnlyCache
	Ctx             protocol.TxSimContext
	BlockHash       *utils.Int
}

func NewStorage(extStorage IExternalStorage) *ContractStorage {
	s := &ContractStorage{
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

func (c *ContractStorage) GetBalance(address *utils.Int) (*utils.Int, error) {
	return utils.New(0), nil
}

func (c *ContractStorage) CanTransfer(from, to, val *utils.Int) bool {
	return false
}

func (c *ContractStorage) GetCode(address *utils.Int) (code []byte, err error) {
	if contractName, err := c.Ctx.Get(address.String(), []byte(protocol.ContractAddress)); err == nil {
		if contractVersion, err := c.Ctx.Get(address.String(), []byte(protocol.ContractVersion)); err == nil {
			versionedByteCodeKey := append([]byte(protocol.ContractByteCode), contractVersion...)
			code, err = c.Ctx.Get(string(contractName), versionedByteCodeKey)
			return code, err
		}
	}
	log.Error("failed to get other contract  code :", err.Error())
	return nil, err
}

func (c *ContractStorage) GetCodeSize(address *utils.Int) (size *utils.Int, err error) {
	if contractName, err := c.Ctx.Get(address.String(), []byte(protocol.ContractAddress)); err == nil {
		if contractVersion, err := c.Ctx.Get(address.String(), []byte(protocol.ContractVersion)); err == nil {
			versionedByteCodeKey := append([]byte(protocol.ContractByteCode), contractVersion...)
			code, err := c.Ctx.Get(string(contractName), versionedByteCodeKey)
			return utils.New(int64(len(code))), err
		}
	}
	log.Error("failed to get other conteact  code size :", err.Error())
	return nil, err
}

func (c *ContractStorage) GetCodeHash(address *utils.Int) (codeHase *utils.Int, err error) {
	if contractName, err := c.Ctx.Get(address.String(), []byte(protocol.ContractAddress)); err == nil {
		if contractVersion, err := c.Ctx.Get(address.String(), []byte(protocol.ContractVersion)); err == nil {
			versionedByteCodeKey := append([]byte(protocol.ContractByteCode), contractVersion...)
			code, err := c.Ctx.Get(string(contractName), versionedByteCodeKey)
			hash := utils.Keccak256(code)
			i := utils.New(0)
			i.SetBytes(hash)
			return i, err
		}
	}
	log.Error("failed to get other conteact  code hash :", err.Error())
	return nil, err
}

func (c *ContractStorage) GetBlockHash(block *utils.Int) (*utils.Int, error) {
	currentHight := c.Ctx.GetBlockHeight() - 1
	high := utils.MinI(currentHight, block.Int64())
	Block, err := c.Ctx.GetBlockchainStore().GetBlock(high)
	if err != nil {
		return utils.New(0), err
	}
	hash, err := utils.HashBytesToEVMInt(Block.GetHeader().GetBlockHash())
	if err != nil {
		return utils.New(0), err
	}
	return hash, nil
}

func (c *ContractStorage) CreateAddress(caller *utils.Int, tx environment.Transaction) *utils.Int {
	//in seal abc smart assets application, we always create fixed contract address.
	return c.CreateFixedAddress(caller, nil, tx)
}

func (c *ContractStorage) CreateFixedAddress(caller *utils.Int, salt *utils.Int, tx environment.Transaction) *utils.Int {
	data := append(caller.Bytes(), tx.TxHash...)
	if salt != nil {
		data = append(data, salt.Bytes()...)
	}
	return utils.MakeAddress(data)
}
func (c *ContractStorage) Load(n string, k string) (*utils.Int, error) {
	val, err := c.Ctx.Get(n, []byte(k))
	if err != nil {
		return nil, err
	}
	r := utils.New(0)
	r.SetBytes(val)
	return r, err
}

func (c ContractStorage) Store(address string, key string, val []byte) {
	c.Ctx.Put(address, []byte(key), val)
}
