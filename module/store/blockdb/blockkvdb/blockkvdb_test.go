/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package blockkvdb

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"testing"

	storePb "chainmaker.org/chainmaker/pb-go/store"

	"chainmaker.org/chainmaker-go/store/cache"
	"chainmaker.org/chainmaker-go/store/dbprovider/leveldbprovider"
	"chainmaker.org/chainmaker-go/store/serialization"
	acPb "chainmaker.org/chainmaker/pb-go/accesscontrol"
	commonPb "chainmaker.org/chainmaker/pb-go/common"
	"chainmaker.org/chainmaker/protocol/test"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/semaphore"
)

var (
	logger  = &test.GoLogger{}
	chainId = "chain1"
)

func TestBlockKvDB_GetTxWithBlockInfo(t *testing.T) {
	block := createBlock(chainId, 1, 2)
	blockDB := &BlockKvDB{
		WorkersSemaphore: semaphore.NewWeighted(int64(1)),
		Cache:            cache.NewStoreCacheMgr(chainId, logger),
		Logger:           logger,
		DbHandle:         leveldbprovider.NewMemdbHandle(),
	}
	_, sb, _ := serialization.SerializeBlock(&storePb.BlockWithRWSet{Block: block})
	blockDB.InitGenesis(sb)
	tx, err := blockDB.GetTxWithBlockInfo(block.Txs[1].Payload.TxId)
	assert.Nil(t, err)
	t.Logf("%+v", tx)
	assert.EqualValues(t, 1, tx.BlockHeight)
	assert.EqualValues(t, 1, tx.TxIndex)
}

func createBlock(chainId string, height uint64, txCount int) *commonPb.Block {
	block := &commonPb.Block{
		Header: &commonPb.BlockHeader{
			ChainId:     chainId,
			BlockHeight: height,
			Proposer:    &acPb.Member{MemberInfo: []byte("User1")},
		},
		Txs: []*commonPb.Transaction{},
	}

	for i := 0; i < txCount; i++ {
		tx := &commonPb.Transaction{
			Payload: &commonPb.Payload{
				ChainId:      chainId,
				TxType:       commonPb.TxType_INVOKE_CONTRACT,
				TxId:         generateTxId(chainId, height, 0),
				ContractName: "contract" + strconv.Itoa(i),
				Method:       "func1",
			},
			Sender: &commonPb.EndorsementEntry{Signer: &acPb.Member{OrgId: "org1", MemberInfo: []byte("cert1...")},
				Signature: []byte("sign1"),
			},
			Result: &commonPb.Result{
				Code: commonPb.TxStatusCode_SUCCESS,
				ContractResult: &commonPb.ContractResult{
					Result: []byte("ok"),
				},
			},
		}
		block.Txs = append(block.Txs, tx)
	}

	block.Header.BlockHash = generateBlockHash(chainId, height)
	return block
}
func generateBlockHash(chainId string, height uint64) []byte {
	blockHash := sha256.Sum256([]byte(fmt.Sprintf("%s-%d", chainId, height)))
	return blockHash[:]
}

func generateTxId(chainId string, height uint64, index int) string {
	txIdBytes := sha256.Sum256([]byte(fmt.Sprintf("%s-%d-%d", chainId, height, index)))
	return hex.EncodeToString(txIdBytes[:32])
}
