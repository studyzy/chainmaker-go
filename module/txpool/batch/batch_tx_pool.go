/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package batch

import (
	"errors"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	commonErrors "chainmaker.org/chainmaker-go/common/errors"
	"chainmaker.org/chainmaker-go/common/msgbus"
	"chainmaker.org/chainmaker-go/common/queue/lockfreequeue"
	"chainmaker.org/chainmaker-go/logger"
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	netPb "chainmaker.org/chainmaker-go/pb/protogo/net"
	txpoolPb "chainmaker.org/chainmaker-go/pb/protogo/txpool"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/utils"

	"github.com/gogo/protobuf/proto"
)

const (
	DefaultBatchMaxSize       = 50000
	DefaultBatchCreateTimeout = 1000 * time.Millisecond
	DefaultPoolSize           = 10000
	errNews                   = "implement me"
)

type BatchTxPool struct {
	stat               int32         // Identification of module service startup
	nodeId             string        // The ID of node
	chainId            string        // The ID of chain
	maxTxCount         int32         // The maximum number of transactions cached by the txPool
	currentTxCount     int32         //	The number of transactions currently cached in the txPool
	batchMaxSize       int32         // Maximum number of transactions included in each batch
	currentBatchId     int32         // The current batch ID
	batchCreateTimeout time.Duration // The creation time of each batch
	batchFetchHeight   sync.Map

	txQueue           *lockfreequeue.Queue // Temporarily cache common transactions received from the network
	commonBatchPool   *nodeBatchPool       // Stores batches of common transactions
	cfgTxQueue        *lockfreequeue.Queue // Temporarily cache config transactions received from the network
	configBatchPool   *nodeBatchPool       // Stores batches of configuration transactions
	batchTxIdRecorder *batchTxIdRecorder   // Stores transaction information within batches
	pendingPool       *pendingBatchPool    // Stores batches to be deleted

	mb         msgbus.MessageBus              // Receive messages from other modules
	ac         protocol.AccessControlProvider // Verify transaction signature
	log        *logger.CMLogger               //
	chainConf  protocol.ChainConf             //
	chainStore protocol.BlockchainStore       // Access information on the chain

	fetchLock sync.Mutex    // The protection of the FETCH function allows only one FETCH at a time
	stopCh    chan struct{} // Signal notification of service exit
}

func NewBatchTxPool(nodeId string, chainId string) *BatchTxPool {
	return &BatchTxPool{
		nodeId:  nodeId,
		chainId: chainId,
		stopCh:  make(chan struct{}),
		log:     logger.GetLoggerByChain(logger.MODULE_TXPOOL, chainId),

		maxTxCount:         int32(DefaultPoolSize),
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

	tx := val.(*commonPb.Transaction)
	batchId := atomic.AddInt32(&p.currentBatchId, 1)
	batch := &txpoolPb.TxBatch{
		BatchId: batchId,
		NodeId:  p.nodeId,

		Txs:      []*commonPb.Transaction{tx},
		TxIdsMap: make(map[string]int32),
		Size_:    1,
	}
	batch.TxIdsMap[tx.GetHeader().GetTxId()] = 0

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
	p.batchFetchHeight.Store(batch.BatchId, int64(0))
	// tell core engine to package
	p.publishSignal()
	// broadcast batch to other nodes
	if err := p.broadcastTxBatch(batchId, batchMsg); err != nil {
		p.log.Errorf("broadcast cfg batch failed, %s", err.Error())
		return
	}
}

func (p *BatchTxPool) createCommonTxBatch() {
	val, ok, _ := p.txQueue.Pull()
	if !ok {
		return
	}

	var (
		err      error
		timeout  bool
		batchMsg []byte

		tx    = val.(*commonPb.Transaction)
		txs   = make([]*commonPb.Transaction, 1)
		txIds = make(map[string]int32)
		timer = time.NewTimer(p.batchCreateTimeout)
	)
	txs[0] = tx
	txIds[tx.GetHeader().GetTxId()] = 0
	defer timer.Stop()

	for i := 1; i < int(p.batchMaxSize); i++ {
		if val, ok, _ = p.txQueue.Pull(); ok {
			tx := val.(*commonPb.Transaction)
			txs = append(txs, tx)
			txIds[tx.GetHeader().GetTxId()] = int32(i)
			continue
		}
		select {
		case <-timer.C:
			timeout = true
		default:
			runtime.Gosched()
			time.Sleep(500 * time.Millisecond)
		}
		if timeout {
			break
		}
	}

	batchId := atomic.AddInt32(&p.currentBatchId, 1)
	batch := &txpoolPb.TxBatch{
		BatchId: batchId,
		NodeId:  p.nodeId,

		Txs:      txs,
		TxIdsMap: txIds,
		Size_:    int32(len(txs)),
	}
	p.log.Infof("create txBatch size: %d, batchId: %d, txMapLen: %d, txsLen: %d", batch.Size(), batch.BatchId, len(batch.TxIdsMap), len(batch.Txs))
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
	p.batchFetchHeight.Store(batch.BatchId, int64(0))
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

func (p *BatchTxPool) GetTxsByTxIds(txIds []string) (map[string]*commonPb.Transaction, map[string]int64) {
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
	if batch := p.configBatchPool.GetBatch(batchId); batch != nil {
		return batch
	}
	return nil
}

func (p *BatchTxPool) generateTxsInfoFromBatch(batch *txpoolPb.TxBatch) (map[string]*commonPb.Transaction, map[string]int64) {
	var height int64 = -1
	heightVal, ok := p.batchFetchHeight.Load(batch.BatchId)
	if !ok {
		height = heightVal.(int64)
	}
	txsRet := make(map[string]*commonPb.Transaction, batch.Size_)
	txsHeightInfo := make(map[string]int64, batch.Size_)

	for _, tx := range batch.Txs {
		txsRet[tx.Header.TxId] = tx
		txsHeightInfo[tx.Header.TxId] = height
	}
	return txsRet, txsHeightInfo
}

func (p *BatchTxPool) GetTxByTxId(txId string) (tx *commonPb.Transaction, inBlockHeight int64) {
	if atomic.LoadInt32(&p.stat) == 0 {
		p.log.Errorf(commonErrors.ErrTxPoolHasStopped.String())
		return nil, -1
	}
	batchId, txIndex, exist := p.batchTxIdRecorder.FindBatchIdWithTxId(txId)
	if !exist {
		return nil, -1
	}
	if val, ok := p.batchFetchHeight.Load(batchId); ok {
		inBlockHeight = val.(int64)
	}
	batch := p.commonBatchPool.GetBatch(batchId)
	if batch != nil {
		return batch.Txs[txIndex], inBlockHeight
	}
	batch = p.configBatchPool.GetBatch(batchId)
	if batch != nil {
		return batch.Txs[txIndex], inBlockHeight
	}
	return nil, -1
}

func (p *BatchTxPool) TxExists(tx *commonPb.Transaction) bool {
	if atomic.LoadInt32(&p.stat) == 0 {
		p.log.Errorf("batch tx pool not started, start it first pls")
		return false
	}
	_, _, exist := p.batchTxIdRecorder.FindBatchIdWithTxId(tx.Header.TxId)
	return exist
}

func (p *BatchTxPool) FetchTxBatch(blockHeight int64) []*commonPb.Transaction {
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

func (p *BatchTxPool) fetchTxsFromCfgPool(blockHeight int64) []*commonPb.Transaction {
	var (
		batch   *txpoolPb.TxBatch
		cfgPool = p.configBatchPool
	)
	cfgPool.pool.Range(func(val interface{}) (isContinue bool) {
		if val == nil {
			return true
		}
		batch = val.(*txpoolPb.TxBatch)
		return false
	})
	if batch != nil {
		if pendingOk := p.pendingPool.PutIfNotExist(batch); !pendingOk {
			p.log.Errorf("pending batch failed, batch(batch id:%d) exist in pending batch pool", batch.GetBatchId())
			return nil
		}
		if ok := cfgPool.RemoveIfExist(batch); !ok {
			p.log.Errorf("remove batch failed, batch(batch id:%d) not exist in normal batch pool", batch.GetBatchId())
			return nil
		}
		p.batchFetchHeight.Store(batch.BatchId, blockHeight)
		return p.filterTxs(batch.GetTxs())
	}
	return nil
}

func (p *BatchTxPool) filterTxs(txs []*commonPb.Transaction) []*commonPb.Transaction {
	noExistInDb := make([]*commonPb.Transaction, 0, len(txs))
	for _, tx := range txs {
		if !p.isTxExistInDB(tx) {
			noExistInDb = append(noExistInDb, tx)
		}
	}
	return noExistInDb
}

func (p *BatchTxPool) fetchTxsFromCommonPool(blockHeight int64) []*commonPb.Transaction {
	var (
		batch         *txpoolPb.TxBatch
		commonTxsPool = p.commonBatchPool
	)
	commonTxsPool.pool.Range(func(val interface{}) (isContinue bool) {
		if val == nil {
			return true
		}
		batch = val.(*txpoolPb.TxBatch)
		return false
	})
	if batch != nil {
		pendingOk := p.pendingPool.PutIfNotExist(batch)
		if !pendingOk {
			p.log.Errorf("pending batch failed, batch(batch id:%d) exist in pending batch pool", batch.GetBatchId())
			return nil
		}
		ok := commonTxsPool.RemoveIfExist(batch)
		if !ok {
			p.log.Errorf("remove batch failed, batch(batch id:%d) not exist in normal batch pool", batch.GetBatchId())
			return nil
		}
		p.batchFetchHeight.Store(batch.BatchId, blockHeight)
		return p.filterTxs(batch.GetTxs())
	}
	return nil
}

func (p *BatchTxPool) RetryAndRemoveTxs(retryTxs []*commonPb.Transaction, removeTxs []*commonPb.Transaction) {
	if atomic.LoadInt32(&p.stat) == 0 {
		return
	}
	defer p.publishSignal()
	p.log.Debugf("retry txs num: %d, remove txs num: %d", len(retryTxs), len(removeTxs))
	p.retryTxBatch(retryTxs)
	p.removeTxBatch(removeTxs)
	return
}

// todo. may be update logic
func (p *BatchTxPool) retryTxBatch(txs []*commonPb.Transaction) {
	if len(txs) == 0 {
		return
	}
	if _, _, ok := p.batchTxIdRecorder.FindBatchIdWithTxId(txs[0].Header.TxId); ok {
		return
	}

	batch := &txpoolPb.TxBatch{
		BatchId:  atomic.AddInt32(&p.currentBatchId, 1),
		NodeId:   p.nodeId,
		Size_:    int32(len(txs)),
		Txs:      txs,
		TxIdsMap: createTxIdsMap(txs),
	}
	p.batchTxIdRecorder.AddRecordWithBatch(batch)
	if utils.IsConfigTx(txs[0]) {
		p.configBatchPool.PutIfNotExist(batch)
	} else {
		p.commonBatchPool.PutIfNotExist(batch)
	}
	atomic.AddInt32(&p.currentTxCount, batch.Size_)
	return
}

func createTxIdsMap(txs []*commonPb.Transaction) map[string]int32 {
	txIdsMap := make(map[string]int32)
	for idx, tx := range txs {
		txIdsMap[tx.GetHeader().GetTxId()] = int32(idx)
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
		txId    = txs[0].GetHeader().GetTxId()
	)
	if batchId, _, ok = p.batchTxIdRecorder.FindBatchIdWithTxId(txId); !ok {
		p.log.Warnf("batch id not found,ignored. (tx id:%s) when removeTxBatch", txId)
		return
	}

	batch := &txpoolPb.TxBatch{NodeId: p.nodeId, BatchId: batchId, Txs: txs, TxIdsMap: createTxIdsMap(txs), Size_: int32(len(txs))}
	if p.pendingPool.RemoveIfExist(batch) {
		remove = true
		p.log.Infof("remove txs[%d] from pending pool, current pool size [%d]", len(txs), atomic.LoadInt32(&p.currentTxCount)-int32(len(txs)))
	} else if utils.IsConfigTx(txs[0]) {
		if p.configBatchPool.RemoveIfExist(batch) {
			p.log.Infof("remove txs[%d] from config tx pool, current pool size [%d]", len(txs), atomic.LoadInt32(&p.currentTxCount)-int32(len(txs)))
			remove = true
		}
	} else {
		if p.commonBatchPool.RemoveIfExist(batch) {
			p.log.Infof("remove txs[%d] from common tx pool, current pool size [%d]", len(txs), atomic.LoadInt32(&p.currentTxCount)-int32(len(txs)))
			remove = true
		}
	}
	if remove {
		atomic.AddInt32(&p.currentTxCount, 0-batch.GetSize_())
		p.log.Infof("current txs num: %d", atomic.LoadInt32(&p.currentTxCount))
		p.batchTxIdRecorder.RemoveRecordWithBatch(batch)
		p.batchFetchHeight.Delete(batchId)
	} else {
		p.log.Errorf("remove batch failed, batch(batch id:%d) not found in all batch pool", batchId)
	}
	return
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
	p.batchFetchHeight.Store(batch.BatchId, int64(0))
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

func (p *BatchTxPool) AddTxsToPendingCache(txs []*commonPb.Transaction, blockHeight int64) {
	if len(txs) == 0 {
		return
	}

	var (
		exist   bool
		batchId int32 = -1
	)
	for _, tx := range txs {
		if batchId, _, exist = p.batchTxIdRecorder.FindBatchIdWithTxId(tx.Header.TxId); !exist {
			continue
		}
	}
	if !exist {
		batchId = atomic.AddInt32(&p.currentBatchId, 1)
		batch := &txpoolPb.TxBatch{NodeId: p.nodeId, BatchId: batchId, Txs: txs, TxIdsMap: createTxIdsMap(txs), Size_: int32(len(txs))}
		p.pendingPool.PutIfNotExist(batch)
		return
	}

	batch := p.getBatch(batchId)
	if utils.IsConfigTx(txs[0]) {
		p.configBatchPool.RemoveIfExist(batch)
	} else {
		p.commonBatchPool.RemoveIfExist(batch)
	}
	p.pendingPool.PutIfNotExist(batch)
}
