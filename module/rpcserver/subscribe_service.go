/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package rpcserver

import (
	"errors"
	"fmt"
	"strings"

	"chainmaker.org/chainmaker/common/v2/bytehelper"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"
	"chainmaker.org/chainmaker/utils/v2"

	"chainmaker.org/chainmaker-go/subscriber"
	"chainmaker.org/chainmaker-go/subscriber/model"
	commonErr "chainmaker.org/chainmaker/common/v2/errors"
	apiPb "chainmaker.org/chainmaker/pb-go/v2/api"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	storePb "chainmaker.org/chainmaker/pb-go/v2/store"
	"chainmaker.org/chainmaker/protocol/v2"
	"github.com/gogo/protobuf/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	// TRUE true string
	TRUE = "true"
)

// Subscribe - deal block/tx subscribe request
func (s *ApiService) Subscribe(req *commonPb.TxRequest, server apiPb.RpcNode_SubscribeServer) error {
	var (
		errCode commonErr.ErrCode
		errMsg  string
	)

	tx := &commonPb.Transaction{
		Payload:   req.Payload,
		Sender:    req.Sender,
		Endorsers: req.Endorsers,
		Result:    nil}

	errCode, errMsg = s.validate(tx)
	if errCode != commonErr.ERR_CODE_OK {
		return status.Error(codes.Unauthenticated, errMsg)
	}

	switch req.Payload.Method {
	case syscontract.SubscribeFunction_SUBSCRIBE_BLOCK.String():
		return s.dealBlockSubscription(tx, server)
	case syscontract.SubscribeFunction_SUBSCRIBE_TX.String():
		return s.dealTxSubscription(tx, server)
	case syscontract.SubscribeFunction_SUBSCRIBE_CONTRACT_EVENT.String():
		return s.dealContractEventSubscription(tx, server)
	}

	return nil
}

// dealBlockSubscription - deal block subscribe request
func (s *ApiService) dealBlockSubscription(tx *commonPb.Transaction, server apiPb.RpcNode_SubscribeServer) error {
	var (
		err             error
		errMsg          string
		errCode         commonErr.ErrCode
		db              protocol.BlockchainStore
		lastBlockHeight int64
		payload         = tx.Payload
		startBlock      int64
		endBlock        int64
		withRWSet       = false
		onlyHeader      = false
		reqSender       protocol.Role
	)

	for _, kv := range payload.Parameters {
		if kv.Key == syscontract.SubscribeBlock_START_BLOCK.String() {
			startBlock, err = bytehelper.BytesToInt64(kv.Value)
		} else if kv.Key == syscontract.SubscribeBlock_END_BLOCK.String() {
			endBlock, err = bytehelper.BytesToInt64(kv.Value)
		} else if kv.Key == syscontract.SubscribeBlock_WITH_RWSET.String() {
			if string(kv.Value) == TRUE {
				withRWSet = true
			}
		} else if kv.Key == syscontract.SubscribeBlock_ONLY_HEADER.String() {
			if string(kv.Value) == TRUE {
				onlyHeader = true
				withRWSet = false
			}
		}

		if err != nil {
			errCode = commonErr.ERR_CODE_CHECK_PAYLOAD_PARAM_SUBSCRIBE_BLOCK
			errMsg = s.getErrMsg(errCode, err)
			s.log.Error(errMsg)
			return status.Error(codes.InvalidArgument, errMsg)
		}
	}

	if err = s.checkSubscribeBlockHeight(startBlock, endBlock); err != nil {
		errCode = commonErr.ERR_CODE_CHECK_PAYLOAD_PARAM_SUBSCRIBE_BLOCK
		errMsg = s.getErrMsg(errCode, err)
		s.log.Error(errMsg)
		return status.Error(codes.InvalidArgument, errMsg)
	}

	s.log.Infof("Recv block subscribe request: [start:%d]/[end:%d]/[withRWSet:%v]/[onlyHeader:%v]",
		startBlock, endBlock, withRWSet, onlyHeader)

	chainId := tx.Payload.ChainId
	if db, err = s.chainMakerServer.GetStore(chainId); err != nil {
		errCode = commonErr.ERR_CODE_GET_STORE
		errMsg = s.getErrMsg(errCode, err)
		s.log.Error(errMsg)
		return status.Error(codes.Internal, errMsg)
	}

	if lastBlockHeight, err = s.checkAndGetLastBlockHeight(db, startBlock); err != nil {
		errCode = commonErr.ERR_CODE_GET_LAST_BLOCK
		errMsg = s.getErrMsg(errCode, err)
		s.log.Error(errMsg)
		return status.Error(codes.Internal, errMsg)
	}

	reqSender, err = s.getRoleFromTx(tx)
	reqSenderOrgId := tx.Sender.Signer.OrgId
	if err != nil {
		return err
	}

	var startBlockHeight int64
	if startBlock > startBlockHeight {
		startBlockHeight = startBlock
	}

	if startBlock == -1 && endBlock == -1 {
		return s.sendNewBlock(db, tx, server, endBlock, withRWSet, onlyHeader,
			-1, reqSender, reqSenderOrgId)
	}

	if endBlock != -1 && endBlock <= lastBlockHeight {
		_, err = s.sendHistoryBlock(db, server, startBlockHeight, endBlock,
			withRWSet, onlyHeader, reqSender, reqSenderOrgId)

		if err != nil {
			s.log.Errorf("sendHistoryBlock failed, %s", err)
			return err
		}

		return status.Error(codes.OK, "OK")
	}

	alreadySendHistoryBlockHeight, err := s.sendHistoryBlock(db, server, startBlockHeight, endBlock,
		withRWSet, onlyHeader, reqSender, reqSenderOrgId)

	if err != nil {
		s.log.Errorf("sendHistoryBlock failed, %s", err)
		return err
	}

	s.log.Debugf("after sendHistoryBlock, alreadySendHistoryBlockHeight is %d", alreadySendHistoryBlockHeight)

	return s.sendNewBlock(db, tx, server, endBlock, withRWSet, onlyHeader, alreadySendHistoryBlockHeight,
		reqSender, reqSenderOrgId)
}

// dealTxSubscription - deal tx subscribe request
func (s *ApiService) dealTxSubscription(tx *commonPb.Transaction, server apiPb.RpcNode_SubscribeServer) error {
	var (
		err          error
		errMsg       string
		errCode      commonErr.ErrCode
		db           protocol.BlockchainStore
		payload      = tx.Payload
		startBlock   int64
		endBlock     int64
		contractName string
		txIds        []string
		reqSender    protocol.Role
	)

	for _, kv := range payload.Parameters {
		if kv.Key == syscontract.SubscribeTx_START_BLOCK.String() {
			startBlock, err = bytehelper.BytesToInt64(kv.Value)
		} else if kv.Key == syscontract.SubscribeTx_END_BLOCK.String() {
			endBlock, err = bytehelper.BytesToInt64(kv.Value)
		} else if kv.Key == syscontract.SubscribeTx_CONTRACT_NAME.String() {
			contractName = string(kv.Value)
		} else if kv.Key == syscontract.SubscribeTx_TX_IDS.String() {
			if kv.Value != nil {
				txIds = strings.Split(string(kv.Value), ",")
			}
		}

		if err != nil {
			errCode = commonErr.ERR_CODE_CHECK_PAYLOAD_PARAM_SUBSCRIBE_TX
			errMsg = s.getErrMsg(errCode, err)
			s.log.Error(errMsg)
			return status.Error(codes.InvalidArgument, errMsg)
		}
	}

	if err = s.checkSubscribeBlockHeight(startBlock, endBlock); err != nil {
		errCode = commonErr.ERR_CODE_CHECK_PAYLOAD_PARAM_SUBSCRIBE_TX
		errMsg = s.getErrMsg(errCode, err)
		s.log.Error(errMsg)
		return status.Error(codes.InvalidArgument, errMsg)
	}

	s.log.Infof("Recv block subscribe request: [start:%d]/[end:%d]/[contractName:%s]/[txIds:%+v]",
		startBlock, endBlock, contractName, txIds)

	chainId := tx.Payload.ChainId
	if db, err = s.chainMakerServer.GetStore(chainId); err != nil {
		errCode = commonErr.ERR_CODE_GET_STORE
		errMsg = s.getErrMsg(errCode, err)
		s.log.Error(errMsg)
		return status.Error(codes.Internal, errMsg)
	}

	reqSender, err = s.getRoleFromTx(tx)
	if err != nil {
		return err
	}
	reqSenderOrgId := tx.Sender.Signer.OrgId
	return s.doSendTx(tx, db, server, startBlock, endBlock, contractName, txIds, reqSender, reqSenderOrgId)
}

//dealContractEventSubscription - deal contract event subscribe request
func (s *ApiService) dealContractEventSubscription(tx *commonPb.Transaction,
	server apiPb.RpcNode_SubscribeServer) error {

	var (
		err          error
		errMsg       string
		errCode      commonErr.ErrCode
		payload      = tx.Payload
		topic        string
		contractName string
	)

	for _, kv := range payload.Parameters {
		if kv.Key == syscontract.SubscribeContractEvent_TOPIC.String() {
			topic = string(kv.Value)
		} else if kv.Key == syscontract.SubscribeContractEvent_CONTRACT_NAME.String() {
			contractName = string(kv.Value)
		}
	}

	if err = s.checkSubscribeContractEventPayload(topic, contractName); err != nil {
		errCode = commonErr.ERR_CODE_CHECK_PAYLOAD_PARAM_SUBSCRIBE_CONTRACT_EVENT
		errMsg = s.getErrMsg(errCode, err)
		s.log.Error(errMsg)
		return status.Error(codes.InvalidArgument, errMsg)
	}
	s.log.Infof("Recv contractEventInfo subscribe request: [topic:%v]/[contractName:%v]",
		topic, contractName)

	return s.doSendContractEvent(tx, server, topic, contractName)

}

func (s *ApiService) checkSubscribeContractEventPayload(topic, contractName string) error {
	if topic == "" || contractName == "" {
		return errors.New("invalid topic or contract name")
	}

	return nil
}

func (s *ApiService) doSendContractEvent(tx *commonPb.Transaction, server apiPb.RpcNode_SubscribeServer,
	topic, contractName string) error {

	var (
		errCode         commonErr.ErrCode
		err             error
		errMsg          string
		eventSubscriber *subscriber.EventSubscriber
		result          *commonPb.SubscribeResult
	)

	eventCh := make(chan model.NewContractEvent)

	chainId := tx.Payload.ChainId
	if eventSubscriber, err = s.chainMakerServer.GetEventSubscribe(chainId); err != nil {
		errCode = commonErr.ERR_CODE_GET_SUBSCRIBER
		errMsg = s.getErrMsg(errCode, err)
		s.log.Error(errMsg)
		return status.Error(codes.Internal, errMsg)
	}

	sub := eventSubscriber.SubscribeContractEvent(eventCh)
	defer sub.Unsubscribe()
	for {
		select {
		case ev := <-eventCh:
			contractEventInfoList := ev.ContractEventInfoList.ContractEvents
			sendEventInfoList := &commonPb.ContractEventInfoList{}
			for _, EventInfo := range contractEventInfoList {
				if EventInfo.ContractName != contractName || EventInfo.Topic != topic {
					continue
				}
				sendEventInfoList.ContractEvents = append(sendEventInfoList.ContractEvents, EventInfo)
			}
			if len(sendEventInfoList.ContractEvents) > 0 {
				if result, err = s.getContractEventSubscribeResult(sendEventInfoList); err != nil {
					s.log.Error(err.Error())
					return status.Error(codes.Internal, err.Error())
				}
				if err := server.Send(result); err != nil {
					err = fmt.Errorf("send block info by realtime failed, %s", err)
					s.log.Error(err.Error())
					return status.Error(codes.Internal, err.Error())
				}
			}
		case <-server.Context().Done():
			return nil
		case <-s.ctx.Done():
			return nil
		}
	}
}

func (s *ApiService) doSendTx(tx *commonPb.Transaction, db protocol.BlockchainStore,
	server apiPb.RpcNode_SubscribeServer, startBlock, endBlock int64, contractName string,
	txIds []string, reqSender protocol.Role, reqSenderOrgId string) error {

	var (
		txIdsMap                      = make(map[string]struct{})
		alreadySendHistoryBlockHeight int64
		err                           error
	)

	for _, txId := range txIds {
		txIdsMap[txId] = struct{}{}
	}

	if startBlock == -1 && endBlock == -1 {
		return s.sendNewTx(db, tx, server, startBlock, endBlock, contractName, txIds,
			txIdsMap, -1, reqSender, reqSenderOrgId)
	}

	if alreadySendHistoryBlockHeight, err = s.doSendHistoryTx(db, server, startBlock, endBlock,
		contractName, txIds, txIdsMap, reqSender, reqSenderOrgId); err != nil {
		return err
	}

	if alreadySendHistoryBlockHeight == 0 {
		return status.Error(codes.OK, "OK")
	}

	return s.sendNewTx(db, tx, server, startBlock, endBlock, contractName, txIds, txIdsMap,
		alreadySendHistoryBlockHeight, reqSender, reqSenderOrgId)
}

func (s *ApiService) doSendHistoryTx(db protocol.BlockchainStore, server apiPb.RpcNode_SubscribeServer,
	startBlock, endBlock int64, contractName string, txIds []string,
	txIdsMap map[string]struct{}, reqSender protocol.Role, reqSenderOrgId string) (int64, error) {

	var (
		err             error
		errMsg          string
		errCode         commonErr.ErrCode
		lastBlockHeight int64
	)

	var startBlockHeight int64
	if startBlock > startBlockHeight {
		startBlockHeight = startBlock
	}

	if lastBlockHeight, err = s.checkAndGetLastBlockHeight(db, startBlock); err != nil {
		errCode = commonErr.ERR_CODE_GET_LAST_BLOCK
		errMsg = s.getErrMsg(errCode, err)
		s.log.Error(errMsg)
		return -1, status.Error(codes.Internal, errMsg)
	}

	if endBlock != -1 && endBlock <= lastBlockHeight {
		_, err = s.sendHistoryTx(db, server, startBlockHeight, endBlock, contractName,
			txIds, txIdsMap, reqSender, reqSenderOrgId)

		if err != nil {
			s.log.Errorf("sendHistoryTx failed, %s", err)
			return -1, err
		}

		return 0, status.Error(codes.OK, "OK")
	}

	if len(txIds) > 0 && len(txIdsMap) == 0 {
		return 0, status.Error(codes.OK, "OK")
	}

	alreadySendHistoryBlockHeight, err := s.sendHistoryTx(db, server, startBlockHeight, endBlock, contractName,
		txIds, txIdsMap, reqSender, reqSenderOrgId)

	if err != nil {
		s.log.Errorf("sendHistoryTx failed, %s", err)
		return -1, err
	}

	if len(txIds) > 0 && len(txIdsMap) == 0 {
		return 0, status.Error(codes.OK, "OK")
	}

	s.log.Debugf("after sendHistoryBlock, alreadySendHistoryBlockHeight is %d", alreadySendHistoryBlockHeight)

	return alreadySendHistoryBlockHeight, nil
}

// sendNewBlock - send new block to subscriber
func (s *ApiService) sendNewBlock(store protocol.BlockchainStore, tx *commonPb.Transaction,
	server apiPb.RpcNode_SubscribeServer,
	endBlockHeight int64, withRWSet, onlyHeader bool, alreadySendHistoryBlockHeight int64,
	reqSender protocol.Role, reqSenderOrgId string) error {

	var (
		errCode         commonErr.ErrCode
		err             error
		errMsg          string
		eventSubscriber *subscriber.EventSubscriber
		blockInfo       *commonPb.BlockInfo
	)

	blockCh := make(chan model.NewBlockEvent)

	chainId := tx.Payload.ChainId
	if eventSubscriber, err = s.chainMakerServer.GetEventSubscribe(chainId); err != nil {
		errCode = commonErr.ERR_CODE_GET_SUBSCRIBER
		errMsg = s.getErrMsg(errCode, err)
		s.log.Error(errMsg)
		return status.Error(codes.Internal, errMsg)
	}

	sub := eventSubscriber.SubscribeBlockEvent(blockCh)
	defer sub.Unsubscribe()

	for {
		select {
		case ev := <-blockCh:
			blockInfo = ev.BlockInfo

			if alreadySendHistoryBlockHeight != -1 && int64(blockInfo.Block.Header.BlockHeight) > alreadySendHistoryBlockHeight {
				_, err = s.sendHistoryBlock(store, server, alreadySendHistoryBlockHeight+1,
					int64(blockInfo.Block.Header.BlockHeight), withRWSet, onlyHeader, reqSender, reqSenderOrgId)
				if err != nil {
					s.log.Errorf("send history block failed, %s", err)
					return err
				}

				alreadySendHistoryBlockHeight = -1
				continue
			}

			if reqSender == protocol.RoleLight {
				newBlock := utils.FilterBlockTxs(reqSenderOrgId, blockInfo.Block)
				blockInfo = &commonPb.BlockInfo{
					Block:     newBlock,
					RwsetList: ev.BlockInfo.RwsetList,
				}
			}

			//printAllTxsOfBlock(blockInfo, reqSender, reqSenderOrgId)

			if err = s.dealBlockSubscribeResult(server, blockInfo, withRWSet, onlyHeader); err != nil {
				s.log.Errorf(err.Error())
				return status.Error(codes.Internal, err.Error())
			}

			if endBlockHeight != -1 && int64(blockInfo.Block.Header.BlockHeight) >= endBlockHeight {
				return status.Error(codes.OK, "OK")
			}

		case <-server.Context().Done():
			return nil
		case <-s.ctx.Done():
			return nil
		}
	}
}

func (s *ApiService) dealBlockSubscribeResult(server apiPb.RpcNode_SubscribeServer, blockInfo *commonPb.BlockInfo,
	withRWSet, onlyHeader bool) error {

	var (
		err    error
		result *commonPb.SubscribeResult
	)

	if !withRWSet {
		blockInfo = &commonPb.BlockInfo{
			Block:     blockInfo.Block,
			RwsetList: nil,
		}
	}

	if result, err = s.getBlockSubscribeResult(blockInfo, onlyHeader); err != nil {
		return fmt.Errorf("get block subscribe result failed, %s", err)
	}

	if err := server.Send(result); err != nil {
		return fmt.Errorf("send block subscribe result by realtime failed, %s", err)
	}

	return nil
}

// sendNewTx - send new tx to subscriber
func (s *ApiService) sendNewTx(store protocol.BlockchainStore, tx *commonPb.Transaction,
	server apiPb.RpcNode_SubscribeServer, startBlock, endBlock int64, contractName string,
	txIds []string, txIdsMap map[string]struct{}, alreadySendHistoryBlockHeight int64,
	reqSender protocol.Role, reqSenderOrgId string) error {

	var (
		errCode         commonErr.ErrCode
		err             error
		errMsg          string
		eventSubscriber *subscriber.EventSubscriber
		block           *commonPb.Block
	)

	blockCh := make(chan model.NewBlockEvent)

	chainId := tx.Payload.ChainId
	if eventSubscriber, err = s.chainMakerServer.GetEventSubscribe(chainId); err != nil {
		errCode = commonErr.ERR_CODE_GET_SUBSCRIBER
		errMsg = s.getErrMsg(errCode, err)
		s.log.Error(errMsg)
		return status.Error(codes.Internal, errMsg)
	}

	sub := eventSubscriber.SubscribeBlockEvent(blockCh)
	defer sub.Unsubscribe()

	for {
		select {
		case ev := <-blockCh:
			block = ev.BlockInfo.Block

			if alreadySendHistoryBlockHeight != -1 && int64(block.Header.BlockHeight) > alreadySendHistoryBlockHeight {
				_, err = s.sendHistoryTx(store, server, alreadySendHistoryBlockHeight+1,
					int64(block.Header.BlockHeight), contractName, txIds, txIdsMap, reqSender, reqSenderOrgId)
				if err != nil {
					s.log.Errorf("send history block failed, %s", err)
					return err
				}

				alreadySendHistoryBlockHeight = -1
				continue
			}

			if err := s.sendSubscribeTx(server, block.Txs, contractName, txIds, txIdsMap,
				reqSender, reqSenderOrgId); err != nil {
				errMsg = fmt.Sprintf("send subscribe tx failed, %s", err)
				s.log.Error(errMsg)
				return status.Error(codes.Internal, errMsg)
			}

			if s.checkIsFinish(txIds, endBlock, txIdsMap, ev.BlockInfo) {
				return status.Error(codes.OK, "OK")
			}

		case <-server.Context().Done():
			return nil
		case <-s.ctx.Done():
			return nil
		}
	}
}

func (s *ApiService) checkIsFinish(txIds []string, endBlock int64,
	txIdsMap map[string]struct{}, blockInfo *commonPb.BlockInfo) bool {

	if len(txIds) > 0 && len(txIdsMap) == 0 {
		return true
	}

	if endBlock != -1 && int64(blockInfo.Block.Header.BlockHeight) >= endBlock {
		return true
	}

	return false
}

func (s *ApiService) getRateLimitToken() error {
	if s.subscriberRateLimiter != nil {
		if err := s.subscriberRateLimiter.Wait(s.ctx); err != nil {
			errMsg := fmt.Sprintf("subscriber rateLimiter wait token failed, %s", err.Error())
			s.log.Error(errMsg)
			return errors.New(errMsg)
		}
	}

	return nil
}

// sendHistoryBlock - send history block to subscriber
func (s *ApiService) sendHistoryBlock(store protocol.BlockchainStore, server apiPb.RpcNode_SubscribeServer,
	startBlockHeight, endBlockHeight int64, withRWSet, onlyHeader bool, reqSender protocol.Role,
	reqSenderOrgId string) (int64, error) {

	var (
		err    error
		errMsg string
		result *commonPb.SubscribeResult
	)

	i := startBlockHeight
	for {
		select {
		case <-s.ctx.Done():
			return -1, status.Error(codes.Internal, "chainmaker is restarting, please retry later")
		default:
			if err = s.getRateLimitToken(); err != nil {
				return -1, status.Error(codes.Internal, err.Error())
			}

			if endBlockHeight != -1 && i > endBlockHeight {
				return i - 1, nil
			}

			blockInfo, alreadySendHistoryBlockHeight, err := s.getBlockInfoFromStore(store, i, withRWSet,
				reqSender, reqSenderOrgId)

			if err != nil {
				return -1, status.Error(codes.Internal, errMsg)
			}

			if blockInfo == nil || alreadySendHistoryBlockHeight > 0 {
				return alreadySendHistoryBlockHeight, nil
			}

			if result, err = s.getBlockSubscribeResult(blockInfo, onlyHeader); err != nil {
				errMsg = fmt.Sprintf("get block subscribe result failed, %s", err)
				s.log.Error(errMsg)
				return -1, errors.New(errMsg)
			}

			if err := server.Send(result); err != nil {
				errMsg = fmt.Sprintf("send block info by history failed, %s", err)
				s.log.Error(errMsg)
				return -1, status.Error(codes.Internal, errMsg)
			}

			i++
		}
	}
}

func (s *ApiService) getBlockInfoFromStore(store protocol.BlockchainStore, curblockHeight int64, withRWSet bool,
	reqSender protocol.Role, reqSenderOrgId string) (blockInfo *commonPb.BlockInfo,
	alreadySendHistoryBlockHeight int64, err error) {

	var (
		errMsg         string
		block          *commonPb.Block
		blockWithRWSet *storePb.BlockWithRWSet
	)

	if withRWSet {
		blockWithRWSet, err = store.GetBlockWithRWSets(uint64(curblockHeight))
	} else {
		block, err = store.GetBlock(uint64(curblockHeight))
	}

	if err != nil {
		if withRWSet {
			errMsg = fmt.Sprintf("get block with rwset failed, at [height:%d], %s", curblockHeight, err)
		} else {
			errMsg = fmt.Sprintf("get block failed, at [height:%d], %s", curblockHeight, err)
		}
		s.log.Error(errMsg)
		return nil, -1, errors.New(errMsg)
	}

	if withRWSet {
		if blockWithRWSet == nil {
			return nil, curblockHeight - 1, nil
		}

		blockInfo = &commonPb.BlockInfo{
			Block:     blockWithRWSet.Block,
			RwsetList: blockWithRWSet.TxRWSets,
		}

		// filter txs so that only related ones get passed
		if reqSender == protocol.RoleLight {
			newBlock := utils.FilterBlockTxs(reqSenderOrgId, blockWithRWSet.Block)
			blockInfo = &commonPb.BlockInfo{
				Block:     newBlock,
				RwsetList: blockWithRWSet.TxRWSets,
			}
		}
	} else {
		if block == nil {
			return nil, curblockHeight - 1, nil
		}

		blockInfo = &commonPb.BlockInfo{
			Block:     block,
			RwsetList: nil,
		}

		// filter txs so that only related ones get passed
		if reqSender == protocol.RoleLight {
			newBlock := utils.FilterBlockTxs(reqSenderOrgId, block)
			blockInfo = &commonPb.BlockInfo{
				Block:     newBlock,
				RwsetList: nil,
			}
		}
	}

	//printAllTxsOfBlock(blockInfo, reqSender, reqSenderOrgId)

	return blockInfo, -1, nil
}

// sendHistoryTx - send history tx to subscriber
func (s *ApiService) sendHistoryTx(store protocol.BlockchainStore,
	server apiPb.RpcNode_SubscribeServer,
	startBlockHeight, endBlockHeight int64,
	contractName string, txIds []string, txIdsMap map[string]struct{},
	reqSender protocol.Role, reqSenderOrgId string) (int64, error) {

	var (
		err    error
		errMsg string
		block  *commonPb.Block
	)

	i := startBlockHeight
	for {
		select {
		case <-s.ctx.Done():
			return -1, status.Error(codes.Internal, "chainmaker is restarting, please retry later")
		default:
			if err = s.getRateLimitToken(); err != nil {
				return -1, status.Error(codes.Internal, err.Error())
			}

			if endBlockHeight != -1 && i > endBlockHeight {
				return i - 1, nil
			}

			if len(txIds) > 0 && len(txIdsMap) == 0 {
				return i - 1, nil
			}

			block, err = store.GetBlock(uint64(i))

			if err != nil {
				errMsg = fmt.Sprintf("get block failed, at [height:%d], %s", i, err)
				s.log.Error(errMsg)
				return -1, status.Error(codes.Internal, errMsg)
			}

			if block == nil {
				return i - 1, nil
			}

			if err := s.sendSubscribeTx(server, block.Txs, contractName, txIds, txIdsMap,
				reqSender, reqSenderOrgId); err != nil {
				errMsg = fmt.Sprintf("send subscribe tx failed, %s", err)
				s.log.Error(errMsg)
				return -1, status.Error(codes.Internal, errMsg)
			}

			i++
		}
	}
}

// checkSubscribeBlockHeight - check subscriber payload info
func (s *ApiService) checkSubscribeBlockHeight(startBlockHeight, endBlockHeight int64) error {
	if startBlockHeight < -1 || endBlockHeight < -1 ||
		(endBlockHeight != -1 && startBlockHeight > endBlockHeight) {

		return errors.New("invalid start block height or end block height")
	}

	return nil
}

func (s *ApiService) getTxSubscribeResult(tx *commonPb.Transaction) (*commonPb.SubscribeResult, error) {
	txBytes, err := proto.Marshal(tx)
	if err != nil {
		errMsg := fmt.Sprintf("marshal tx info failed, %s", err)
		s.log.Error(errMsg)
		return nil, errors.New(errMsg)
	}

	result := &commonPb.SubscribeResult{
		Data: txBytes,
	}

	return result, nil
}

func (s *ApiService) getBlockSubscribeResult(blockInfo *commonPb.BlockInfo,
	onlyHeader bool) (*commonPb.SubscribeResult, error) {

	var (
		resultBytes []byte
		err         error
	)

	if onlyHeader {
		resultBytes, err = proto.Marshal(blockInfo.Block.Header)
	} else {
		resultBytes, err = proto.Marshal(blockInfo)
	}

	if err != nil {
		errMsg := fmt.Sprintf("marshal block subscribe result failed, %s", err)
		s.log.Error(errMsg)
		return nil, errors.New(errMsg)
	}

	result := &commonPb.SubscribeResult{
		Data: resultBytes,
	}

	return result, nil
}

func (s *ApiService) getContractEventSubscribeResult(contractEventsInfoList *commonPb.ContractEventInfoList) (
	*commonPb.SubscribeResult, error) {

	eventBytes, err := proto.Marshal(contractEventsInfoList)
	if err != nil {
		errMsg := fmt.Sprintf("marshal contract event info failed, %s", err)
		s.log.Error(errMsg)
		return nil, errors.New(errMsg)
	}

	result := &commonPb.SubscribeResult{
		Data: eventBytes,
	}

	return result, nil
}
func (s *ApiService) sendSubscribeTx(server apiPb.RpcNode_SubscribeServer,
	txs []*commonPb.Transaction, contractName string, txIds []string,
	txIdsMap map[string]struct{}, reqSender protocol.Role, reqSenderOrgId string) error {

	var (
		err error
	)

	for _, tx := range txs {
		if contractName == "" && len(txIds) == 0 {
			if err = s.doSendSubscribeTx(server, tx, reqSender, reqSenderOrgId); err != nil {
				return err
			}
			continue
		}

		if s.checkIsContinue(tx, contractName, txIds, txIdsMap) {
			continue
		}

		if err = s.doSendSubscribeTx(server, tx, reqSender, reqSenderOrgId); err != nil {
			return err
		}
	}

	return nil
}

func (s *ApiService) checkIsContinue(tx *commonPb.Transaction, contractName string, txIds []string,
	txIdsMap map[string]struct{}) bool {

	if contractName != "" && tx.Payload.ContractName != contractName {
		return true
	}

	if len(txIds) > 0 {
		_, ok := txIdsMap[tx.Payload.TxId]
		if !ok {
			return true
		}

		delete(txIdsMap, tx.Payload.TxId)
	}

	return false
}

func (s *ApiService) doSendSubscribeTx(server apiPb.RpcNode_SubscribeServer, tx *commonPb.Transaction,
	reqSender protocol.Role, reqSenderOrgId string) error {

	var (
		err    error
		errMsg string
		result *commonPb.SubscribeResult
	)

	isReqSenderLightNode := reqSender == protocol.RoleLight
	isTxRelatedToSender := (tx.Sender != nil) && reqSenderOrgId == tx.Sender.Signer.OrgId

	if result, err = s.getTxSubscribeResult(tx); err != nil {
		errMsg = fmt.Sprintf("get tx subscribe result failed, %s", err)
		s.log.Error(errMsg)
		return errors.New(errMsg)
	}

	if isReqSenderLightNode {
		if isTxRelatedToSender {
			if err := server.Send(result); err != nil {
				errMsg = fmt.Sprintf("send subscribe tx result failed, %s", err)
				s.log.Error(errMsg)
				return errors.New(errMsg)
			}
		}
	} else {
		if err := server.Send(result); err != nil {
			errMsg = fmt.Sprintf("send subscribe tx result failed, %s", err)
			s.log.Error(errMsg)
			return errors.New(errMsg)
		}
	}

	return nil
}

func (s *ApiService) checkAndGetLastBlockHeight(store protocol.BlockchainStore,
	payloadStartBlockHeight int64) (int64, error) {

	var (
		err             error
		errMsg          string
		errCode         commonErr.ErrCode
		lastBlock       *commonPb.Block
		lastBlockHeight uint64
	)

	if lastBlock, err = store.GetLastBlock(); err != nil {
		errCode = commonErr.ERR_CODE_GET_LAST_BLOCK
		errMsg = s.getErrMsg(errCode, err)
		s.log.Error(errMsg)
		return -1, status.Error(codes.Internal, errMsg)
	}

	lastBlockHeight = lastBlock.Header.BlockHeight

	if int64(lastBlockHeight) < payloadStartBlockHeight {
		errMsg = fmt.Sprintf("payload start block height:%d > last block height:%d",
			payloadStartBlockHeight, lastBlockHeight)

		s.log.Error(errMsg)
		return -1, status.Error(codes.InvalidArgument, errMsg)
	}

	return int64(lastBlock.Header.BlockHeight), nil
}

//func printAllTxsOfBlock(blockInfo *commonPb.BlockInfo, reqSender protocol.Role, reqSenderOrgId string) {
//	fmt.Printf("Verifying subscribed block of height: %d\n", blockInfo.Block.Header.BlockHeight)
//	fmt.Printf("verify: the role of request sender is Light [%t]\n", reqSender == protocol.RoleLight)
//	fmt.Printf("the block has %d txs\n", len(blockInfo.Block.Txs))
//	for i, tx := range blockInfo.Block.Txs {
//
//		if tx.Sender != nil {
//
//			fmt.Printf("Tx [%d] of subscribed block, from org %v, TxSenderOrgId is %s, "+
//				"verify: this tx is of the same organization [%t]\n", i, tx.Sender.Signer.OrgId,
//				reqSenderOrgId, tx.Sender.Signer.OrgId == reqSenderOrgId)
//		}
//	}
//	fmt.Println()
//}

func (s *ApiService) getRoleFromTx(tx *commonPb.Transaction) (protocol.Role, error) {
	bc, err := s.chainMakerServer.GetBlockchain(tx.Payload.ChainId)
	if err != nil {
		errCode := commonErr.ERR_CODE_GET_BLOCKCHAIN
		errMsg := s.getErrMsg(errCode, err)
		s.log.Error(errMsg)
		return "", err
	}

	ac := bc.GetAccessControl()
	return utils.GetRoleFromTx(tx, ac)
}
