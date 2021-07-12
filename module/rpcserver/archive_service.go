package rpcserver

import (
	"chainmaker.org/chainmaker-go/utils"
	commonErr "chainmaker.org/chainmaker/common/errors"
	commonPb "chainmaker.org/chainmaker/pb-go/common"
	"chainmaker.org/chainmaker/pb-go/consts"
	"chainmaker.org/chainmaker/protocol"
	"errors"
	"fmt"
)

func (s *ApiService) doArchive(tx *commonPb.Transaction) *commonPb.TxResponse {
	switch tx.Payload.TxType {
	case commonPb.TxType_ARCHIVE_FULL_BLOCK:
		return s.doArchiveBlock(tx)
	case commonPb.TxType_RESTORE_FULL_BLOCK:
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

	key := consts.ArchiveBlockPayload_BlockHeight.String()
	if params[0].Key != key {
		return 0, errors.New(fmt.Sprintf("invalid key, must be %s", key))
	}

	blockHeight, err := utils.BytesToUint64(params[0].Value)
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
		resp        = &commonPb.TxResponse{}
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

	key := consts.RestoreBlockPayload_FullBlock.String()
	if params[0].Key != key {
		return nil, errors.New(fmt.Sprintf("invalid key, must be %s", key))
	}

	fullBlock := params[0].Value
	if len(fullBlock) == 0 {
		return nil, errors.New("empty restore block data")
	}

	return fullBlock, nil
}

func (s *ApiService) doRestoreBlock(tx *commonPb.Transaction) *commonPb.TxResponse {

	var (
		err         error
		errMsg      string
		fullBlock   []byte
		errCode     commonErr.ErrCode
		store       protocol.BlockchainStore
		resp        = &commonPb.TxResponse{}
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
