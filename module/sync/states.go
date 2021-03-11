/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sync

// blockState state of the block in specified height
type blockState int

const (
	newBlock      = iota // First time to see a block of this height
	pendingBlock         // will fetch block data of the specified height
	receivedBlock        // has got the block data of the specified height
)
