/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package dposmgr

import (
	"fmt"
	"testing"

	"chainmaker.org/chainmaker/protocol/v2/mock"
	"github.com/golang/mock/gomock"

	"chainmaker.org/chainmaker/logger/v2"
	acPb "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	"chainmaker.org/chainmaker/protocol/v2"
	"github.com/stretchr/testify/require"
)

var (
	Owner            = []byte("GMx5CwXvH9FyGwD5CbHsCXfM6XmAyzjb9iVRDiYBTxdB")
	TransferTo       = []byte("4yp3FUSrc1jyCgHMXswPeSE9N4Dnys1Hsg3NtBbzu2F4")
	TransferFrom     = []byte("9WqpbNL8q2qfo67WMjEB1iFrKb1VsGRgim4eW49R5fhj")
	Decimals         = []byte("18")
	TotalSupply      = []byte("100000000")
	TransferValue    = []byte("1000000")
	TransferBigValue = []byte("3000000")
	ApproveValue     = []byte("2000000")
)

var isFromAccount = false

func TestDPoSRuntime_Owner(t *testing.T) {
	dPoSRuntime, txSimContext, fn := initEnv(t)
	defer fn()
	// 获取owner
	result, err := dPoSRuntime.Owner(txSimContext, nil)
	require.Nil(t, err)
	require.Equal(t, string(result), string(Owner))
}

func TestDPoSRuntime_Decimals(t *testing.T) {
	dPoSRuntime, txSimContext, fn := initEnv(t)
	defer fn()
	result, err := dPoSRuntime.Decimals(txSimContext, nil)
	t.Logf("Result:%s", string(result))
	require.Nil(t, err)
	require.Equal(t, string(result), string(Decimals))
}

func TestDPoSRuntime_Mint(t *testing.T) {
	initEnv(t)
}

func TestDPoSRuntime_Transfer(t *testing.T) {
	dPoSRuntime, txSimContext, fn := initEnv(t)
	defer fn()
	// 从owner中转出
	params := make(map[string][]byte, 32)
	params[paramNameTo] = TransferTo
	params[paramNameValue] = TransferValue
	result, err := dPoSRuntime.Transfer(txSimContext, params)
	require.Nil(t, err)
	require.Equal(t, string(result), "99000000")
}

func TestDPoSRuntime_BalanceOf(t *testing.T) {
	dPoSRuntime, txSimContext, fn := initEnv(t)
	defer fn()
	// 从owner中转出
	params := make(map[string][]byte, 32)
	params[paramNameTo] = TransferTo
	params[paramNameValue] = TransferValue
	result, err := dPoSRuntime.Transfer(txSimContext, params)
	require.Nil(t, err)
	require.Equal(t, string(result), "99000000")
	// 查询owner的balance
	params = make(map[string][]byte, 32)
	params[paramNameOwner] = Owner
	result, err = dPoSRuntime.BalanceOf(txSimContext, params)
	require.Nil(t, err)
	require.Equal(t, string(result), "99000000")
	// 查询to的balance
	params = make(map[string][]byte, 32)
	params[paramNameOwner] = TransferTo
	result, err = dPoSRuntime.BalanceOf(txSimContext, params)
	require.Nil(t, err)
	require.EqualValues(t, result, TransferValue)
}

func TestDPoSRuntime_TransferOwnership(t *testing.T) {
	dPoSRuntime, txSimContext, fn := initEnv(t)
	defer fn()
	params := make(map[string][]byte, 32)
	params[paramNameTo] = TransferTo
	result, err := dPoSRuntime.TransferOwnership(txSimContext, params)
	require.Nil(t, err)
	require.EqualValues(t, string(result), TransferTo)
	// 查询新的owner
	result, err = dPoSRuntime.Owner(txSimContext, nil)
	require.Nil(t, err)
	require.EqualValues(t, string(result), TransferTo)
}

func TestDPoSRuntime_Approve(t *testing.T) {
	dPoSRuntime, txSimContext, fn := initEnv(t)
	defer fn()
	params := make(map[string][]byte, 32)
	params[paramNameTo] = TransferFrom
	params[paramNameValue] = ApproveValue
	result, err := dPoSRuntime.Approve(txSimContext, params)
	require.Nil(t, err)
	require.EqualValues(t, string(result), ApproveValue)
}

func TestDPoSRuntime_Allowance(t *testing.T) {
	dPoSRuntime, txSimContext, fn := initEnv(t)
	defer fn()
	params := make(map[string][]byte, 32)
	params[paramNameTo] = TransferFrom
	params[paramNameValue] = ApproveValue
	result, err := dPoSRuntime.Approve(txSimContext, params)
	require.Nil(t, err)
	require.EqualValues(t, string(result), ApproveValue)
	params = make(map[string][]byte, 32)
	params[paramNameFrom] = Owner
	params[paramNameTo] = TransferFrom
	result, err = dPoSRuntime.Allowance(txSimContext, params)
	require.Nil(t, err)
	require.EqualValues(t, string(result), ApproveValue)
}

func TestDPoSRuntime_TransferFrom(t *testing.T) {
	dPoSRuntime, txSimContext, fn := initEnv(t)
	defer fn()
	// 首先进行转账，给From用户指定金额
	params := make(map[string][]byte, 32)
	params[paramNameTo] = TransferFrom
	params[paramNameValue] = TransferBigValue
	result, err := dPoSRuntime.Transfer(txSimContext, params)
	require.Nil(t, err)
	require.Equal(t, string(result), "97000000")
	// 使用owner用户进行transferFrom操作
	params = make(map[string][]byte, 32)
	params[paramNameFrom] = TransferFrom
	params[paramNameTo] = TransferTo
	params[paramNameValue] = TransferValue
	// owner用户直接操作from账号会报错，因为未批准
	result, err = dPoSRuntime.TransferFrom(txSimContext, params)
	require.NotNil(t, err)
	require.Nil(t, result)
	// 进行批准
	params = make(map[string][]byte, 32)
	params[paramNameTo] = Owner
	params[paramNameValue] = ApproveValue
	// 此处需要进行设置，表示本次操作是由
	isFromAccount = true
	result, err = dPoSRuntime.Approve(txSimContext, params)
	require.Nil(t, err)
	require.EqualValues(t, string(result), ApproveValue)
	isFromAccount = false
	// 再次进行转账操作
	params = make(map[string][]byte, 32)
	params[paramNameFrom] = TransferFrom
	params[paramNameTo] = TransferTo
	params[paramNameValue] = TransferValue
	result, err = dPoSRuntime.TransferFrom(txSimContext, params)
	require.Nil(t, err)
	require.EqualValues(t, string(result), "2000000")
	// 再次进行allowance查询
	params = make(map[string][]byte, 32)
	params[paramNameFrom] = TransferFrom
	params[paramNameTo] = Owner
	result, err = dPoSRuntime.Allowance(txSimContext, params)
	require.Nil(t, err)
	require.EqualValues(t, string(result), "1000000")
}

func TestOwnerCert(t *testing.T) {
	address, err := parseUserAddress(ownerCert())
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(address)
	require.EqualValues(t, address, Owner)
}

func initEnv(t *testing.T) (*DPoSRuntime, protocol.TxSimContext, func()) {
	dPoSRuntime := NewDPoSRuntime(NewLogger())
	ctrl := gomock.NewController(t)
	txSimContext := mock.NewMockTxSimContext(ctrl)

	cache := NewCacheMock()
	txSimContext.EXPECT().Put(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(name string, key []byte, value []byte) error {
			cache.Put(name, string(key), value)
			return nil
		}).AnyTimes()
	txSimContext.EXPECT().GetSender().DoAndReturn(func() *acPb.Member {
		return &acPb.Member{
			OrgId:      "wx-org1.chainmaker.org",
			MemberInfo: ownerCert(),
			//IsFullCert: true,
		}
	}).AnyTimes()
	txSimContext.EXPECT().Get(gomock.Any(), gomock.Any()).DoAndReturn(
		func(name string, key []byte) ([]byte, error) {
			return cache.Get(name, string(key)), nil
		}).AnyTimes()

	err := dPoSRuntime.setOwner(txSimContext, string(Owner))
	require.Nil(t, err)
	err = dPoSRuntime.setDecimals(txSimContext, string(Decimals))
	require.Nil(t, err)
	// 增发指定数量的token
	params := make(map[string][]byte, 32)
	params[paramNameTo] = Owner
	params[paramNameValue] = TotalSupply
	result, err := dPoSRuntime.Mint(txSimContext, params)
	require.Nil(t, err)
	require.Equal(t, string(result), string(TotalSupply))
	return dPoSRuntime, txSimContext, func() { ctrl.Finish() }
}

func NewLogger() protocol.Logger {
	cmLogger := logger.GetLogger("DPoS")
	return cmLogger
}

func ownerCert() []byte {
	var certStr string
	if isFromAccount {
		certStr = "-----BEGIN CERTIFICATE-----\nMIICijCCAi+gAwIBAgIDBS9vMAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ\nMA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt\nb3JnMS5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD\nExljYS53eC1vcmcxLmNoYWlubWFrZXIub3JnMB4XDTIwMTIwODA2NTM0M1oXDTI1\nMTIwNzA2NTM0M1owgZExCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw\nDgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmcxLmNoYWlubWFrZXIub3Jn\nMQ8wDQYDVQQLEwZjbGllbnQxLDAqBgNVBAMTI2NsaWVudDEuc2lnbi53eC1vcmcx\nLmNoYWlubWFrZXIub3JnMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE56xayRx0\n/a8KEXPxRfiSzYgJ/sE4tVeI/ZbjpiUX9m0TCJX7W/VHdm6WeJLOdCDuLLNvjGTy\nt8LLyqyubJI5AKN7MHkwDgYDVR0PAQH/BAQDAgGmMA8GA1UdJQQIMAYGBFUdJQAw\nKQYDVR0OBCIEIMjAiM2eMzlQ9HzV9ePW69rfUiRZVT2pDBOMqM4WVJSAMCsGA1Ud\nIwQkMCKAIDUkP3EcubfENS6TH3DFczH5dAnC2eD73+wcUF/bEIlnMAoGCCqGSM49\nBAMCA0kAMEYCIQCWUHL0xisjQoW+o6VV12pBXIRJgdeUeAu2EIjptSg2GAIhAIxK\nLXpHIBFxIkmWlxUaanCojPSZhzEbd+8LRrmhEO8n\n-----END CERTIFICATE-----\n"
	} else {
		certStr = "-----BEGIN CERTIFICATE-----\nMIIChzCCAi2gAwIBAgIDAwGbMAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ\nMA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt\nb3JnMS5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD\nExljYS53eC1vcmcxLmNoYWlubWFrZXIub3JnMB4XDTIwMTIwODA2NTM0M1oXDTI1\nMTIwNzA2NTM0M1owgY8xCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw\nDgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmcxLmNoYWlubWFrZXIub3Jn\nMQ4wDAYDVQQLEwVhZG1pbjErMCkGA1UEAxMiYWRtaW4xLnNpZ24ud3gtb3JnMS5j\naGFpbm1ha2VyLm9yZzBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABORqoYNAw8ax\n9QOD94VaXq1dCHguarSKqAruEI39dRkm8Vu2gSHkeWlxzvSsVVqoN6ATObi2ZohY\nKYab2s+/QA2jezB5MA4GA1UdDwEB/wQEAwIBpjAPBgNVHSUECDAGBgRVHSUAMCkG\nA1UdDgQiBCDZOtAtHzfoZd/OQ2Jx5mIMgkqkMkH4SDvAt03yOrRnBzArBgNVHSME\nJDAigCA1JD9xHLm3xDUukx9wxXMx+XQJwtng+9/sHFBf2xCJZzAKBggqhkjOPQQD\nAgNIADBFAiEAiGjIB8Wb8mhI+ma4F3kCW/5QM6tlxiKIB5zTcO5E890CIBxWDICm\nAod1WZHJajgnDQ2zEcFF94aejR9dmGBB/P//\n-----END CERTIFICATE-----"
	}
	return []byte(certStr)
}

const KeyFormat = "%s/%s"

func realKey(name, key string) string {
	return fmt.Sprintf(KeyFormat, name, key)
}

type CacheMock struct {
	content map[string][]byte
}

func NewCacheMock() *CacheMock {
	return &CacheMock{
		content: make(map[string][]byte, 64),
	}
}

func (c *CacheMock) Put(name, key string, value []byte) {
	c.content[realKey(name, key)] = value
}

func (c *CacheMock) Get(name, key string) []byte {
	return c.content[realKey(name, key)]
}

func (c *CacheMock) Del(name, key string) error {
	delete(c.content, realKey(name, key))
	return nil
}

func (c *CacheMock) GetByKey(key string) []byte {
	return c.content[key]
}

func (c *CacheMock) Keys() []string {
	sc := make([]string, 0)
	for k := range c.content {
		sc = append(sc, k)
	}
	return sc
}
