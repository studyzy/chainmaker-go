package rpcserver

import (
	commonErr "chainmaker.org/chainmaker/common/errors"
	commonPb "chainmaker.org/chainmaker/pb-go/common"
	"chainmaker.org/chainmaker/protocol"
	"fmt"
	"github.com/golang/protobuf/proto"
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

func (s *ApiService) doArchiveBlock(tx *commonPb.Transaction) *commonPb.TxResponse {
	var (
		err     error
		errMsg  string
		errCode commonErr.ErrCode
		payload commonPb.ArchiveBlockPayload
		store   protocol.BlockchainStore
		resp    = &commonPb.TxResponse{}
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

	if err = proto.Unmarshal(tx.RequestPayload, &payload); err != nil {
		errCode = commonErr.ERR_CODE_SYSTEM_CONTRACT_PB_UNMARSHAL
		errMsg = s.getErrMsg(errCode, err)
		s.log.Error(errMsg)
		resp.Code = commonPb.TxStatusCode_INTERNAL_ERROR
		resp.Message = errMsg
		return resp
	}

	if err = store.ArchiveBlock(uint64(payload.BlockHeight)); err != nil {
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

func (s *ApiService) doRestoreBlock(tx *commonPb.Transaction) *commonPb.TxResponse {

	var (
		err     error
		errMsg  string
		errCode commonErr.ErrCode
		payload commonPb.RestoreBlockPayload
		store   protocol.BlockchainStore
		resp    = &commonPb.TxResponse{}
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

	if err = proto.Unmarshal(tx.RequestPayload, &payload); err != nil {
		errCode = commonErr.ERR_CODE_SYSTEM_CONTRACT_PB_UNMARSHAL
		errMsg = s.getErrMsg(errCode, err)
		s.log.Error(errMsg)
		resp.Code = commonPb.TxStatusCode_INTERNAL_ERROR
		resp.Message = errMsg
		return resp
	}

	if err = store.RestoreBlocks([][]byte{payload.FullBlock}); err != nil {
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
