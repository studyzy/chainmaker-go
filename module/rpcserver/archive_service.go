/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package rpcserver

import (
	"errors"
	"fmt"

	"chainmaker.org/chainmaker/common/v2/bytehelper"
	commonErr "chainmaker.org/chainmaker/common/v2/errors"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"

	"chainmaker.org/chainmaker/pb-go/v2/syscontract"
	"chainmaker.org/chainmaker/protocol/v2"
)

func (s *ApiService) doArchive(tx *commonPb.Transaction) *commonPb.TxResponse {
	if tx.Payload.TxType != commonPb.TxType_ARCHIVE {
		return &commonPb.TxResponse{
			Code:    commonPb.TxStatusCode_INTERNAL_ERROR,
			Message: commonErr.ERR_CODE_TXTYPE.String(),
			TxId:    tx.Payload.TxId,
		}
	}

	switch tx.Payload.Method {

	case syscontract.ArchiveFunction_ARCHIVE_BLOCK.String():
		return s.doArchiveBlock(tx)
	case syscontract.ArchiveFunction_RESTORE_BLOCK.String():
		return s.doRestoreBlock(tx)
	default:
		return &commonPb.TxResponse{
			Code:    commonPb.TxStatusCode_INTERNAL_ERROR,
			Message: commonErr.ERR_CODE_TXTYPE.String(),
		}
	}
}

func (s *ApiService) getArchiveBlockHeight(params []*commonPb.KeyValuePair) (uint64, error) {
	if len(params) != 1 {
		return 0, errors.New("params count != 1")
	}

	key := syscontract.ArchiveBlock_BLOCK_HEIGHT.String()
	if params[0].Key != key {
		return 0, fmt.Errorf("invalid key, must be %s", key)
	}

	blockHeight, err := bytehelper.BytesToUint64(params[0].Value)
	if err != nil {
		return 0, errors.New("convert blockHeight from bytes to uint64 failed")
	}

	return blockHeight, nil
}

func (s *ApiService) doArchiveBlock(tx *commonPb.Transaction) *commonPb.TxResponse {
	var (
		err         error
		errMsg      string
		blockHeight uint64
		errCode     commonErr.ErrCode
		store       protocol.BlockchainStore
		resp        = &commonPb.TxResponse{TxId: tx.Payload.TxId}
	)

	chainId := tx.Payload.ChainId

	if store, err = s.chainMakerServer.GetStore(chainId); err != nil {
		errCode = commonErr.ERR_CODE_GET_STORE
		errMsg = s.getErrMsg(errCode, err)
		s.log.Error(errMsg)
		resp.Code = commonPb.TxStatusCode_INTERNAL_ERROR
		resp.Message = errMsg
		return resp
	}

	if blockHeight, err = s.getArchiveBlockHeight(tx.Payload.Parameters); err != nil {
		errCode = commonErr.ERR_CODE_CHECK_PAYLOAD_PARAM_ARCHIVE_BLOCK
		errMsg = s.getErrMsg(errCode, err)
		s.log.Error(errMsg)
		resp.Code = commonPb.TxStatusCode_INTERNAL_ERROR
		resp.Message = errMsg
		return resp
	}

	if err = store.ArchiveBlock(blockHeight); err != nil {
		errMsg = fmt.Sprintf("archive block failed, %s", err.Error())
		s.log.Error(errMsg)
		resp.Code = commonPb.TxStatusCode_INTERNAL_ERROR
		resp.Message = errMsg
		return resp
	}

	resp.Code = commonPb.TxStatusCode_SUCCESS
	resp.Message = commonPb.TxStatusCode_SUCCESS.String()
	return resp
}

func (s *ApiService) getRestoreBlock(params []*commonPb.KeyValuePair) ([]byte, error) {
	if len(params) != 1 {
		return nil, errors.New("params count != 1")
	}

	key := syscontract.RestoreBlock_FULL_BLOCK.String()
	if params[0].Key != key {
		return nil, fmt.Errorf("invalid key, must be %s", key)
	}

	fullBlock := params[0].Value
	if len(fullBlock) == 0 {
		return nil, errors.New("empty restore block data")
	}

	return fullBlock, nil
}

func (s *ApiService) doRestoreBlock(tx *commonPb.Transaction) *commonPb.TxResponse {

	var (
		err       error
		errMsg    string
		fullBlock []byte
		errCode   commonErr.ErrCode
		store     protocol.BlockchainStore
		resp      = &commonPb.TxResponse{TxId: tx.Payload.TxId}
	)

	chainId := tx.Payload.ChainId

	if store, err = s.chainMakerServer.GetStore(chainId); err != nil {
		errCode = commonErr.ERR_CODE_GET_STORE
		errMsg = s.getErrMsg(errCode, err)
		s.log.Error(errMsg)
		resp.Code = commonPb.TxStatusCode_INTERNAL_ERROR
		resp.Message = errMsg
		return resp
	}

	if fullBlock, err = s.getRestoreBlock(tx.Payload.Parameters); err != nil {
		errCode = commonErr.ERR_CODE_CHECK_PAYLOAD_PARAM_ARCHIVE_BLOCK
		errMsg = s.getErrMsg(errCode, err)
		s.log.Error(errMsg)
		resp.Code = commonPb.TxStatusCode_INTERNAL_ERROR
		resp.Message = errMsg
		return resp
	}

	if err = store.RestoreBlocks([][]byte{fullBlock}); err != nil {
		errMsg = fmt.Sprintf("restore block failed, %s", err.Error())
		s.log.Error(errMsg)
		resp.Code = commonPb.TxStatusCode_INTERNAL_ERROR
		resp.Message = errMsg
		return resp
	}

	resp.Code = commonPb.TxStatusCode_SUCCESS
	resp.Message = commonPb.TxStatusCode_SUCCESS.String()
	return resp
}
