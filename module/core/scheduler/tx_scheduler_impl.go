/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package scheduler

import (
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"runtime"
	"sync"
	"time"

	"chainmaker.org/chainmaker-go/localconf"
	"chainmaker.org/chainmaker-go/logger"
	"chainmaker.org/chainmaker-go/monitor"
	acpb "chainmaker.org/chainmaker-go/pb/protogo/accesscontrol"
	commonpb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/utils"
	"github.com/gogo/protobuf/proto"
	"github.com/panjf2000/ants/v2"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	ScheduleTimeout        = 10
	ScheduleWithDagTimeout = 20
)

// TxSchedulerImpl transaction scheduler structure
type TxSchedulerImpl struct {
	lock            sync.Mutex
	VmManager       protocol.VmManager
	scheduleFinishC chan bool
	log             *logger.CMLogger
	chainConf       protocol.ChainConf // chain config

	metricVMRunTime *prometheus.HistogramVec
}

// Transaction dependency in adjacency table representation
type dagNeighbors map[int]bool

// NewTxScheduler building a transaction scheduler
func NewTxScheduler(vmMgr protocol.VmManager, chainConf protocol.ChainConf) *TxSchedulerImpl {
	txSchedulerImpl := &TxSchedulerImpl{
		lock:            sync.Mutex{},
		VmManager:       vmMgr,
		scheduleFinishC: make(chan bool),
		log:             logger.GetLoggerByChain(logger.MODULE_CORE, chainConf.ChainConfig().ChainId),
		chainConf:       chainConf,
	}
	if localconf.ChainMakerConfig.MonitorConfig.Enabled {
		txSchedulerImpl.metricVMRunTime = monitor.NewHistogramVec(monitor.SUBSYSTEM_CORE_PROPOSER_SCHEDULER, "metric_vm_run_time",
			"VM run time metric", []float64{0.005, 0.01, 0.015, 0.05, 0.1, 1, 10}, "chainId")
	}
	return txSchedulerImpl
}

func newTxSimContext(vmManager protocol.VmManager, snapshot protocol.Snapshot, tx *commonpb.Transaction) protocol.TxSimContext {
	return &txSimContextImpl{
		txExecSeq:     snapshot.GetSnapshotSize(),
		tx:            tx,
		txReadKeyMap:  make(map[string]*commonpb.TxRead, 8),
		txWriteKeyMap: make(map[string]*commonpb.TxWrite, 8),
		sqlRowCache:   make(map[int32]protocol.SqlRows, 0),
		txWriteKeySql: make([]*commonpb.TxWrite, 0),
		snapshot:      snapshot,
		vmManager:     vmManager,
		gasUsed:       0,
		currentDepth:  0,
		hisResult:     make([]*callContractResult, 0),
	}
}

// Schedule according to a batch of transactions, and generating DAG according to the conflict relationship
func (ts *TxSchedulerImpl) Schedule(block *commonpb.Block, txBatch []*commonpb.Transaction, snapshot protocol.Snapshot) (map[string]*commonpb.TxRWSet, map[string][]*commonpb.ContractEvent, error) {

	ts.lock.Lock()
	defer ts.lock.Unlock()
	txRWSetMap := make(map[string]*commonpb.TxRWSet)
	txBatchSize := len(txBatch)
	runningTxC := make(chan *commonpb.Transaction, txBatchSize)
	timeoutC := time.After(ScheduleTimeout * time.Second)
	finishC := make(chan bool)
	ts.log.Infof("schedule tx batch start, size %d", txBatchSize)
	var goRoutinePool *ants.Pool
	var err error
	poolCapacity := runtime.NumCPU() * 4
	if ts.chainConf.ChainConfig().Contract.EnableSqlSupport {
		poolCapacity = 1
	}
	if goRoutinePool, err = ants.NewPool(poolCapacity, ants.WithPreAlloc(true)); err != nil {
		return  nil, nil, err
	}
	defer goRoutinePool.Release()
	startTime := time.Now()
	go func() {
		for {
			select {
			case tx := <-runningTxC:
				err := goRoutinePool.Submit(func() {
					// If snapshot is sealed, no more transaction will be added into snapshot
					if snapshot.IsSealed() {
						return
					}
					ts.log.Debugf("run vm for tx id:%s", tx.Header.GetTxId())
					txSimContext := newTxSimContext(ts.VmManager, snapshot, tx)
					runVmSuccess := true
					var txResult *commonpb.Result
					var err error
					var start time.Time
					if localconf.ChainMakerConfig.MonitorConfig.Enabled {
						start = time.Now()
					}
					//交易结果
					if txResult, err = ts.runVM(tx, txSimContext); err != nil {
						runVmSuccess = false
						tx.Result = txResult
						txSimContext.SetTxResult(txResult)
						ts.log.Errorf("failed to run vm for tx id:%s during schedule, tx result:%+v, error:%+v", tx.Header.GetTxId(), txResult, err)
					} else {
						tx.Result = txResult
						txSimContext.SetTxResult(txResult)
					}
					applyResult, applySize := snapshot.ApplyTxSimContext(txSimContext, runVmSuccess)
					if !applyResult {
						runningTxC <- tx
					} else {
						if localconf.ChainMakerConfig.MonitorConfig.Enabled {
							elapsed := time.Since(start)
							ts.metricVMRunTime.WithLabelValues(tx.Header.ChainId).Observe(elapsed.Seconds())
						}
						ts.log.Debugf("apply to snapshot tx id:%s, result:%+v, apply count:%d", tx.Header.GetTxId(), txResult, applySize)
					}
					// If all transactions have been successfully added to dag
					if applySize >= txBatchSize {
						finishC <- true
					}
				})
				if err != nil {
					ts.log.Warnf("failed to submit tx id %s during schedule, %+v", tx.Header.GetTxId(), err)
				}
			case <-timeoutC:
				ts.scheduleFinishC <- true
				ts.log.Debugf("schedule reached time limit")
				return
			case <-finishC:
				ts.log.Debugf("schedule finish")
				ts.scheduleFinishC <- true
				return
			}
		}
	}()
	// Put the pending transaction into the running queue
	go func() {
		if len(txBatch) > 0 {
			for _, tx := range txBatch {
				runningTxC <- tx
			}
		} else {
			finishC <- true
		}
	}()
	// Wait for schedule finish signal
	<-ts.scheduleFinishC
	// Build DAG from read-write table
	snapshot.Seal()
	timeCostA := time.Since(startTime)
	block.Dag = snapshot.BuildDAG()
	block.Txs = snapshot.GetTxTable()
	timeCostB := time.Since(startTime)
	ts.log.Infof("schedule tx batch end, success %d, time cost %v, time cost(dag include) %v ",
		len(block.Dag.Vertexes), timeCostA, timeCostB)
	txRWSetTable := snapshot.GetTxRWSetTable()
	for _, txRWSet := range txRWSetTable {
		if txRWSet != nil {
			txRWSetMap[txRWSet.TxId] = txRWSet
		}
	}
	contractEventMap := make(map[string][]*commonpb.ContractEvent)
	for _, tx := range block.Txs {
		event := tx.Result.ContractResult.ContractEvent
		contractEventMap[tx.Header.TxId] = event
	}
	//ts.dumpDAG(block.Dag, block.Txs)
	return txRWSetMap, contractEventMap, nil
}

// SimulateWithDag based on the dag in the block, perform scheduling and execution transactions
func (ts *TxSchedulerImpl) SimulateWithDag(block *commonpb.Block, snapshot protocol.Snapshot) (map[string]*commonpb.TxRWSet, map[string]*commonpb.Result, error) {
	ts.lock.Lock()
	defer ts.lock.Unlock()

	var (
		startTime  = time.Now()
		txRWSetMap = make(map[string]*commonpb.TxRWSet)
	)
	if len(block.Txs) == 0 {
		ts.log.Debugf("no txs in block[%x] when simulate", block.Header.BlockHash)
		return txRWSetMap, snapshot.GetTxResultMap(), nil
	}
	ts.log.Debugf("simulate with dag start, size %d", len(block.Txs))
	txMapping := make(map[int]*commonpb.Transaction)
	for index, tx := range block.Txs {
		txMapping[index] = tx
	}

	// Construct the adjacency list of dag, which describes the subsequent adjacency transactions of all transactions
	dag := block.Dag
	dagRemain := make(map[int]dagNeighbors)
	for txIndex, neighbors := range dag.Vertexes {
		dn := make(dagNeighbors)
		for _, neighbor := range neighbors.Neighbors {
			dn[int(neighbor)] = true
		}
		dagRemain[txIndex] = dn
	}

	txBatchSize := len(block.Dag.Vertexes)
	runningTxC := make(chan int, txBatchSize)
	doneTxC := make(chan int, txBatchSize)

	timeoutC := time.After(ScheduleWithDagTimeout * time.Second)
	finishC := make(chan bool)

	var goRoutinePool *ants.Pool
	var err error
	poolCapacity := runtime.NumCPU() * 4
	if ts.chainConf.ChainConfig().Contract.EnableSqlSupport {
		poolCapacity = 1
	}
	if goRoutinePool, err = ants.NewPool(poolCapacity, ants.WithPreAlloc(true)); err != nil {
		return nil, nil, err
	}
	defer goRoutinePool.Release()

	go func() {
		for {
			select {
			case txIndex := <-runningTxC:
				tx := txMapping[txIndex]
				err := goRoutinePool.Submit(func() {
					ts.log.Debugf("run vm with dag for tx id %s", tx.Header.GetTxId())
					txSimContext := newTxSimContext(ts.VmManager, snapshot, tx)
					runVmSuccess := true
					var txResult *commonpb.Result
					var err error

					if txResult, err = ts.runVM(tx, txSimContext); err != nil {
						runVmSuccess = false
						txSimContext.SetTxResult(txResult)
						ts.log.Errorf("failed to run vm for tx id:%s during simulate with dag, tx result:%+v, error:%+v", tx.Header.GetTxId(), txResult, err)
					} else {
						//ts.log.Debugf("success to run vm for tx id:%s during simulate with dag, tx result:%+v", tx.Header.GetTxId(), txResult)
						txSimContext.SetTxResult(txResult)
					}

					applyResult, applySize := snapshot.ApplyTxSimContext(txSimContext, runVmSuccess)
					if !applyResult {
						ts.log.Debugf("failed to apply according to dag with tx %s ", tx.Header.TxId)
						runningTxC <- txIndex
					} else {
						ts.log.Debugf("apply to snapshot tx id:%s, result:%+v, apply count:%d", tx.Header.GetTxId(), txResult, applySize)
						doneTxC <- txIndex
					}
					// If all transactions in current batch have been successfully added to dag
					if applySize >= txBatchSize {
						finishC <- true
					}
				})
				if err != nil {
					ts.log.Warnf("failed to submit tx id %s during simulate with dag, %+v", tx.Header.GetTxId(), err)
				}
			case doneTxIndex := <-doneTxC:
				ts.shrinkDag(doneTxIndex, dagRemain)

				txIndexBatch := ts.popNextTxBatchFromDag(dagRemain)
				ts.log.Debugf("pop next tx index batch %v", txIndexBatch)
				for _, tx := range txIndexBatch {
					runningTxC <- tx
				}
			case <-finishC:
				ts.log.Debugf("schedule with dag finish")
				ts.scheduleFinishC <- true
				return
			case <-timeoutC:
				ts.log.Errorf("schedule with dag timeout")
				ts.scheduleFinishC <- true
				return
			}
		}
	}()

	txIndexBatch := ts.popNextTxBatchFromDag(dagRemain)

	go func() {
		for _, tx := range txIndexBatch {
			runningTxC <- tx
		}
	}()

	<-ts.scheduleFinishC
	snapshot.Seal()

	ts.log.Infof("simulate with dag end, size %d, time cost %+v", len(block.Txs), time.Since(startTime))

	// Return the read and write set after the scheduled execution

	for _, txRWSet := range snapshot.GetTxRWSetTable() {
		if txRWSet != nil {
			txRWSetMap[txRWSet.TxId] = txRWSet
		}
	}
	return txRWSetMap, snapshot.GetTxResultMap(), nil
}

func (ts *TxSchedulerImpl) shrinkDag(txIndex int, dagRemain map[int]dagNeighbors) {
	for _, neighbors := range dagRemain {
		delete(neighbors, txIndex)
	}
}

func (ts *TxSchedulerImpl) popNextTxBatchFromDag(dagRemain map[int]dagNeighbors) []int {
	var txIndexBatch []int
	for checkIndex, neighbors := range dagRemain {
		if len(neighbors) == 0 {
			txIndexBatch = append(txIndexBatch, checkIndex)
			delete(dagRemain, checkIndex)
		}
	}
	return txIndexBatch
}

func (ts *TxSchedulerImpl) Halt() {
	ts.scheduleFinishC <- true
}

func (ts *TxSchedulerImpl) runVM(tx *commonpb.Transaction, txSimContext protocol.TxSimContext) (*commonpb.Result, error) {
	var contractId *commonpb.ContractId
	var contractName string
	var runtimeType commonpb.RuntimeType
	var contractVersion string
	var method string
	var byteCode []byte
	var parameterPairs []*commonpb.KeyValuePair
	var parameters map[string]string
	var endorsements []*commonpb.EndorsementEntry
	var sequence uint64

	result := &commonpb.Result{
		Code: commonpb.TxStatusCode_SUCCESS,
		ContractResult: &commonpb.ContractResult{
			Code:    commonpb.ContractResultCode_OK,
			Result:  nil,
			Message: "",
		},
		RwSetHash: nil,
	}

	switch tx.Header.TxType {
	case commonpb.TxType_QUERY_SYSTEM_CONTRACT, commonpb.TxType_QUERY_USER_CONTRACT:
		var payload commonpb.QueryPayload
		if err := proto.Unmarshal(tx.RequestPayload, &payload); err == nil {
			contractName = payload.ContractName
			method = payload.Method
			parameterPairs = payload.Parameters
			parameters = ts.parseParameter(parameterPairs)
		} else {
			return errResult(result, fmt.Errorf("failed to unmarshal query payload for tx %s, %s", tx.Header.TxId, err))
		}
	case commonpb.TxType_INVOKE_USER_CONTRACT:
		var payload commonpb.TransactPayload
		if err := proto.Unmarshal(tx.RequestPayload, &payload); err == nil {
			contractName = payload.ContractName
			method = payload.Method
			parameterPairs = payload.Parameters
			parameters = ts.parseParameter(parameterPairs)
		} else {
			return errResult(result, fmt.Errorf("failed to unmarshal transact payload for tx %s, %s", tx.Header.TxId, err))
		}
	case commonpb.TxType_INVOKE_SYSTEM_CONTRACT:
		var payload commonpb.SystemContractPayload
		if err := proto.Unmarshal(tx.RequestPayload, &payload); err == nil {
			contractName = payload.ContractName
			method = payload.Method
			parameterPairs = payload.Parameters
			parameters = ts.parseParameter(parameterPairs)
		} else {
			return errResult(result, fmt.Errorf("failed to unmarshal invoke payload for tx %s, %s", tx.Header.TxId, err))
		}
	case commonpb.TxType_UPDATE_CHAIN_CONFIG:
		var payload commonpb.SystemContractPayload
		if err := proto.Unmarshal(tx.RequestPayload, &payload); err == nil {
			contractName = payload.ContractName
			method = payload.Method
			parameterPairs = payload.Parameters
			parameters = ts.parseParameter(parameterPairs)
			endorsements = payload.Endorsement
			sequence = payload.Sequence

			if endorsements == nil {
				return errResult(result, fmt.Errorf("endorsements not found in config update payload, tx id:%s", tx.Header.TxId))
			}
			payload.Endorsement = nil
			verifyPayloadBytes, err := proto.Marshal(&payload)

			if err = ts.acVerify(txSimContext, method, endorsements, verifyPayloadBytes, parameters); err != nil {
				return errResult(result, err)
			}

			ts.log.Debugf("chain config update [%d] [%v]", sequence, endorsements)
		} else {
			return errResult(result, fmt.Errorf("failed to unmarshal system contract payload for tx %s, %s", tx.Header.TxId, err.Error()))
		}
	case commonpb.TxType_MANAGE_USER_CONTRACT:
		var payload commonpb.ContractMgmtPayload
		if err := proto.Unmarshal(tx.RequestPayload, &payload); err == nil {
			if payload.ContractId == nil {
				return errResult(result, fmt.Errorf("param is null"))
			}
			contractName = payload.ContractId.ContractName
			runtimeType = payload.ContractId.RuntimeType
			contractVersion = payload.ContractId.ContractVersion
			method = payload.Method
			byteCode = payload.ByteCode
			parameterPairs = payload.Parameters
			parameters = ts.parseParameter(parameterPairs)
			endorsements = payload.Endorsement

			if endorsements == nil {
				return errResult(result, fmt.Errorf("endorsements not found in contract mgmt payload, tx id:%s", tx.Header.TxId))
			}

			payload.Endorsement = nil
			verifyPayloadBytes, err := proto.Marshal(&payload)

			if err = ts.acVerify(txSimContext, method, endorsements, verifyPayloadBytes, parameters); err != nil {
				return errResult(result, err)
			}
		} else {
			return errResult(result, fmt.Errorf("failed to unmarshal contract mgmt payload for tx %s, %s", tx.Header.TxId, err.Error()))
		}
	default:
		return errResult(result, fmt.Errorf("no such tx type: %s", tx.Header.TxType))
	}

	contractId = &commonpb.ContractId{
		ContractName:    contractName,
		ContractVersion: contractVersion,
		RuntimeType:     runtimeType,
	}

	// verify parameters
	if len(parameters) > protocol.ParametersKeyMaxCount {
		return errResult(result, fmt.Errorf("expect less than %d parameters, but get %d, tx id:%s", protocol.ParametersKeyMaxCount, len(parameters),
			tx.Header.TxId))
	}
	for key, val := range parameters {
		if len(key) > protocol.DefaultStateLen {
			return errResult(result, fmt.Errorf("expect key length less than %d, but get %d, tx id:%s", protocol.DefaultStateLen, len(key), tx.Header.TxId))
		}
		match, err := regexp.MatchString(protocol.DefaultStateRegex, key)
		if err != nil || !match {
			return errResult(result, fmt.Errorf("expect key no special characters, but get key:[%s]. letter, number, dot and underline are allowed, tx id:[%s]", key, tx.Header.TxId))
		}
		if len(val) > protocol.ParametersValueMaxLength {
			return errResult(result, fmt.Errorf("expect value length less than %d, but get %d, tx id:%s", protocol.ParametersValueMaxLength, len(val), tx.Header.TxId))
		}
	}
	contractResultPayload, txStatusCode := ts.VmManager.RunContract(contractId, method, byteCode, parameters, txSimContext, 0, tx.Header.TxType)

	result.Code = txStatusCode
	result.ContractResult = contractResultPayload

	if txStatusCode == commonpb.TxStatusCode_SUCCESS {
		return result, nil
	} else {
		return result, errors.New(contractResultPayload.Message)
	}
}

func errResult(result *commonpb.Result, err error) (*commonpb.Result, error) {
	result.ContractResult.Message = err.Error()
	result.Code = commonpb.TxStatusCode_INVALID_PARAMETER
	result.ContractResult.Code = commonpb.ContractResultCode_FAIL
	return result, err
}
func (ts *TxSchedulerImpl) parseParameter(parameterPairs []*commonpb.KeyValuePair) map[string]string {
	parameters := make(map[string]string, 16)
	for i := 0; i < len(parameterPairs); i++ {
		key := parameterPairs[i].Key
		// ignore the following input from the user's invoke parameters
		if key == protocol.ContractCreatorOrgIdParam ||
			key == protocol.ContractCreatorRoleParam ||
			key == protocol.ContractCreatorPkParam ||
			key == protocol.ContractSenderOrgIdParam ||
			key == protocol.ContractSenderRoleParam ||
			key == protocol.ContractSenderPkParam ||
			key == protocol.ContractBlockHeightParam ||
			key == protocol.ContractTxIdParam {
			continue
		}
		value := parameterPairs[i].Value
		parameters[key] = value
	}
	return parameters
}

func (ts *TxSchedulerImpl) acVerify(txSimContext protocol.TxSimContext, methodName string, endorsements []*commonpb.EndorsementEntry, msg []byte, parameters map[string]string) error {
	var ac protocol.AccessControlProvider
	var targetOrgId string
	var err error

	tx := txSimContext.GetTx()

	if ac, err = txSimContext.GetAccessControl(); err != nil {
		return fmt.Errorf("failed to get access control from tx sim context for tx: %s, error: %s", tx.Header.TxId, err.Error())
	}
	if orgId, ok := parameters[protocol.ConfigNameOrgId]; ok {
		targetOrgId = orgId
	} else {
		targetOrgId = ""
	}

	var fullCertEndorsements []*commonpb.EndorsementEntry
	for _, endorsement := range endorsements {
		if endorsement == nil || endorsement.Signer == nil {
			return fmt.Errorf("failed to get endorsement signer for tx: %s, endorsement: %+v", tx.Header.TxId, endorsement)
		}
		if endorsement.Signer.IsFullCert {
			fullCertEndorsements = append(fullCertEndorsements, endorsement)
		} else {
			fullCertEndorsement := &commonpb.EndorsementEntry{
				Signer: &acpb.SerializedMember{
					OrgId:      endorsement.Signer.OrgId,
					MemberInfo: nil,
					IsFullCert: true,
				},
				Signature: endorsement.Signature,
			}
			memberInfoHex := hex.EncodeToString(endorsement.Signer.MemberInfo)
			if fullMemberInfo, err := txSimContext.Get(commonpb.ContractName_SYSTEM_CONTRACT_CERT_MANAGE.String(), []byte(memberInfoHex)); err != nil {
				return fmt.Errorf("failed to get full cert from tx sim context for tx: %s, error: %s", tx.Header.TxId, err.Error())
			} else {
				fullCertEndorsement.Signer.MemberInfo = fullMemberInfo
			}
			fullCertEndorsements = append(fullCertEndorsements, fullCertEndorsement)
		}
	}
	if verifyResult, err := utils.VerifyConfigUpdateTx(methodName, fullCertEndorsements, msg, targetOrgId, ac); err != nil {
		return fmt.Errorf("failed to verify endorsements for tx: %s, error: %s", tx.Header.TxId, err.Error())
	} else if !verifyResult {
		return fmt.Errorf("failed to verify endorsements for tx: %s", tx.Header.TxId)
	} else {
		return nil
	}
}

func (ts *TxSchedulerImpl) dumpDAG(dag *commonpb.DAG, txs []*commonpb.Transaction) {
	dagString := "digraph DAG {\n"
	for i, ns := range dag.Vertexes {
		if len(ns.Neighbors) == 0 {
			dagString += fmt.Sprintf("id_%s -> begin;\n", txs[i].Header.TxId[:8])
			continue
		}
		for _, n := range ns.Neighbors {
			dagString += fmt.Sprintf("id_%s -> id_%s;\n", txs[i].Header.TxId[:8], txs[n].Header.TxId[:8])
		}
	}
	dagString += "}"
	ts.log.Infof("Dump Dag: %s", dagString)
}
