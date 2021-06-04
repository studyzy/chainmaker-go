/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package utils

import "math/big"

const DecBase = 10

// BigInteger wrapper for big.Int
type BigInteger struct {
	Value *big.Int
}

func NewBigInteger(value string) *BigInteger {
	var val big.Int
	newVal, ok := val.SetString(value, DecBase)
	if ok {
		return &BigInteger{
			Value: newVal,
		}
	}
	return nil
}

func NewZeroBigInteger() *BigInteger {
	return &BigInteger{
		Value: big.NewInt(0),
	}
}

func (x *BigInteger) Add(y *BigInteger) {
	x.Value = x.Value.Add(x.Value, y.Value)
}

func (x *BigInteger) Sub(y *BigInteger) {
	x.Value = x.Value.Sub(x.Value, y.Value)
}

// Cmp compares x and y and returns:
//
//   -1 if x <  y
//    0 if x == y
//   +1 if x >  y
func (x *BigInteger) Cmp(y *BigInteger) int {
	return x.Value.Cmp(y.Value)
}

func (x *BigInteger) String() string {
	return x.Value.String()
}

func Sub(x, y *BigInteger) *BigInteger {
	z := NewBigInteger(x.String())
	z.Sub(y)
	return z
}

func Sum(x, y *BigInteger) *BigInteger {
	z := NewZeroBigInteger()
	z.Add(x)
	z.Add(y)
	return z
}
