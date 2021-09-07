/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package snapshot

import (
	"chainmaker.org/chainmaker/logger/v2"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/utils/v2"
)

var log = logger.GetLogger(logger.MODULE_SNAPSHOT)

type Factory struct {
}

func (f *Factory) NewSnapshotManager(blockchainStore protocol.BlockchainStore) protocol.SnapshotManager {
	log.Debugf("use the common Snapshot.")
	return &ManagerImpl{
		snapshots: make(map[utils.BlockFingerPrint]*SnapshotImpl, 1024),
		delegate: &ManagerDelegate{
			blockchainStore: blockchainStore,
		},
	}
}

func (f *Factory) NewSnapshotEvidenceMgr(blockchainStore protocol.BlockchainStore) protocol.SnapshotManager {
	log.Debugf("use the evidence Snapshot.")
	return &ManagerEvidence{
		snapshots: make(map[utils.BlockFingerPrint]*SnapshotEvidence, 1024),
		delegate: &ManagerDelegate{
			blockchainStore: blockchainStore,
		},
	}
}
