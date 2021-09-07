/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package snapshot

import (
	"fmt"
	"testing"

	"chainmaker.org/chainmaker/pb-go/v2/accesscontrol"

	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/utils/v2"
	"github.com/stretchr/testify/require"
)

func TestChainedSnapshot(t *testing.T) {
	snapshotMgr := &ManagerImpl{
		snapshots: make(map[utils.BlockFingerPrint]*SnapshotImpl, 1024),
		delegate: &ManagerDelegate{
			blockchainStore: nil,
		},
	}

	genesis := createNewBlock(0, 0)

	block1 := createNewBlock(1, 1)
	snapshot1 := snapshotMgr.NewSnapshot(genesis, block1)

	block2 := createNewBlock(2, 2)
	snapshot2 := snapshotMgr.NewSnapshot(block1, block2)

	block3 := createNewBlock(3, 3)
	snapshot3 := snapshotMgr.NewSnapshot(block2, block3)

	block3a := createNewBlock(3, 4)
	snapshot3a := snapshotMgr.NewSnapshot(block2, block3a)

	fmt.Printf("%v\n", snapshot1)
	fmt.Printf("%v\n", snapshot2)
	fmt.Printf("%v\n", snapshot3)
	fmt.Printf("%v\n", snapshot3a)

	require.Equal(t, snapshot1, snapshot2.GetPreSnapshot())
	require.Equal(t, snapshot2, snapshot3.GetPreSnapshot())
	require.Equal(t, nil, snapshotMgr.NotifyBlockCommitted(block1))
	require.Equal(t, nil, snapshot2.GetPreSnapshot())
	require.Equal(t, nil, snapshotMgr.NotifyBlockCommitted(block2))
	require.Equal(t, nil, snapshot3.GetPreSnapshot())
}

func createNewBlock(height uint64, timeStamp int64) *commonPb.Block {
	block := &commonPb.Block{
		Header: &commonPb.BlockHeader{
			BlockHeight:    height,
			PreBlockHash:   nil,
			BlockHash:      nil,
			BlockVersion:   0,
			DagHash:        nil,
			RwSetRoot:      nil,
			BlockTimestamp: timeStamp,
			Proposer:       &accesscontrol.Member{MemberInfo: []byte{1, 2, 3}},
			ConsensusArgs:  nil,
			TxCount:        0,
			Signature:      nil,
		},
		Dag: &commonPb.DAG{
			Vertexes: nil,
		},
		Txs: nil,
	}
	block.Header.PreBlockHash = nil
	return block
}
