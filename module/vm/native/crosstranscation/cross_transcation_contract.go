/*
 Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 SPDX-License-Identifier: Apache-2.0
*/
package crosstranscation

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"chainmaker.org/chainmaker-go/utils"

	"chainmaker.org/chainmaker/pb-go/accesscontrol"
	configPb "chainmaker.org/chainmaker/pb-go/config"

	"chainmaker.org/chainmaker-go/vm/native/common"
	"chainmaker.org/chainmaker/common/serialize"
	commonPb "chainmaker.org/chainmaker/pb-go/common"
	"chainmaker.org/chainmaker/pb-go/syscontract"
	"chainmaker.org/chainmaker/protocol"
	"github.com/mr-tron/base58"
	"github.com/pkg/errors"

	//"google.golang.org/protobuf/proto"
	"github.com/gogo/protobuf/proto"
)

type cacheKey []byte

var (
	crossTxContractName = syscontract.SystemContract_CROSS_TRANSACTION.String()

	paramCrossID      = "crossID"
	paramExecData     = "execData"
	paramRollbackData = "rollbackData"
	paramProofKey     = "proofKey"
	paramTxProof      = "txProof"
	paramArbitrateCmd = "command"

	paramContract   = "contract"
	paramMethod     = "method"
	paramCallParams = "params"
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
	queryMethodMap[syscontract.CrossTransactionFunction_ARBITRATE.String()] = crossTransactionRuntime.Arbitrate

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
	if state != syscontract.CrossTxState_NON_EXIST {
		r.log.Infof("crossID [%s] state is [%s], repeated tx execution", crossID, state.String())
		return nil, fmt.Errorf("crossID [%s] repeated tx execution ", crossID)
	}
	r.cache.SetCrossState(ctx, crossID, syscontract.CrossTxState_INIT)
	//存储执行数据
	r.cache.Set(ctx, crossID, r.cache.ExecParamKey, executeData)
	//探测一下回滚数据是否可用
	if _, err := parseContractCallParams(crossID, rollbackData); err != nil {
		r.log.Errorf("crossID [%s] parse rollback params error: [%v]", crossID, err)
		return nil, errors.WithMessage(err, "rollback params parse")
	}
	r.cache.Set(ctx, crossID, r.cache.RollbackParamKey, rollbackData)
	return r.execute(ctx, crossID, executeData)
}

func (r *CrossTransactionRuntime) execute(ctx protocol.TxSimContext, crossID, executeData []byte) ([]byte, error) {
	result, err := callBusinessContract(ctx, crossID, executeData)
	//调用失败，退出
	if err != nil {
		r.log.Errorf("crossID [%s] call execute business contract error: [%v]", crossID, err)
		return nil, err
	}
	r.log.Infof("executeCall result: %v", result)
	//执行结果OK, 则
	if contractProcessSuccess(result) {
		r.cache.SetCrossState(ctx, crossID, syscontract.CrossTxState_EXECUTE_OK)
		return result.Result, nil
	}
	r.cache.SetCrossState(ctx, crossID, syscontract.CrossTxState_EXECUTE_FAIL)
	return nil, errors.New(result.Message)
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
	if state != syscontract.CrossTxState_EXECUTE_OK {
		err = fmt.Errorf("crossID [%s] tx's state is [%s], cannot be committed", crossID, state.String())
		r.log.Info(err)
		return nil, err
	}
	return nil, r.cache.SetCrossState(ctx, crossID, syscontract.CrossTxState_COMMIT_OK)
}

func (r *CrossTransactionRuntime) commit(ctx protocol.TxSimContext, crossID []byte) ([]byte, error) {
	state := r.cache.GetCrossState(ctx, crossID)
	if state != syscontract.CrossTxState_EXECUTE_OK {
		err := fmt.Errorf("crossID [%s] tx's state is [%s], cannot be committed", crossID, state.String())
		r.log.Info(err)
		return nil, err
	}
	return nil, r.cache.SetCrossState(ctx, crossID, syscontract.CrossTxState_COMMIT_OK)
}

func (r *CrossTransactionRuntime) Rollback(ctx protocol.TxSimContext, params map[string][]byte) ([]byte, error) {
	err := checkParams(params, paramCrossID)
	if err != nil {
		r.log.Errorf("CrossTransactionRuntime.Rollback checkParams param error: [%v]", err)
		return nil, err
	}
	//获取参数crossID
	crossID := params[paramCrossID]
	return r.rollback(ctx, crossID)
}

func (r *CrossTransactionRuntime) rollback(ctx protocol.TxSimContext, crossID []byte) ([]byte, error) {
	state := r.cache.GetCrossState(ctx, crossID)
	r.log.Infof("crossID [%s] state is [%s]", crossID, state.String())
	switch state {
	case syscontract.CrossTxState_NON_EXIST:
		return []byte{}, nil
	case syscontract.CrossTxState_ROLLBACK_OK: //应该有个message去表示[]byte("已回滚,重复回滚")
		return []byte{}, nil
	case syscontract.CrossTxState_EXECUTE_FAIL:
		r.cache.SetCrossState(ctx, crossID, syscontract.CrossTxState_ROLLBACK_OK)
		return []byte{}, nil
	case syscontract.CrossTxState_EXECUTE_OK, syscontract.CrossTxState_ROLLBACK_FAIL:
		result, err := r.rollbackCall(ctx, crossID)
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
	r.log.Infof("SaveProof crossID[%s] proofKey[%s] proof [%s]", crossID, proofKey, proof)
	//检测是否已经存储proof 是则返回存储的proof， 否则存储
	ret, err := r.cache.GetProof(ctx, crossID, proofKey)
	if err == nil && len(ret) > 0 {
		r.log.Infof("crossID[%s] proofKey[%s] already exists: [%v]", crossID, proofKey, ret)
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
		r.log.Errorf("CrossTransactionRuntime.ReadState checkParams param error: [%v]", err)
		return nil, err
	}
	//获取参数crossID
	crossID := params[paramCrossID]
	state := r.cache.GetCrossState(ctx, crossID)
	if state == syscontract.CrossTxState_NON_EXIST {
		return nil, fmt.Errorf("crossID [%s] transaction is not exist", crossID)
	}
	result := syscontract.CrossState{
		State: state,
	}
	return result.Marshal()
}

//仲裁
func (r *CrossTransactionRuntime) Arbitrate(ctx protocol.TxSimContext, params map[string][]byte) ([]byte, error) {
	err := checkParams(params, paramCrossID, paramArbitrateCmd)
	if err != nil {
		r.log.Errorf("CrossTransactionRuntime.Arbitrate checkParams param error: [%v]", err)
		return nil, err
	}
	crossID := params[paramCrossID]
	cmd := string(params[paramArbitrateCmd])
	r.log.Infof("crossID [%s] arbitrate cmd is [%s]", crossID, cmd)
	ok, err := arbitrateAuth(ctx)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("authentication fail")
	}
	switch cmd {
	case syscontract.CrossArbitrateCmd_EXECUTE_CMD.String():
		return r.arbitrateExec(ctx, crossID)
	case syscontract.CrossArbitrateCmd_COMMIT_CMD.String():
		return r.arbitrateCommit(ctx, crossID)
	case syscontract.CrossArbitrateCmd_ROLLBACK_CMD.String():
		return r.arbitrateRollback(ctx, crossID)
	//case syscontract.CrossArbitrateCmd_AUTO_CMD.String():
	default:
		return nil, fmt.Errorf("unrecognized command:[%s]", cmd)
	}
}

func (r *CrossTransactionRuntime) arbitrateExec(ctx protocol.TxSimContext, crossID []byte) ([]byte, error) {
	switch r.cache.GetCrossState(ctx, crossID) {
	//case syscontract.CrossTxState_NON_EXIST:
	//	return nil, fmt.Errorf("crossID [%s] transaction is not exist", crossID)
	case syscontract.CrossTxState_NON_EXIST, syscontract.CrossTxState_INIT, syscontract.CrossTxState_EXECUTE_FAIL:
		execParams, err := r.cache.Get(ctx, crossID, r.cache.ExecParamKey)
		if err != nil {
			return nil, err
		}
		return r.execute(ctx, crossID, execParams)
	default:
		return []byte{}, nil
	}
}

func (r *CrossTransactionRuntime) arbitrateCommit(ctx protocol.TxSimContext, crossID []byte) ([]byte, error) {
	return r.commit(ctx, crossID)
}

func (r *CrossTransactionRuntime) arbitrateRollback(ctx protocol.TxSimContext, crossID []byte) ([]byte, error) {
	return r.rollback(ctx, crossID)
}

//func (r *CrossTransactionRuntime) genCrossResult(code syscontract.CrossCallCode, message string, data []byte) ([]byte, error) {
//	result := syscontract.CrossCallResult{
//		Code:    code,
//		Message: message,
//		Result:  data,
//	}
//	return result.Marshal()
//}

func (r *CrossTransactionRuntime) rollbackCall(ctx protocol.TxSimContext, crossID []byte) (*commonPb.ContractResult, error) {
	rollbackParams, err := r.cache.Get(ctx, crossID, r.cache.RollbackParamKey)
	if err != nil {
		return nil, err
	}
	result, err := callBusinessContract(ctx, crossID, rollbackParams)
	if err != nil {
		return nil, err
	}
	r.log.Infof("rollbackCall result: %v", result)
	if contractProcessSuccess(result) {
		r.cache.SetCrossState(ctx, crossID, syscontract.CrossTxState_ROLLBACK_OK)
		return result, nil
	}
	r.cache.SetCrossState(ctx, crossID, syscontract.CrossTxState_ROLLBACK_FAIL)
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

func arbitrateAuth(ctx protocol.TxSimContext) (bool, error) {
	nodeIDs, err := getAllOrgNodeIDS(ctx)
	if err != nil {
		return false, err
	}
	nodeID, err := loadSenderAddress(ctx)
	if err != nil {
		return false, err
	}
	for _, id := range nodeIDs {
		if id == nodeID {
			return true, nil
		}
	}
	return false, nil
}

func getAllOrgNodeIDS(ctx protocol.TxSimContext) ([]string, error) {
	//result, err := callContract(ctx, &Contract{
	//	Name:   syscontract.SystemContract_CHAIN_CONFIG.String(),
	//	Method: syscontract.ChainConfigFunction_GET_CHAIN_CONFIG.String(),
	//	Params: map[string][]byte{},
	//})
	//if !contractProcessSuccess(result) {
	//	return nil, fmt.Errorf("obtain chain config faile: [%s]", result.Message)
	//}
	chainConfigName := syscontract.SystemContract_CHAIN_CONFIG.String()
	bytes, err := ctx.Get(chainConfigName, []byte(chainConfigName))
	if err != nil {
		msg := fmt.Errorf("get chain config faile: [%v]", err)
		return nil, msg
	}

	chainConfig := &configPb.ChainConfig{}
	err = proto.Unmarshal(bytes, chainConfig)
	if err != nil {
		return nil, err
	}
	nodeIDs := make([]string, 0, len(chainConfig.Consensus.Nodes))
	for _, node := range chainConfig.Consensus.Nodes {
		nodeIDs = append(nodeIDs, node.NodeId...)
	}
	return nodeIDs, nil
}

func loadSenderAddress(txSimContext protocol.TxSimContext) (string, error) {
	sender := txSimContext.GetSender()
	if sender != nil {
		// 将sender转换为用户地址
		var member []byte
		if sender.MemberType == accesscontrol.MemberType_CERT {
			// 长证书
			member = sender.MemberInfo
		} else if sender.MemberType == accesscontrol.MemberType_CERT_HASH {
			// 短证书
			memberInfoHex := hex.EncodeToString(sender.MemberInfo)
			certInfo, err := getWholeCertInfo(txSimContext, memberInfoHex)
			if err != nil {
				return "", fmt.Errorf(
					"can not load whole cert info , contract[%s] member[%s]",
					crossTxContractName, memberInfoHex)
			}
			member = certInfo.Cert
		} else {
			return "", errors.New("invalid member type")
		}
		return parseUserAddress(member)
	}
	return "", fmt.Errorf("can not find sender from tx, contract[%s]", crossTxContractName)
}

// parseUserAddress
func parseUserAddress(member []byte) (string, error) {
	certificate, err := utils.ParseCert(member)
	if err != nil {
		msg := fmt.Errorf("parse cert failed, name[%s] err: %+v", crossTxContractName, err)
		return "", msg
	}
	pubKeyBytes, err := certificate.PublicKey.Bytes()
	if err != nil {
		msg := fmt.Errorf("load public key from cert failed, name[%s] err: %+v", crossTxContractName, err)
		return "", msg
	}
	// 转换为SHA-256
	addressBytes := sha256.Sum256(pubKeyBytes)
	return base58.Encode(addressBytes[:]), nil
}

func getWholeCertInfo(txSimContext protocol.TxSimContext, certHash string) (*commonPb.CertInfo, error) {
	certBytes, err := txSimContext.Get(syscontract.SystemContract_CERT_MANAGE.String(), []byte(certHash))
	if err != nil {
		return nil, err
	}
	return &commonPb.CertInfo{
		Hash: certHash,
		Cert: certBytes,
	}, nil
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
	return callContract(ctx, contract)
}

func callContract(ctx protocol.TxSimContext, contract *Contract) (*commonPb.ContractResult, error) {
	result, code := ctx.CallContract(&commonPb.Contract{Name: contract.Name}, contract.Method, nil, contract.Params, 0, commonPb.TxType_INVOKE_CONTRACT)
	if code != commonPb.TxStatusCode_SUCCESS {
		if result != nil {
			return nil, fmt.Errorf("invoke contract [%s/%s] %s %v", contract.Name, contract.Method, code.String(), result.Message)
		}
		return nil, fmt.Errorf("invoke contract [%s/%s] %s", contract.Name, contract.Method, code.String())
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
		return syscontract.CrossTxState_NON_EXIST
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
	return crossTxContractName + "/" + string(crossID)
}

func (c *cache) Get(ctx protocol.TxSimContext, crossID []byte, key []byte) ([]byte, error) {
	//key := c.genKey(crossID, suffix)
	return ctx.Get(c.genName(crossID), key)
}
func (c *cache) Set(ctx protocol.TxSimContext, crossID []byte, key []byte, value []byte) error {
	//key := c.genKey(crossID, suffix)
	return ctx.Put(c.genName(crossID), key, value)
}
