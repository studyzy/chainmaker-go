/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package scheduler

import (
	//	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"sync"
	"time"

	"chainmaker.org/chainmaker-go/core/provider/conf"
	"chainmaker.org/chainmaker-go/localconf"
	"chainmaker.org/chainmaker-go/utils"
	commonpb "chainmaker.org/chainmaker/pb-go/common"
	"chainmaker.org/chainmaker/protocol"
	"github.com/panjf2000/ants/v2"
	"github.com/prometheus/client_golang/prometheus"
	//	acpb "chainmaker.org/chainmaker/pb-go/accesscontrol"
	//	"chainmaker.org/chainmaker/pb-go/syscontract"
)

const (
	ScheduleTimeout        = 10
	ScheduleWithDagTimeout = 20
)

// TxScheduler transaction scheduler structure
type TxScheduler struct {
	lock            sync.Mutex
	VmManager       protocol.VmManager
	scheduleFinishC chan bool
	log             protocol.Logger
	chainConf       protocol.ChainConf // chain config

	metricVMRunTime *prometheus.HistogramVec
	StoreHelper     conf.StoreHelper
}

// Transaction dependency in adjacency table representation
type dagNeighbors map[int]bool

func NewTxSimContext(vmManager protocol.VmManager, snapshot protocol.Snapshot, tx *commonpb.Transaction,
	blockVersion uint32) protocol.TxSimContext {
	return &txSimContextImpl{
		txExecSeq:        snapshot.GetSnapshotSize(),
		tx:               tx,
		txReadKeyMap:     make(map[string]*commonpb.TxRead, 8),
		txWriteKeyMap:    make(map[string]*commonpb.TxWrite, 8),
		sqlRowCache:      make(map[int32]protocol.SqlRows),
		kvRowCache:       make(map[int32]protocol.StateIterator),
		txWriteKeySql:    make([]*commonpb.TxWrite, 0),
		txWriteKeyDdlSql: make([]*commonpb.TxWrite, 0),
		snapshot:         snapshot,
		vmManager:        vmManager,
		gasUsed:          0,
		currentDepth:     0,
		hisResult:        make([]*callContractResult, 0),
		blockVersion:     blockVersion,
	}
}

// Schedule according to a batch of transactions, and generating DAG according to the conflict relationship
func (ts *TxScheduler) Schedule(block *commonpb.Block, txBatch []*commonpb.Transaction,
	snapshot protocol.Snapshot) (map[string]*commonpb.TxRWSet, map[string][]*commonpb.ContractEvent, error) {

	ts.lock.Lock()
	defer ts.lock.Unlock()
	txRWSetMap := make(map[string]*commonpb.TxRWSet)
	txBatchSize := len(txBatch)
	runningTxC := make(chan *commonpb.Transaction, txBatchSize)
	timeoutC := time.After(ScheduleTimeout * time.Second)
	finishC := make(chan bool)
	var goRoutinePool *ants.Pool
	var err error
	ts.log.Infof("schedule tx batch start, size %d", txBatchSize)

	poolCapacity := ts.StoreHelper.GetPoolCapacity()
	if goRoutinePool, err = ants.NewPool(poolCapacity, ants.WithPreAlloc(true)); err != nil {
		return nil, nil, err
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
					ts.log.Debugf("run vm for tx id:%s", tx.Payload.GetTxId())
					txSimContext := NewTxSimContext(ts.VmManager, snapshot, tx, block.Header.BlockVersion)
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
						ts.log.Errorf(
							"failed to run vm for tx id:%s during schedule, tx result:%+v, error:%+v",
							tx.Payload.GetTxId(),
							txResult,
							err,
						)
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
							ts.metricVMRunTime.WithLabelValues(tx.Payload.ChainId).Observe(elapsed.Seconds())
						}
						ts.log.Debugf("apply to snapshot tx id:%s, result:%+v, apply count:%d", tx.Payload.GetTxId(), txResult, applySize)
					}
					// If all transactions have been successfully added to dag
					if applySize >= txBatchSize {
						finishC <- true
					}
				})
				if err != nil {
					ts.log.Warnf("failed to submit tx id %s during schedule, %+v", tx.Payload.GetTxId(), err)
				}
			case <-timeoutC:
				ts.scheduleFinishC <- true
				ts.log.Warnf("block [%d] schedule reached time limit", block.Header.BlockHeight)
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
	block.Dag = snapshot.BuildDAG(ts.chainConf.ChainConfig().Contract.EnableSqlSupport)
	block.Txs = snapshot.GetTxTable()
	timeCostB := time.Since(startTime)
	ts.log.Infof("schedule tx batch end, success %d, time used %v, time used (dag include) %v ",
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
		contractEventMap[tx.Payload.TxId] = event
	}
	//ts.dumpDAG(block.Dag, block.Txs)
	if localconf.ChainMakerConfig.SchedulerConfig.RWSetLog {
		ts.log.Debugf("rwset %v", txRWSetMap)
	}
	return txRWSetMap, contractEventMap, nil
}

// SimulateWithDag based on the dag in the block, perform scheduling and execution transactions
func (ts *TxScheduler) SimulateWithDag(block *commonpb.Block, snapshot protocol.Snapshot) (
	map[string]*commonpb.TxRWSet, map[string]*commonpb.Result, error) {
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
	poolCapacity := ts.StoreHelper.GetPoolCapacity()
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
					ts.log.Debugf("run vm with dag for tx id %s", tx.Payload.GetTxId())
					txSimContext := NewTxSimContext(ts.VmManager, snapshot, tx, block.Header.BlockVersion)
					runVmSuccess := true
					var txResult *commonpb.Result
					var err error

					if txResult, err = ts.runVM(tx, txSimContext); err != nil {
						runVmSuccess = false
						txSimContext.SetTxResult(txResult)
						ts.log.Errorf(
							"failed to run vm for tx id:%s during simulate with dag, tx result:%+v, error:%+v",
							tx.Payload.GetTxId(),
							txResult,
							err,
						)
					} else {
						//ts.log.Debugf(
						//	"success to run vm for tx id:%s during simulate with dag, tx result:%+v",
						//	tx.Payload.GetTxId(),
						//	txResult,
						//)
						txSimContext.SetTxResult(txResult)
					}

					applyResult, applySize := snapshot.ApplyTxSimContext(txSimContext, runVmSuccess)
					if !applyResult {
						ts.log.Debugf("failed to apply according to dag with tx %s ", tx.Payload.TxId)
						runningTxC <- txIndex
					} else {
						ts.log.Debugf("apply to snapshot tx id:%s, result:%+v, apply count:%d", tx.Payload.GetTxId(), txResult, applySize)
						doneTxC <- txIndex
					}
					// If all transactions in current batch have been successfully added to dag
					if applySize >= txBatchSize {
						finishC <- true
					}
				})
				if err != nil {
					ts.log.Warnf("failed to submit tx id %s during simulate with dag, %+v", tx.Payload.GetTxId(), err)
				}
			case doneTxIndex := <-doneTxC:
				ts.shrinkDag(doneTxIndex, dagRemain)

				txIndexBatch := ts.popNextTxBatchFromDag(dagRemain)
				//ts.log.Debugf("pop next tx index batch %v", txIndexBatch)
				for _, tx := range txIndexBatch {
					runningTxC <- tx
				}
			case <-finishC:
				ts.log.Debugf("schedule with dag finish")
				ts.scheduleFinishC <- true
				return
			case <-timeoutC:
				ts.log.Errorf("block [%d] schedule with dag timeout", block.Header.BlockHeight)
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

	ts.log.Infof("simulate with dag end, size %d, time used %+v", len(block.Txs), time.Since(startTime))

	// Return the read and write set after the scheduled execution

	for _, txRWSet := range snapshot.GetTxRWSetTable() {
		if txRWSet != nil {
			txRWSetMap[txRWSet.TxId] = txRWSet
		}
	}
	if localconf.ChainMakerConfig.SchedulerConfig.RWSetLog {
		ts.log.Debugf("rwset %v", txRWSetMap)
	}
	return txRWSetMap, snapshot.GetTxResultMap(), nil
}

func (ts *TxScheduler) shrinkDag(txIndex int, dagRemain map[int]dagNeighbors) {
	for _, neighbors := range dagRemain {
		delete(neighbors, txIndex)
	}
}

func (ts *TxScheduler) popNextTxBatchFromDag(dagRemain map[int]dagNeighbors) []int {
	var txIndexBatch []int
	for checkIndex, neighbors := range dagRemain {
		if len(neighbors) == 0 {
			txIndexBatch = append(txIndexBatch, checkIndex)
			delete(dagRemain, checkIndex)
		}
	}
	return txIndexBatch
}

func (ts *TxScheduler) Halt() {
	ts.scheduleFinishC <- true
}

func (ts *TxScheduler) runVM(tx *commonpb.Transaction, txSimContext protocol.TxSimContext) (*commonpb.Result, error) {
	//var contractId *commonpb.Contract
	var contractName string
	//var runtimeType commonpb.RuntimeType
	//var contractVersion string
	var method string
	var byteCode []byte
	//var endorsements []*commonpb.EndorsementEntry
	//var sequence uint64

	result := &commonpb.Result{
		Code: commonpb.TxStatusCode_SUCCESS,
		ContractResult: &commonpb.ContractResult{
			Code:    uint32(0),
			Result:  nil,
			Message: "",
		},
		RwSetHash: nil,
	}
	payload := tx.Payload
	switch tx.Payload.TxType {
	case commonpb.TxType_QUERY_CONTRACT:
		//var payload commonpb.Payload
		//if err := proto.Unmarshal(tx.RequestPayload, &payload); err == nil {
		//contractName = payload.ContractName
		//method = payload.Method
		//parameterPairs = payload.Parameters
		//parameters, err = ts.parseParameter(parameterPairs)
		//} else {
		//return errResult(
		//	result,
		//	fmt.Errorf("failed to unmarshal query payload for tx %s, %s", tx.Payload.TxId, err),
		//)
		//}
	case commonpb.TxType_INVOKE_CONTRACT:
		//var payload commonpb.TransactPayload
		//if err := proto.Unmarshal(tx.RequestPayload, &payload); err == nil {
		//contractName = payload.ContractName
		//method = payload.Method
		//parameterPairs = payload.Parameters
		//parameters, err = ts.parseParameter(parameterPairs)
		//} else {
		//return errResult(
		//	result,
		//	fmt.Errorf("failed to unmarshal transact payload for tx %s, %s", tx.Payload.TxId, err),
		//)
		//}
		//case commonpb.TxType_INVOKE_CONTRACT:
		//	var payload commonpb.Payload
		//	if err := proto.Unmarshal(tx.RequestPayload, &payload); err == nil {
		//		contractName = payload.ContractName
		//		method = payload.Method
		//		parameterPairs = payload.Parameters
		//		parameters = ts.parseParameter(parameterPairs)
		//	} else {
		//return errResult(
		//	result,
		//	fmt.Errorf("failed to unmarshal invoke payload for tx %s, %s", tx.Payload.TxId, err),
		//)
		//	}
		//case commonpb.TxType_INVOKE_CONTRACT:
		//	var payload commonpb.Payload
		//	if err := proto.Unmarshal(tx.RequestPayload, &payload); err == nil {
		//		contractName = payload.ContractName
		//		method = payload.Method
		//		parameterPairs = payload.Parameters
		//		parameters = ts.parseParameter(parameterPairs)
		//		endorsements = payload.Endorsement
		//		sequence = payload.Sequence
		//
		//if endorsements == nil {
		//	return errResult(
		//		result, fmt.Errorf(
		//			"endorsements not found in config update payload, tx id:%s",
		//			tx.Payload.TxId,
		//		),
		//	)
		//}
		//		payload.Endorsement = nil
		//		verifyPayloadBytes, err := proto.Marshal(&payload)
		//
		//		if err = ts.acVerify(txSimContext, method, endorsements, verifyPayloadBytes, parameters); err != nil {
		//			return errResult(result, err)
		//		}
		//
		//		ts.log.Debugf("chain config update [%d] [%v]", sequence, endorsements)
		//	} else {
		//return errResult(
		//	result,
		//	fmt.Errorf("failed to unmarshal system contract payload for tx %s, %s", tx.Payload.TxId, err.Error()),
		//)
		//	}
		//case commonpb.TxType_MANAGE_USER_CONTRACT:
		//	var payload commonpb.Payload
		//	if err := proto.Unmarshal(tx.RequestPayload, &payload); err == nil {
		//		if payload.Contract == nil {
		//			return errResult(result, fmt.Errorf("param is null"))
		//		}
		//		contractName = payload.Contract.Name
		//		runtimeType = payload.Contract.RuntimeType
		//		contractVersion = payload.Contract.Version
		//		method = payload.Method
		//		byteCode = payload.ByteCode
		//		parameterPairs = payload.Parameters
		//		parameters = ts.parseParameter(parameterPairs)
		//		endorsements = payload.Endorsement
		//
		//if endorsements == nil {
		//	return errResult(
		//		result,
		//		fmt.Errorf("endorsements not found in contract mgmt payload, tx id:%s", tx.Payload.TxId),
		//	)
		//}
		//
		//		payload.Endorsement = nil
		//		verifyPayloadBytes, err := proto.Marshal(&payload)
		//
		//		if err = ts.acVerify(txSimContext, method, endorsements, verifyPayloadBytes, parameters); err != nil {
		//			return errResult(result, err)
		//		}
		//	} else {
		//return errResult(
		//	result,
		//	fmt.Errorf("failed to unmarshal contract mgmt payload for tx %s, %s", tx.Payload.TxId, err.Error()),
		//)
		//}
	default:
		return errResult(result, fmt.Errorf("no such tx type: %s", tx.Payload.TxType))
	}

	contractName = payload.ContractName
	method = payload.Method
	parameters, err := ts.parseParameter(payload.Parameters)
	if err != nil {
		ts.log.Errorf("parse contract[%s] parameters error:%s", contractName, err)
		return errResult(result, fmt.Errorf(
			"parse tx[%s] contract[%s] parameters error:%s",
			payload.TxId,
			contractName,
			err.Error()),
		)
	}

	contract, err := utils.GetContractByName(txSimContext.Get, contractName)
	if err != nil {
		ts.log.Errorf("Get contract info by name[%s] error:%s", contractName, err)
		return nil, err
	}
	if contract.RuntimeType != commonpb.RuntimeType_NATIVE {
		byteCode, err = utils.GetContractBytecode(txSimContext.Get, contractName)
		if err != nil {
			ts.log.Errorf("Get contract bytecode by name[%s] error:%s", contractName, err)
			return nil, err
		}
	}
	//contract = &commonpb.Contract{
	//	ContractName:    contractName,
	//	ContractVersion: contractVersion,
	//	RuntimeType:     runtimeType,
	//}

	contractResultPayload, txStatusCode := ts.VmManager.RunContract(
		contract, method, byteCode, parameters, txSimContext, 0, tx.Payload.TxType)

	result.Code = txStatusCode
	result.ContractResult = contractResultPayload

	if txStatusCode == commonpb.TxStatusCode_SUCCESS {
		return result, nil
	}
	return result, errors.New(contractResultPayload.Message)
}

func errResult(result *commonpb.Result, err error) (*commonpb.Result, error) {
	result.ContractResult.Message = err.Error()
	result.Code = commonpb.TxStatusCode_INVALID_PARAMETER
	result.ContractResult.Code = 1
	return result, err
}
func (ts *TxScheduler) parseParameter(parameterPairs []*commonpb.KeyValuePair) (map[string][]byte, error) {
	// verify parameters
	if len(parameterPairs) > protocol.ParametersKeyMaxCount {
		return nil, fmt.Errorf(
			"expect parameters length less than %d, but got %d",
			protocol.ParametersKeyMaxCount,
			len(parameterPairs),
		)
	}
	parameters := make(map[string][]byte, 16)
	for i := 0; i < len(parameterPairs); i++ {
		key := parameterPairs[i].Key
		value := parameterPairs[i].Value
		if len(key) > protocol.DefaultMaxStateKeyLen {
			return nil, fmt.Errorf(
				"expect key length less than %d, but got %d",
				protocol.DefaultMaxStateKeyLen,
				len(key),
			)
		}

		re, err := regexp.Compile(protocol.DefaultStateRegex)
		match := re.MatchString(key)
		if err != nil || !match {
			return nil, fmt.Errorf(
				"expect key no special characters, but got key:[%s]. letter, number, dot and underline are allowed",
				key,
			)
		}
		if len(value) > protocol.ParametersValueMaxLength {
			return nil, fmt.Errorf(
				"expect value length less than %d, but got %d",
				protocol.ParametersValueMaxLength,
				len(value),
			)
		}

		parameters[key] = value
	}
	return parameters, nil
}

/*
func (ts *TxScheduler) acVerify(txSimContext protocol.TxSimContext, methodName string,
	endorsements []*commonpb.EndorsementEntry, msg []byte, parameters map[string][]byte) error {
	var ac protocol.AccessControlProvider
	var targetOrgId string
	var err error

	tx := txSimContext.GetTx()

	if ac, err = txSimContext.GetAccessControl(); err != nil {
		return fmt.Errorf(
			"failed to get access control from tx sim context for tx: %s, error: %s",
			tx.Payload.TxId,
			err.Error(),
		)
	}
	if orgId, ok := parameters[protocol.ConfigNameOrgId]; ok {
		targetOrgId = string(orgId)
	} else {
		targetOrgId = ""
	}

	var fullCertEndorsements []*commonpb.EndorsementEntry
	for _, endorsement := range endorsements {
		if endorsement == nil || endorsement.Signer == nil {
			return fmt.Errorf("failed to get endorsement signer for tx: %s, endorsement: %+v", tx.Payload.TxId, endorsement)
		}
		if endorsement.Signer.MemberType == acpb.MemberType_CERT {
			fullCertEndorsements = append(fullCertEndorsements, endorsement)
		} else {
			fullCertEndorsement := &commonpb.EndorsementEntry{
				Signer: &acpb.Member{
					OrgId:      endorsement.Signer.OrgId,
					MemberInfo: nil,
					//IsFullCert: true,
				},
				Signature: endorsement.Signature,
			}
			memberInfoHex := hex.EncodeToString(endorsement.Signer.MemberInfo)
			if fullMemberInfo, err := txSimContext.Get(
				syscontract.SystemContract_CERT_MANAGE.String(), []byte(memberInfoHex)); err != nil {
				return fmt.Errorf(
					"failed to get full cert from tx sim context for tx: %s,
					error: %s",
					tx.Payload.TxId,
					err.Error(),
				)
			} else {
				fullCertEndorsement.Signer.MemberInfo = fullMemberInfo
			}
			fullCertEndorsements = append(fullCertEndorsements, fullCertEndorsement)
		}
	}
	if verifyResult, err := utils.VerifyConfigUpdateTx(
		methodName, fullCertEndorsements, msg, targetOrgId, ac); err != nil {
		return fmt.Errorf("failed to verify endorsements for tx: %s, error: %s", tx.Payload.TxId, err.Error())
	} else if !verifyResult {
		return fmt.Errorf("failed to verify endorsements for tx: %s", tx.Payload.TxId)
	} else {
		return nil
	}
}
*/

//nolint: unused
func (ts *TxScheduler) dumpDAG(dag *commonpb.DAG, txs []*commonpb.Transaction) {
	dagString := "digraph DAG {\n"
	for i, ns := range dag.Vertexes {
		if len(ns.Neighbors) == 0 {
			dagString += fmt.Sprintf("id_%s -> begin;\n", txs[i].Payload.TxId[:8])
			continue
		}
		for _, n := range ns.Neighbors {
			dagString += fmt.Sprintf("id_%s -> id_%s;\n", txs[i].Payload.TxId[:8], txs[n].Payload.TxId[:8])
		}
	}
	dagString += "}"
	ts.log.Infof("Dump Dag: %s", dagString)
}
