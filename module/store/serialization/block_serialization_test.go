/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package serialization

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"

	acPb "chainmaker.org/chainmaker/pb-go/accesscontrol"

	commonPb "chainmaker.org/chainmaker/pb-go/common"
	storePb "chainmaker.org/chainmaker/pb-go/store"
	"github.com/stretchr/testify/assert"
)

var chainId = "testchain1"

func generateBlockHash(chainId string, height uint64) []byte {
	blockHash := sha256.Sum256([]byte(fmt.Sprintf("%s-%d", chainId, height)))
	return blockHash[:]
}

func generateTxId(chainId string, height uint64, index int) string {
	txIdBytes := sha256.Sum256([]byte(fmt.Sprintf("%s-%d-%d", chainId, height, index)))
	return hex.EncodeToString(txIdBytes[:32])
}

func createBlockAndRWSets(chainId string, height uint64, txNum int) *storePb.BlockWithRWSet {
	block := &commonPb.Block{
		Header: &commonPb.BlockHeader{
			ChainId:     chainId,
			BlockHeight: height,
		},
	}

	for i := 0; i < txNum; i++ {
		tx := &commonPb.Transaction{
			Payload: &commonPb.Payload{
				ChainId: chainId,
				TxId:    generateTxId(chainId, height, i),
			},
			Sender: &commonPb.EndorsementEntry{
				Signer: &acPb.Member{
					OrgId:      "org1",
					MemberInfo: []byte("User1"),
				},
				Signature: []byte("signature1"),
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
	var txRWSets []*commonPb.TxRWSet
	for i := 0; i < txNum; i++ {
		key := fmt.Sprintf("key_%d", i)
		value := fmt.Sprintf("value_%d", i)
		txRWset := &commonPb.TxRWSet{
			TxId: block.Txs[i].Payload.TxId,
			TxWrites: []*commonPb.TxWrite{
				{
					Key:          []byte(key),
					Value:        []byte(value),
					ContractName: "contract1",
				},
			},
		}
		txRWSets = append(txRWSets, txRWset)
	}

	return &storePb.BlockWithRWSet{Block: block, TxRWSets: txRWSets}
}

func TestSerializeBlock(t *testing.T) {
	for i := 0; i < 10; i++ {
		block := createBlockAndRWSets(chainId, uint64(i), 5000)
		bytes, blockInfo, err := SerializeBlock(block)
		assert.Nil(t, err)
		assert.Equal(t, blockInfo.Block.String(), block.Block.String())
		assert.Equal(t, len(block.Block.Txs), len(blockInfo.SerializedTxs))
		assert.Equal(t, len(block.TxRWSets), len(blockInfo.TxRWSets))
		result, err := DeserializeBlock(bytes)
		assert.Nil(t, err)
		assert.Equal(t, block.Block.String(), result.Block.String())
	}
}
