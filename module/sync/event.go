/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sync

import (
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	syncPb "chainmaker.org/chainmaker/pb-go/v2/sync"
	"github.com/Workiva/go-datastructures/queue"
)

type EqualLevel struct{}

func (e EqualLevel) Compare(other queue.Item) int {
	return 0
}

type SyncedBlockMsg struct {
	EqualLevel
	msg  []byte
	from string
}

type NodeStatusMsg struct {
	EqualLevel
	msg  syncPb.BlockHeightBCM
	from string
}

type SchedulerMsg struct {
	EqualLevel
}

type LivenessMsg struct {
	EqualLevel
}

type ReceivedBlocks struct {
	blks []*commonPb.Block
	from string
	EqualLevel
}

// processor events

type ProcessBlockMsg struct {
	EqualLevel
}

type DataDetection struct {
	EqualLevel
}

type processedBlockStatus int64

const (
	ok processedBlockStatus = iota
	dbErr
	addErr
	hasProcessed
	validateFailed
)

type ProcessedBlockResp struct {
	height uint64
	status processedBlockStatus
	from   string
	EqualLevel
}
