/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package raft

// SnapshotHeight stores block height in raft snapshot.
type SnapshotHeight struct {
	Height uint64
}

// AdditionalData contains consensus specified data to be store in block
type AdditionalData struct {
	Signature []byte
}
