/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package snapshot

import (
	"fmt"
	"sync"

	"go.uber.org/atomic"

	"chainmaker.org/chainmaker/pb-go/v2/accesscontrol"

	"chainmaker.org/chainmaker/localconf/v2"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"

	"chainmaker.org/chainmaker/common/v2/bitmap"
	"chainmaker.org/chainmaker/protocol/v2"
)

// The record value is written by the SEQ corresponding to TX
type sv struct {
	seq   int
	value []byte
}

type SnapshotImpl struct {
	lock            sync.Mutex
	blockchainStore protocol.BlockchainStore

	// If the snapshot has been sealed, the results of subsequent vm execution will not be added to the snapshot
	sealed *atomic.Bool

	chainId        string
	blockTimestamp int64
	blockProposer  *accesscontrol.Member
	blockHeight    uint64
	preBlockHash   []byte

	preSnapshot protocol.Snapshot

	txRWSetTable   []*commonPb.TxRWSet
	txTable        []*commonPb.Transaction
	specialTxTable []*commonPb.Transaction
	txResultMap    map[string]*commonPb.Result
	readTable      map[string]*sv
	writeTable     map[string]*sv

	txRoot    []byte
	dagHash   []byte
	rwSetHash []byte
}

func (s *SnapshotImpl) GetPreSnapshot() protocol.Snapshot {
	return s.preSnapshot
}

func (s *SnapshotImpl) SetPreSnapshot(snapshot protocol.Snapshot) {
	s.preSnapshot = snapshot
}

func (s *SnapshotImpl) GetBlockchainStore() protocol.BlockchainStore {
	return s.blockchainStore
}

func (s *SnapshotImpl) GetSnapshotSize() int {
	s.lock.Lock()
	defer s.lock.Unlock()
	return len(s.txTable)
}

func (s *SnapshotImpl) GetTxTable() []*commonPb.Transaction {
	return s.txTable
}

func (s *SnapshotImpl) GetSpecialTxTable() []*commonPb.Transaction {
	return s.specialTxTable
}

// After the scheduling is completed, get the result from the current snapshot
func (s *SnapshotImpl) GetTxResultMap() map[string]*commonPb.Result {
	return s.txResultMap
}

func (s *SnapshotImpl) GetTxRWSetTable() []*commonPb.TxRWSet {
	if localconf.ChainMakerConfig.SchedulerConfig.RWSetLog {
		log.DebugDynamic(func() string {

			info := "rwset: "
			for i, txRWSet := range s.txRWSetTable {
				info += fmt.Sprintf("read set for tx id:[%s], count [%d]<", s.txTable[i].Payload.TxId, len(txRWSet.TxReads))
				//for _, txRead := range txRWSet.TxReads {
				//	if !strings.HasPrefix(string(txRead.Key), protocol.ContractByteCode) {
				//		info += fmt.Sprintf("[%v] -> [%v], contract name [%v], version [%v],",
				//		txRead.Key, txRead.Value, txRead.ContractName, txRead.Version)
				//	}
				//}
				info += "> "
				info += fmt.Sprintf("write set for tx id:[%s], count [%d]<", s.txTable[i].Payload.TxId, len(txRWSet.TxWrites))
				for _, txWrite := range txRWSet.TxWrites {
					info += fmt.Sprintf("[%v] -> [%v], contract name [%v], ", txWrite.Key, txWrite.Value, txWrite.ContractName)
				}
				info += ">"
			}
			return info
		})
		//log.Debugf(info)
	}

	//for _, txRWSet := range s.txRWSetTable {
	//	for _, txRead := range txRWSet.TxReads {
	//		if strings.HasPrefix(string(txRead.Key), protocol.ContractByteCode) ||
	//			strings.HasPrefix(string(txRead.Key), protocol.ContractCreator) ||
	//			txRead.ContractName == syscontract.SystemContract_CERT_MANAGE.String() {
	//			txRead.Value = nil
	//		}
	//	}
	//}
	return s.txRWSetTable
}

func (s *SnapshotImpl) GetKey(txExecSeq int, contractName string, key []byte) ([]byte, error) {
	// get key before txExecSeq
	snapshotSize := s.GetSnapshotSize()

	s.lock.Lock()
	defer s.lock.Unlock()

	{
		if txExecSeq > snapshotSize || txExecSeq < 0 {
			txExecSeq = snapshotSize //nolint: ineffassign, staticcheck
		}
		finalKey := constructKey(contractName, key)
		if sv, ok := s.writeTable[finalKey]; ok {
			return sv.value, nil
		}

		if sv, ok := s.readTable[finalKey]; ok {
			return sv.value, nil
		}
	}

	iter := s.preSnapshot
	for iter != nil {
		if value, err := iter.GetKey(-1, contractName, key); err == nil {
			return value, nil
		}
		iter = iter.GetPreSnapshot()
	}

	return s.blockchainStore.ReadObject(contractName, key)
}

// ApplyTxSimContext After the read-write set is generated, add TxSimContext to the snapshot
// return the result of application(successfully or not) and current applied tx num
func (s *SnapshotImpl) ApplyTxSimContext(txSimContext protocol.TxSimContext, specialTxType protocol.ExecOrderTxType,
	runVmSuccess bool, applySpecialTx bool) (bool, int) {
	tx := txSimContext.GetTx()
	log.Debugf("apply tx: %s, execOrderTxType:%d, runVmSuccess:%v, applySpecialTx:%v", tx.Payload.TxId,
		specialTxType, runVmSuccess, applySpecialTx)
	if !applySpecialTx && s.IsSealed() {
		return false, s.GetSnapshotSize()
	}

	s.lock.Lock()
	defer s.lock.Unlock()
	// it is necessary to check sealed secondly
	if !applySpecialTx && s.IsSealed() {
		return false, s.GetSnapshotSize()
	}

	txExecSeq := txSimContext.GetTxExecSeq()
	var txRWSet *commonPb.TxRWSet
	var txResult *commonPb.Result

	if !applySpecialTx && specialTxType == protocol.ExecOrderTxTypeIterator {
		s.specialTxTable = append(s.specialTxTable, tx)
		return true, len(s.txTable) + len(s.specialTxTable)
	}

	// Only when the virtual machine is running normally can the read-write set be saved, or write fake conflicted key
	txRWSet = txSimContext.GetTxRWSet(runVmSuccess)
	txResult = txSimContext.GetTxResult()

	if specialTxType == protocol.ExecOrderTxTypeIterator || txExecSeq >= len(s.txTable) {
		s.apply(tx, txRWSet, txResult)
		return true, len(s.txTable)
	}

	// Check whether the dependent state has been modified during the run
	for _, txRead := range txRWSet.TxReads {
		finalKey := constructKey(txRead.ContractName, txRead.Key)
		if sv, ok := s.writeTable[finalKey]; ok {
			if sv.seq >= txExecSeq {
				log.Debugf("Key Conflicted %+v-%+v", sv.seq, txExecSeq)
				return false, len(s.txTable)
			}
		}
	}

	s.apply(tx, txRWSet, txResult)
	return true, len(s.txTable)
}

// After the read-write set is generated, add TxSimContext to the snapshot
func (s *SnapshotImpl) apply(tx *commonPb.Transaction, txRWSet *commonPb.TxRWSet, txResult *commonPb.Result) {
	// Append to read table
	applySeq := len(s.txTable)
	for _, txRead := range txRWSet.TxReads {
		finalKey := constructKey(txRead.ContractName, txRead.Key)
		s.readTable[finalKey] = &sv{
			seq:   applySeq,
			value: txRead.Value,
		}
	}

	// Append to write table
	for _, txWrite := range txRWSet.TxWrites {
		finalKey := constructKey(txWrite.ContractName, txWrite.Key)
		s.writeTable[finalKey] = &sv{
			seq:   applySeq,
			value: txWrite.Value,
		}
	}

	// Append to read-write-set table
	s.txRWSetTable = append(s.txRWSetTable, txRWSet)
	log.Debugf("apply tx: %s, rwset table size %d", tx.Payload.TxId, len(s.txRWSetTable))

	// Add to tx result map
	s.txResultMap[tx.Payload.TxId] = txResult

	// Add to transaction table
	s.txTable = append(s.txTable, tx)
}

// check if snapshot is sealed
func (s *SnapshotImpl) IsSealed() bool {
	return s.sealed.Load()
}

// get block height for current snapshot
func (s *SnapshotImpl) GetBlockHeight() uint64 {
	return s.blockHeight
}

// GetBlockTimestamp returns current block timestamp
func (s *SnapshotImpl) GetBlockTimestamp() int64 {
	return s.blockTimestamp
}

// Get Block Proposer for current snapshot
func (s *SnapshotImpl) GetBlockProposer() *accesscontrol.Member {
	return s.blockProposer
}

// seal the snapshot
func (s *SnapshotImpl) Seal() {
	s.sealed.Store(true)
}

// Build txs' read bitmap and write bitmap, so we can use AND to simplify read/write set conflict detection process.
// keyDict: key string -> key index in bitmap, e.g., key1 -> 0, key2 -> 1, key3 -> 2
// read/write Table:	tx1: {key1->value1, key3->value3}; tx2: {key2->value2, key3->value4}
// read/write bitmap: 			key1	key2	key3
//						tx1		1		0		1
// 						tx2		0		1		1
func (s *SnapshotImpl) buildRWBitmaps() ([]*bitmap.Bitmap, []*bitmap.Bitmap) {
	dictIndex := 0
	txCount := len(s.txTable)
	readBitmap := make([]*bitmap.Bitmap, txCount)
	writeBitmap := make([]*bitmap.Bitmap, txCount)
	keyDict := make(map[string]int, 1024)
	for i := 0; i < txCount; i++ {
		readTableItemForI := s.txRWSetTable[i].TxReads
		writeTableItemForI := s.txRWSetTable[i].TxWrites

		readBitmap[i] = &bitmap.Bitmap{}
		for _, keyForI := range readTableItemForI {
			if existIndex, ok := keyDict[string(keyForI.Key)]; !ok {
				keyDict[string(keyForI.Key)] = dictIndex
				readBitmap[i].Set(dictIndex)
				dictIndex++
			} else {
				readBitmap[i].Set(existIndex)
			}
		}

		writeBitmap[i] = &bitmap.Bitmap{}
		for _, keyForI := range writeTableItemForI {
			if existIndex, ok := keyDict[string(keyForI.Key)]; !ok {
				keyDict[string(keyForI.Key)] = dictIndex
				writeBitmap[i].Set(dictIndex)
				dictIndex++
			} else {
				writeBitmap[i].Set(existIndex)
			}
		}
	}
	return readBitmap, writeBitmap
}

func (s *SnapshotImpl) buildCumulativeBitmap(readBitmap []*bitmap.Bitmap,
	writeBitmap []*bitmap.Bitmap) ([]*bitmap.Bitmap, []*bitmap.Bitmap) {
	cumulativeReadBitmap := make([]*bitmap.Bitmap, len(readBitmap))
	cumulativeWriteBitmap := make([]*bitmap.Bitmap, len(writeBitmap))

	for i, b := range readBitmap {
		cumulativeReadBitmap[i] = b.Clone()
		if i > 0 {
			cumulativeReadBitmap[i].Or(cumulativeReadBitmap[i-1])
		}
	}
	for i, b := range writeBitmap {
		cumulativeWriteBitmap[i] = b.Clone()
		if i > 0 {
			cumulativeWriteBitmap[i].Or(cumulativeWriteBitmap[i-1])
		}
	}
	return cumulativeReadBitmap, cumulativeWriteBitmap
}

// According to the read-write table, the read-write dependency is checked from back to front to determine whether
// the transaction can be executed concurrently.
// From the process of building the read-write table, we have known that every transaction is based on a known
// world state, or cache state. As long as the world state or cache state that the tx depends on does not
// change during the execution, then the execution result of the transaction is determined.
// We need to ensure that when validating the DAG, there is no possibility that the execution of other
// transactions will affect the dependence of the current transaction
func (s *SnapshotImpl) BuildDAG(isSql bool) *commonPb.DAG {
	if !s.IsSealed() {
		log.Warnf("you need to execute Seal before you can build DAG of snapshot with height %d", s.blockHeight)
	}
	s.lock.Lock()
	defer s.lock.Unlock()

	txCount := len(s.txTable)
	log.Debugf("start building DAG for block %d with %d txs", s.blockHeight, txCount)

	// build read-write bitmap for all transactions
	readBitmaps, writeBitmaps := s.buildRWBitmaps()
	cumulativeReadBitmap, cumulativeWriteBitmap := s.buildCumulativeBitmap(readBitmaps, writeBitmaps)

	dag := &commonPb.DAG{}
	if txCount == 0 {
		return dag
	}

	dag.Vertexes = make([]*commonPb.DAG_Neighbor, txCount)

	// build DAG base on read and write bitmaps
	// reachMap describes reachability from tx i to tx j in DAG.
	// For example, if the DAG is tx3 -> tx2 -> tx1 -> begin, the reachMap is
	// 		tx1		tx2		tx3
	// tx1	0		0		0
	// tx2	1		0		0
	// tx3	1		1		0
	reachMap := make([]*bitmap.Bitmap, txCount)
	if isSql {
		for i := 0; i < txCount; i++ {
			dag.Vertexes[i] = &commonPb.DAG_Neighbor{
				Neighbors: make([]uint32, 0, 1),
			}
			if i != 0 {
				dag.Vertexes[i].Neighbors = append(dag.Vertexes[i].Neighbors, uint32(i-1))
			}
		}
	} else {
		for i := 0; i < txCount; i++ {
			// 1、get read and write bitmap for tx i
			readBitmapForI := readBitmaps[i]
			writeBitmapForI := writeBitmaps[i]

			// directReachFromI is used to build DAG, it's the direct neighbors of the ith tx
			directReachFromI := &bitmap.Bitmap{}
			// reachFromI is used to save reachability we have already known, it's the all neighbors of the ith tx
			reachFromI := &bitmap.Bitmap{}
			reachFromI.Set(i)

			if i > 0 && s.fastConflicted(
				readBitmapForI, writeBitmapForI, cumulativeReadBitmap[i-1], cumulativeWriteBitmap[i-1]) {
				// check reachability one by one, then build table
				s.buildReach(i, reachFromI, readBitmaps, writeBitmaps, readBitmapForI, writeBitmapForI, directReachFromI, reachMap)
			}
			reachMap[i] = reachFromI

			// build DAG based on directReach bitmap
			dag.Vertexes[i] = &commonPb.DAG_Neighbor{
				Neighbors: make([]uint32, 0, 16),
			}
			for _, j := range directReachFromI.Pos1() {
				dag.Vertexes[i].Neighbors = append(dag.Vertexes[i].Neighbors, uint32(j))
			}
		}
	}
	log.Debugf("build DAG for block %d finished", s.blockHeight)
	return dag
}

// check reachability one by one, then build table
func (s *SnapshotImpl) buildReach(i int, reachFromI *bitmap.Bitmap,
	readBitmaps []*bitmap.Bitmap, writeBitmaps []*bitmap.Bitmap,
	readBitmapForI *bitmap.Bitmap, writeBitmapForI *bitmap.Bitmap,
	directReachFromI *bitmap.Bitmap, reachMap []*bitmap.Bitmap) {

	for j := i - 1; j >= 0; j-- {
		if reachFromI.Has(j) {
			continue
		}

		readBitmapForJ := readBitmaps[j]
		writeBitmapForJ := writeBitmaps[j]
		if s.conflicted(readBitmapForI, writeBitmapForI, readBitmapForJ, writeBitmapForJ) {
			directReachFromI.Set(j)
			reachFromI.Or(reachMap[j])
		}
	}
}

// Conflict cases: I read & J write; I write & J read; I write & J write
func (s *SnapshotImpl) conflicted(readBitmapForI, writeBitmapForI,
	readBitmapForJ, writeBitmapForJ *bitmap.Bitmap) bool {
	if readBitmapForI.InterExist(writeBitmapForJ) ||
		writeBitmapForI.InterExist(writeBitmapForJ) ||
		writeBitmapForI.InterExist(readBitmapForJ) {
		return true
	}
	return false
}

// fast conflict cases: I read & J write; I write & J read; I write & J write
func (s *SnapshotImpl) fastConflicted(readBitmapForI, writeBitmapForI, cumulativeReadBitmap,
	cumulativeWriteBitmap *bitmap.Bitmap) bool {
	if readBitmapForI.InterExist(cumulativeWriteBitmap) ||
		writeBitmapForI.InterExist(cumulativeWriteBitmap) ||
		writeBitmapForI.InterExist(cumulativeReadBitmap) {
		return true
	}
	return false
}

func constructKey(contractName string, key []byte) string {
	return contractName + string(key)
}
