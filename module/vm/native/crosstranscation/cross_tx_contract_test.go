/*
 Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 SPDX-License-Identifier: Apache-2.0
*/
package crosstranscation

import (
	"fmt"
	"testing"

	"chainmaker.org/chainmaker-go/logger"

	"chainmaker.org/chainmaker/common/serialize"
	commonPb "chainmaker.org/chainmaker/pb-go/common"
	"chainmaker.org/chainmaker/pb-go/syscontract"
	"chainmaker.org/chainmaker/protocol/mock"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

var (
	store = &cache{
		ExecParamKey:     cacheKey("exec_param"),
		RollbackParamKey: cacheKey("rollback_param"),
		StateKey:         cacheKey("state"),
		ProofPreKey:      cacheKey("proof"),
	}
	crossContract = NewCrossTransactionContract(logger.GetLogger("CrossTx"))
	gCache        = NewCacheMock()
	crossID       = []byte(uuid.New().String())
	gProofKey     = []byte("1233211234567")
)

func Test_Execute(t *testing.T) {
	CrossID := crossID
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	txSimContext := mock.NewMockTxSimContext(ctrl)
	txSimContext.EXPECT().Get(gomock.Eq(store.genName(CrossID)), gomock.Not(nil)).DoAndReturn(
		func(name string, key []byte) ([]byte, error) {
			return gCache.Get(name, key)
		},
	)
	txSimContext.EXPECT().Put(gomock.Eq(store.genName(CrossID)), gomock.Not(nil), gomock.Not(nil)).DoAndReturn(
		func(name string, key []byte, value []byte) error {
			gCache.Put(name, key, value)
			return nil
		},
	).AnyTimes()
	txSimContext.EXPECT().CallContract(gomock.Not(nil), gomock.Eq("exec"), gomock.Nil(), gomock.Any(), gomock.Eq(uint64(0)), gomock.Eq(commonPb.TxType_INVOKE_CONTRACT)).DoAndReturn(
		func(contract *commonPb.Contract, method string, byteCode []byte, parameter map[string][]byte, gasUsed uint64, refTxType commonPb.TxType) (*commonPb.ContractResult, commonPb.TxStatusCode) {
			if contract.Name == "tx" {
				return &commonPb.ContractResult{
					Code:   0,
					Result: []byte("hello world"),
				}, commonPb.TxStatusCode_SUCCESS
			}
			return nil, commonPb.TxStatusCode_CONTRACT_FAIL
		},
	)
	exec := crossContract.GetMethod(syscontract.CrossTransactionFunction_EXECUTE.String())
	params := genExecParams(CrossID)
	ret, err := exec(txSimContext, params)
	require.Nil(t, err)
	t.Log(string(ret))
}
func genExecParams(CrossID []byte) map[string][]byte {
	eParams := map[string][]byte{
		paramCrossID:    CrossID,
		paramContract:   []byte("tx"),
		paramMethod:     []byte("exec"),
		paramCallParams: serialize.EasyMarshal(serialize.ParamsMapToEasyCodecItem(map[string][]byte{})),
	}
	rParams := map[string][]byte{
		paramCrossID:    CrossID,
		paramContract:   []byte("tx"),
		paramMethod:     []byte("rollback"),
		paramCallParams: serialize.EasyMarshal(serialize.ParamsMapToEasyCodecItem(map[string][]byte{})),
	}

	return map[string][]byte{
		paramCrossID:      CrossID,
		paramExecData:     serialize.EasyMarshal(serialize.ParamsMapToEasyCodecItem(eParams)),
		paramRollbackData: serialize.EasyMarshal(serialize.ParamsMapToEasyCodecItem(rParams)),
	}
}

func Test_Commit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	CrossID := crossID
	txSimContext := mock.NewMockTxSimContext(ctrl)
	txSimContext.EXPECT().Get(gomock.Eq(store.genName(CrossID)), gomock.Not(nil)).DoAndReturn(
		func(name string, key []byte) ([]byte, error) {
			return gCache.Get(name, key)
		},
	)
	txSimContext.EXPECT().Put(gomock.Eq(store.genName(CrossID)), gomock.Not(nil), gomock.Not(nil)).DoAndReturn(
		func(name string, key []byte, value []byte) error {
			gCache.Put(name, key, value)
			return nil
		},
	).AnyTimes()

	commit := crossContract.GetMethod(syscontract.CrossTransactionFunction_COMMIT.String())
	params := map[string][]byte{paramCrossID: CrossID}
	ret, err := commit(txSimContext, params)
	require.Nil(t, err)
	t.Log(string(ret))
}

func Test_Rollback(t *testing.T) {
	CrossID := crossID
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	gCache.Put(store.genName(CrossID), store.StateKey, []byte{byte(syscontract.CrossTxState_ExecOK)})
	txSimContext := mock.NewMockTxSimContext(ctrl)
	txSimContext.EXPECT().Get(gomock.Eq(store.genName(CrossID)), gomock.Not(nil)).DoAndReturn(
		func(name string, key []byte) ([]byte, error) {
			return gCache.Get(name, key)
		},
	).AnyTimes()
	txSimContext.EXPECT().Put(gomock.Eq(store.genName(CrossID)), gomock.Not(nil), gomock.Not(nil)).DoAndReturn(
		func(name string, key []byte, value []byte) error {
			gCache.Put(name, key, value)
			return nil
		},
	).AnyTimes()
	//if state == syscontract.CrossTxState_ExecOK || state == syscontract.CrossTxState_RollbackFail {
	txSimContext.EXPECT().CallContract(gomock.Not(nil), gomock.Eq("rollback"), gomock.Nil(), gomock.Any(), gomock.Eq(uint64(0)), gomock.Eq(commonPb.TxType_INVOKE_CONTRACT)).DoAndReturn(
		func(contract *commonPb.Contract, method string, byteCode []byte, parameter map[string][]byte, gasUsed uint64, refTxType commonPb.TxType) (*commonPb.ContractResult, commonPb.TxStatusCode) {
			if contract.Name == "tx" && method == "rollback" {
				return &commonPb.ContractResult{
					Code:   0,
					Result: []byte("hello world"),
				}, commonPb.TxStatusCode_SUCCESS
			}
			return nil, commonPb.TxStatusCode_CONTRACT_FAIL
		},
	)
	//}

	call := crossContract.GetMethod(syscontract.CrossTransactionFunction_ROLLBACK.String())
	params := map[string][]byte{paramCrossID: CrossID}
	ret, err := call(txSimContext, params)
	require.Nil(t, err)
	t.Log(string(ret))
}

func Test_ReadState(t *testing.T) {
	CrossID := crossID
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	txSimContext := mock.NewMockTxSimContext(ctrl)
	txSimContext.EXPECT().Get(gomock.Eq(store.genName(CrossID)), gomock.Not(nil)).DoAndReturn(
		func(name string, key []byte) ([]byte, error) {
			return gCache.Get(name, key)
		},
	).AnyTimes()

	call := crossContract.GetMethod(syscontract.CrossTransactionFunction_READ_STATE.String())
	params := map[string][]byte{paramCrossID: CrossID}
	ret, err := call(txSimContext, params)
	require.Nil(t, err)
	result := syscontract.CrossState{}
	result.Unmarshal(ret)
	t.Log(result)
}

func Test_SaveProof(t *testing.T) {
	CrossID := crossID
	proofKey := gProofKey
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	txSimContext := mock.NewMockTxSimContext(ctrl)
	txSimContext.EXPECT().Get(gomock.Eq(store.genName(CrossID)), gomock.Not(nil)).DoAndReturn(
		func(name string, key []byte) ([]byte, error) {
			return gCache.Get(name, key)
		},
	).AnyTimes()

	txSimContext.EXPECT().Put(gomock.Eq(store.genName(CrossID)), gomock.Not(nil), gomock.Not(nil)).DoAndReturn(
		func(name string, key []byte, value []byte) error {
			gCache.Put(name, key, value)
			return nil
		},
	).AnyTimes()

	call := crossContract.GetMethod(syscontract.CrossTransactionFunction_SAVE_PROOF.String())
	params := map[string][]byte{paramCrossID: CrossID, paramProofKey: proofKey, paramTxProof: []byte("中国奥运健儿加油")}
	ret, err := call(txSimContext, params)
	require.Nil(t, err)
	t.Log(string(ret))
}

func Test_ReadProof(t *testing.T) {
	CrossID := crossID
	proofKey := gProofKey
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	txSimContext := mock.NewMockTxSimContext(ctrl)
	txSimContext.EXPECT().Get(gomock.Eq(store.genName(CrossID)), gomock.Not(nil)).DoAndReturn(
		func(name string, key []byte) ([]byte, error) {
			return gCache.Get(name, key)
		},
	).AnyTimes()

	call := crossContract.GetMethod(syscontract.CrossTransactionFunction_READ_PROOF.String())
	params := map[string][]byte{paramCrossID: CrossID, paramProofKey: proofKey}
	ret, err := call(txSimContext, params)
	require.Nil(t, err)
	t.Log(string(ret))
}

func realKey(name string, key []byte) string {
	return fmt.Sprintf("%s/%s", name, key)
}

type CacheMock struct {
	content map[string][]byte
}

func NewCacheMock() *CacheMock {
	return &CacheMock{
		content: make(map[string][]byte, 64),
	}
}

func (c *CacheMock) Put(name string, key, value []byte) {
	c.content[realKey(name, key)] = value
}

func (c *CacheMock) Get(name string, key []byte) ([]byte, error) {
	k := realKey(name, key)
	v, ok := c.content[k]
	if !ok {
		return nil, errors.New(k + " not exists")
	}
	return v, nil
}

func (c *CacheMock) Del(name string, key []byte) error {
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
