/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package native

import (
	"chainmaker.org/chainmaker-go/logger"
	acPb "chainmaker.org/chainmaker-go/pb/protogo/accesscontrol"
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/protocol"
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
)

const (
	Owner = "GMx5CwXvH9FyGwD5CbHsCXfM6XmAyzjb9iVRDiYBTxdB"
	TransferTo = "4yp3FUSrc1jyCgHMXswPeSE9N4Dnys1Hsg3NtBbzu2F4"
	TransferFrom = "9WqpbNL8q2qfo67WMjEB1iFrKb1VsGRgim4eW49R5fhj"
	Decimals = "18"
	TotalSupply = "100000000"
	TransferValue = "1000000"
	TransferBigValue = "3000000"
	ApproveValue  = "2000000"

)

var isFromAccount = false

func TestBigInteger(t *testing.T) {
	bigInteger := NewBigInteger("1024000000000000000000000000000000000000000000")
	require.NotNil(t, bigInteger)
	bigInteger.Add(NewBigInteger("1024"))
	require.Equal(t, "1024000000000000000000000000000000000000001024", bigInteger.String())
	bigInteger.Sub(NewBigInteger("1024"))
	require.Equal(t, "1024000000000000000000000000000000000000000000", bigInteger.String())
}

func TestDPoSRuntime_Owner(t *testing.T) {
	dPoSRuntime, txSimContext := initEnv(t)
	// 获取owner
	result, err := dPoSRuntime.Owner(txSimContext, nil)
	require.Nil(t, err)
	require.Equal(t, string(result), Owner)
}

func TestDPoSRuntime_Decimals(t *testing.T) {
	dPoSRuntime, txSimContext := initEnv(t)
	result, err := dPoSRuntime.Decimals(txSimContext, nil)
	require.Nil(t, err)
	require.Equal(t, string(result), Decimals)
}

func TestDPoSRuntime_Mint(t *testing.T) {
	initEnv(t)
}

func TestDPoSRuntime_Transfer(t *testing.T) {
	dPoSRuntime, txSimContext := initEnv(t)
	// 从owner中转出
	params := make(map[string]string, 32)
	params[paramNameTo] = TransferTo
	params[paramNameValue] = TransferValue
	result, err := dPoSRuntime.Transfer(txSimContext, params)
	require.Nil(t, err)
	require.Equal(t, string(result), "99000000")
}

func TestDPoSRuntime_BalanceOf(t *testing.T) {
	dPoSRuntime, txSimContext := initEnv(t)
	// 从owner中转出
	params := make(map[string]string, 32)
	params[paramNameTo] = TransferTo
	params[paramNameValue] = TransferValue
	result, err := dPoSRuntime.Transfer(txSimContext, params)
	require.Nil(t, err)
	require.Equal(t, string(result), "99000000")
	// 查询owner的balance
	params = make(map[string]string, 32)
	params[paramNameOwner] = Owner
	result, err = dPoSRuntime.BalanceOf(txSimContext, params)
	require.Nil(t, err)
	require.Equal(t, string(result), "99000000")
	// 查询to的balance
	params = make(map[string]string, 32)
	params[paramNameOwner] = TransferTo
	result, err = dPoSRuntime.BalanceOf(txSimContext, params)
	require.Nil(t, err)
	require.Equal(t, string(result), TransferValue)
}

func TestDPoSRuntime_TransferOwnership(t *testing.T) {
	dPoSRuntime, txSimContext := initEnv(t)
	params := make(map[string]string, 32)
	params[paramNameTo] = TransferTo
	result, err := dPoSRuntime.TransferOwnership(txSimContext, params)
	require.Nil(t, err)
	require.Equal(t, string(result), TransferTo)
	// 查询新的owner
	result, err = dPoSRuntime.Owner(txSimContext, nil)
	require.Nil(t, err)
	require.Equal(t, string(result), TransferTo)
}


func TestDPoSRuntime_Approve(t *testing.T) {
	dPoSRuntime, txSimContext := initEnv(t)
	params := make(map[string]string, 32)
	params[paramNameTo] = TransferFrom
	params[paramNameValue] = ApproveValue
	result, err := dPoSRuntime.Approve(txSimContext, params)
	require.Nil(t, err)
	require.Equal(t, string(result), ApproveValue)
}

func TestDPoSRuntime_Allowance(t *testing.T) {
	dPoSRuntime, txSimContext := initEnv(t)
	params := make(map[string]string, 32)
	params[paramNameTo] = TransferFrom
	params[paramNameValue] = ApproveValue
	result, err := dPoSRuntime.Approve(txSimContext, params)
	require.Nil(t, err)
	require.Equal(t, string(result), ApproveValue)
	params = make(map[string]string, 32)
	params[paramNameFrom] = Owner
	params[paramNameTo] = TransferFrom
	result, err = dPoSRuntime.Allowance(txSimContext, params)
	require.Nil(t, err)
	require.Equal(t, string(result), ApproveValue)
}

func TestDPoSRuntime_TransferFrom(t *testing.T) {
	dPoSRuntime, txSimContext := initEnv(t)
	// 首先进行转账，给From用户指定金额
	params := make(map[string]string, 32)
	params[paramNameTo] = TransferFrom
	params[paramNameValue] = TransferBigValue
	result, err := dPoSRuntime.Transfer(txSimContext, params)
	require.Nil(t, err)
	require.Equal(t, string(result), "97000000")
	// 使用owner用户进行transferFrom操作
	params = make(map[string]string, 32)
	params[paramNameFrom] = TransferFrom
	params[paramNameTo] = TransferTo
	params[paramNameValue] = TransferValue
	// owner用户直接操作from账号会报错，因为未批准
	result, err = dPoSRuntime.TransferFrom(txSimContext, params)
	require.NotNil(t, err)
	require.Nil(t, result)
	// 进行批准
	params = make(map[string]string, 32)
	params[paramNameTo] = Owner
	params[paramNameValue] = ApproveValue
	// 此处需要进行设置，表示本次操作是由
	isFromAccount = true
	result, err = dPoSRuntime.Approve(txSimContext, params)
	require.Nil(t, err)
	require.Equal(t, string(result), ApproveValue)
	isFromAccount = false
	// 再次进行转账操作
	params = make(map[string]string, 32)
	params[paramNameFrom] = TransferFrom
	params[paramNameTo] = TransferTo
	params[paramNameValue] = TransferValue
	result, err = dPoSRuntime.TransferFrom(txSimContext, params)
	require.Nil(t, err)
	require.Equal(t, string(result), "2000000")
	// 再次进行allowance查询
	params = make(map[string]string, 32)
	params[paramNameFrom] = TransferFrom
	params[paramNameTo] = Owner
	result, err = dPoSRuntime.Allowance(txSimContext, params)
	require.Nil(t, err)
	require.Equal(t, string(result), "1000000")
}

func TestOwnerCert(t *testing.T) {
	address, err := parseUserAddress(ownerCert())
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(address)
	require.Equal(t, address, Owner)
}

func initEnv(t *testing.T) (*DPoSRuntime, *TxSimContextMock) {
	dPoSRuntime := NewDPoSRuntime(NewLogger())
	// 进行准备工作
	txSimContext := NewTxSimContextMock()
	err := dPoSRuntime.setOwner(txSimContext, Owner)
	require.Nil(t, err)
	err = dPoSRuntime.setDecimals(txSimContext, Decimals)
	require.Nil(t, err)
	// 增发指定数量的token
	params := make(map[string]string, 32)
	params[paramNameTo] = Owner
	params[paramNameValue] = TotalSupply
	result, err := dPoSRuntime.Mint(txSimContext, params)
	require.Nil(t, err)
	require.Equal(t, string(result), TotalSupply)
	return dPoSRuntime, txSimContext
}

func NewLogger() *logger.CMLogger{
	cmLogger := logger.GetLogger("DPoS")
	return cmLogger
}

type TxSimContextMock struct {
	cache *CacheMock
}

func NewTxSimContextMock() *TxSimContextMock {
	return &TxSimContextMock{
		cache: NewCacheMock(),
	}
}

func (t *TxSimContextMock) Get(name string, key []byte) ([]byte, error) {
	return t.cache.Get(name, string(key)), nil
}

func (t *TxSimContextMock) Put(name string, key []byte, value []byte) error {
	t.cache.Put(name, string(key), value)
	return nil
}

func (t *TxSimContextMock) PutRecord(contractName string, value []byte) {
	panic("implement me")
}

func (t *TxSimContextMock) Del(name string, key []byte) error {
	panic("implement me")
}

func (t *TxSimContextMock) Select(name string, startKey []byte, limit []byte) (protocol.StateIterator, error) {
	panic("implement me")
}

func (t *TxSimContextMock) CallContract(contractId *commonPb.ContractId, method string, byteCode []byte, parameter map[string]string, gasUsed uint64, refTxType commonPb.TxType) (*commonPb.ContractResult, commonPb.TxStatusCode) {
	panic("implement me")
}

func (t *TxSimContextMock) GetCurrentResult() []byte {
	panic("implement me")
}

func (t *TxSimContextMock) GetTx() *commonPb.Transaction {
	panic("implement me")
}

func (t *TxSimContextMock) GetBlockHeight() int64 {
	panic("implement me")
}

func (t *TxSimContextMock) GetBlockProposer() []byte {
	panic("implement me")
}

func (t *TxSimContextMock) GetTxResult() *commonPb.Result {
	panic("implement me")
}

func (t *TxSimContextMock) SetTxResult(result *commonPb.Result) {
	panic("implement me")
}

func (t *TxSimContextMock) GetTxRWSet() *commonPb.TxRWSet {
	panic("implement me")
}

func (t *TxSimContextMock) GetCreator(namespace string) *acPb.SerializedMember {
	panic("implement me")
}

func (t *TxSimContextMock) GetSender() *acPb.SerializedMember {
	// 生成证书
	member := &acPb.SerializedMember {
		OrgId: "wx-org1.chainmaker.org",
		MemberInfo: ownerCert(),
		IsFullCert: true,
	}
	return member
}

func (t *TxSimContextMock) GetBlockchainStore() protocol.BlockchainStore {
	panic("implement me")
}

func (t *TxSimContextMock) GetAccessControl() (protocol.AccessControlProvider, error) {
	panic("implement me")
}

func (t *TxSimContextMock) GetChainNodesInfoProvider() (protocol.ChainNodesInfoProvider, error) {
	panic("implement me")
}

func (t *TxSimContextMock) GetTxExecSeq() int {
	panic("implement me")
}

func (t *TxSimContextMock) SetTxExecSeq(i int) {
	panic("implement me")
}

func (t *TxSimContextMock) GetDepth() int {
	panic("implement me")
}

func (t *TxSimContextMock) SetStateSqlHandle(i int32, rows protocol.SqlRows) {
	panic("implement me")
}

func (t *TxSimContextMock) GetStateSqlHandle(i int32) (protocol.SqlRows, bool) {
	panic("implement me")
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

func realKey(name, key string) string {
	return fmt.Sprintf(KeyFormat, name, key)
}