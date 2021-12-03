/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package scheduler

import (
	"errors"
	"fmt"
	"regexp"
	"sync"
	"time"

	"chainmaker.org/chainmaker-go/core/provider/conf"
	"chainmaker.org/chainmaker/localconf/v2"
	commonpb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/vm/v2"
	"github.com/panjf2000/ants/v2"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	ScheduleTimeout        = 10
	ScheduleWithDagTimeout = 10
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

// Schedule according to a batch of transactions, and generating DAG according to the conflict relationship
func (ts *TxScheduler) Schedule(block *commonpb.Block, txBatch []*commonpb.Transaction, snapshot protocol.Snapshot) (
	map[string]*commonpb.TxRWSet, map[string][]*commonpb.ContractEvent, error) {

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

					var start time.Time
					if localconf.ChainMakerConfig.MonitorConfig.Enabled {
						start = time.Now()
					}
					txSimContext, specialTxType, runVmSuccess := ts.executeTx(tx, snapshot, block)
					tx.Result = txSimContext.GetTxResult()

					// Apply failed means this tx's read set conflict with other txs' write set
					applyResult, applySize := snapshot.ApplyTxSimContext(txSimContext, specialTxType,
						runVmSuccess, false)
					if !applyResult {
						runningTxC <- tx
					} else {
						if localconf.ChainMakerConfig.MonitorConfig.Enabled {
							elapsed := time.Since(start)
							ts.metricVMRunTime.WithLabelValues(tx.Payload.ChainId).Observe(elapsed.Seconds())
						}
						ts.log.Debugf("apply to snapshot tx id:%s, result:%+v, apply count:%d",
							tx.Payload.GetTxId(), txSimContext.GetTxResult(), applySize)
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

	// Execute special tx sequentially, and add to dag
	if len(snapshot.GetSpecialTxTable()) > 0 {
		ts.simulateSpecialTxs(block.Dag, snapshot, block, txBatchSize)
	}

	timeCostB := time.Since(startTime)
	ts.log.Infof("schedule tx batch finished, success %d, time used %v, time used (dag include) %v ",
		len(block.Dag.Vertexes), timeCostA, timeCostB)
	block.Txs = snapshot.GetTxTable()
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
func (ts *TxScheduler) SimulateWithDag(block *commonpb.Block, snapshot protocol.Snapshot) (map[string]*commonpb.TxRWSet,
	map[string]*commonpb.Result, error) {
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
					txSimContext, specialTxType, runVmSuccess := ts.executeTx(tx, snapshot, block)
					// if apply failed means this tx's read set conflict with other txs' write set
					applyResult, applySize := snapshot.ApplyTxSimContext(txSimContext, specialTxType,
						runVmSuccess, true)
					if !applyResult {
						ts.log.Debugf("failed to apply according to dag with tx %s ", tx.Payload.TxId)
						runningTxC <- txIndex
					} else {
						ts.log.Debugf("apply to snapshot tx id:%s, result:%+v, apply count:%d, tx batch size:%d",
							tx.Payload.GetTxId(), txSimContext.GetTxResult(), applySize, txBatchSize)
						doneTxC <- txIndex
					}
					// If all transactions in current batch have been successfully added to dag
					if applySize >= txBatchSize {
						ts.log.Debugf("finished 1 batch, apply size:%d, tx batch size:%d, dagRemain size:%d",
							applySize, txBatchSize, len(dagRemain))
						finishC <- true
					}
				})
				if err != nil {
					ts.log.Warnf("failed to submit tx id %s during simulate with dag, %+v",
						tx.Payload.GetTxId(), err)
				}
			case doneTxIndex := <-doneTxC:
				ts.shrinkDag(doneTxIndex, dagRemain)
				txIndexBatch := ts.popNextTxBatchFromDag(dagRemain)
				ts.log.Debugf("block [%d] schedule with dag, pop next tx index batch size:%d", len(txIndexBatch))
				for _, tx := range txIndexBatch {
					runningTxC <- tx
				}
				ts.log.Debugf("shrinkDag and pop next tx batch size:%d, dagRemain size:%d",
					len(txIndexBatch), len(dagRemain))
			case <-finishC:
				ts.log.Debugf("block [%d] schedule with dag finish", block.Header.BlockHeight)
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
	ts.log.Debugf("simulate with dag first batch size:%d, total batch size:%d", len(txIndexBatch), txBatchSize)
	go func() {
		for _, tx := range txIndexBatch {
			runningTxC <- tx
		}
	}()

	<-ts.scheduleFinishC
	snapshot.Seal()

	ts.log.Infof("simulate with dag finished, size %d, time used %+v", len(block.Txs), time.Since(startTime))

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

func (ts *TxScheduler) executeTx(tx *commonpb.Transaction, snapshot protocol.Snapshot, block *commonpb.Block) (
	protocol.TxSimContext, protocol.ExecOrderTxType, bool) {
	ts.log.Debugf("run vm start for tx:%s", tx.Payload.GetTxId())
	txSimContext := vm.NewTxSimContext(ts.VmManager, snapshot, tx, block.Header.BlockVersion)
	ts.log.Debugf("new tx simulate context for tx:%s", tx.Payload.GetTxId())
	runVmSuccess := true
	var txResult *commonpb.Result
	var err error
	var specialTxType protocol.ExecOrderTxType
	if txResult, specialTxType, err = ts.runVM(tx, txSimContext); err != nil {
		runVmSuccess = false
		ts.log.Errorf("failed to run vm for tx id:%s, tx result:%+v, error:%+v",
			tx.Payload.GetTxId(), txResult, err)
	}
	ts.log.Debugf("run vm finished for tx:%s, runVmSuccess:%v", tx.Payload.TxId, runVmSuccess)
	txSimContext.SetTxResult(txResult)
	return txSimContext, specialTxType, runVmSuccess
}

func (ts *TxScheduler) simulateSpecialTxs(dag *commonpb.DAG, snapshot protocol.Snapshot, block *commonpb.Block,
	txBatchSize int) {
	specialTxs := snapshot.GetSpecialTxTable()
	specialTxsLen := len(specialTxs)
	var firstTx *commonpb.Transaction
	runningTxC := make(chan *commonpb.Transaction, specialTxsLen)
	scheduleFinishC := make(chan bool)
	timeoutC := time.After(ScheduleWithDagTimeout * time.Second)
	go func() {
		for _, tx := range specialTxs {
			runningTxC <- tx
		}
	}()

	go func() {
		for {
			select {
			case tx := <-runningTxC:
				// simulate tx
				txSimContext, specialTxType, runVmSuccess := ts.executeTx(tx, snapshot, block)
				tx.Result = txSimContext.GetTxResult()
				// apply tx
				applyResult, applySize := snapshot.ApplyTxSimContext(txSimContext, specialTxType, runVmSuccess, true)
				if !applyResult {
					ts.log.Debugf("failed to apply according to dag with tx %s ", tx.Payload.TxId)
					runningTxC <- tx
					continue
				}
				if firstTx == nil {
					firstTx = tx
					dagNeighbors := &commonpb.DAG_Neighbor{
						Neighbors: make([]uint32, 0, snapshot.GetSnapshotSize()-1),
					}
					for i := uint32(0); i < uint32(snapshot.GetSnapshotSize()-1); i++ {
						dagNeighbors.Neighbors = append(dagNeighbors.Neighbors, i)
					}
					dag.Vertexes = append(dag.Vertexes, dagNeighbors)
				} else {
					dagNeighbors := &commonpb.DAG_Neighbor{
						Neighbors: make([]uint32, 0, 1),
					}
					dagNeighbors.Neighbors = append(dagNeighbors.Neighbors, uint32(snapshot.GetSnapshotSize())-2)
					dag.Vertexes = append(dag.Vertexes, dagNeighbors)
				}
				if applySize >= txBatchSize {
					ts.log.Debugf("block [%d] schedule special txs finished, apply size:%d, len of txs:%d, "+
						"len of special txs:%d", block.Header.BlockHeight, applySize, txBatchSize, specialTxsLen)
					scheduleFinishC <- true
					return
				}
			case <-timeoutC:
				ts.log.Errorf("block [%d] schedule special txs timeout", block.Header.BlockHeight)
				scheduleFinishC <- true
				return
			}
		}
	}()
	<-scheduleFinishC
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

func (ts *TxScheduler) runVM(tx *commonpb.Transaction, txSimContext protocol.TxSimContext) (
	*commonpb.Result, protocol.ExecOrderTxType, error) {
	var contractName string
	var method string
	var byteCode []byte

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
	if payload.TxType != commonpb.TxType_QUERY_CONTRACT && payload.TxType != commonpb.TxType_INVOKE_CONTRACT {
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

	contract, err := txSimContext.GetContractByName(contractName)
	if err != nil {
		ts.log.Errorf("Get contract info by name[%s] error:%s", contractName, err)
		return errResult(result, err)
	}
	if contract.RuntimeType != commonpb.RuntimeType_NATIVE {
		byteCode, err = txSimContext.GetContractBytecode(contractName)
		if err != nil {
			ts.log.Errorf("Get contract bytecode by name[%s] error:%s", contractName, err)
			return errResult(result, err)
		}
	}
	contractResultPayload, specialTxType, txStatusCode := ts.VmManager.RunContract(contract, method, byteCode,
		parameters, txSimContext, 0, tx.Payload.TxType)

	result.Code = txStatusCode
	result.ContractResult = contractResultPayload

	if txStatusCode == commonpb.TxStatusCode_SUCCESS {
		return result, specialTxType, nil
	}
	return result, specialTxType, errors.New(contractResultPayload.Message)
}

func errResult(result *commonpb.Result, err error) (*commonpb.Result, protocol.ExecOrderTxType, error) {
	result.ContractResult.Message = err.Error()
	result.Code = commonpb.TxStatusCode_INVALID_PARAMETER
	result.ContractResult.Code = 1
	return result, protocol.ExecOrderTxTypeNormal, err
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

//func (ts *TxScheduler) dumpDAG(dag *commonpb.DAG, txs []*commonpb.Transaction) {
//	dagString := "digraph DAG {\n"
//	for i, ns := range dag.Vertexes {
//		if len(ns.Neighbors) == 0 {
//			dagString += fmt.Sprintf("id_%s -> begin;\n", txs[i].Payload.TxId[:8])
//			continue
//		}
//		for _, n := range ns.Neighbors {
//			dagString += fmt.Sprintf("id_%s -> id_%s;\n", txs[i].Payload.TxId[:8], txs[n].Payload.TxId[:8])
//		}
//	}
//	dagString += "}"
//	ts.log.Infof("Dump Dag: %s", dagString)
//}
