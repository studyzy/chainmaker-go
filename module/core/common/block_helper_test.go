/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"chainmaker.org/chainmaker-go/logger"
	commonpb "chainmaker.org/chainmaker/pb-go/common"
	"fmt"
	"github.com/stretchr/testify/require"
	_ "net/http/pprof"
	"testing"
	"time"
)

func TestFinalizeBlock_Async(t *testing.T) {

	log := logger.GetLogger("core")
	block := createBlock(10)
	txs := make([]*commonpb.Transaction, 0)
	txRWSetMap := make(map[string]*commonpb.TxRWSet)
	for i := 0; i < 1000; i++ {
		txId := "0x123456789" + fmt.Sprint(i)
		tx := createNewTestTx(txId)
		txs = append(txs, tx)
		txRWSetMap[txId] = &commonpb.TxRWSet{
			TxId:     txId,
			TxReads:  nil,
			TxWrites: nil,
		}
	}
	block.Txs = txs

	var err error
	timeStart := CurrentTimeMillisSeconds()
	err = FinalizeBlock(block, txRWSetMap, nil, "SHA256", log)
	timeEnd := CurrentTimeMillisSeconds()
	require.Equal(t, nil, err)
	log.Infof("finalize block cost:[%d]", timeEnd-timeStart)

}

func createBlock(height int64) *commonpb.Block {
	var hash = []byte("0123456789")
	var version = []byte("0")
	var block = &commonpb.Block{
		Header: &commonpb.BlockHeader{
			ChainId:        "Chain1",
			BlockHeight:    height,
			PreBlockHash:   hash,
			BlockHash:      hash,
			PreConfHeight:  0,
			BlockVersion:   version,
			DagHash:        hash,
			RwSetRoot:      hash,
			TxRoot:         hash,
			BlockTimestamp: 0,
			Proposer:       hash,
			ConsensusArgs:  nil,
			TxCount:        1,
			Signature:      []byte(""),
		},
		Dag: &commonpb.DAG{
			Vertexes: nil,
		},
		Txs: nil,
	}

	return block
}

func createNewTestTx(txID string) *commonpb.Transaction {
	var hash = []byte("0123456789")
	return &commonpb.Transaction{
		Header: &commonpb.TxHeader{
			ChainId:        "Chain1",
			Sender:         nil,
			TxType:         0,
			TxId:           txID,
			Timestamp:      CurrentTimeMillisSeconds(),
			ExpirationTime: 0,
		},
		RequestPayload:   hash,
		RequestSignature: hash,
		Result: &commonpb.Result{
			Code:           commonpb.TxStatusCode_SUCCESS,
			ContractResult: nil,
			RwSetHash:      nil,
		},
	}
}

func CurrentTimeMillisSeconds() int64 {
	return time.Now().UnixNano() / 1e6
}
