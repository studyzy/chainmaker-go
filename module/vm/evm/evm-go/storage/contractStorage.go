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
	"chainmaker.org/chainmaker-go/logger"
	"chainmaker.org/chainmaker-go/utils"
	"chainmaker.org/chainmaker/common/evmutils"
	"chainmaker.org/chainmaker/protocol"
)

var log = logger.GetLogger(logger.MODULE_VM)

type ContractStorage struct {
	ResultCache     ResultCache
	ExternalStorage IExternalStorage
	readOnlyCache   readOnlyCache
	Ctx             protocol.TxSimContext
	BlockHash       *evmutils.Int
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

func (c *ContractStorage) GetBalance(address *evmutils.Int) (*evmutils.Int, error) {
	return evmutils.New(0), nil
}

func (c *ContractStorage) CanTransfer(from, to, val *evmutils.Int) bool {
	return false
}

func (c *ContractStorage) GetCode(address *evmutils.Int) (code []byte, err error) {
	return utils.GetContractBytecode(c.Ctx.Get, address.String())
	//if contractName, err := c.Ctx.Get(address.String(), []byte(protocol.ContractAddress)); err == nil {
	//	versionKey := []byte(protocol.ContractVersion + address.String())
	//	if contractVersion, err := c.Ctx.Get(syscontract.SystemContract_CONTRACT_MANAGE.String(), versionKey); err == nil {
	//		versionedByteCodeKey := append([]byte(protocol.ContractByteCode), contractName...)
	//		versionedByteCodeKey = append(versionedByteCodeKey, contractVersion...)
	//		code, err = c.Ctx.Get(syscontract.SystemContract_CONTRACT_MANAGE.String(), versionedByteCodeKey)
	//		return code, err
	//	} else {
	//		log.Errorf("failed to get other contract byte code version, address [%s] , error :", address.String(), err.Error())
	//	}
	//}
	//log.Error("failed to get other contract  code :", err.Error())
	//return nil, err
}

func (c *ContractStorage) GetCodeSize(address *evmutils.Int) (size *evmutils.Int, err error) {
	code, err := c.GetCode(address)
	if err != nil {
		log.Error("failed to get other contract code size :", err.Error())
		return nil, err
	}
	return evmutils.New(int64(len(code))), err
}

func (c *ContractStorage) GetCodeHash(address *evmutils.Int) (codeHase *evmutils.Int, err error) {
	code, err := c.GetCode(address)
	if err != nil {
		log.Error("failed to get other contract code hash :", err.Error())
		return nil, err
	}
	hash := evmutils.Keccak256(code)
	i := evmutils.New(0)
	i.SetBytes(hash)
	return i, err
	return evmutils.New(int64(len(code))), err
}

func (c *ContractStorage) GetBlockHash(block *evmutils.Int) (*evmutils.Int, error) {
	currentHight := c.Ctx.GetBlockHeight() - 1
	high := evmutils.MinI(int64(currentHight), block.Int64())
	Block, err := c.Ctx.GetBlockchainStore().GetBlock(uint64(high))
	if err != nil {
		return evmutils.New(0), err
	}
	hash, err := evmutils.HashBytesToEVMInt(Block.GetHeader().GetBlockHash())
	if err != nil {
		return evmutils.New(0), err
	}
	return hash, nil
}

func (c *ContractStorage) CreateAddress(caller *evmutils.Int, tx environment.Transaction) *evmutils.Int {
	//in seal abc smart assets application, we always create fixed contract address.
	return c.CreateFixedAddress(caller, nil, tx)
}

func (c *ContractStorage) CreateFixedAddress(caller *evmutils.Int, salt *evmutils.Int, tx environment.Transaction) *evmutils.Int {
	data := append(caller.Bytes(), tx.TxHash...)
	if salt != nil {
		data = append(data, salt.Bytes()...)
	}
	return evmutils.MakeAddress(data)
}
func (c *ContractStorage) Load(n string, k string) (*evmutils.Int, error) {
	val, err := c.Ctx.Get(n, []byte(k))
	if err != nil {
		return nil, err
	}
	r := evmutils.New(0)
	r.SetBytes(val)
	return r, err
}

func (c ContractStorage) Store(address string, key string, val []byte) {
	c.Ctx.Put(address, []byte(key), val)
}
