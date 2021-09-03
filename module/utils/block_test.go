/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package utils

import (
	"testing"
	"time"

	"chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	"github.com/stretchr/testify/assert"
)

func TestCalcBlockFingerPrint(t *testing.T) {
	h1 := &common.BlockHeader{BlockHeight: 0, ChainId: "chain1", BlockTimestamp: time.Now().Unix()}
	b1 := &common.Block{Header: h1}
	fp1 := CalcBlockFingerPrint(b1)
	t.Log(fp1)
	h2 := *h1
	h2.Proposer = &accesscontrol.Member{OrgId: "org1", MemberInfo: []byte("User1")}
	b2 := &common.Block{Header: &h2}
	fp2 := CalcBlockFingerPrint(b2)
	assert.NotEqual(t, fp1, fp2)
}
func TestCalcUnsignedBlockBytes(t *testing.T) {
	h1 := &common.BlockHeader{BlockHeight: 0,
		ChainId:        "chain1",
		BlockTimestamp: time.Now().Unix(),
		TxCount:        1,
		TxRoot:         []byte("hash root"),
		BlockHash:      []byte("hash1"),
		Signature:      []byte("sign1")}
	b1 := &common.Block{Header: h1}
	bytes, err := calcUnsignedBlockBytes(b1)
	assert.Nil(t, err)
	t.Logf("%x", bytes)
	assert.NotNil(t, b1.Header.Signature)
	assert.NotNil(t, b1.Header.BlockHash)
	b1.Header.BlockHash = nil
	b1.Header.Signature = nil
	data2, err := b1.Header.Marshal()
	assert.Nil(t, err)
	assert.Equal(t, bytes, data2)
}
