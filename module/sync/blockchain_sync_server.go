/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sync

import (
	"fmt"
	"sync/atomic"
	"time"

	commonErrors "chainmaker.org/chainmaker/common/v2/errors"
	"chainmaker.org/chainmaker/common/v2/msgbus"
	"chainmaker.org/chainmaker/localconf/v2"
	"chainmaker.org/chainmaker/logger/v2"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	netPb "chainmaker.org/chainmaker/pb-go/v2/net"
	storePb "chainmaker.org/chainmaker/pb-go/v2/store"
	syncPb "chainmaker.org/chainmaker/pb-go/v2/sync"
	"chainmaker.org/chainmaker/protocol/v2"
	"github.com/gogo/protobuf/proto"
)

var _ protocol.SyncService = (*BlockChainSyncServer)(nil)

type BlockChainSyncServer struct {
	chainId string

	net             protocol.NetService      // receive/broadcast messages from net module
	msgBus          msgbus.MessageBus        // receive/broadcast messages from internal modules
	blockChainStore protocol.BlockchainStore // The module that provides blocks storage/query
	ledgerCache     protocol.LedgerCache     // Provides the latest chain state for the node
	blockVerifier   protocol.BlockVerifier   // Verify Block Validity
	blockCommitter  protocol.BlockCommitter  // Adds a validated block to the chain to update the state of the chain

	log   *logger.CMLogger
	conf  *BlockSyncServerConf // The configuration in sync module
	start int32                // Identification of module startup
	close chan bool            // Identification of module close

	scheduler *Routine // Service that get blocks from other nodes
	processor *Routine // Service that processes block data, adding valid blocks to the chain
}

func NewBlockChainSyncServer(chainId string,
	net protocol.NetService,
	msgBus msgbus.MessageBus,
	blockchainStore protocol.BlockchainStore,
	ledgerCache protocol.LedgerCache,
	blockVerifier protocol.BlockVerifier,
	blockCommitter protocol.BlockCommitter) protocol.SyncService {

	syncServer := &BlockChainSyncServer{
		chainId:         chainId,
		net:             net,
		msgBus:          msgBus,
		blockChainStore: blockchainStore,
		ledgerCache:     ledgerCache,
		blockVerifier:   blockVerifier,
		blockCommitter:  blockCommitter,
		close:           make(chan bool),
		log:             logger.GetLoggerByChain(logger.MODULE_SYNC, chainId),
	}
	return syncServer
}

func (sync *BlockChainSyncServer) Start() error {
	if !atomic.CompareAndSwapInt32(&sync.start, 0, 1) {
		return commonErrors.ErrSyncServiceHasStarted
	}

	// 1. init conf
	sync.initSyncConfIfRequire()
	processor := newProcessor(sync, sync.ledgerCache, sync.log)
	scheduler := newScheduler(sync, sync.ledgerCache,
		sync.conf.blockPoolSize, sync.conf.timeOut, sync.conf.reqTimeThreshold, sync.conf.batchSizeFromOneNode, sync.log)
	if scheduler == nil {
		return fmt.Errorf("init scheduler failed")
	}
	sync.scheduler = NewRoutine("scheduler", scheduler.handler, scheduler.getServiceState, sync.log)
	sync.processor = NewRoutine("processor", processor.handler, processor.getServiceState, sync.log)

	// 2. register msgs handler
	if sync.msgBus != nil {
		sync.msgBus.Register(msgbus.BlockInfo, sync)
	}
	if err := sync.net.Subscribe(netPb.NetMsg_SYNC_BLOCK_MSG, sync.blockSyncMsgHandler); err != nil {
		return err
	}
	if err := sync.net.ReceiveMsg(netPb.NetMsg_SYNC_BLOCK_MSG, sync.blockSyncMsgHandler); err != nil {
		return err
	}

	// 3. start internal service
	if err := sync.scheduler.begin(); err != nil {
		return err
	}
	if err := sync.processor.begin(); err != nil {
		return err
	}
	go sync.loop()
	return nil
}

func (sync *BlockChainSyncServer) initSyncConfIfRequire() {
	defer func() {
		sync.log.Infof(sync.conf.print())
	}()
	if sync.conf != nil {
		return
	}
	sync.conf = NewBlockSyncServerConf()
	if localconf.ChainMakerConfig.SyncConfig.BlockPoolSize > 0 {
		sync.conf.SetBlockPoolSize(uint64(localconf.ChainMakerConfig.SyncConfig.BlockPoolSize))
	}
	if localconf.ChainMakerConfig.SyncConfig.WaitTimeOfBlockRequestMsg > 0 {
		sync.conf.SetWaitTimeOfBlockRequestMsg(int64(localconf.ChainMakerConfig.SyncConfig.WaitTimeOfBlockRequestMsg))
	}
	if localconf.ChainMakerConfig.SyncConfig.BatchSizeFromOneNode > 0 {
		sync.conf.SetBatchSizeFromOneNode(uint64(localconf.ChainMakerConfig.SyncConfig.BatchSizeFromOneNode))
	}
	if localconf.ChainMakerConfig.SyncConfig.LivenessTick > 0 {
		sync.conf.SetLivenessTicker(localconf.ChainMakerConfig.SyncConfig.LivenessTick)
	}
	if localconf.ChainMakerConfig.SyncConfig.NodeStatusTick > 0 {
		sync.conf.SetNodeStatusTicker(localconf.ChainMakerConfig.SyncConfig.NodeStatusTick)
	}
	if localconf.ChainMakerConfig.SyncConfig.DataDetectionTick > 0 {
		sync.conf.SetDataDetectionTicker(localconf.ChainMakerConfig.SyncConfig.DataDetectionTick)
	}
	if localconf.ChainMakerConfig.SyncConfig.ProcessBlockTick > 0 {
		sync.conf.SetProcessBlockTicker(localconf.ChainMakerConfig.SyncConfig.ProcessBlockTick)
	}
	if localconf.ChainMakerConfig.SyncConfig.SchedulerTick > 0 {
		sync.conf.SetSchedulerTicker(localconf.ChainMakerConfig.SyncConfig.SchedulerTick)
	}
	if localconf.ChainMakerConfig.SyncConfig.ReqTimeThreshold > 0 {
		sync.conf.SetReqTimeThreshold(localconf.ChainMakerConfig.SyncConfig.ReqTimeThreshold)
	}
}

func (sync *BlockChainSyncServer) blockSyncMsgHandler(from string, msg []byte, msgType netPb.NetMsg_MsgType) error {
	if atomic.LoadInt32(&sync.start) != 1 {
		return commonErrors.ErrSyncServiceHasStoped
	}
	if msgType != netPb.NetMsg_SYNC_BLOCK_MSG {
		return nil
	}
	var (
		err     error
		syncMsg = syncPb.SyncMsg{}
	)
	if err = proto.Unmarshal(msg, &syncMsg); err != nil {
		sync.log.Errorf("fail to proto.Unmarshal the syncPb.SyncMsg:%s", err.Error())
		return err
	}
	sync.log.Debugf("receive the NetMsg_SYNC_BLOCK_MSG:the Type is %d", syncMsg.Type)

	switch syncMsg.Type {
	case syncPb.SyncMsg_NODE_STATUS_REQ:
		return sync.handleNodeStatusReq(from)
	case syncPb.SyncMsg_NODE_STATUS_RESP:
		return sync.handleNodeStatusResp(&syncMsg, from)
	case syncPb.SyncMsg_BLOCK_SYNC_REQ:
		return sync.handleBlockReq(&syncMsg, from)
	case syncPb.SyncMsg_BLOCK_SYNC_RESP:
		return sync.scheduler.addTask(&SyncedBlockMsg{msg: syncMsg.Payload, from: from})
	}
	return fmt.Errorf("not support the syncPb.SyncMsg.Type as %d", syncMsg.Type)
}

func (sync *BlockChainSyncServer) handleNodeStatusReq(from string) error {
	var (
		height uint64
		bz     []byte
		err    error
	)
	if height, err = sync.ledgerCache.CurrentHeight(); err != nil {
		return err
	}
	archivedHeight := sync.blockChainStore.GetArchivedPivot()
	sync.log.Debugf("receive node status request from node [%s]", from)
	if bz, err = proto.Marshal(&syncPb.BlockHeightBCM{BlockHeight: height, ArchivedHeight: archivedHeight}); err != nil {
		return err
	}
	return sync.sendMsg(syncPb.SyncMsg_NODE_STATUS_RESP, bz, from)
}

func (sync *BlockChainSyncServer) handleNodeStatusResp(syncMsg *syncPb.SyncMsg, from string) error {
	msg := syncPb.BlockHeightBCM{}
	if err := proto.Unmarshal(syncMsg.Payload, &msg); err != nil {
		return err
	}
	sync.log.Debugf("receive node[%s] status, height [%d], archived height [%d]", from, msg.BlockHeight,
		msg.ArchivedHeight)
	return sync.scheduler.addTask(NodeStatusMsg{msg: msg, from: from})
}

func (sync *BlockChainSyncServer) handleBlockReq(syncMsg *syncPb.SyncMsg, from string) error {
	var (
		err error
		req syncPb.BlockSyncReq
	)
	if err = proto.Unmarshal(syncMsg.Payload, &req); err != nil {
		sync.log.Errorf("fail to proto.Unmarshal the syncPb.SyncMsg:%s", err.Error())
		return err
	}
	sync.log.Debugf("receive request to get block [height: %d, batch_size: %d] from "+
		"node [%s]", req.BlockHeight, req.BatchSize, from)
	if req.WithRwset {
		return sync.sendInfos(&req, from)
	}
	return sync.sendBlocks(&req, from)
}

func (sync *BlockChainSyncServer) sendBlocks(req *syncPb.BlockSyncReq, from string) error {
	var (
		bz  []byte
		err error
		blk *commonPb.Block
	)

	for i := uint64(0); i < req.BatchSize; i++ {
		if blk, err = sync.blockChainStore.GetBlock(req.BlockHeight + i); err != nil || blk == nil {
			return err
		}
		if bz, err = proto.Marshal(&syncPb.SyncBlockBatch{
			Data: &syncPb.SyncBlockBatch_BlockBatch{BlockBatch: &syncPb.BlockBatch{Batches: []*commonPb.Block{blk}}},
		}); err != nil {
			return err
		}
		if err := sync.sendMsg(syncPb.SyncMsg_BLOCK_SYNC_RESP, bz, from); err != nil {
			return err
		}
	}
	return nil
}

func (sync *BlockChainSyncServer) sendInfos(req *syncPb.BlockSyncReq, from string) error {
	var (
		bz        []byte
		err       error
		blkRwInfo *storePb.BlockWithRWSet
	)

	for i := uint64(0); i < req.BatchSize; i++ {
		if blkRwInfo, err = sync.blockChainStore.GetBlockWithRWSets(req.BlockHeight + i); err != nil || blkRwInfo == nil {
			return err
		}
		info := &commonPb.BlockInfo{Block: blkRwInfo.Block, RwsetList: blkRwInfo.TxRWSets}
		if bz, err = proto.Marshal(&syncPb.SyncBlockBatch{
			Data: &syncPb.SyncBlockBatch_BlockinfoBatch{BlockinfoBatch: &syncPb.BlockInfoBatch{
				Batch: []*commonPb.BlockInfo{info}}},
		}); err != nil {
			return err
		}
		if err := sync.sendMsg(syncPb.SyncMsg_BLOCK_SYNC_RESP, bz, from); err != nil {
			return err
		}
	}
	return nil
}

func (sync *BlockChainSyncServer) sendMsg(msgType syncPb.SyncMsg_MsgType, msg []byte, to string) error {
	var (
		bs  []byte
		err error
	)
	if bs, err = proto.Marshal(&syncPb.SyncMsg{
		Type:    msgType,
		Payload: msg,
	}); err != nil {
		sync.log.Error(err)
		return err
	}
	if err = sync.net.SendMsg(bs, netPb.NetMsg_SYNC_BLOCK_MSG, to); err != nil {
		sync.log.Error(err)
		return err
	}
	return nil
}

func (sync *BlockChainSyncServer) broadcastMsg(msgType syncPb.SyncMsg_MsgType, msg []byte) error {
	var (
		bs  []byte
		err error
	)
	if bs, err = proto.Marshal(&syncPb.SyncMsg{
		Type:    msgType,
		Payload: msg,
	}); err != nil {
		sync.log.Error(err)
		return err
	}
	if err = sync.net.BroadcastMsg(bs, netPb.NetMsg_SYNC_BLOCK_MSG); err != nil {
		sync.log.Error(err)
		return err
	}
	return nil
}

func (sync *BlockChainSyncServer) loop() {
	var (
		// task: trigger the flow of the block process
		doProcessBlockTk = time.NewTicker(sync.conf.processBlockTick)
		// task: trigger the state acquisition process for the node
		doScheduleTk = time.NewTicker(sync.conf.schedulerTick)
		// task: trigger the flow of the node status acquisition from connected peers
		doNodeStatusTk = time.NewTicker(sync.conf.nodeStatusTick)
		// task: trigger the check of the liveness with connected peers
		doLivenessTk = time.NewTicker(sync.conf.livenessTick)
		// task: trigger the check of the data in processor and scheduler
		doDataDetect = time.NewTicker(sync.conf.dataDetectionTick)
	)
	defer func() {
		doProcessBlockTk.Stop()
		doScheduleTk.Stop()
		doLivenessTk.Stop()
		doNodeStatusTk.Stop()
		doDataDetect.Stop()
	}()

	for {
		select {
		case <-sync.close:
			return

			// Timing task
		case <-doProcessBlockTk.C:
			if err := sync.processor.addTask(ProcessBlockMsg{}); err != nil {
				sync.log.Errorf("add process block task to processor failed, reason: %s", err)
			}
		case <-doScheduleTk.C:
			if err := sync.scheduler.addTask(SchedulerMsg{}); err != nil {
				sync.log.Errorf("add scheduler task to scheduler failed, reason: %s", err)
			}
		case <-doLivenessTk.C:
			if err := sync.scheduler.addTask(LivenessMsg{}); err != nil {
				sync.log.Errorf("add livenessMsg task to scheduler failed, reason: %s", err)
			}
		case <-doNodeStatusTk.C:
			sync.log.Debugf("broadcast request of the node status")
			if err := sync.broadcastMsg(syncPb.SyncMsg_NODE_STATUS_REQ, nil); err != nil {
				sync.log.Errorf("request node status failed by broadcast", err)
			}
		case <-doDataDetect.C:
			if err := sync.processor.addTask(DataDetection{}); err != nil {
				sync.log.Errorf("add data detection task to processor failed, reason: %s", err)
			}
			if err := sync.scheduler.addTask(DataDetection{}); err != nil {
				sync.log.Errorf("add data detection task to scheduler failed, reason: %s", err)
			}

		// State processing results in state machine
		case resp := <-sync.scheduler.out:
			if err := sync.processor.addTask(resp); err != nil {
				sync.log.Errorf("add scheduler task to processor failed, reason: %s", err)
			}
		case resp := <-sync.processor.out:
			if err := sync.scheduler.addTask(resp); err != nil {
				sync.log.Errorf("add processor task to scheduler failed, reason: %s", err)
			}
		}
	}
}

func (sync *BlockChainSyncServer) validateAndCommitBlock(block *commonPb.Block) processedBlockStatus {
	if blk := sync.ledgerCache.GetLastCommittedBlock(); blk != nil && blk.Header.BlockHeight >= block.Header.BlockHeight {
		sync.log.Infof("the block: %d has been committed in the blockChainStore ", block.Header.BlockHeight)
		return hasProcessed
	}
	if err := sync.blockVerifier.VerifyBlock(block, protocol.SYNC_VERIFY); err != nil {
		if err == commonErrors.ErrBlockHadBeenCommited {
			sync.log.Warnf("the block: %d has been committed in the blockChainStore ", block.Header.BlockHeight)
			return hasProcessed
		}
		sync.log.Warnf("fail to verify the block whose height is %d, err: %s", block.Header.BlockHeight, err)
		return validateFailed
	}
	if err := sync.blockCommitter.AddBlock(block); err != nil {
		if err == commonErrors.ErrBlockHadBeenCommited {
			sync.log.Warnf("the block: %d has been committed in the blockChainStore ", block.Header.BlockHeight)
			return hasProcessed
		}
		sync.log.Warnf("fail to commit the block whose height is %d, err: %s", block.Header.BlockHeight, err)
		return addErr
	}
	return ok
}

func (sync *BlockChainSyncServer) Stop() {
	if !atomic.CompareAndSwapInt32(&sync.start, 1, 0) {
		return
	}
	sync.scheduler.end()
	sync.processor.end()
	close(sync.close)
}

func (sync *BlockChainSyncServer) OnMessage(message *msgbus.Message) {
	if message == nil || message.Payload == nil {
		sync.log.Errorf("receive the empty message")
		return
	}
	if message.Topic != msgbus.BlockInfo {
		sync.log.Errorf("receive the message from the topic as %d, but not msgbus.BlockInfo ", message.Topic)
		return
	}
	switch blockInfo := message.Payload.(type) {
	case *commonPb.BlockInfo:
		if blockInfo == nil || blockInfo.Block == nil {
			sync.log.Errorf("error message BlockInfo = nil")
			return
		}
		height := blockInfo.Block.Header.BlockHeight
		if height%3 != 0 {
			return
		}
		bz, err := proto.Marshal(&syncPb.BlockHeightBCM{BlockHeight: height})
		if err != nil {
			sync.log.Errorf("marshal BlockHeightBCM failed, reason: %s", err)
			return
		}
		if err := sync.broadcastMsg(syncPb.SyncMsg_NODE_STATUS_RESP, bz); err != nil {
			sync.log.Errorf("fail to broadcast the height as %d, and the error is %s", height, err)
		}
	default:
		sync.log.Errorf("not support the message type as %T", message.Payload)
	}
}

func (sync *BlockChainSyncServer) OnQuit() {
	sync.log.Infof("stop to listen the msgbus.BlockInfo")
}
