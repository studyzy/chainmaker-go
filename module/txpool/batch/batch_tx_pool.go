/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package batch

import (
	commonErrors "chainmaker.org/chainmaker-go/common/errors"
	"chainmaker.org/chainmaker-go/common/msgbus"
	"chainmaker.org/chainmaker-go/common/queue/lockfreequeue"
	"chainmaker.org/chainmaker-go/logger"
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	netPb "chainmaker.org/chainmaker-go/pb/protogo/net"
	txpoolPb "chainmaker.org/chainmaker-go/pb/protogo/txpool"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/utils"
	"errors"
	"github.com/gogo/protobuf/proto"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

const (
	DefaultBatchMaxSize       = 50000
	DefaultBatchCreateTimeout = 1000 * time.Millisecond
	DefaultPoolSize           = 100000
	errNews                   = "implement me"
)

type BatchTxPool struct {
	batchMaxSize       int32
	batchCreateTimeout time.Duration
	maxTxCount         int32
	currentTxCount     int32
	nodeId             string
	chainId            string
	currentBatchId     int32
	stat               int32
	testMode           bool
	txQueue            *lockfreequeue.Queue
	cfgTxQueue         *lockfreequeue.Queue

	configBatchPool   *cfgBatchPool
	commonBatchPool   *nodeBatchPool
	batchTxIdRecorder *batchTxIdRecorder
	pendingPool       *pendingBatchPool
	mb                msgbus.MessageBus

	log *logger.CMLogger

	fetchLock sync.Mutex
	lock      sync.Mutex
	stopCh    chan struct{}
}

func NewBatchTxPool(nodeId string, chainId string) (*BatchTxPool, error) {
	return &BatchTxPool{
		nodeId:             nodeId,
		chainId:            chainId,
		batchMaxSize:       int32(DefaultBatchMaxSize),
		batchCreateTimeout: DefaultBatchCreateTimeout,
		maxTxCount:         int32(DefaultPoolSize),
		configBatchPool:    newCfgBatchPool(),
		commonBatchPool:    newNodeBatchPool(),
		batchTxIdRecorder:  newBatchTxIdRecorder(),
		pendingPool:        newPendingBatchPool(),
		stopCh:             make(chan struct{}),
		log:                logger.GetLoggerByChain(logger.MODULE_TXPOOL, chainId),
	}, nil
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
	p.lock.Lock()
	defer p.lock.Unlock()
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
	p.lock.Lock()
	defer p.lock.Unlock()
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
		return errors.New("batch tx pool is full")
	}
	if utils.IsConfigTx(tx) {
		ok, _ := p.cfgTxQueue.Push(tx)
		if !ok {
			return errors.New("push cfg tx to cfg queue failed")
		}
		atomic.AddInt32(&p.currentTxCount, 1)
		return nil
	}
	ok, _ := p.txQueue.Push(tx)
	if !ok {
		return errors.New("push tx to tx queue failed")
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
		BatchId:  batchId,
		NodeId:   p.nodeId,
		Size_:    1,
		Txs:      make([]*commonPb.Transaction, 1),
		TxIdsMap: make(map[string]int32),
	}
	batch.Txs[0] = tx
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
		tx    = val.(*commonPb.Transaction)
		txs   = make([]*commonPb.Transaction, 1)
		txIds = make(map[string]int32)
		timer = time.NewTimer(p.batchCreateTimeout)

		err      error
		timeout  bool
		batchMsg []byte
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
		BatchId:  batchId,
		NodeId:   p.nodeId,
		Size_:    int32(len(txs)),
		Txs:      txs,
		TxIdsMap: txIds,
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
	panic(errNews)
}

func (p *BatchTxPool) GetTxByTxId(txId string) (tx *commonPb.Transaction, inBlockHeight int64) {
	if atomic.LoadInt32(&p.stat) == 0 {
		p.log.Errorf(commonErrors.ErrTxPoolHasStopped.String())
		return nil, -1
	}
	searchFunc := func(val interface{}) (isContinue bool) {
		batch := val.(*txpoolPb.TxBatch)
		if batch.GetTxIdsMap() == nil {
			return true
		}
		if txIdx, ok := batch.GetTxIdsMap()[txId]; ok {
			p.log.Debugf("txMapLen(%d), txsLen(%d)", len(batch.GetTxIdsMap()), len(batch.GetTxs()))
			tx = batch.GetTxs()[txIdx]
			return false
		}
		return true
	}

	p.commonBatchPool.pool.Range(searchFunc)
	if tx != nil {
		return tx, 0
	}
	p.configBatchPool.pool.Range(searchFunc)
	if tx != nil {
		return tx, 0
	}

	p.pendingPool.Range(func(batch *txpoolPb.TxBatch) (isContinue bool) {
		if batch.GetTxIdsMap() == nil {
			return true
		}
		if txIdx, ok := batch.GetTxIdsMap()[txId]; ok {
			tx = batch.GetTxs()[txIdx]
			return false
		}
		return true
	})
	return tx, 0
}

func (p *BatchTxPool) TxExists(tx *commonPb.Transaction) bool {
	if atomic.LoadInt32(&p.stat) == 0 {
		p.log.Errorf("batch tx pool not started, start it first pls")
		return false
	}
	_, exist := p.batchTxIdRecorder.FindBatchIdWithTxId(tx.Header.TxId)
	return exist
}

func (p *BatchTxPool) FetchTxBatch(blockHeight int64) []*commonPb.Transaction {
	if atomic.LoadInt32(&p.stat) == 0 {
		p.log.Errorf("batch tx pool not started, start it first pls")
		return nil
	}
	p.fetchLock.Lock()
	defer p.fetchLock.Unlock()
	if txs := p.fetchTxsFromCfgPool(); len(txs) > 0 {
		return txs
	}
	return p.fetchTxsFromCommonPool()
}

func (p *BatchTxPool) fetchTxsFromCfgPool() []*commonPb.Transaction {
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
		}
		if ok := cfgPool.RemoveIfExist(batch); !ok {
			p.log.Errorf("remove batch failed, batch(batch id:%d) not exist in normal batch pool", batch.GetBatchId())
		}
		return batch.GetTxs()
	}
	return nil
}

func (p *BatchTxPool) fetchTxsFromCommonPool() []*commonPb.Transaction {
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
		}
		ok := commonTxsPool.RemoveIfExist(batch)
		if !ok {
			p.log.Errorf("remove batch failed, batch(batch id:%d) not exist in normal batch pool", batch.GetBatchId())
		}
		return batch.GetTxs()
	}
	return nil
}

func (p *BatchTxPool) RetryAndRemoveTxs(retryTxs []*commonPb.Transaction, removeTxs []*commonPb.Transaction) {
	if atomic.LoadInt32(&p.stat) == 0 {
		return
	}
	defer p.publishSignal()
	p.retryTxBatch(retryTxs)
	p.removeTxBatch(removeTxs)
	return
}

func (p *BatchTxPool) retryTxBatch(txs []*commonPb.Transaction) {
	if len(txs) == 0 {
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
		batchId int32
		remove  = false
		txId    = txs[0].GetHeader().GetTxId()
	)
	if batchId, ok = p.batchTxIdRecorder.FindBatchIdWithTxId(txId); !ok {
		p.log.Errorf("batch id not found,ignored. (tx id:%s)", txId)
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
		p.batchTxIdRecorder.RemoveRecordWithBatch(batch)
	} else {
		p.log.Debugf("remove batch failed, batch(batch id:%d) not found in batch pool", batchId)
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
	panic(errNews)
}
