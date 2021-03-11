/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package snapshot

import (
	"chainmaker.org/chainmaker-go/logger"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/utils"
)

var log = logger.GetLogger(logger.MODULE_SNAPSHOT)

type Factory struct {
}

func (f *Factory) NewSnapshotManager(blockchainStore protocol.BlockchainStore) protocol.SnapshotManager {
	return &ManagerImpl{
		snapshots:       make(map[utils.BlockFingerPrint]*SnapshotImpl, 1024),
		blockchainStore: blockchainStore,
	}
}
