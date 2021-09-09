/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package batch

import (
	"errors"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"chainmaker.org/chainmaker-go/txpool/poolconf"
	commonErrors "chainmaker.org/chainmaker/common/v2/errors"
	"chainmaker.org/chainmaker/common/v2/msgbus"
	"chainmaker.org/chainmaker/common/v2/queue/lockfreequeue"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	netPb "chainmaker.org/chainmaker/pb-go/v2/net"
	txpoolPb "chainmaker.org/chainmaker/pb-go/v2/txpool"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/utils/v2"

	"github.com/gogo/protobuf/proto"
)

const (
	DefaultBatchMaxSize       = 50000
	DefaultBatchCreateTimeout = 1000 * time.Millisecond
	DefaultPoolSize           = 10000
)

// BatchTxPool Another implementation of tx pool, which can only be used in non-Hotstuff consensus algorithms
type BatchTxPool struct {
	stat               int32         // Identification of module service startup
	nodeId             string        // The ID of node
	chainId            string        // The ID of chain
	maxTxCount         int32         // The maximum number of transactions cached by the txPool
	currentTxCount     int32         //	The number of transactions currently cached in the txPool
	batchMaxSize       int32         // Maximum number of transactions included in each batch
	currentBatchId     int32         // The current batch ID
	batchCreateTimeout time.Duration // The creation time of each batch

	txQueue           *lockfreequeue.Queue // Temporarily cache common transactions received from the network
	commonBatchPool   *nodeBatchPool       // Stores batches of common transactions
	cfgTxQueue        *lockfreequeue.Queue // Temporarily cache config transactions received from the network
	configBatchPool   *nodeBatchPool       // Stores batches of configuration transactions
	batchTxIdRecorder *batchTxIdRecorder   // Stores transaction information within batches
	pendingPool       *pendingBatchPool    // Stores batches to be deleted

	mb         msgbus.MessageBus              // Receive messages from other modules
	ac         protocol.AccessControlProvider // Verify transaction signature
	log        protocol.Logger                //
	chainConf  protocol.ChainConf             //
	chainStore protocol.BlockchainStore       // Access information on the chain

	fetchLock sync.Mutex    // The protection of the FETCH function allows only one FETCH at a time
	stopCh    chan struct{} // Signal notification of service exit
}

func NewBatchTxPool(nodeId string, chainId string, chainConf protocol.ChainConf,
	chainStore protocol.BlockchainStore, ac protocol.AccessControlProvider, log protocol.Logger) *BatchTxPool {

	return &BatchTxPool{
		nodeId:  nodeId,
		chainId: chainId,

		ac:         ac,
		chainConf:  chainConf,
		chainStore: chainStore,
		stopCh:     make(chan struct{}),
		log:        log,

		maxTxCount:         int32(poolconf.DefaultMaxTxPoolSize),
		batchMaxSize:       int32(DefaultBatchMaxSize),
		batchCreateTimeout: DefaultBatchCreateTimeout,
		pendingPool:        newPendingBatchPool(),
		configBatchPool:    newNodeBatchPool(),
		commonBatchPool:    newNodeBatchPool(),
		batchTxIdRecorder:  newBatchTxIdRecorder(),
	}
}

func (p *BatchTxPool) SetPoolSize(size int) {
	atomic.StoreInt32(&p.maxTxCount, int32(size))
}

func (p *BatchTxPool) SetBatchMaxSize(size int) {
	atomic.StoreInt32(&p.batchMaxSize, int32(size))
}

func (p *BatchTxPool) SetBatchCreateTimeout(timeout time.Duration) {
	p.batchCreateTimeout = timeout
}

func (p *BatchTxPool) SetMsgBus(msgBus msgbus.MessageBus) {
	p.mb = msgBus
}

func (p *BatchTxPool) Start() error {
	if !atomic.CompareAndSwapInt32(&p.stat, 0, 1) {
		return commonErrors.ErrTxPoolHasStarted
	}
	p.txQueue = lockfreequeue.NewQueue(uint32(p.maxTxCount))
	p.cfgTxQueue = lockfreequeue.NewQueue(uint32(10))
	if p.mb != nil {
		p.mb.Register(msgbus.RecvTxPoolMsg, p)
	}
	go p.createBatchLoop()
	return nil
}

func (p *BatchTxPool) Stop() error {
	if !atomic.CompareAndSwapInt32(&p.stat, 1, 0) {
		return commonErrors.ErrTxPoolHasStopped
	}
	close(p.stopCh)
	return nil
}

func (p *BatchTxPool) AddTx(tx *commonPb.Transaction, src protocol.TxSource) error {
	if atomic.LoadInt32(&p.stat) == 0 {
		return errors.New("batch tx pool not started")
	}
	if atomic.LoadInt32(&p.currentTxCount) >= p.maxTxCount {
		return fmt.Errorf("batch tx pool is full, count: %d, maxCount: %d", p.currentTxCount, p.maxTxCount)
	}
	if err := p.validate(tx, src); err != nil {
		return err
	}
	// 1. push tx to config queue
	if utils.IsConfigTx(tx) {
		if ok, _ := p.cfgTxQueue.Push(tx); !ok {
			return errors.New("push cfg tx to cfg queue failed because queue is full")
		}
		atomic.AddInt32(&p.currentTxCount, 1)
		return nil
	}

	// 2. push tx to common queue
	if ok, _ := p.txQueue.Push(tx); !ok {
		return errors.New("push tx to tx queue failed because queue is full")
	}
	atomic.AddInt32(&p.currentTxCount, 1)
	return nil
}

func (p *BatchTxPool) createBatchLoop() {
	for {
		select {
		case <-p.stopCh:
			return
		default:
		}
		p.createConfigTxBatch()
		p.createCommonTxBatch()
		time.Sleep(500 * time.Millisecond)
	}
}

func (p *BatchTxPool) createConfigTxBatch() {
	val, ok, _ := p.cfgTxQueue.Pull()
	if !ok {
		return
	}

	tx, ok := val.(*commonPb.Transaction)
	if !ok {
		p.log.Errorf("transfer val interface into *commonPb.Transaction failed")
		return
	}
	batchId := atomic.AddInt32(&p.currentBatchId, 1)
	batch := &txpoolPb.TxBatch{
		BatchId: batchId,
		NodeId:  p.nodeId,

		Txs:      []*commonPb.Transaction{tx},
		TxIdsMap: make(map[string]int32),
		Size_:    1,
	}
	batch.TxIdsMap[tx.Payload.GetTxId()] = 0

	batchMsg, err := proto.Marshal(batch)
	if err != nil {
		p.log.Errorf("marshal batch failed, %s", err.Error())
		return
	}
	// put batch to cfg batch pool
	if !p.configBatchPool.PutIfNotExist(batch) {
		p.log.Errorf("put cfg batch to config batch pool failed")
		return
	}
	p.batchTxIdRecorder.AddRecordWithBatch(batch)
	// tell core engine to package
	p.publishSignal()
	// broadcast batch to other nodes
	if err := p.broadcastTxBatch(batchId, batchMsg); err != nil {
		p.log.Errorf("broadcast cfg batch failed, %s", err.Error())
		return
	}
}

func (p *BatchTxPool) popTxsFromQueue() ([]*commonPb.Transaction, map[string]int32) {
	var (
		txs         = make([]*commonPb.Transaction, 0, p.batchMaxSize/2)
		txIdToIndex = make(map[string]int32, p.batchMaxSize/2)
		timer       = time.NewTimer(p.batchCreateTimeout)
	)
	defer timer.Stop()
	for i := 0; i < int(p.batchMaxSize); {
		if val, ok, _ := p.txQueue.Pull(); ok {
			tx, ok := val.(*commonPb.Transaction)
			if !ok {
				p.log.Errorf("transfer val interface into *commonPb.Transaction failed")
			}
			if _, ok := txIdToIndex[tx.Payload.GetTxId()]; ok {
				continue
			}
			txs = append(txs, tx)
			txIdToIndex[tx.Payload.GetTxId()] = int32(i)
			i++
			continue
		}
		select {
		case <-timer.C:
			return txs, txIdToIndex
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
	return txs, txIdToIndex
}

func (p *BatchTxPool) createCommonTxBatch() {
	if p.txQueue.Quantity() < 1 {
		return
	}

	txs, txIds := p.popTxsFromQueue()
	batchId := atomic.AddInt32(&p.currentBatchId, 1)
	batch := &txpoolPb.TxBatch{
		BatchId: batchId,
		NodeId:  p.nodeId,

		Txs:      txs,
		TxIdsMap: txIds,
		Size_:    int32(len(txs)),
	}
	p.log.Infof("create txBatch size: %d, batchId: %d, txMapLen: %d, txsLen: %d, totalTxCount: %d",
		batch.GetSize_(), batch.BatchId, len(batch.TxIdsMap), len(batch.Txs),
		atomic.LoadInt32(&p.currentTxCount)+int32(batch.GetSize_()))

	var (
		err      error
		batchMsg []byte
	)
	if batchMsg, err = proto.Marshal(batch); err != nil {
		p.log.Errorf("marshal batch failed, %s", err.Error())
		return
	}

	// put batch to normal batch pool
	if !p.commonBatchPool.PutIfNotExist(batch) {
		p.log.Errorf("put batch to normal batch pool failed")
		return
	}
	p.batchTxIdRecorder.AddRecordWithBatch(batch)
	// tell core engine to package
	p.publishSignal()
	// broadcast batch to other nodes
	if err := p.broadcastTxBatch(batchId, batchMsg); err != nil {
		p.log.Errorf("broadcast normal batch failed, %s", err.Error())
		return
	}
}

func (p *BatchTxPool) broadcastTxBatch(batchId int32, batchMsg []byte) error {
	if p.mb == nil {
		return nil
	}
	netMsg := &netPb.NetMsg{
		Payload: batchMsg,
		Type:    netPb.NetMsg_TX,
	}
	p.mb.Publish(msgbus.SendTxPoolMsg, netMsg)
	p.log.Infof("broadcast tx batch [%d]", batchId)
	return nil
}

func (p *BatchTxPool) GetTxsByTxIds(txIds []string) (map[string]*commonPb.Transaction, map[string]uint64) {
	if atomic.LoadInt32(&p.stat) == 0 {
		p.log.Errorf(commonErrors.ErrTxPoolHasStopped.String())
		return nil, nil
	}
	for _, txId := range txIds {
		batchId, _, ok := p.batchTxIdRecorder.FindBatchIdWithTxId(txId)
		if !ok {
			continue
		}
		if batch := p.getBatch(batchId); batch != nil {
			return p.generateTxsInfoFromBatch(batch)
		}
	}
	return nil, nil
}

func (p *BatchTxPool) getBatch(batchId int32) *txpoolPb.TxBatch {
	if batch := p.pendingPool.GetBatch(batchId); batch != nil {
		return batch
	}
	if batch := p.commonBatchPool.GetBatch(batchId); batch != nil {
		return batch
	}
	batch := p.configBatchPool.GetBatch(batchId)
	return batch

}

func (p *BatchTxPool) generateTxsInfoFromBatch(batch *txpoolPb.TxBatch) (map[string]*commonPb.Transaction,
	map[string]uint64) {
	txsRet := make(map[string]*commonPb.Transaction, batch.Size_)
	txsHeightInfo := make(map[string]uint64)

	for _, tx := range batch.Txs {
		txsRet[tx.Payload.TxId] = tx
	}
	return txsRet, txsHeightInfo
}

func (p *BatchTxPool) GetTxByTxId(txId string) (tx *commonPb.Transaction, inBlockHeight uint64) {
	if atomic.LoadInt32(&p.stat) == 0 {
		p.log.Errorf(commonErrors.ErrTxPoolHasStopped.String())
		return nil, math.MaxUint64
	}
	batchId, txIndex, exist := p.batchTxIdRecorder.FindBatchIdWithTxId(txId)
	if !exist {
		return nil, math.MaxUint64
	}
	if batch := p.getBatch(batchId); batch != nil {
		return batch.Txs[txIndex], inBlockHeight
	}
	return nil, math.MaxUint64
}

func (p *BatchTxPool) TxExists(tx *commonPb.Transaction) bool {
	if atomic.LoadInt32(&p.stat) == 0 {
		p.log.Errorf("batch tx pool not started, start it first pls")
		return false
	}
	_, _, exist := p.batchTxIdRecorder.FindBatchIdWithTxId(tx.Payload.TxId)
	return exist
}

func (p *BatchTxPool) FetchTxBatch(blockHeight uint64) []*commonPb.Transaction {
	if atomic.LoadInt32(&p.stat) == 0 {
		p.log.Errorf("batch tx pool not started, start it first pls")
		return nil
	}
	p.fetchLock.Lock()
	defer p.fetchLock.Unlock()
	if txs := p.fetchTxsFromCfgPool(blockHeight); len(txs) > 0 {
		return txs
	}
	return p.fetchTxsFromCommonPool(blockHeight)
}

func (p *BatchTxPool) fetchTxsFromCfgPool(blockHeight uint64) []*commonPb.Transaction {
	var (
		batch   *txpoolPb.TxBatch
		cfgPool = p.configBatchPool
	)
	cfgPool.pool.Range(func(val interface{}) (isContinue bool) {
		if val == nil {
			return true
		}
		var ok bool
		batch, ok = val.(*txpoolPb.TxBatch)
		if !ok {
			p.log.Errorf("transfer val interface into *txpoolPb.TxBatch failed")
		}
		return false
	})
	if batch != nil {
		filterTxs := p.moveBatch(cfgPool, batch)
		return filterTxs
	}
	return nil
}

func (p *BatchTxPool) moveBatch(removePool *nodeBatchPool, batch *txpoolPb.TxBatch) []*commonPb.Transaction {
	if ok := removePool.RemoveIfExist(batch); !ok {
		p.log.Errorf("remove batch failed, batch(batch id:%d) not exist in normal batch pool", batch.GetBatchId())
		return nil
	}
	if pendingOk := p.pendingPool.PutIfNotExist(batch); !pendingOk {
		p.log.Errorf("pending batch failed, batch(batch id:%d) exist in pending batch pool", batch.GetBatchId())
		return nil
	}
	return batch.Txs
}

func (p *BatchTxPool) fetchTxsFromCommonPool(blockHeight uint64) []*commonPb.Transaction {
	var (
		batch         *txpoolPb.TxBatch
		commonTxsPool = p.commonBatchPool
		ok            bool
	)
	commonTxsPool.pool.Range(func(val interface{}) (isContinue bool) {
		if val == nil {
			return true
		}
		batch, ok = val.(*txpoolPb.TxBatch)
		if !ok {
			p.log.Errorf("transfer val interface into *txpoolPb.TxBatch failed")
		}
		return false
	})
	if batch != nil {
		filterTxs := p.moveBatch(commonTxsPool, batch)
		return filterTxs
	}
	return nil
}

//func (p *BatchTxPool) filterTxs(txs []*commonPb.Transaction) []*commonPb.Transaction {
//	noExistInDb := make([]*commonPb.Transaction, 0, len(txs))
//	for _, tx := range txs {
//		if !p.isTxExistInDB(tx) {
//			noExistInDb = append(noExistInDb, tx)
//		}
//	}
//	return noExistInDb
//}

func (p *BatchTxPool) RetryAndRemoveTxs(retryTxs []*commonPb.Transaction, removeTxs []*commonPb.Transaction) {
	if atomic.LoadInt32(&p.stat) == 0 {
		return
	}
	defer p.publishSignal()
	p.log.Debugf("retry txs num: %d, remove txs num: %d", len(retryTxs), len(removeTxs))
	p.retryTxBatch(retryTxs)
	p.removeTxBatch(removeTxs)
}

// todo. may be update logic
func (p *BatchTxPool) retryTxBatch(txs []*commonPb.Transaction) {
	if len(txs) == 0 {
		return
	}
	if _, _, ok := p.batchTxIdRecorder.FindBatchIdWithTxId(txs[0].Payload.TxId); ok {
		return
	}

	batch := &txpoolPb.TxBatch{
		Txs:      txs,
		NodeId:   p.nodeId,
		Size_:    int32(len(txs)),
		TxIdsMap: createTxIdsMap(txs),
		BatchId:  atomic.AddInt32(&p.currentBatchId, 1),
	}
	p.batchTxIdRecorder.AddRecordWithBatch(batch)
	if utils.IsConfigTx(txs[0]) {
		p.configBatchPool.PutIfNotExist(batch)
	} else {
		p.commonBatchPool.PutIfNotExist(batch)
	}
	atomic.AddInt32(&p.currentTxCount, batch.Size_)
}

func createTxIdsMap(txs []*commonPb.Transaction) map[string]int32 {
	txIdsMap := make(map[string]int32)
	for idx, tx := range txs {
		txIdsMap[tx.Payload.GetTxId()] = int32(idx)
	}
	return txIdsMap
}

func (p *BatchTxPool) removeTxBatch(txs []*commonPb.Transaction) {
	if len(txs) == 0 {
		return
	}
	var (
		ok      bool
		remove  = false
		batchId int32
		txId    = txs[0].Payload.GetTxId()
	)
	if batchId, _, ok = p.batchTxIdRecorder.FindBatchIdWithTxId(txId); !ok {
		p.log.Warnf("batch id not found,ignored. (tx id:%s) when removeTxBatch", txId)
		return
	}

	batch := &txpoolPb.TxBatch{NodeId: p.nodeId, BatchId: batchId, Txs: txs, TxIdsMap: createTxIdsMap(txs),
		Size_: int32(len(txs))}
	if p.pendingPool.GetBatch(batchId) != nil {
		batch.Size_ = p.pendingPool.GetBatch(batchId).GetSize_()
	} else if utils.IsConfigTx(txs[0]) {
		if p.configBatchPool.GetBatch(batchId) != nil {
			batch.Size_ = p.configBatchPool.GetBatch(batchId).GetSize_()
		}
	} else {
		if p.commonBatchPool.GetBatch(batchId) != nil {
			batch.Size_ = p.commonBatchPool.GetBatch(batchId).GetSize_()
		}
	}

	if p.pendingPool.RemoveIfExist(batch) {
		remove = true
		p.log.Infof("remove txs[%d] from pending pool, current pool size [%d]", len(txs),
			atomic.LoadInt32(&p.currentTxCount)-int32(len(txs)))
	} else if utils.IsConfigTx(txs[0]) {
		if p.configBatchPool.RemoveIfExist(batch) {
			p.log.Infof("remove txs[%d] from config tx pool, current pool size [%d]", len(txs),
				atomic.LoadInt32(&p.currentTxCount)-int32(len(txs)))
			remove = true
		}
	} else {
		if p.commonBatchPool.RemoveIfExist(batch) {
			p.log.Infof("remove txs[%d] from common tx pool, current pool size [%d]", len(txs),
				atomic.LoadInt32(&p.currentTxCount)-int32(len(txs)))
			remove = true
		}
	}
	if remove {
		atomic.AddInt32(&p.currentTxCount, 0-batch.GetSize_())
		p.log.Infof("current txs num: %d", atomic.LoadInt32(&p.currentTxCount))
		p.batchTxIdRecorder.RemoveRecordWithBatch(batch)
	} else {
		p.log.Errorf("remove batch failed, batch(batch id:%d) not found in all batch pool", batchId)
	}
}

func (p *BatchTxPool) publishSignal() {
	if p.mb == nil {
		return
	}
	if p.configBatchPool.currentSize() > 0 || p.commonBatchPool.currentSize() > 0 {
		p.mb.Publish(msgbus.TxPoolSignal, &txpoolPb.TxPoolSignal{
			SignalType: txpoolPb.SignalType_BLOCK_PROPOSE,
			ChainId:    p.chainId,
		})
	}
}

func (p *BatchTxPool) OnMessage(message *msgbus.Message) {
	switch message.Topic {
	case msgbus.RecvTxPoolMsg:
		batch := &txpoolPb.TxBatch{}
		if err := proto.Unmarshal(message.Payload.(*netPb.NetMsg).Payload, batch); err != nil {
			p.log.Errorf("unmarshal batch failed, %s", err.Error())
			return
		}
		if len(batch.Txs) != len(batch.TxIdsMap) && len(batch.Txs) != int(batch.Size_) {
			p.log.Errorf("invalid batch info, txs num[%d], txsMap num[%d], "+
				"internal size[%d]\n", len(batch.Txs), len(batch.TxIdsMap), batch.Size_)
			return
		}

		filterTxs := make([]*commonPb.Transaction, 0, batch.Size_)
		for txId, index := range batch.TxIdsMap {
			if batch.Txs[index].Payload.TxId != txId {
				p.log.Errorf("Malicious batch, %d tx's Id in"+
					" map is %s, but actual is %s", index, txId, batch.Txs[index].Payload.TxId)
				return
			}
			if err := p.validate(batch.Txs[index], protocol.P2P); err == nil {
				filterTxs = append(filterTxs, batch.Txs[index])
			}
		}
		if len(filterTxs) != len(batch.Txs) {
			batch = &txpoolPb.TxBatch{
				Txs:      filterTxs,
				Size_:    int32(len(filterTxs)),
				TxIdsMap: createTxIdsMap(filterTxs),
			}
		}
		p.log.Infof("receive batch from node:[%s], size:[%d], actual "+
			"tx num:[%d], batchId:[%d]", batch.NodeId, batch.Size, len(batch.Txs), batch.BatchId)
		if err := p.addTxBatch(batch); err != nil {
			p.log.Errorf("add batch failed, %s", err.Error())
		}
	default:
		p.log.Errorf("msg is not from topic wanted(wanted:%s,curr:%s)", msgbus.RecvTxPoolMsg.String(), message.Topic.String())
	}
}

func (p *BatchTxPool) addTxBatch(batch *txpoolPb.TxBatch) error {
	if atomic.LoadInt32(&p.stat) == 0 {
		return errors.New("batch tx pool not started")
	}
	if atomic.LoadInt32(&p.currentTxCount) >= p.maxTxCount {
		return errors.New("batch tx pool is full")
	}
	batch.NodeId = p.nodeId
	batch.BatchId = atomic.AddInt32(&p.currentBatchId, 1)
	p.batchTxIdRecorder.AddRecordWithBatch(batch)
	firstTx := batch.GetTxs()[0]
	if utils.IsConfigTx(firstTx) {
		p.configBatchPool.PutIfNotExist(batch)
	} else {
		p.commonBatchPool.PutIfNotExist(batch)
	}
	atomic.AddInt32(&p.currentTxCount, batch.Size_)
	return nil
}

func (p *BatchTxPool) OnQuit() {
	// no implement
}

func (p *BatchTxPool) AddTxsToPendingCache(txs []*commonPb.Transaction, blockHeight uint64) {
	// no implement, Because it's only implemented in the Hotstuff algorithm.
}
