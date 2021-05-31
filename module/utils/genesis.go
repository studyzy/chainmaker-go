/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package utils

import (
	"chainmaker.org/chainmaker/common/crypto/hash"
	commonPb "chainmaker.org/chainmaker/pb-go/common"
	configPb "chainmaker.org/chainmaker/pb-go/config"
	"fmt"
	"github.com/gogo/protobuf/proto"
)

// default timestamp is "2020-11-30 0:0:0"
const (
	defaultTimestamp           = int64(1606669261)
	errMsgMarshalChainConfFail = "proto marshal chain config failed, %s"
)

// CreateGenesis create genesis block (with read-write set) based on chain config
func CreateGenesis(cc *configPb.ChainConfig) (*commonPb.Block, []*commonPb.TxRWSet, error) {
	var (
		err      error
		tx       *commonPb.Transaction
		rwSet    *commonPb.TxRWSet
		txHash   []byte
		hashType = cc.Crypto.Hash
	)

	// generate config tx, read-write set, and hash
	if tx, err = genConfigTx(cc); err != nil {
		return nil, nil, fmt.Errorf("create genesis config tx failed, %s", err)
	}

	if rwSet, err = genConfigTxRWSet(cc); err != nil {
		return nil, nil, fmt.Errorf("create genesis config tx read-write set failed, %s", err)
	}

	if tx.Result.RwSetHash, err = CalcRWSetHash(cc.Crypto.Hash, rwSet); err != nil {
		return nil, nil, fmt.Errorf("calculate genesis config tx read-write set hash failed, %s", err)
	}

	if txHash, err = CalcTxHash(cc.Crypto.Hash, tx); err != nil {
		return nil, nil, fmt.Errorf("calculate tx hash failed, %s", err)
	}

	// generate genesis block
	genesisBlock := &commonPb.Block{
		Header: &commonPb.BlockHeader{
			ChainId:        cc.ChainId,
			BlockHeight:    0,
			PreBlockHash:   nil,
			BlockHash:      nil,
			PreConfHeight:  0,
			BlockVersion:   []byte(cc.Version),
			DagHash:        nil,
			RwSetRoot:      nil,
			TxRoot:         nil,
			BlockTimestamp: defaultTimestamp,
			Proposer:       nil,
			ConsensusArgs:  nil,
			TxCount:        1,
			Signature:      nil,
		},
		Dag: &commonPb.DAG{
			Vertexes: []*commonPb.DAG_Neighbor{
				&commonPb.DAG_Neighbor{
					Neighbors: nil,
				},
			},
		},
		Txs: []*commonPb.Transaction{tx},
	}

	if genesisBlock.Header.TxRoot, err = hash.GetMerkleRoot(hashType, [][]byte{txHash}); err != nil {
		return nil, nil, fmt.Errorf("calculate genesis block tx root failed, %s", err)
	}

	if genesisBlock.Header.RwSetRoot, err = CalcRWSetRoot(hashType, genesisBlock.Txs); err != nil {
		return nil, nil, fmt.Errorf("calculate genesis block rwset root failed, %s", err)
	}

	if genesisBlock.Header.DagHash, err = CalcDagHash(hashType, genesisBlock.Dag); err != nil {
		return nil, nil, fmt.Errorf("calculate genesis block DAG hash failed, %s", err)
	}

	if genesisBlock.Header.BlockHash, err = CalcBlockHash(hashType, genesisBlock); err != nil {
		return nil, nil, fmt.Errorf("calculate genesis block hash failed, %s", err)
	}

	return genesisBlock, []*commonPb.TxRWSet{rwSet}, nil
}

func genConfigTx(cc *configPb.ChainConfig) (*commonPb.Transaction, error) {
	var (
		err          error
		ccBytes      []byte
		payloadBytes []byte
	)

	if ccBytes, err = proto.Marshal(cc); err != nil {
		return nil, fmt.Errorf(errMsgMarshalChainConfFail, err.Error())
	}

	payload := &commonPb.SystemContractPayload{
		ChainId:      cc.ChainId,
		ContractName: commonPb.ContractName_SYSTEM_CONTRACT_CHAIN_CONFIG.String(),
		Method:       "Genesis",
		Parameters:   make([]*commonPb.KeyValuePair, 0),
		Sequence:     cc.Sequence,
		Endorsement:  nil,
	}
	payload.Parameters = append(payload.Parameters, &commonPb.KeyValuePair{
		Key:   commonPb.ContractName_SYSTEM_CONTRACT_CHAIN_CONFIG.String(),
		Value: cc.String(),
	})

	if payloadBytes, err = proto.Marshal(payload); err != nil {
		return nil, fmt.Errorf(errMsgMarshalChainConfFail, err.Error())
	}

	tx := &commonPb.Transaction{
		Header: &commonPb.TxHeader{
			ChainId:        cc.ChainId,
			Sender:         nil,
			TxType:         commonPb.TxType_UPDATE_CHAIN_CONFIG,
			TxId:           GetTxIdWithSeed(int64(defaultTimestamp)),
			Timestamp:      defaultTimestamp,
			ExpirationTime: -1,
		},
		RequestPayload:   payloadBytes,
		RequestSignature: nil,
		Result: &commonPb.Result{
			Code: commonPb.TxStatusCode_SUCCESS,
			ContractResult: &commonPb.ContractResult{
				Code:    commonPb.ContractResultCode_OK,
				Message: commonPb.ContractResultCode_OK.String(),
				Result:  ccBytes,
			},
			RwSetHash: nil,
		},
	}

	return tx, nil
}

func genConfigTxRWSet(cc *configPb.ChainConfig) (*commonPb.TxRWSet, error) {
	var (
		err     error
		ccBytes []byte
	)

	if ccBytes, err = proto.Marshal(cc); err != nil {
		return nil, fmt.Errorf(errMsgMarshalChainConfFail, err.Error())
	}

	set := &commonPb.TxRWSet{
		TxId:    GetTxIdWithSeed(int64(defaultTimestamp)),
		TxReads: nil,
		TxWrites: []*commonPb.TxWrite{
			{
				Key:          []byte(commonPb.ContractName_SYSTEM_CONTRACT_CHAIN_CONFIG.String()),
				Value:        ccBytes,
				ContractName: commonPb.ContractName_SYSTEM_CONTRACT_CHAIN_CONFIG.String(),
			},
		},
	}
	return set, nil
}
