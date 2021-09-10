/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sync

import (
	"fmt"
	"sync/atomic"

	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"

	"chainmaker.org/chainmaker/logger/v2"
	"chainmaker.org/chainmaker/protocol/v2"

	"github.com/Workiva/go-datastructures/queue"
)

type verifyAndAddBlock interface {
	validateAndCommitBlock(block *commonPb.Block) processedBlockStatus
}

type blockWithPeerInfo struct {
	id  string
	blk *commonPb.Block
}

type processor struct {
	queue          map[uint64]blockWithPeerInfo // Information about the blocks will be processed
	hasCommitBlock uint64                       // Number of blocks that have been commit

	log         *logger.CMLogger
	ledgerCache protocol.LedgerCache // Provides the latest chain state for the node
	verifyAndAddBlock
}

func newProcessor(verify verifyAndAddBlock, ledgerCache protocol.LedgerCache, log *logger.CMLogger) *processor {
	return &processor{
		ledgerCache:       ledgerCache,
		verifyAndAddBlock: verify,
		queue:             make(map[uint64]blockWithPeerInfo),
		log:               log,
	}
}

func (pro *processor) handler(event queue.Item) (queue.Item, error) {
	switch msg := event.(type) {
	case *ReceivedBlocks:
		pro.handleReceivedBlocks(msg)
	case ProcessBlockMsg:
		return pro.handleProcessBlockMsg()
	case DataDetection:
		pro.handleDataDetection()
	}
	return nil, nil
}

func (pro *processor) handleReceivedBlocks(msg *ReceivedBlocks) {
	lastCommitBlockHeight := pro.lastCommitBlockHeight()
	for _, blk := range msg.blks {
		if blk.Header.BlockHeight <= lastCommitBlockHeight {
			continue
		}
		if _, exist := pro.queue[blk.Header.BlockHeight]; !exist {
			pro.queue[blk.Header.BlockHeight] = blockWithPeerInfo{
				blk: blk, id: msg.from,
			}
			pro.log.Debugf("received block [height: %d] from node [%s]", blk.Header.BlockHeight, msg.from)
		}
	}
}

func (pro *processor) handleProcessBlockMsg() (queue.Item, error) {
	var (
		exist  bool
		info   blockWithPeerInfo
		status processedBlockStatus
	)
	pendingBlockHeight := pro.lastCommitBlockHeight() + 1
	if info, exist = pro.queue[pendingBlockHeight]; !exist {
		//pro.log.Debugf("block [%d] not find in queue.", pendingBlockHeight)
		return nil, nil
	}
	if status = pro.validateAndCommitBlock(info.blk); status == ok || status == hasProcessed {
		pro.hasCommitBlock++
	}
	delete(pro.queue, pendingBlockHeight)
	pro.log.Infof("process block [height: %d], status [%d]", info.blk.Header.BlockHeight, status)
	return ProcessedBlockResp{
		status: status,
		height: info.blk.Header.BlockHeight,
		from:   info.id,
	}, nil
}

func (pro *processor) handleDataDetection() {
	pendingBlockHeight := pro.lastCommitBlockHeight() + 1
	for height := range pro.queue {
		if height < pendingBlockHeight {
			delete(pro.queue, height)
		}
	}
}

func (pro *processor) lastCommitBlockHeight() uint64 {
	return pro.ledgerCache.GetLastCommittedBlock().Header.BlockHeight
}

func (pro *processor) hasProcessedBlock() uint64 {
	return atomic.LoadUint64(&pro.hasCommitBlock)
}

func (pro *processor) getServiceState() string {
	return fmt.Sprintf("pendingBlockHeight: %d, queue num: %d", pro.lastCommitBlockHeight()+1, len(pro.queue))
}
