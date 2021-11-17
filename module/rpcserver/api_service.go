/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package rpcserver

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"chainmaker.org/chainmaker-go/blockchain"
	commonErr "chainmaker.org/chainmaker/common/v2/errors"
	"chainmaker.org/chainmaker/common/v2/monitor"
	"chainmaker.org/chainmaker/localconf/v2"
	"chainmaker.org/chainmaker/logger/v2"
	apiPb "chainmaker.org/chainmaker/pb-go/v2/api"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	configPb "chainmaker.org/chainmaker/pb-go/v2/config"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/store/v2/archive"
	"chainmaker.org/chainmaker/utils/v2"
	native "chainmaker.org/chainmaker/vm-native/v2"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/time/rate"
)

const (
	//SYSTEM_CHAIN the system chain name
	SYSTEM_CHAIN = "system_chain"
)

var _ apiPb.RpcNodeServer = (*ApiService)(nil)

// ApiService struct define
type ApiService struct {
	chainMakerServer      *blockchain.ChainMakerServer
	log                   *logger.CMLogger
	logBrief              *logger.CMLogger
	subscriberRateLimiter *rate.Limiter
	metricQueryCounter    *prometheus.CounterVec
	metricInvokeCounter   *prometheus.CounterVec
	ctx                   context.Context
}

// NewApiService - new ApiService object
func NewApiService(ctx context.Context, chainMakerServer *blockchain.ChainMakerServer) *ApiService {
	log := logger.GetLogger(logger.MODULE_RPC)
	logBrief := logger.GetLogger(logger.MODULE_BRIEF)

	tokenBucketSize := localconf.ChainMakerConfig.RpcConfig.SubscriberConfig.RateLimitConfig.TokenBucketSize
	tokenPerSecond := localconf.ChainMakerConfig.RpcConfig.SubscriberConfig.RateLimitConfig.TokenPerSecond

	var subscriberRateLimiter *rate.Limiter
	if tokenBucketSize >= 0 && tokenPerSecond >= 0 {
		if tokenBucketSize == 0 {
			tokenBucketSize = subscriberRateLimitDefaultTokenBucketSize
		}

		if tokenPerSecond == 0 {
			tokenPerSecond = subscriberRateLimitDefaultTokenPerSecond
		}

		subscriberRateLimiter = rate.NewLimiter(rate.Limit(tokenPerSecond), tokenBucketSize)
	}

	apiService := ApiService{
		chainMakerServer:      chainMakerServer,
		log:                   log,
		logBrief:              logBrief,
		subscriberRateLimiter: subscriberRateLimiter,
		ctx:                   ctx,
	}

	if localconf.ChainMakerConfig.MonitorConfig.Enabled {
		apiService.metricQueryCounter = monitor.NewCounterVec(monitor.SUBSYSTEM_RPCSERVER, "metric_query_request_counter",
			"query request counts metric", "chainId", "state")
		apiService.metricInvokeCounter = monitor.NewCounterVec(monitor.SUBSYSTEM_RPCSERVER, "metric_invoke_request_counter",
			"invoke request counts metric", "chainId", "state")
	}

	return &apiService
}

// SendRequest - deal received TxRequest
func (s *ApiService) SendRequest(ctx context.Context, req *commonPb.TxRequest) (*commonPb.TxResponse, error) {
	s.log.DebugDynamic(func() string {
		return fmt.Sprintf("SendRequest[%s],payload:%#v,\n----signer:%v\n----endorsers:%+v",
			req.Payload.TxId, req.Payload, req.Sender, req.Endorsers)
	})

	resp := s.invoke(&commonPb.Transaction{
		Payload:   req.Payload,
		Sender:    req.Sender,
		Endorsers: req.Endorsers,
		Result:    nil}, protocol.RPC)

	// audit log format: ip:port|orgId|chainId|TxType|TxId|Timestamp|ContractName|Method|retCode|retCodeMsg|retMsg
	s.logBrief.Infof("|%s|%s|%s|%s|%s|%d|%s|%s|%d|%s|%s", GetClientAddr(ctx), req.Sender.Signer.OrgId,
		req.Payload.ChainId, req.Payload.TxType, req.Payload.TxId, req.Payload.Timestamp, req.Payload.ContractName,
		req.Payload.Method, resp.Code, resp.Code, resp.Message)

	return resp, nil
}

// validate tx
func (s *ApiService) validate(tx *commonPb.Transaction) (errCode commonErr.ErrCode, errMsg string) {
	var (
		err error
		bc  *blockchain.Blockchain
	)

	_, err = s.chainMakerServer.GetChainConf(tx.Payload.ChainId)
	if err != nil {
		errCode = commonErr.ERR_CODE_GET_CHAIN_CONF
		errMsg = s.getErrMsg(errCode, err)
		s.log.Error(errMsg)
		return
	}

	bc, err = s.chainMakerServer.GetBlockchain(tx.Payload.ChainId)
	if err != nil {
		errCode = commonErr.ERR_CODE_GET_BLOCKCHAIN
		errMsg = s.getErrMsg(errCode, err)
		s.log.Error(errMsg)
		return
	}

	if err = utils.VerifyTxWithoutPayload(tx, tx.Payload.ChainId, bc.GetAccessControl()); err != nil {
		errCode = commonErr.ERR_CODE_TX_VERIFY_FAILED
		errMsg = fmt.Sprintf("%s, %s, txId:%s, sender:%s", errCode.String(), err.Error(), tx.Payload.TxId,
			hex.EncodeToString(tx.Sender.Signer.MemberInfo))
		s.log.Error(errMsg)
		return
	}

	return commonErr.ERR_CODE_OK, ""
}

func (s *ApiService) getErrMsg(errCode commonErr.ErrCode, err error) string {
	return fmt.Sprintf("%s, %s", errCode.String(), err.Error())
}

// invoke contract according to TxType
func (s *ApiService) invoke(tx *commonPb.Transaction, source protocol.TxSource) *commonPb.TxResponse {
	var (
		errCode commonErr.ErrCode
		errMsg  string
		resp    = &commonPb.TxResponse{}
	)

	if tx.Payload.ChainId != SYSTEM_CHAIN {
		errCode, errMsg = s.validate(tx)
		if errCode != commonErr.ERR_CODE_OK {
			resp.Code = commonPb.TxStatusCode_INTERNAL_ERROR
			resp.Message = errMsg
			resp.TxId = tx.Payload.TxId
			return resp
		}
	}

	switch tx.Payload.TxType {
	case commonPb.TxType_QUERY_CONTRACT:
		return s.dealQuery(tx, source)
	case commonPb.TxType_INVOKE_CONTRACT:
		return s.dealTransact(tx, source)
	case commonPb.TxType_ARCHIVE:
		return s.doArchive(tx)
	default:
		return &commonPb.TxResponse{
			Code:    commonPb.TxStatusCode_INTERNAL_ERROR,
			Message: commonErr.ERR_CODE_TXTYPE.String(),
		}
	}
}

// dealQuery - deal query tx
func (s *ApiService) dealQuery(tx *commonPb.Transaction, source protocol.TxSource) *commonPb.TxResponse {
	var (
		err     error
		errMsg  string
		errCode commonErr.ErrCode
		store   protocol.BlockchainStore
		vmMgr   protocol.VmManager
		resp    = &commonPb.TxResponse{TxId: tx.Payload.TxId}
	)

	chainId := tx.Payload.ChainId

	if store, err = s.chainMakerServer.GetStore(chainId); err != nil {
		errCode = commonErr.ERR_CODE_GET_STORE
		errMsg = s.getErrMsg(errCode, err)
		s.log.Error(errMsg)
		resp.Code = commonPb.TxStatusCode_INTERNAL_ERROR
		resp.Message = errMsg
		resp.TxId = tx.Payload.TxId
		return resp
	}

	if vmMgr, err = s.chainMakerServer.GetVmManager(chainId); err != nil {
		errCode = commonErr.ERR_CODE_GET_VM_MGR
		errMsg = s.getErrMsg(errCode, err)
		s.log.Error(errMsg)
		resp.Code = commonPb.TxStatusCode_INTERNAL_ERROR
		resp.Message = errMsg
		resp.TxId = tx.Payload.TxId
		return resp
	}

	if chainId == SYSTEM_CHAIN {
		return s.dealSystemChainQuery(tx, vmMgr)
	}

	ctx := &txQuerySimContextImpl{
		tx:               tx,
		txReadKeyMap:     map[string]*commonPb.TxRead{},
		txWriteKeyMap:    map[string]*commonPb.TxWrite{},
		txWriteKeySql:    make([]*commonPb.TxWrite, 0),
		txWriteKeyDdlSql: make([]*commonPb.TxWrite, 0),
		rowCache:         make(map[int32]interface{}),
		blockchainStore:  store,
		vmManager:        vmMgr,
		blockVersion:     protocol.DefaultBlockVersion,
	}

	contract, err := store.GetContractByName(tx.Payload.ContractName)
	if err != nil {
		s.log.Error(err)
		resp.Code = commonPb.TxStatusCode_INTERNAL_ERROR
		resp.Message = err.Error()
		resp.TxId = tx.Payload.TxId
		return resp
	}

	var bytecode []byte
	if contract.RuntimeType != commonPb.RuntimeType_NATIVE {
		bytecode, err = store.GetContractBytecode(tx.Payload.ContractName)
		if err != nil {
			s.log.Error(err)
			resp.Code = commonPb.TxStatusCode_INTERNAL_ERROR
			resp.Message = err.Error()
			resp.TxId = tx.Payload.TxId
			return resp
		}
	}
	txResult, _, txStatusCode := vmMgr.RunContract(contract, tx.Payload.Method,
		bytecode, s.kvPair2Map(tx.Payload.Parameters), ctx, 0, tx.Payload.TxType)
	s.log.DebugDynamic(func() string {
		contractJson, _ := json.Marshal(contract)
		return fmt.Sprintf("vmMgr.RunContract: txStatusCode:%d, resultCode:%d, contractName[%s](%s), "+
			"method[%s], txType[%s], message[%s],result len: %d",
			txStatusCode, txResult.Code, tx.Payload.ContractName, string(contractJson), tx.Payload.Method,
			tx.Payload.TxType, txResult.Message, len(txResult.Result))
	})
	if localconf.ChainMakerConfig.MonitorConfig.Enabled {
		if txStatusCode == commonPb.TxStatusCode_SUCCESS && txResult.Code != 1 {
			s.metricQueryCounter.WithLabelValues(chainId, "true").Inc()
		} else {
			s.metricQueryCounter.WithLabelValues(chainId, "false").Inc()
		}
	}
	if txStatusCode != commonPb.TxStatusCode_SUCCESS {
		errMsg = fmt.Sprintf("txStatusCode:%d, resultCode:%d, contractName[%s] method[%s] txType[%s], %s",
			txStatusCode, txResult.Code, tx.Payload.ContractName, tx.Payload.Method, tx.Payload.TxType, txResult.Message)
		s.log.Warn(errMsg)

		resp.Code = txStatusCode
		if txResult.Message == archive.ErrArchivedBlock.Error() {
			resp.Code = commonPb.TxStatusCode_ARCHIVED_BLOCK
		} else if txResult.Message == archive.ErrArchivedTx.Error() {
			resp.Code = commonPb.TxStatusCode_ARCHIVED_TX
		}

		resp.Message = errMsg
		resp.ContractResult = txResult
		resp.TxId = tx.Payload.TxId
		return resp
	}

	if txResult.Code == 1 {
		resp.Code = commonPb.TxStatusCode_CONTRACT_FAIL
		resp.Message = commonPb.TxStatusCode_CONTRACT_FAIL.String()
		resp.ContractResult = txResult
		resp.TxId = tx.Payload.TxId
		return resp
	}

	resp.Code = commonPb.TxStatusCode_SUCCESS
	resp.Message = commonPb.TxStatusCode_SUCCESS.String()
	resp.ContractResult = txResult
	resp.TxId = tx.Payload.TxId
	return resp
}

// dealSystemChainQuery - deal system chain query
func (s *ApiService) dealSystemChainQuery(tx *commonPb.Transaction, vmMgr protocol.VmManager) *commonPb.TxResponse {
	var (
		resp = &commonPb.TxResponse{}
	)

	chainId := tx.Payload.ChainId

	ctx := &txQuerySimContextImpl{
		tx:               tx,
		txReadKeyMap:     map[string]*commonPb.TxRead{},
		txWriteKeyMap:    map[string]*commonPb.TxWrite{},
		txWriteKeySql:    make([]*commonPb.TxWrite, 0),
		txWriteKeyDdlSql: make([]*commonPb.TxWrite, 0),
		rowCache:         make(map[int32]interface{}),
		vmManager:        vmMgr,
		blockVersion:     protocol.DefaultBlockVersion,
	}

	runtimeInstance := native.GetRuntimeInstance(chainId)
	txResult := runtimeInstance.Invoke(&commonPb.Contract{
		Name: tx.Payload.ContractName,
	},
		tx.Payload.Method,
		nil,
		s.kvPair2Map(tx.Payload.Parameters),
		ctx,
	)

	if localconf.ChainMakerConfig.MonitorConfig.Enabled {
		if txResult.Code != 1 {
			s.metricQueryCounter.WithLabelValues(chainId, "true").Inc()
		} else {
			s.metricQueryCounter.WithLabelValues(chainId, "false").Inc()
		}
	}

	if txResult.Code == 1 {
		resp.Code = commonPb.TxStatusCode_CONTRACT_FAIL
		resp.Message = commonPb.TxStatusCode_CONTRACT_FAIL.String()
		resp.ContractResult = txResult
		resp.TxId = tx.Payload.TxId
		return resp
	}

	resp.Code = commonPb.TxStatusCode_SUCCESS
	resp.Message = commonPb.TxStatusCode_SUCCESS.String()
	resp.ContractResult = txResult
	resp.TxId = tx.Payload.TxId
	return resp
}

// kvPair2Map - change []*commonPb.KeyValuePair to map[string]string
func (s *ApiService) kvPair2Map(kvPair []*commonPb.KeyValuePair) map[string][]byte {
	kvMap := make(map[string][]byte)

	for _, kv := range kvPair {
		kvMap[kv.Key] = kv.Value
	}

	return kvMap
}

// dealTransact - deal transact tx
func (s *ApiService) dealTransact(tx *commonPb.Transaction, source protocol.TxSource) *commonPb.TxResponse {
	var (
		err     error
		errMsg  string
		errCode commonErr.ErrCode
		resp    = &commonPb.TxResponse{TxId: tx.Payload.TxId}
	)

	err = s.chainMakerServer.AddTx(tx.Payload.ChainId, tx, source)

	s.incInvokeCounter(tx.Payload.ChainId, err)

	if err != nil {
		s.log.Warnf("Add tx failed, %s, chainId:%s, txId:%s",
			err.Error(), tx.Payload.ChainId, tx.Payload.TxId)

		errCode = commonErr.ERR_CODE_TX_ADD_FAILED
		errMsg = s.getErrMsg(errCode, err)
		s.log.Error(errMsg)
		resp.Code = commonPb.TxStatusCode_INTERNAL_ERROR
		resp.Message = errMsg
		resp.TxId = tx.Payload.TxId
		return resp
	}

	s.log.Debugf("Add tx success, chainId:%s, txId:%s", tx.Payload.ChainId, tx.Payload.TxId)

	errCode = commonErr.ERR_CODE_OK
	resp.Code = commonPb.TxStatusCode_SUCCESS
	resp.Message = errCode.String()
	resp.TxId = tx.Payload.TxId
	return resp
}

func (s *ApiService) incInvokeCounter(chainId string, err error) {
	if localconf.ChainMakerConfig.MonitorConfig.Enabled {
		if err == nil {
			s.metricInvokeCounter.WithLabelValues(chainId, "true").Inc()
		} else {
			s.metricInvokeCounter.WithLabelValues(chainId, "false").Inc()
		}
	}
}

// RefreshLogLevelsConfig - refresh log level
func (s *ApiService) RefreshLogLevelsConfig(ctx context.Context, req *configPb.LogLevelsRequest) (
	*configPb.LogLevelsResponse, error) {

	if err := localconf.RefreshLogLevelsConfig(); err != nil {
		return &configPb.LogLevelsResponse{
			Code:    int32(1),
			Message: err.Error(),
		}, nil
	}
	return &configPb.LogLevelsResponse{
		Code: int32(0),
	}, nil
}

// UpdateDebugConfig - update debug config for test
func (s *ApiService) UpdateDebugConfig(ctx context.Context, req *configPb.DebugConfigRequest) (
	*configPb.DebugConfigResponse, error) {

	if err := localconf.UpdateDebugConfig(req.Pairs); err != nil {
		return &configPb.DebugConfigResponse{
			Code:    int32(1),
			Message: err.Error(),
		}, nil
	}
	return &configPb.DebugConfigResponse{
		Code: int32(0),
	}, nil
}

// CheckNewBlockChainConfig check new block chain config.
func (s *ApiService) CheckNewBlockChainConfig(context.Context, *configPb.CheckNewBlockChainConfigRequest) (
	*configPb.CheckNewBlockChainConfigResponse, error) {

	if err := localconf.CheckNewCmBlockChainConfig(); err != nil {
		return &configPb.CheckNewBlockChainConfigResponse{
			Code:    int32(1),
			Message: err.Error(),
		}, nil
	}
	return &configPb.CheckNewBlockChainConfigResponse{
		Code: int32(0),
	}, nil
}

// GetChainMakerVersion get chainmaker version by rpc request
func (s *ApiService) GetChainMakerVersion(ctx context.Context, req *configPb.ChainMakerVersionRequest) (
	*configPb.ChainMakerVersionResponse, error) {

	return &configPb.ChainMakerVersionResponse{
		Code:    int32(0),
		Version: s.chainMakerServer.Version(),
	}, nil
}
