/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sync

import (
	"fmt"
	"time"
)

type BlockSyncServerConf struct {
	timeOut          time.Duration // Timeout of request, unit nanosecond
	reqTimeThreshold time.Duration // When the difference between the height of the node and the latest height of peers
	// is 1, the time interval for requesting
	processBlockTick  time.Duration // The ticker to process of the block, unit nanosecond
	livenessTick      time.Duration // The ticker to liveness checking, unit nanosecond
	schedulerTick     time.Duration // The ticker to request block from the peer, unit nanosecond
	nodeStatusTick    time.Duration // The ticker to request node status from other peers, unit nanosecond
	dataDetectionTick time.Duration // The ticker to check data in processor

	blockPoolSize        uint64 // Maximum number of blocks to be processed in scheduler
	batchSizeFromOneNode uint64 // The number of blocks received from each node in a request

}

func NewBlockSyncServerConf() *BlockSyncServerConf {
	return &BlockSyncServerConf{
		timeOut:              5 * time.Second,
		blockPoolSize:        bufferSize,
		batchSizeFromOneNode: 1,
		processBlockTick:     20 * time.Millisecond,
		livenessTick:         1 * time.Second,
		nodeStatusTick:       5 * time.Second,
		schedulerTick:        20 * time.Millisecond,
		dataDetectionTick:    time.Minute,
		reqTimeThreshold:     3 * time.Second,
	}
}

func (c *BlockSyncServerConf) SetBlockPoolSize(n uint64) *BlockSyncServerConf {
	c.blockPoolSize = n
	return c
}
func (c *BlockSyncServerConf) SetWaitTimeOfBlockRequestMsg(n int64) *BlockSyncServerConf {
	c.timeOut = time.Duration(n) * time.Second
	return c
}
func (c *BlockSyncServerConf) SetBatchSizeFromOneNode(n uint64) *BlockSyncServerConf {
	c.batchSizeFromOneNode = n
	return c
}
func (c *BlockSyncServerConf) SetProcessBlockTicker(n float64) *BlockSyncServerConf {
	c.processBlockTick = time.Duration(n * float64(time.Second))
	return c
}
func (c *BlockSyncServerConf) SetSchedulerTicker(n float64) *BlockSyncServerConf {
	c.schedulerTick = time.Duration(n * float64(time.Second))
	return c
}
func (c *BlockSyncServerConf) SetLivenessTicker(n float64) *BlockSyncServerConf {
	c.livenessTick = time.Duration(n * float64(time.Second))
	return c
}
func (c *BlockSyncServerConf) SetNodeStatusTicker(n float64) *BlockSyncServerConf {
	c.nodeStatusTick = time.Duration(n * float64(time.Second))
	return c
}
func (c *BlockSyncServerConf) SetDataDetectionTicker(n float64) *BlockSyncServerConf {
	c.dataDetectionTick = time.Duration(n * float64(time.Second))
	return c
}
func (c *BlockSyncServerConf) SetReqTimeThreshold(n float64) *BlockSyncServerConf {
	c.reqTimeThreshold = time.Duration(n * float64(time.Second))
	return c
}
func (c *BlockSyncServerConf) print() string {
	return fmt.Sprintf("blockPoolSize: %d, request timeout: %d, batchSizeFromOneNode: %d"+
		", processBlockTick: %v, schedulerTick: %v, livenessTick: %v, nodeStatusTick: %v\n",
		c.blockPoolSize, c.timeOut, c.batchSizeFromOneNode, c.processBlockTick, c.schedulerTick, c.livenessTick,
		c.nodeStatusTick)
}
