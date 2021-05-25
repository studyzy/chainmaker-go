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

package utils

import (
	"errors"
	"fmt"
)

var ErrStackUnderFlow = errors.New("stack under flow")
var ErrStackOverFlow = errors.New("stack over flow")
var ErrStorageNotInitialized = errors.New("storage not initialized")
var ErrInvalidEVMInstance = errors.New("invalid EVM instance")
var ErrReturnDataCopyOutOfBounds = errors.New("return data copy out of bounds")
var ErrJumpOutOfBounds = errors.New("jump out of range")
var ErrInvalidJumpDest = errors.New("invalid jump dest")
var ErrJumpToNoneOpCode = errors.New("jump to non-OpCode")
var ErrOutOfGas = errors.New("out of gas")
var ErrInsufficientBalance = errors.New("insufficient balance")
var ErrWriteProtection = errors.New("write protection")

var ErrBN256BadPairingInput = errors.New("bn256 bad pairing input")

func InvalidOpCode(code byte) error {
	return errors.New(fmt.Sprintf("invalid op code: 0x%X", code))
}

func NoSuchDataInTheStorage(err error) error {
	return errors.New("no such data in the storage: " + err.Error())
}

var ErrOutOfMemory = errors.New("out of memory")
