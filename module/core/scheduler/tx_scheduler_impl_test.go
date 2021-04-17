/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package scheduler

import (
	"chainmaker.org/chainmaker-go/mock"
	commonpb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"encoding/hex"
	"fmt"
	"github.com/gogo/protobuf/proto"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestDag(t *testing.T) {
	for i := 0; i < 10; i++ {

		neb1 := &commonpb.DAG_Neighbor{
			Neighbors: []int32{1, 2, 3, 4},
		}
		neb2 := &commonpb.DAG_Neighbor{
			Neighbors: []int32{1, 2, 3, 4},
		}
		neb3 := &commonpb.DAG_Neighbor{
			Neighbors: []int32{1, 2, 3, 4},
		}
		vs := make([]*commonpb.DAG_Neighbor, 3)
		vs[0] = neb1
		vs[1] = neb2
		vs[2] = neb3
		dag := &commonpb.DAG{
			Vertexes: vs,
		}
		marshal, _ := proto.Marshal(dag)
		println("Dag", hex.EncodeToString(marshal))
	}
}

func newTx(txId string, contractId *commonpb.ContractId, parameterMap map[string]string) *commonpb.Transaction {

	var parameters []*commonpb.KeyValuePair
	for key, value := range parameterMap {
		parameters = append(parameters, &commonpb.KeyValuePair{
			Key:   key,
			Value: value,
		})
	}
	payload := &commonpb.QueryPayload{
		ContractName: contractId.ContractName,
		Method:       "method",
		Parameters:   parameters,
	}
	payloadBytes, _ := proto.Marshal(payload)
	return &commonpb.Transaction{
		Header: &commonpb.TxHeader{
			ChainId:        "",
			Sender:         nil,
			TxType:         commonpb.TxType_QUERY_USER_CONTRACT,
			TxId:           txId,
			Timestamp:      0,
			ExpirationTime: 0,
		},
		RequestPayload:   payloadBytes,
		RequestSignature: nil,
		Result:           nil,
	}
}

func newBlock() *commonpb.Block {
	return &commonpb.Block{
		Header: &commonpb.BlockHeader{
			ChainId:        "",
			BlockHeight:    0,
			PreBlockHash:   nil,
			BlockHash:      nil,
			BlockVersion:   nil,
			DagHash:        nil,
			RwSetRoot:      nil,
			TxRoot:         nil,
			BlockTimestamp: 0,
			Proposer:       nil,
			ConsensusArgs:  nil,
			TxCount:        0,
			Signature:      nil,
		},
		Dag: &commonpb.DAG{
			Vertexes: nil,
		},
		Txs: nil,
		AdditionalData: &commonpb.AdditionalData{
			ExtraData: nil,
		},
	}
}

func prepare(t *testing.T) (*mock.MockVmManager, []*commonpb.TxRWSet, []*commonpb.Transaction, *mock.MockSnapshot, *TxSchedulerImpl, *commonpb.ContractId, *commonpb.Block) {
	var txRWSetTable = make([]*commonpb.TxRWSet, 8)
	var txTable = make([]*commonpb.Transaction, 8)

	ctl := gomock.NewController(t)
	snapshot := mock.NewMockSnapshot(ctl)
	vmMgr := mock.NewMockVmManager(ctl)

	scheduler := NewTxScheduler(vmMgr, "Chain1")

	contractId := &commonpb.ContractId{
		ContractName:    "ContractName",
		ContractVersion: "1",
		RuntimeType:     commonpb.RuntimeType_WASMER,
	}

	contractResult := &commonpb.ContractResult{
		Code:    0,
		Result:  nil,
		Message: "",
	}
	block := newBlock()

	snapshot.EXPECT().GetTxTable().AnyTimes().Return(txTable)
	snapshot.EXPECT().GetTxRWSetTable().AnyTimes().Return(txRWSetTable)

	vmMgr.EXPECT().RunContract(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(contractResult, commonpb.TxStatusCode_SUCCESS)
	return vmMgr, txRWSetTable, txTable, snapshot, scheduler, contractId, block
}

func TestSchedule(t *testing.T) {

	vmMgr, txRWSetTable, txTable, snapshot, scheduler, contractId, block := prepare(t)

	parameters := make(map[string]string, 8)
	tx0 := newTx("a0000000000000000000000000000001", contractId, parameters)
	tx1 := newTx("a0000000000000000000000000000002", contractId, parameters)

	txTable[0] = tx0
	txTable[1] = tx1
	txRWSetTable[0] = &commonpb.TxRWSet{
		TxId: tx0.Header.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractId.ContractName,
			Key:          []byte("K1"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractId.ContractName,
			Key:          []byte("K2"),
			Value:        []byte("V"),
		}},
	}
	txRWSetTable[1] = &commonpb.TxRWSet{
		TxId: tx1.Header.TxId,
		TxReads: []*commonpb.TxRead{
			{
				ContractName: contractId.ContractName,
				Key:          []byte("K2"),
				Value:        []byte("V"),
			},
			{
				ContractName: contractId.ContractName,
				Key:          []byte("K2"),
				Value:        []byte("V"),
			},
		},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractId.ContractName,
			Key:          []byte("K3"),
			Value:        []byte("V"),
		}},
	}

	txBatch := []*commonpb.Transaction{tx0, tx1}

	txSimCache0 := newTxSimContext(vmMgr, snapshot, tx0)
	txSimCache1 := newTxSimContext(vmMgr, snapshot, tx1)

	snapshot.EXPECT().IsSealed().AnyTimes().Return(false)
	snapshot.EXPECT().Seal().Return()
	snapshot.EXPECT().ApplyTxSimContext(txSimCache0, true).Return(true, 1)
	snapshot.EXPECT().ApplyTxSimContext(txSimCache1, true).Return(false, 1)
	snapshot.EXPECT().ApplyTxSimContext(txSimCache1, true).Return(true, 2)

	dag := &commonpb.DAG{
		Vertexes: []*commonpb.DAG_Neighbor{{}},
	}

	snapshot.EXPECT().BuildDAG().Return(dag)

	_,_,err := scheduler.Schedule(block, txBatch, snapshot)

	if err != nil {
		fmt.Printf("error : %s", err.Error())
	}

	fmt.Printf("GetTxRWSet 0: %q", txSimCache0.GetTxRWSet())
	fmt.Printf("GetTxRWSet 1: %q", txSimCache1.GetTxRWSet())
}

func TestSimulateWithDag(t *testing.T) {

	vmMgr, _, _, snapshot, scheduler, contractId, block := prepare(t)

	parameters := make(map[string]string, 8)
	tx0 := newTx("a0000000000000000000000000000000", contractId, parameters)
	tx1 := newTx("a0000000000000000000000000000001", contractId, parameters)
	tx2 := newTx("a0000000000000000000000000000002", contractId, parameters)

	block.Txs = []*commonpb.Transaction{tx0, tx1, tx2}
	block.Dag = &commonpb.DAG{
		Vertexes: []*commonpb.DAG_Neighbor{
			{
				Neighbors: nil,
			},
			{
				Neighbors: []int32{0},
			},
			{
				Neighbors: []int32{0},
			},
		},
	}

	txSimCache0 := newTxSimContext(vmMgr, snapshot, tx0)
	txSimCache1 := newTxSimContext(vmMgr, snapshot, tx1)
	txSimCache2 := newTxSimContext(vmMgr, snapshot, tx2)

	snapshot.EXPECT().IsSealed().AnyTimes().Return(false)
	snapshot.EXPECT().Seal().Return()
	snapshot.EXPECT().ApplyTxSimContext(txSimCache0, true).Return(true, 1)
	snapshot.EXPECT().ApplyTxSimContext(txSimCache1, true).Return(true, 2)
	snapshot.EXPECT().ApplyTxSimContext(txSimCache2, true).Return(true, 3)

	scheduler.SimulateWithDag(block, snapshot)
}

func TestMarshalDag(t *testing.T) {
	dag := &commonpb.DAG{
		Vertexes: []*commonpb.DAG_Neighbor{
			{
				Neighbors: []int32{0},
			},
			nil,
			{
				Neighbors: []int32{0, 1, 2},
			},
		},
	}

	mar, _ := proto.Marshal(dag)

	dag2 := &commonpb.DAG{}
	proto.Unmarshal(mar, dag2)

	require.Equal(t, len(dag2.Vertexes), 3)
}
