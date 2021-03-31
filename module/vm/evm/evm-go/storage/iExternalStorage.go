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

//Teh External Storage,provding a Storage for touching out of current evm
type IExternalStorage interface {
	GetBalance(address *utils.Int) (*utils.Int, error)
	GetCode(address *utils.Int) ([]byte, error)
	GetCodeSize(address *utils.Int) (*utils.Int, error)
	GetCodeHash(address *utils.Int) (*utils.Int, error)
	GetBlockHash(block *utils.Int) (*utils.Int, error)

	CreateAddress(caller *utils.Int, tx environment.Transaction) *utils.Int
	CreateFixedAddress(caller *utils.Int, salt *utils.Int, tx environment.Transaction) *utils.Int

	CanTransfer(from *utils.Int, to *utils.Int, amount *utils.Int) bool

	Load(n string, k string) (*utils.Int, error)
	Store(address string, key string, val []byte)
}
