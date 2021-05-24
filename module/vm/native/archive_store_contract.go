/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package native

import (
	"chainmaker.org/chainmaker-go/logger"
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/store/types"
	"strconv"
)

const (
	paramTargetBlockHeight = "targetBlockHeight"
	paramBlockWithRWSet    = "blockWithRWSet"
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
	//methodMap[commonPb.ArchiveStoreContractFunction_ARCHIVE_BLOCK.String()] = archiveRuntime.ArchiveBlock
	//methodMap[commonPb.ArchiveStoreContractFunction_RESTORE_BLOCKS.String()] = archiveRuntime.RestoreBlock
	return methodMap
}

type ArchiveStoreRuntime struct {
	log          *logger.CMLogger
}

func (a *ArchiveStoreRuntime) GetArchiveBlockHeight(context protocol.TxSimContext, params map[string]string) ([]byte, error) {
	blockHeight := strconv.FormatInt(int64(context.GetBlockchainStore().GetArchivedPivot()), 10)

	a.log.Infof("get archive block height success blockHeight[%s] ", blockHeight)
	return []byte(blockHeight), nil
}

//func (a *ArchiveStoreRuntime) ArchiveBlock(context protocol.TxSimContext, params map[string]string) ([]byte, error) {
//	var errMsg string
//	var err error
//
//	targetBlockHeightStr, err := a.getValue(params, paramTargetBlockHeight)
//	if err != nil {
//		errMsg = fmt.Sprintf("%s, archive block require param [%s] not found", ErrParams.Error(), paramTargetBlockHeight)
//		a.log.Errorf(errMsg)
//		return nil, fmt.Errorf(errMsg)
//	}
//
//	blockHeight, err := strconv.Atoi(targetBlockHeightStr)
//	if err != nil {
//		errMsg = fmt.Sprintf("convert atoi failed, err: %s", err.Error())
//		a.log.Errorf(errMsg)
//		return nil, fmt.Errorf(errMsg)
//	}
//
//	if err = context.GetBlockchainStore().ArchiveBlock(uint64(blockHeight)); err != nil {
//		errMsg = fmt.Sprintf("archive block  err: %s", err.Error())
//		a.log.Errorf(errMsg)
//		return nil, fmt.Errorf(errMsg)
//	}
//
//	return []byte(""), nil
//}
//
//func (a *ArchiveStoreRuntime) RestoreBlock(context protocol.TxSimContext, params map[string]string) ([]byte, error) {
//	var errMsg string
//	var err error
//
//	blocksWithRWSet, err := a.getValue(params, paramBlockWithRWSet)
//	if err != nil {
//		errMsg = fmt.Sprintf("%s, restore block require param [%s] not found", ErrParams.Error(), paramNameWithRWSet)
//		a.log.Errorf(errMsg)
//		return nil, fmt.Errorf(errMsg)
//	}
//
//	blocksWithRwSetArr := strings.Split(blocksWithRWSet, ";")
//
//	blocksBytes := make([][]byte, 0, len(blocksWithRwSetArr)+1)
//	for _, blockWithRwSetStr := range blocksWithRwSetArr {
//		blockWithRwSetStruct := &storePb.BlockWithRWSet{}
//		blockWithRwSetSlice, err := hex.DecodeString(blockWithRwSetStr)
//		if err != nil {
//			a.log.Errorf("hex decode string is err :%s", err.Error())
//		}
//
//		if err = blockWithRwSetStruct.Unmarshal(blockWithRwSetSlice); err != nil {
//			a.log.Errorf("block with rwset unmarshal  is err :%s", err.Error())
//		}
//
//		blockBytes, _, err := serialization.SerializeBlock(&storePb.BlockWithRWSet{
//			Block:          blockWithRwSetStruct.Block,
//			TxRWSets:       blockWithRwSetStruct.TxRWSets,
//			ContractEvents: blockWithRwSetStruct.ContractEvents,
//		})
//
//		if err != nil {
//			a.log.Errorf("serialize block err:%s ", err.Error())
//		}
//
//		blocksBytes = append(blocksBytes, blockBytes)
//	}
//
//	if err = context.GetBlockchainStore().RestoreBlocks(blocksBytes); err != nil {
//		return nil, err
//	}
//
//	return []byte(""), nil
//}
//
//func (a *ArchiveStoreRuntime) getValue(parameters map[string]string, key string) (string, error) {
//	value, ok := parameters[key]
//	if !ok {
//		errMsg := fmt.Sprintf("miss params %s", key)
//		a.log.Error(errMsg)
//		return "", errors.New(errMsg)
//	}
//	return value, nil
//}
