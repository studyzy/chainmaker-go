/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package native

import (
	"chainmaker.org/chainmaker-go/logger"
	acPb "chainmaker.org/chainmaker-go/pb/protogo/accesscontrol"
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	storePb "chainmaker.org/chainmaker-go/pb/protogo/store"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/store/serialization"
	"chainmaker.org/chainmaker-go/store/types"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
)

const (
	paramTargetBlockHeight = "targetBlockHeight"
	paramStartBlockHeight  = "startBlockHeight"
)

var dbType = types.LevelDb

type ArchiveStoreContract struct {
	methods map[string]ContractFunc
	log     *logger.CMLogger
}

func newArchiveStoreContract(log *logger.CMLogger) *ArchiveStoreContract {
	return &ArchiveStoreContract{
		log:     log,
		methods: registerArchiveStoreContractMethods(log),
	}
}

func (a *ArchiveStoreContract) getMethod(methodName string) ContractFunc {
	return a.methods[methodName]
}

func registerArchiveStoreContractMethods(log *logger.CMLogger) map[string]ContractFunc {
	methodMap := make(map[string]ContractFunc, 64)
	archiveRuntime := &ArchiveStoreRuntime{log: log}
	methodMap[commonPb.ArchiveStoreContractFunction_GET_ARCHIVED_BLOCK_HEIGHT.String()] = archiveRuntime.GetArchiveBlockHeight
	methodMap[commonPb.ArchiveStoreContractFunction_ARCHIVE_BLOCK.String()] = archiveRuntime.ArchiveBlock
	methodMap[commonPb.ArchiveStoreContractFunction_RESTORE_BLOCKS.String()] = archiveRuntime.RestoreBlock
	return methodMap
}

type ArchiveStoreRuntime struct {
	log *logger.CMLogger
	contractName string
}

func (a *ArchiveStoreRuntime) GetArchiveBlockHeight(context protocol.TxSimContext, params map[string]string) ([]byte, error) {
	blockHeight := strconv.FormatInt(int64(context.GetBlockchainStore().GetArchivedPivot()), 10)

	a.log.Infof("get archive block height success blockHeight[%s] ", blockHeight)
	return []byte(blockHeight), nil
}

func (a *ArchiveStoreRuntime) ArchiveBlock(context protocol.TxSimContext, params map[string]string) ([]byte, error) {
	var errMsg string
	var err error

	targetBlockHeightStr, err := a.getValue(params, paramTargetBlockHeight)
	if err != nil {
		errMsg = fmt.Sprintf("%s, archive block require param [%s] not found", ErrParams.Error(), paramTargetBlockHeight)
		a.log.Errorf(errMsg)
		return nil, fmt.Errorf(errMsg)
	}

	blockHeight, err := strconv.Atoi(targetBlockHeightStr)
	if err != nil {
		errMsg = fmt.Sprintf("convert atoi failed, err: %s", err.Error())
		a.log.Errorf(errMsg)
		return nil, fmt.Errorf(errMsg)
	}

	if err = context.GetBlockchainStore().ArchiveBlock(uint64(blockHeight)); err != nil {
		errMsg = fmt.Sprintf("archive block  err: %s", err.Error())
		a.log.Errorf(errMsg)
		return nil, fmt.Errorf(errMsg)
	}

	return []byte(""), nil
}

func (a *ArchiveStoreRuntime) RestoreBlock(context protocol.TxSimContext, params map[string]string) ([]byte, error) {
	var errMsg string
	var err error

	startBlockHeightStr, err := a.getValue(params, paramStartBlockHeight)
	if err != nil {
		errMsg = fmt.Sprintf("%s, restore block require param [%s] not found", ErrParams.Error(), paramStartBlockHeight)
		a.log.Errorf(errMsg)
		return nil, fmt.Errorf(errMsg)
	}

	startBlockHeight, err := strconv.Atoi(startBlockHeightStr)
	if err != nil {
		errMsg = fmt.Sprintf("convert atoi failed, err: %s", err.Error())
		a.log.Error(errMsg)
		return nil, fmt.Errorf(errMsg)
	}

	//init contractName
	a.contractName, err = context.GetTx().GetContractName()
	if err != nil {
		errMsg = fmt.Sprintf("get contract name failed, err: %s", err.Error())
		a.log.Error(errMsg)
		return nil, fmt.Errorf(errMsg)
	}

	//getArchivedBlockHeight
	archiveBlockHeight := context.GetBlockchainStore().GetArchivedPivot()
	if archiveBlockHeight <= uint64(startBlockHeight) {
		return nil, errors.New("archived block height not low start block height")
	}

	//Prepare block data
	blocks := make([]*commonPb.Block, 0, archiveBlockHeight)
	TxRWSetMp := make(map[int64][]*commonPb.TxRWSet)
	for i := 0; i < int(archiveBlockHeight); i++ {
		block, txRWSet := a.createBlockAndRWSets(context.GetTx().Header.ChainId, int64(i), 100)
		if err = context.GetBlockchainStore().PutBlock(block, txRWSet); err != nil {
			return nil, err
		}
		blocks = append(blocks, block)
		TxRWSetMp[block.Header.BlockHeight] = txRWSet
	}

	//Prepare restore data
	blocksBytes := make([][]byte, 0, startBlockHeight+1)
	for i := 0; i <= startBlockHeight; i++ {
		blockBytes, _, err := serialization.SerializeBlock(&storePb.BlockWithRWSet{
			Block:          blocks[i],
			TxRWSets:       TxRWSetMp[blocks[i].Header.BlockHeight],
			ContractEvents: nil,
		})
		if err != nil {
			a.log.Errorf("serialize block err:%s ", err.Error())
		}
		blocksBytes = append(blocksBytes, blockBytes)
	}

	if err = context.GetBlockchainStore().RestoreBlocks(blocksBytes); err != nil {
		return nil, err
	}

	return []byte(""), nil
}

func (a *ArchiveStoreRuntime) createBlockAndRWSets(chainId string, height int64, txNum int) (*commonPb.Block, []*commonPb.TxRWSet) {
	block := &commonPb.Block{
		Header: &commonPb.BlockHeader{
			ChainId:     chainId,
			BlockHeight: height,
		},
	}

	for i := 0; i < txNum; i++ {
		tx := &commonPb.Transaction{
			Header: &commonPb.TxHeader{
				ChainId: chainId,
				TxId:    a.generateTxId(chainId, height, i),
				Sender: &acPb.SerializedMember{
					OrgId: "org1",
				},
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

	block.Header.BlockHash = a.generateBlockHash(chainId, height)
	var txRWSets []*commonPb.TxRWSet
	for i := 0; i < txNum; i++ {
		key := fmt.Sprintf("key_%d", i)
		value := fmt.Sprintf("value_%d", i)
		txRWset := &commonPb.TxRWSet{
			TxId: block.Txs[i].Header.TxId,
			TxWrites: []*commonPb.TxWrite{
				{
					Key:          []byte(key),
					Value:        []byte(value),
					ContractName: a.contractName,
				},
			},
		}
		txRWSets = append(txRWSets, txRWset)
	}

	return block, txRWSets
}

func (a *ArchiveStoreRuntime) generateTxId(chainId string, height int64, index int) string {
	txIdBytes := sha256.Sum256([]byte(fmt.Sprintf("%s-%d-%d", chainId, height, index)))
	return hex.EncodeToString(txIdBytes[:])
}

func (a *ArchiveStoreRuntime) generateBlockHash(chainId string, height int64) []byte {
	blockHash := sha256.Sum256([]byte(fmt.Sprintf("%s-%d", chainId, height)))
	return blockHash[:]
}

func (a *ArchiveStoreRuntime) getValue(parameters map[string]string, key string) (string, error) {
	value, ok := parameters[key]
	if !ok {
		errMsg := fmt.Sprintf("miss params %s", key)
		a.log.Error(errMsg)
		return "", errors.New(errMsg)
	}
	return value, nil
}
