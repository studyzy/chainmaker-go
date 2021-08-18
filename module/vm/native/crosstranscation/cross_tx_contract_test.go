/*
 Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 SPDX-License-Identifier: Apache-2.0
*/
package crosstranscation

import (
	"fmt"
	"testing"

	"chainmaker.org/chainmaker/pb-go/accesscontrol"
	pbac "chainmaker.org/chainmaker/pb-go/accesscontrol"

	configPb "chainmaker.org/chainmaker/pb-go/config"
	"github.com/gogo/protobuf/proto"

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

const (
	contractName   = "tx"
	rollbackMethod = "rollback"
)

func Test_Execute(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	txSimContext := mock.NewMockTxSimContext(ctrl)
	txSimContext.EXPECT().Get(gomock.Eq(store.genName(crossID)), gomock.Not(nil)).DoAndReturn(
		func(name string, key []byte) ([]byte, error) {
			return gCache.Get(name, key)
		},
	)
	txSimContext.EXPECT().Put(gomock.Eq(store.genName(crossID)), gomock.Not(nil), gomock.Not(nil)).DoAndReturn(
		func(name string, key []byte, value []byte) error {
			gCache.Put(name, key, value)
			return nil
		},
	).AnyTimes()
	txSimContext.EXPECT().CallContract(gomock.Not(nil), gomock.Eq("exec"), gomock.Nil(), gomock.Any(), gomock.Eq(uint64(0)), gomock.Eq(commonPb.TxType_INVOKE_CONTRACT)).DoAndReturn(
		func(contract *commonPb.Contract, method string, byteCode []byte, parameter map[string][]byte, gasUsed uint64, refTxType commonPb.TxType) (*commonPb.ContractResult, commonPb.TxStatusCode) {
			if contract.Name == contractName {
				return &commonPb.ContractResult{
					Code:   0,
					Result: []byte("hello world"),
				}, commonPb.TxStatusCode_SUCCESS
			}
			return nil, commonPb.TxStatusCode_CONTRACT_FAIL
		},
	)
	exec := crossContract.GetMethod(syscontract.CrossTransactionFunction_EXECUTE.String())
	params := genExecParams(crossID)
	ret, err := exec(txSimContext, params)
	require.Nil(t, err)
	t.Log(string(ret))
}
func genExecParams(crossID []byte) map[string][]byte {
	eParams := map[string][]byte{
		paramCrossID:    crossID,
		paramContract:   []byte("tx"),
		paramMethod:     []byte("exec"),
		paramCallParams: serialize.EasyMarshal(serialize.ParamsMapToEasyCodecItem(map[string][]byte{})),
	}
	rParams := map[string][]byte{
		paramCrossID:    crossID,
		paramContract:   []byte("tx"),
		paramMethod:     []byte("rollback"),
		paramCallParams: serialize.EasyMarshal(serialize.ParamsMapToEasyCodecItem(map[string][]byte{})),
	}

	return map[string][]byte{
		paramCrossID:      crossID,
		paramExecData:     serialize.EasyMarshal(serialize.ParamsMapToEasyCodecItem(eParams)),
		paramRollbackData: serialize.EasyMarshal(serialize.ParamsMapToEasyCodecItem(rParams)),
	}
}

func Test_Commit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	txSimContext := mock.NewMockTxSimContext(ctrl)
	txSimContext.EXPECT().Get(gomock.Eq(store.genName(crossID)), gomock.Not(nil)).DoAndReturn(
		func(name string, key []byte) ([]byte, error) {
			return gCache.Get(name, key)
		},
	)
	txSimContext.EXPECT().Put(gomock.Eq(store.genName(crossID)), gomock.Not(nil), gomock.Not(nil)).DoAndReturn(
		func(name string, key []byte, value []byte) error {
			gCache.Put(name, key, value)
			return nil
		},
	).AnyTimes()

	commit := crossContract.GetMethod(syscontract.CrossTransactionFunction_COMMIT.String())
	params := map[string][]byte{paramCrossID: crossID}
	ret, err := commit(txSimContext, params)
	require.Nil(t, err)
	t.Log(string(ret))
}

func Test_Rollback(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	gCache.Put(store.genName(crossID), store.StateKey, []byte{byte(syscontract.CrossTxState_EXECUTE_OK)})
	txSimContext := mock.NewMockTxSimContext(ctrl)
	txSimContext.EXPECT().Get(gomock.Eq(store.genName(crossID)), gomock.Not(nil)).DoAndReturn(
		func(name string, key []byte) ([]byte, error) {
			return gCache.Get(name, key)
		},
	).AnyTimes()
	txSimContext.EXPECT().Put(gomock.Eq(store.genName(crossID)), gomock.Not(nil), gomock.Not(nil)).DoAndReturn(
		func(name string, key []byte, value []byte) error {
			gCache.Put(name, key, value)
			return nil
		},
	).AnyTimes()
	//if state == syscontract.CrossTxState_ExecOK || state == syscontract.CrossTxState_RollbackFail {
	txSimContext.EXPECT().CallContract(gomock.Not(nil), gomock.Eq("rollback"), gomock.Nil(), gomock.Any(), gomock.Eq(uint64(0)), gomock.Eq(commonPb.TxType_INVOKE_CONTRACT)).DoAndReturn(
		func(contract *commonPb.Contract, method string, byteCode []byte, parameter map[string][]byte, gasUsed uint64, refTxType commonPb.TxType) (*commonPb.ContractResult, commonPb.TxStatusCode) {
			if contract.Name == contractName && method == rollbackMethod {
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
	params := map[string][]byte{paramCrossID: crossID}
	ret, err := call(txSimContext, params)
	require.Nil(t, err)
	t.Log(string(ret))
}

func Test_ReadState(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	txSimContext := mock.NewMockTxSimContext(ctrl)
	txSimContext.EXPECT().Get(gomock.Eq(store.genName(crossID)), gomock.Not(nil)).DoAndReturn(
		func(name string, key []byte) ([]byte, error) {
			return gCache.Get(name, key)
		},
	).AnyTimes()

	call := crossContract.GetMethod(syscontract.CrossTransactionFunction_READ_STATE.String())
	params := map[string][]byte{paramCrossID: crossID}
	ret, err := call(txSimContext, params)
	require.Nil(t, err)
	result := syscontract.CrossState{}
	result.Unmarshal(ret)
	t.Log(result)
}

func Test_SaveProof(t *testing.T) {
	proofKey := gProofKey
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	txSimContext := mock.NewMockTxSimContext(ctrl)
	txSimContext.EXPECT().Get(gomock.Eq(store.genName(crossID)), gomock.Not(nil)).DoAndReturn(
		func(name string, key []byte) ([]byte, error) {
			return gCache.Get(name, key)
		},
	).AnyTimes()

	txSimContext.EXPECT().Put(gomock.Eq(store.genName(crossID)), gomock.Not(nil), gomock.Not(nil)).DoAndReturn(
		func(name string, key []byte, value []byte) error {
			gCache.Put(name, key, value)
			return nil
		},
	).AnyTimes()

	call := crossContract.GetMethod(syscontract.CrossTransactionFunction_SAVE_PROOF.String())
	params := map[string][]byte{paramCrossID: crossID, paramProofKey: proofKey, paramTxProof: []byte("中国奥运健儿加油")}
	ret, err := call(txSimContext, params)
	require.Nil(t, err)
	t.Log(string(ret))
}

func Test_ReadProof(t *testing.T) {
	proofKey := gProofKey
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	txSimContext := mock.NewMockTxSimContext(ctrl)
	txSimContext.EXPECT().Get(gomock.Eq(store.genName(crossID)), gomock.Not(nil)).DoAndReturn(
		func(name string, key []byte) ([]byte, error) {
			return gCache.Get(name, key)
		},
	).AnyTimes()

	call := crossContract.GetMethod(syscontract.CrossTransactionFunction_READ_PROOF.String())
	params := map[string][]byte{paramCrossID: crossID, paramProofKey: proofKey}
	ret, err := call(txSimContext, params)
	require.Nil(t, err)
	t.Log(string(ret))
}

func Test_Arbitrate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	gCache.Put(store.genName(crossID), store.StateKey, []byte{byte(syscontract.CrossTxState_EXECUTE_OK)})
	chainConfig := &configPb.ChainConfig{
		Consensus: &configPb.ConsensusConfig{
			Nodes: []*configPb.OrgConfig{
				{
					NodeId: []string{"hello", "9HdRUYfrzSER2EbY8b1NFuVSFp4cKNznE1ucRgtHoK6s"},
				},
			},
		},
	}
	pbccPayload, _ := proto.Marshal(chainConfig)
	gCache.Put(syscontract.SystemContract_CHAIN_CONFIG.String(), []byte(syscontract.SystemContract_CHAIN_CONFIG.String()), pbccPayload)
	txSimContext := mock.NewMockTxSimContext(ctrl)
	txSimContext.EXPECT().Get(gomock.Not(nil), gomock.Not(nil)).DoAndReturn(
		func(name string, key []byte) ([]byte, error) {
			return gCache.Get(name, key)
		},
	).AnyTimes()
	txSimContext.EXPECT().Put(gomock.Eq(store.genName(crossID)), gomock.Not(nil), gomock.Not(nil)).DoAndReturn(
		func(name string, key []byte, value []byte) error {
			gCache.Put(name, key, value)
			return nil
		},
	).AnyTimes()

	txSimContext.EXPECT().GetSender().DoAndReturn(
		func() *pbac.Member {
			return &pbac.Member{
				MemberType: accesscontrol.MemberType_CERT,
				MemberInfo: []byte(`-----BEGIN CERTIFICATE-----
MIICnTCCAYUCCQDNeorE6MGDgjANBgkqhkiG9w0BAQUFADANMQswCQYDVQQKDAJD
QTAeFw0yMTA4MDUxMjQ2MDdaFw0zNTA0MTQxMjQ2MDdaMBQxEjAQBgNVBAMMCWxv
Y2FsaG9zdDCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAMeCwVXCFqCg
jJWgNpOxPRkOHAg0sX5RMzgQ+5T313d6Qr5WPDa/rQEhNr0kf63m5x51l/hz0Sgs
pMYJdZEm70vj8dBPz8HJ/+WhMFd89Rj9lo8zqj3sK6jWqkmSdewzoin3r6Cx2FHz
RT6T1c0wo+pwWeikARfW+UD+u+yticQkkUrziooFQrrukU+FwAM8q3ZXEj32Asqn
/6rkGsdwUTs3E8M+nD+D9chmEuxOk3QJ6RqTAPEFehfDeTfniWOw/oEKmUlJ9Qqa
zBO6Yk2UMvuQsXlEK0ynXNGT6OFNPOQf6N1WFWHSWV/d6reJLGgt+D6Ld9mBSAuL
XvXbilf5VVkCAwEAATANBgkqhkiG9w0BAQUFAAOCAQEAAI2BeblnAFw+0rhNEGln
Kpieomz+7lBYOiXzLEf9nqcFiYsUL7YQjflXfxFTiPES+Q2L+Tyxm8IhILHhy2h8
ICl60gIAAZAu/M2hclOekzLA7W7s3kyh40s2eKMh4E+4dJtUqEd+dmyElhCJlLNA
D2IzK4Bz/FvnSxjgv2psjjq/g41mrsm0+J5ZqeCLbaKoFqA7+QA7f/dkHwPVrZ8n
9ip8iY4YVB6jIiDRpnjmPD8P9s7ztFVqQ46a9wShWzZYCaSq2whxyjcakKE4PxSm
MmUZz2wJML7wFsZw+IZ1MH28g3IRc67NcHiV7TX97kqwcTrfD10aV8UZn/+8aDQ5
+g==
-----END CERTIFICATE-----`),
			}
		},
	).AnyTimes()

	txSimContext.EXPECT().CallContract(gomock.Not(nil), gomock.Not(nil), gomock.Nil(), gomock.Any(), gomock.Eq(uint64(0)), gomock.Eq(commonPb.TxType_INVOKE_CONTRACT)).DoAndReturn(
		func(contract *commonPb.Contract, method string, byteCode []byte, parameter map[string][]byte, gasUsed uint64, refTxType commonPb.TxType) (*commonPb.ContractResult, commonPb.TxStatusCode) {
			if contract.Name == "tx" {
				return &commonPb.ContractResult{
					Code:   0,
					Result: []byte("hello world"),
				}, commonPb.TxStatusCode_SUCCESS
			}
			return nil, commonPb.TxStatusCode_CONTRACT_FAIL
		},
	).AnyTimes()

	call := crossContract.GetMethod(syscontract.CrossTransactionFunction_ARBITRATE.String())
	params := map[string][]byte{paramCrossID: crossID, paramArbitrateCmd: []byte(syscontract.CrossArbitrateCmd_ROLLBACK_CMD.String())}
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
