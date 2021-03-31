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
	"encoding/hex"
	"golang.org/x/crypto/sha3"
)

const (
	hashLength    = 32
	AddressLength = 20
)

type Address [AddressLength]byte

func (a *Address) SetBytes(b []byte) {
	if len(b) > len(a) {
		b = b[len(b)-AddressLength:]
	}
	copy(a[AddressLength-len(b):], b)
}

var (
	BlankHash = make([]byte, hashLength, hashLength)
	ZeroHash  = Keccak256(nil)
)

func EVMIntToHashBytes(i *Int) [hashLength]byte {
	iBytes := i.Bytes()
	iLen := len(iBytes)

	var hash [hashLength]byte
	if iLen > hashLength {
		copy(hash[:], iBytes[iLen-hashLength:])
	} else {
		copy(hash[hashLength-iLen:], iBytes)
	}

	return hash
}

func EthHashBytesToEVMInt(hash [hashLength]byte) (*Int, error) {

	i := New(0)
	i.SetBytes(hash[:])

	return i, nil
}
func HashBytesToEVMInt(hash []byte) (*Int, error) {
	i := New(0)
	i.SetBytes(hash[:])
	return i, nil
}
func BytesDataToEVMIntHash(data []byte) *Int {
	var hashBytes []byte
	srcLen := len(data)
	if srcLen < hashLength {
		hashBytes = LeftPaddingSlice(data, hashLength)
	} else {
		hashBytes = data[:hashLength]
	}

	i := New(0)
	i.SetBytes(hashBytes)

	return i
}

func GetDataFrom(src []byte, offset uint64, size uint64) []byte {
	ret := make([]byte, size, size)
	dLen := uint64(len(src))
	if dLen < offset {
		return ret
	}

	end := offset + size
	if dLen < end {
		end = dLen
	}

	copy(ret, src[offset:end])
	return ret
}

func LeftPaddingSlice(src []byte, toSize int) []byte {
	sLen := len(src)
	if toSize <= sLen {
		return src
	}

	ret := make([]byte, toSize, toSize)
	copy(ret[toSize-sLen:], src)

	return ret
}

func RightPaddingSlice(src []byte, toSize int) []byte {
	sLen := len(src)
	if toSize <= sLen {
		return src
	}

	ret := make([]byte, toSize, toSize)
	copy(ret, src)

	return ret
}

func Keccak256(data []byte) []byte {
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write(data)
	return hasher.Sum(nil)
}

// LeftPadBytes zero-pads slice to the left up to length l.
func LeftPadBytes(slice []byte, l int) []byte {
	if l <= len(slice) {
		return slice
	}

	padded := make([]byte, l)
	copy(padded[l-len(slice):], slice)

	return padded
}

func MakeAddressFromHex(str string) (*Int, error) {
	data, err := hex.DecodeString(str)
	if err != nil {
		return nil, err
	}
	return MakeAddress(data), nil
}
func MakeAddressFromString(str string) (*Int, error) {
	return MakeAddress([]byte(str)), nil
}

func MakeAddress(data []byte) *Int {
	address := Keccak256(data)
	addr := hex.EncodeToString(address)[24:]
	return FromHexString(addr)
}

func BytesToAddress(b []byte) Address {
	var a Address
	a.SetBytes(b)
	return a
}

func StringToAddress(s string) Address { return BytesToAddress([]byte(s)) }

func BigToAddress(b *Int) Address { return BytesToAddress(b.Bytes()) }

func HexToAddress(s string) Address { return BytesToAddress(FromHex(s)) }

func FromHex(s string) []byte {
	if Has0xPrefix(s) {
		s = s[2:]
	}
	if len(s)%2 == 1 {
		s = "0" + s
	}
	return Hex2Bytes(s)
}

func Has0xPrefix(str string) bool {
	return len(str) >= 2 && str[0] == '0' && (str[1] == 'x' || str[1] == 'X')
}

func Hex2Bytes(str string) []byte {
	h, _ := hex.DecodeString(str)
	return h
}
