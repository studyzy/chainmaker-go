/*
 Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 SPDX-License-Identifier: Apache-2.0
*/
package crosstranscation

import (
	"fmt"

	"chainmaker.org/chainmaker-go/vm/native/common"
	"chainmaker.org/chainmaker/common/serialize"
	commonPb "chainmaker.org/chainmaker/pb-go/common"
	"chainmaker.org/chainmaker/pb-go/syscontract"
	"chainmaker.org/chainmaker/protocol"
	"github.com/pkg/errors"
)

type cacheKey []byte

var (
	paramCrossID      = syscontract.CrossParamKey_crossID.String()
	paramExecData     = syscontract.CrossParamKey_execData.String()
	paramRollbackData = syscontract.CrossParamKey_rollbackData.String()
	paramProofKey     = syscontract.CrossParamKey_proofKey.String()
	paramTxProof      = syscontract.CrossParamKey_txProof.String()

	paramContract   = syscontract.CrossParamKey_contract.String()
	paramMethod     = syscontract.CrossParamKey_method.String()
	paramCallParams = syscontract.CrossParamKey_params.String()
)

type CrossTransactionContract struct {
	methods map[string]common.ContractFunc
	log     protocol.Logger
}

func NewCrossTransactionContract(log protocol.Logger) *CrossTransactionContract {
	return &CrossTransactionContract{
		log:     log,
		methods: registerPrivateComputeContractMethods(log),
	}
}

func (c *CrossTransactionContract) GetMethod(methodName string) common.ContractFunc {
	return c.methods[methodName]
}

type CrossTransactionRuntime struct {
	log   protocol.Logger
	cache *cache
}

func registerPrivateComputeContractMethods(log protocol.Logger) map[string]common.ContractFunc {
	queryMethodMap := make(map[string]common.ContractFunc, 64)
	crossTransactionRuntime := &CrossTransactionRuntime{
		log: log,
		cache: &cache{
			ExecParamKey:     cacheKey("exec_param"),
			RollbackParamKey: cacheKey("rollback_param"),
			StateKey:         cacheKey("state"),
			ProofPreKey:      cacheKey("proof"),
		},
	}

	queryMethodMap[syscontract.CrossTransactionFunction_EXECUTE.String()] = crossTransactionRuntime.Execute
	queryMethodMap[syscontract.CrossTransactionFunction_COMMIT.String()] = crossTransactionRuntime.Commit
	queryMethodMap[syscontract.CrossTransactionFunction_ROLLBACK.String()] = crossTransactionRuntime.Rollback
	queryMethodMap[syscontract.CrossTransactionFunction_READ_STATE.String()] = crossTransactionRuntime.ReadState
	queryMethodMap[syscontract.CrossTransactionFunction_SAVE_PROOF.String()] = crossTransactionRuntime.SaveProof
	queryMethodMap[syscontract.CrossTransactionFunction_READ_PROOF.String()] = crossTransactionRuntime.ReadProof

	return queryMethodMap
}

func (r *CrossTransactionRuntime) Execute(ctx protocol.TxSimContext, params map[string][]byte) ([]byte, error) {
	err := checkParams(params, paramCrossID, paramExecData, paramRollbackData)
	if err != nil {
		r.log.Errorf("CrossTransactionRuntime.Execute checkParams param error: [%v]", err)
		return nil, err
	}
	//获取参数crossID
	crossID := params[paramCrossID]
	executeData := params[paramExecData]
	rollbackData := params[paramRollbackData]
	//检测crossID对应状态是否存在，存在的话直接返回error，不存在进行状态初始化
	state := r.cache.GetCrossState(ctx, crossID)
	if state != syscontract.CrossTxState_NonExist {
		r.log.Infof("crossID [%s] state is [%s], repeated tx execution", crossID, state.String())
		return nil, fmt.Errorf("crossID [%s] repeated tx execution ", crossID)
	}
	r.cache.SetCrossState(ctx, crossID, syscontract.CrossTxState_Init)
	//存储执行数据
	r.cache.Set(ctx, crossID, r.cache.ExecParamKey, executeData)
	//探测一下回滚数据是否可用
	if _, err := parseContractCallParams(crossID, rollbackData); err != nil {
		r.log.Errorf("crossID [%s] parse rollback params error: [%v]", crossID, err)
		return nil, errors.WithMessage(err, "rollback params parse")
	}
	r.cache.Set(ctx, crossID, r.cache.RollbackParamKey, rollbackData)

	callResp, err := callBusinessContract(ctx, crossID, executeData)
	//调用失败，退出
	if err != nil {
		r.log.Errorf("crossID [%s] call execute business contract error: [%v]", crossID, err)
		return nil, err
	}
	//执行结果OK, 则
	if contractProcessSuccess(callResp) {
		r.cache.SetCrossState(ctx, crossID, syscontract.CrossTxState_ExecOK)
		return callResp.Result, nil
	}
	r.cache.SetCrossState(ctx, crossID, syscontract.CrossTxState_ExecFail)
	return nil, errors.New(callResp.Message)
}

func (r *CrossTransactionRuntime) Commit(ctx protocol.TxSimContext, params map[string][]byte) ([]byte, error) {
	err := checkParams(params, paramCrossID)
	if err != nil {
		r.log.Errorf("CrossTransactionRuntime.Commit checkParams param error: [%v]", err)
		return nil, err
	}
	//获取参数crossID
	crossID := params[paramCrossID]
	state := r.cache.GetCrossState(ctx, crossID)
	if state != syscontract.CrossTxState_ExecOK {
		err = fmt.Errorf("crossID [%s] tx's state is [%s], cannot be committed", crossID, state.String())
		r.log.Info(err)
		return nil, err
	}
	return nil, r.cache.SetCrossState(ctx, crossID, syscontract.CrossTxState_CommitOK)
}

func (r *CrossTransactionRuntime) Rollback(ctx protocol.TxSimContext, params map[string][]byte) ([]byte, error) {
	err := checkParams(params, paramCrossID)
	if err != nil {
		r.log.Errorf("CrossTransactionRuntime.Rollback checkParams param error: [%v]", err)
		return nil, err
	}
	//获取参数crossID
	crossID := params[paramCrossID]
	state := r.cache.GetCrossState(ctx, crossID)
	r.log.Info("crossID [%s] state is [%s]", crossID, state.String())
	switch state {
	case syscontract.CrossTxState_RollbackOK: //应该有个message去表示[]byte("已回滚,重复回滚")
		return nil, nil
	case syscontract.CrossTxState_ExecFail:
		r.cache.SetCrossState(ctx, crossID, syscontract.CrossTxState_RollbackOK)
		return nil, nil
	case syscontract.CrossTxState_ExecOK, syscontract.CrossTxState_RollbackFail:
		result, err := r.rollback(ctx, crossID)
		if err != nil {
			r.log.Error("crossID [%s] rollback failed:[%v]", crossID, err)
			return nil, err
		}
		return result.Result, nil
	default:
		return nil, fmt.Errorf("crossID [%s] tx's state is [%s], cannot be rollback", crossID, state.String())
	}
}

func (r *CrossTransactionRuntime) SaveProof(ctx protocol.TxSimContext, params map[string][]byte) ([]byte, error) {
	err := checkParams(params, paramCrossID, paramProofKey, paramTxProof)
	if err != nil {
		r.log.Errorf("CrossTransactionRuntime.SaveProof checkParams param error: [%v]", err)
		return nil, err
	}
	//获取参数crossID
	crossID := params[paramCrossID]
	//获取参数proofKey
	proofKey := params[paramProofKey]
	//获取参数TxProof
	proof := params[paramTxProof]
	//检测是否已经存储proof 是则返回存储的proof， 否则存储
	ret, err := r.cache.GetProof(ctx, crossID, proofKey)
	if err == nil && len(ret) > 0 {
		r.log.Infof("crossID[%s] proofKey[%s] already exists", crossID, proofKey)
		return ret, nil
	}
	return proof, r.cache.SetProof(ctx, crossID, proofKey, proof)
}

func (r *CrossTransactionRuntime) ReadProof(ctx protocol.TxSimContext, params map[string][]byte) ([]byte, error) {
	err := checkParams(params, paramCrossID, paramProofKey)
	if err != nil {
		r.log.Errorf("CrossTransactionRuntime.ReadProof checkParams param error: [%v]", err)
		return nil, err
	}
	crossID := params[paramCrossID]
	//获取参数proofKey
	proofKey := params[paramProofKey]
	ret, err := r.cache.GetProof(ctx, crossID, proofKey)
	if err == nil && len(ret) > 0 {
		return ret, nil
	}
	return nil, fmt.Errorf("crossID [%s], proof_key [%s]'s proof is not exist", crossID, proofKey)
}

func (r *CrossTransactionRuntime) ReadState(ctx protocol.TxSimContext, params map[string][]byte) ([]byte, error) {
	err := checkParams(params, paramCrossID)
	if err != nil {
		return nil, err
	}
	//获取参数crossID
	crossID := params[paramCrossID]
	state := r.cache.GetCrossState(ctx, crossID)
	if state == syscontract.CrossTxState_NonExist {
		return nil, fmt.Errorf("crossID [%s] transaction is not exist", paramCrossID)
	}
	result := syscontract.CrossState{
		State: state,
	}
	return result.Marshal()
}

//仲裁
func (r *CrossTransactionRuntime) Arbitrate(ctx protocol.TxSimContext, params map[string][]byte) ([]byte, error) {
	return nil, nil
}

//func (r *CrossTransactionRuntime) genCrossResult(code syscontract.CrossCallCode, message string, data []byte) ([]byte, error) {
//	result := syscontract.CrossCallResult{
//		Code:    code,
//		Message: message,
//		Result:  data,
//	}
//	return result.Marshal()
//}

func (r *CrossTransactionRuntime) rollback(ctx protocol.TxSimContext, crossID []byte) (*commonPb.ContractResult, error) {
	rollbackParams, err := r.cache.Get(ctx, crossID, r.cache.RollbackParamKey)
	if err != nil {
		return nil, err
	}
	result, err := callBusinessContract(ctx, crossID, rollbackParams)
	if err != nil {
		return nil, err
	}
	if contractProcessSuccess(result) {
		r.cache.SetCrossState(ctx, crossID, syscontract.CrossTxState_RollbackOK)
		return result, nil
	}
	r.cache.SetCrossState(ctx, crossID, syscontract.CrossTxState_RollbackFail)
	return nil, errors.New(result.Message)
}

func checkParams(params map[string][]byte, keys ...string) error {
	if params == nil {
		return fmt.Errorf("params is nil")
	}
	for _, key := range keys {
		if v, ok := params[key]; !ok {
			return fmt.Errorf("params has no such key: [%s]", key)
		} else if len(v) == 0 {
			fmt.Errorf("param [%s] is invalid: value is nil", key)
		}
	}
	return nil
}

type Contract struct {
	Name   string
	Method string
	Params map[string][]byte
}

func parseContractCallParams(crossID, in []byte) (contract *Contract, err error) {
	params := &codec{serialize.NewEasyCodecWithItems(serialize.EasyUnmarshal(in))}
	contract = &Contract{}
	contract.Name, err = params.GetString(paramContract)
	if err != nil {
		return nil, fmt.Errorf("crossID [%s] parse contract name from params error: [%v]", string(crossID), err)
	}
	contract.Method, err = params.GetString(paramMethod)
	if err != nil {
		return nil, fmt.Errorf("crossID [%s] parse method name from params error: [%v]", string(crossID), err)
	}
	paramsBz, err := params.GetBytes(paramCallParams)
	if err != nil {
		return nil, fmt.Errorf("crossID [%s] parse method params from params error: [%v]", string(crossID), err)
	}
	contract.Params = serialize.NewEasyCodecWithBytes(paramsBz).ToMap()
	return
}

type codec struct {
	*serialize.EasyCodec
}

func (c *codec) GetBytes(key string) ([]byte, error) {
	item, err := c.GetItem(key, serialize.EasyKeyType_USER)
	if err == nil {
		if item.ValueType == serialize.EasyValueType_BYTES {
			return item.Value.([]byte), nil
		} else if item.ValueType == serialize.EasyValueType_STRING {
			return []byte(item.Value.(string)), nil
		}
		errors.New("value type not bytes")
	}
	return nil, errors.New("not found key")
}

func (c *codec) GetString(key string) (string, error) {
	item, err := c.GetItem(key, serialize.EasyKeyType_USER)
	if err == nil {
		if item.ValueType == serialize.EasyValueType_BYTES {
			return string(item.Value.([]byte)), nil
		} else if item.ValueType == serialize.EasyValueType_STRING {
			return item.Value.(string), nil
		}
		errors.New("value type not string")
	}
	return "", errors.New("not found key")
}

func callBusinessContract(ctx protocol.TxSimContext, crossID, params []byte) (*commonPb.ContractResult, error) {
	contract, err := parseContractCallParams(crossID, params)
	if err != nil {
		return nil, err
	}
	result, code := ctx.CallContract(&commonPb.Contract{Name: contract.Name}, contract.Method, nil, contract.Params, 0, commonPb.TxType_INVOKE_CONTRACT)
	if code != commonPb.TxStatusCode_SUCCESS {
		return nil, errors.New(code.String())
	}
	return result, nil
}

func contractProcessSuccess(result *commonPb.ContractResult) bool {
	return result != nil && result.Code == 0
}

type cache struct {
	ExecParamKey     cacheKey
	RollbackParamKey cacheKey
	StateKey         cacheKey
	ProofPreKey      cacheKey
}

func (c *cache) GetCrossState(ctx protocol.TxSimContext, crossID []byte) syscontract.CrossTxState {
	ret, err := c.Get(ctx, crossID, c.StateKey)
	if err != nil || len(ret) == 0 {
		return syscontract.CrossTxState_NonExist
	}
	return syscontract.CrossTxState(ret[0])
}

func (c *cache) SetCrossState(ctx protocol.TxSimContext, crossID []byte, state syscontract.CrossTxState) error {
	return c.Set(ctx, crossID, c.StateKey, []byte{byte(state)})
}

func (c *cache) GetProof(ctx protocol.TxSimContext, crossID []byte, proofKey []byte) ([]byte, error) {
	key := c.genKey(c.ProofPreKey, proofKey)
	return c.Get(ctx, crossID, key)
}

func (c *cache) SetProof(ctx protocol.TxSimContext, crossID []byte, proofKey []byte, proof []byte) error {
	key := c.genKey(c.ProofPreKey, proofKey)
	return c.Set(ctx, crossID, key, proof)
}

func (c *cache) genKey(crossID []byte, suffix []byte) []byte {
	key := make([]byte, len(crossID)+len(suffix)+1)
	i := copy(key, crossID)
	key[i] = '_'
	copy(key[i+1:], suffix)
	return key
}

func (c *cache) genName(crossID []byte) string {
	return syscontract.SystemContract_Cross_Transaction.String() + "/" + string(crossID)
}

func (c *cache) Get(ctx protocol.TxSimContext, crossID []byte, key []byte) ([]byte, error) {
	//key := c.genKey(crossID, suffix)
	return ctx.Get(c.genName(crossID), key)
}
func (c *cache) Set(ctx protocol.TxSimContext, crossID []byte, key []byte, value []byte) error {
	//key := c.genKey(crossID, suffix)
	return ctx.Put(c.genName(crossID), key, value)
}
