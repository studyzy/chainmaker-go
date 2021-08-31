/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

//import (
//	evm "chainmaker.org/chainmaker/common/v2/evmutils"
//	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
//	"encoding/hex"
//	"github.com/ethereum/go-ethereum/accounts/abi"
//	"github.com/stretchr/testify/require"
//	"math/big"
//	"strings"
//	"testing"
//)
//
//const AbiJson = "[{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"address\"}],\"name\":\"balances\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_newBalance\",\"type\":\"uint256\"},{\"name\":\"_to\",\"type\":\"address\"}],\"name\":\"updateBalance\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"newBalance\",\"type\":\"uint256\"}],\"name\":\"updateMyBalance\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"c\",\"type\":\"string\"}],\"name\":\"paramTest2\",\"outputs\":[{\"name\":\"success\",\"type\":\"bool\"},{\"name\":\"abcdef\",\"type\":\"string\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_to\",\"type\":\"address\"},{\"name\":\"_value\",\"type\":\"uint256\"}],\"name\":\"transfer\",\"outputs\":[{\"name\":\"success\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"a\",\"type\":\"bool\"},{\"name\":\"b\",\"type\":\"int256\"},{\"name\":\"c\",\"type\":\"string\"}],\"name\":\"paramTest\",\"outputs\":[{\"name\":\"success\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"name\":\"_addressFounder\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"_from\",\"type\":\"address\"},{\"indexed\":true,\"name\":\"_to\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"_value\",\"type\":\"uint256\"}],\"name\":\"Transfer\",\"type\":\"event\"}]"
//const certSki = "9dbf916da9f5ae892e0991d82b30e1366fe7aa76a6e18767783c9fa3c0921f3b"
//const evmAddr = "5924544a57e26b52231597aaa5e0374748c0a127"
//
//func TestMakePairs(t *testing.T) {
//	isEvm = true
//	method = "updateBalance"
//	pairs := []*commonPb.KeyValuePair{
//		{
//			Key:   "_newBalance",
//			Value: "100000002",
//		},
//		{
//			Key:   "_to",
//			Value: "../../config/wx-org1/certs/user/client1/client1.tls.crt", // [ski str] or [cert path]
//		},
//	}
//
//	pairs, err := makePairs(method, AbiJson, pairs)
//	require.Nil(t, err)
//
//	myAbi, err := abi.JSON(strings.NewReader(AbiJson))
//	require.Nil(t, err)
//	addr, err := evm.MakeAddressFromHex(certSki)
//	require.Nil(t, err)
//	dataByte, err := myAbi.Pack(method, big.NewInt(100000002), evm.BigToAddress(addr))
//	require.Nil(t, err)
//	data := hex.EncodeToString(dataByte)
//
//	for _, pair := range pairs {
//		require.EqualValues(t, data, pair.Value)
//	}
//}
