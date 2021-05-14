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
	"chainmaker.org/chainmaker-go/utils"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
)

const (
	defaultContractName    = "contract1"
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
}

func (a *ArchiveStoreRuntime) GetArchiveBlockHeight(context protocol.TxSimContext, params map[string]string) ([]byte, error) {

	blockHeight := strconv.FormatInt(int64(context.GetBlockchainStore().GetArchivedPivot()), 10)

	a.log.Infof("GetArchiveBlockHeight success blockHeight[%s] ", blockHeight)
	return []byte(blockHeight), nil
}

func (a *ArchiveStoreRuntime) ArchiveBlock(context protocol.TxSimContext, params map[string]string) ([]byte, error) {
	targetBlockHeightStr := params[paramTargetBlockHeight]

	if utils.IsAnyBlank(targetBlockHeightStr) {
		err := fmt.Errorf("%s, archive block require param [%s] not found", ErrParams.Error(), paramNameCertHashes)
		a.log.Error(err)
		return nil, err
	}

	blockHeight, err := strconv.Atoi(targetBlockHeightStr)
	if err != nil {
		err = fmt.Errorf(" failed, err: %s", err.Error())
		a.log.Error(err)
		return nil, err
	}

	if err = context.GetBlockchainStore().ArchiveBlock(uint64(blockHeight)); err != nil {
		err = fmt.Errorf(" archive block  err: %s", err.Error())
		a.log.Error(err)
		return nil, err
	}

	return []byte(""), nil
}

func (a *ArchiveStoreRuntime) RestoreBlock(context protocol.TxSimContext, params map[string]string) ([]byte, error) {
	startBlockHeightStr := params[paramStartBlockHeight]
	if utils.IsAnyBlank(startBlockHeightStr) {
		err := fmt.Errorf("%s, archive block require param [%s] not found", ErrParams.Error(), paramNameCertHashes)
		a.log.Error(err)
		return nil, err
	}

	startBlockHeight, err := strconv.Atoi(startBlockHeightStr)
	if err != nil {
		err = fmt.Errorf(" failed, err: %s", err.Error())
		a.log.Error(err)
		return nil, err
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
		block, txRWSet := createBlockAndRWSets(context.GetTx().Header.ChainId, int64(i), 100)
		if err := context.GetBlockchainStore().PutBlock(block, txRWSet, nil); err != nil {
			return nil, err //todo log
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
			a.log.Error(err) //todo log
		}
		blocksBytes = append(blocksBytes, blockBytes)
	}

	if err = context.GetBlockchainStore().RestoreBlocks(blocksBytes); err != nil {
		return nil, err
	}

	return []byte(""), nil
}

func createBlockAndRWSets(chainId string, height int64, txNum int) (*commonPb.Block, []*commonPb.TxRWSet) {
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
				TxId:    generateTxId(chainId, height, i),
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

	block.Header.BlockHash = generateBlockHash(chainId, height)
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
					ContractName: defaultContractName,
				},
			},
		}
		txRWSets = append(txRWSets, txRWset)
	}

	return block, txRWSets
}

func generateTxId(chainId string, height int64, index int) string {
	txIdBytes := sha256.Sum256([]byte(fmt.Sprintf("%s-%d-%d", chainId, height, index)))
	return hex.EncodeToString(txIdBytes[:])
}

func generateBlockHash(chainId string, height int64) []byte {
	blockHash := sha256.Sum256([]byte(fmt.Sprintf("%s-%d", chainId, height)))
	return blockHash[:]
}
