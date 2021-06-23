/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dpos

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBytesEqual(t *testing.T) {
	bz := make([]byte, 0)
	require.True(t, bytes.Equal(bz, nil))
}

func TestDPoSImpl_VerifyConsensusArgs(t *testing.T) {

}

func TestDPoSImpl_CreateDPoSRWSet(t *testing.T) {

}

func TestDPoSImpl_CreateNewEpoch(t *testing.T) {

}
