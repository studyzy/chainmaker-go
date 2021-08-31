/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dpos

import (
	"bytes"
	"testing"

	commonpb "chainmaker.org/chainmaker/pb-go/v2/common"
	consensuspb "chainmaker.org/chainmaker/pb-go/v2/consensus"

	"github.com/stretchr/testify/require"
)

func TestBytesEqual(t *testing.T) {
	bz := make([]byte, 0)
	require.True(t, bytes.Equal(bz, nil))
}

func TestDPoSImpl_VerifyConsensusArgs(t *testing.T) {

}

func TestDPoSImpl_CreateDPoSRWSet(t *testing.T) {
	impl, fn := initTestImpl(t)
	defer fn()

	proposedBlk := &consensuspb.ProposalBlock{Block: &commonpb.Block{Header: &commonpb.BlockHeader{BlockHeight: 99}}}
	rwSet, err := impl.createDPoSRWSet(nil, proposedBlk)
	require.NoError(t, err)
	require.Nil(t, rwSet)

	proposedBlk.Block.Header.BlockHeight = 100
	rwSet, err = impl.createDPoSRWSet(nil, proposedBlk)
	require.EqualError(t, err, "not found candidates from contract")
	require.Nil(t, rwSet)
}

func TestDPoSImpl_CreateNewEpoch(t *testing.T) {

}

func TestDPoSImpl_selectValidators(t *testing.T) {

}
