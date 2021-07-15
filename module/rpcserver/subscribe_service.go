/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package rpcserver

import (
	apiPb "chainmaker.org/chainmaker/pb-go/api"
	commonPb "chainmaker.org/chainmaker/pb-go/common"
)

//import (
//	"errors"
//	"fmt"
//
//	"chainmaker.org/chainmaker-go/subscriber"
//	"chainmaker.org/chainmaker-go/subscriber/model"
//	commonErr "chainmaker.org/chainmaker/common/errors"
//	apiPb "chainmaker.org/chainmaker/pb-go/api"
//	commonPb "chainmaker.org/chainmaker/pb-go/common"
//	storePb "chainmaker.org/chainmaker/pb-go/store"
//	"chainmaker.org/chainmaker/protocol"
//	"github.com/gogo/protobuf/proto"
//	"google.golang.org/grpc/codes"
//	"google.golang.org/grpc/status"
//)
//


func (s *ApiService) Subscribe(request *commonPb.TxRequest, server apiPb.RpcNode_SubscribeServer) error {
	panic("implement me")
}
//// Subscribe - deal block/tx subscribe request
//func (s *ApiService) Subscribe(req *commonPb.TxRequest, server apiPb.RpcNode_SubscribeServer) error {
//	var (
//		errCode commonErr.ErrCode
//		errMsg  string
//	)
//
//	tx := &commonPb.Transaction{
//		Header:           req.Header,
//		RequestPayload:   req.Payload,
//		RequestSignature: req.Signature,
//		Result:           nil}
//
//	errCode, errMsg = s.validate(tx)
//	if errCode != commonErr.ERR_CODE_OK {
//		return status.Error(codes.Unauthenticated, errMsg)
//	}
//
//	switch req.Header.TxType {
//	case commonPb.TxType_SUBSCRIBE:
//		return s.dealBlockSubscription(tx, server)
//	case commonPb.TxType_SUBSCRIBE:
//		return s.dealTxSubscription(tx, server)
//	case commonPb.TxType_SUBSCRIBE:
//		return s.dealContractEventSubscription(tx, server)
//	}
//
//	return nil
//}
//
////dealContractEventSubscription - deal contract event subscribe request
//func (s *ApiService) dealContractEventSubscription(tx *commonPb.Transaction, server apiPb.RpcNode_SubscribeServer) error {
//	var (
//		err     error
//		errMsg  string
//		errCode commonErr.ErrCode
//		payload commonPb.SubscribeContractEventPayload
//	)
//
//	if err = proto.Unmarshal(tx.RequestPayload, &payload); err != nil {
//		errCode = commonErr.ERR_CODE_SYSTEM_CONTRACT_PB_UNMARSHAL
//		errMsg = s.getErrMsg(errCode, err)
//		s.log.Error(errMsg)
//		return status.Error(codes.InvalidArgument, errMsg)
//	}
//
//	if err = s.checkSubscribeContractEventPayload(&payload); err != nil {
//		errCode = commonErr.ERR_CODE_CHECK_PAYLOAD_PARAM_SUBSCRIBE_CONTRACT_EVENT
//		errMsg = s.getErrMsg(errCode, err)
//		s.log.Error(errMsg)
//		return status.Error(codes.InvalidArgument, errMsg)
//	}
//	s.log.Infof("Recv contractEventInfo subscribe request: [topic:%v]/[contractName:%v]",
//		payload.Topic, payload.ContractName)
//
//	return s.doSendContractEvent(tx, server, payload)
//
//}
//
//func (s *ApiService) checkSubscribeContractEventPayload(payload *commonPb.SubscribeContractEventPayload) error {
//	if payload.Topic == "" || payload.ContractName == "" {
//		return errors.New("invalid topic or contract name")
//	}
//	return nil
//}
//func (s *ApiService) doSendContractEvent(tx *commonPb.Transaction, server apiPb.RpcNode_SubscribeServer, payload commonPb.SubscribeContractEventPayload) error {
//
//	var (
//		errCode         commonErr.ErrCode
//		err             error
//		errMsg          string
//		eventSubscriber *subscriber.EventSubscriber
//		result          *commonPb.SubscribeResult
//	)
//
//	eventCh := make(chan model.NewContractEvent)
//
//	chainId := tx.Payload.ChainId
//	if eventSubscriber, err = s.chainMakerServer.GetEventSubscribe(chainId); err != nil {
//		errCode = commonErr.ERR_CODE_GET_SUBSCRIBER
//		errMsg = s.getErrMsg(errCode, err)
//		s.log.Error(errMsg)
//		return status.Error(codes.Internal, errMsg)
//	}
//
//	sub := eventSubscriber.SubscribeContractEvent(eventCh)
//	defer sub.Unsubscribe()
//	for {
//		select {
//		case ev := <-eventCh:
//			contractEventInfoList := ev.ContractEventInfoList.ContractEvents
//			sendEventInfoList := &commonPb.ContractEventInfoList{}
//			for _, EventInfo := range contractEventInfoList {
//				if EventInfo.ContractName != payload.ContractName || EventInfo.Topic != payload.Topic {
//					continue
//				}
//				sendEventInfoList.ContractEvents = append(sendEventInfoList.ContractEvents, EventInfo)
//			}
//			if len(sendEventInfoList.ContractEvents) > 0 {
//				if result, err = s.getContractEventSubscribeResult(sendEventInfoList); err != nil {
//					s.log.Error(err.Error())
//					return status.Error(codes.Internal, err.Error())
//				}
//				if err := server.Send(result); err != nil {
//					err = fmt.Errorf("send block info by realtime failed, %s", err)
//					s.log.Error(err.Error())
//					return status.Error(codes.Internal, err.Error())
//				}
//			}
//		case <-server.Context().Done():
//			return nil
//		case <-s.ctx.Done():
//			return nil
//		}
//	}
//}
//
//// dealTxSubscription - deal tx subscribe request
//func (s *ApiService) dealTxSubscription(tx *commonPb.Transaction, server apiPb.RpcNode_SubscribeServer) error {
//	var (
//		err     error
//		errMsg  string
//		errCode commonErr.ErrCode
//		payload commonPb.SubscribeTxPayload
//		db      protocol.BlockchainStore
//	)
//
//	if err = proto.Unmarshal(tx.RequestPayload, &payload); err != nil {
//		errCode = commonErr.ERR_CODE_SYSTEM_CONTRACT_PB_UNMARSHAL
//		errMsg = s.getErrMsg(errCode, err)
//		s.log.Error(errMsg)
//		return status.Error(codes.InvalidArgument, errMsg)
//	}
//
//	if err = s.checkSubscribePayload(payload.StartBlock, payload.EndBlock); err != nil {
//		errCode = commonErr.ERR_CODE_CHECK_PAYLOAD_PARAM_SUBSCRIBE_TX
//		errMsg = s.getErrMsg(errCode, err)
//		s.log.Error(errMsg)
//		return status.Error(codes.InvalidArgument, errMsg)
//	}
//
//	s.log.Infof("Recv block subscribe request: [start:%d]/[end:%d]/[txType:%d]/[txIds:%+v]",
//		payload.StartBlock, payload.EndBlock, payload.TxType, payload.TxIds)
//
//	chainId := tx.Payload.ChainId
//	if db, err = s.chainMakerServer.GetStore(chainId); err != nil {
//		errCode = commonErr.ERR_CODE_GET_STORE
//		errMsg = s.getErrMsg(errCode, err)
//		s.log.Error(errMsg)
//		return status.Error(codes.Internal, errMsg)
//	}
//
//	return s.doSendTx(tx, db, server, payload)
//}
//
//func (s *ApiService) doSendTx(tx *commonPb.Transaction, db protocol.BlockchainStore,
//	server apiPb.RpcNode_SubscribeServer, payload commonPb.SubscribeTxPayload) error {
//
//	var (
//		txIdsMap                      = make(map[string]struct{})
//		alreadySendHistoryBlockHeight int64
//		err                           error
//	)
//
//	for _, txId := range payload.TxIds {
//		txIdsMap[txId] = struct{}{}
//	}
//
//	if payload.StartBlock == -1 && payload.EndBlock == -1 {
//		return s.sendNewTx(db, tx, server, payload, txIdsMap, -1)
//	}
//
//	if alreadySendHistoryBlockHeight, err = s.doSendHistoryTx(db, server, payload, txIdsMap); err != nil {
//		return err
//	}
//
//	if alreadySendHistoryBlockHeight == 0 {
//		return status.Error(codes.OK, "OK")
//	}
//
//	return s.sendNewTx(db, tx, server, payload, txIdsMap, alreadySendHistoryBlockHeight)
//}
//
//func (s *ApiService) doSendHistoryTx(db protocol.BlockchainStore, server apiPb.RpcNode_SubscribeServer,
//	payload commonPb.SubscribeTxPayload, txIdsMap map[string]struct{}) (int64, error) {
//
//	var (
//		err             error
//		errMsg          string
//		errCode         commonErr.ErrCode
//		lastBlockHeight int64
//	)
//
//	var startBlockHeight int64
//	if payload.StartBlock > startBlockHeight {
//		startBlockHeight = payload.StartBlock
//	}
//
//	if lastBlockHeight, err = s.checkAndGetLastBlockHeight(db, payload.StartBlock); err != nil {
//		errCode = commonErr.ERR_CODE_GET_LAST_BLOCK
//		errMsg = s.getErrMsg(errCode, err)
//		s.log.Error(errMsg)
//		return -1, status.Error(codes.Internal, errMsg)
//	}
//
//	if payload.EndBlock != -1 && payload.EndBlock <= lastBlockHeight {
//		err, _ := s.sendHistoryTx(db, server, startBlockHeight, payload.EndBlock, payload.TxType, payload.TxIds, txIdsMap)
//		if err != nil {
//			s.log.Errorf("sendHistoryTx failed, %s", err)
//			return -1, err
//		}
//
//		return 0, status.Error(codes.OK, "OK")
//	}
//
//	if len(payload.TxIds) > 0 && len(txIdsMap) == 0 {
//		return 0, status.Error(codes.OK, "OK")
//	}
//
//	err, alreadySendHistoryBlockHeight := s.sendHistoryTx(db, server, startBlockHeight, payload.EndBlock, payload.TxType, payload.TxIds, txIdsMap)
//	if err != nil {
//		s.log.Errorf("sendHistoryTx failed, %s", err)
//		return -1, err
//	}
//
//	if len(payload.TxIds) > 0 && len(txIdsMap) == 0 {
//		return 0, status.Error(codes.OK, "OK")
//	}
//
//	s.log.Debugf("after sendHistoryBlock, alreadySendHistoryBlockHeight is %d", alreadySendHistoryBlockHeight)
//
//	return alreadySendHistoryBlockHeight, nil
//}
//
//// dealBlockSubscription - deal block subscribe request
//func (s *ApiService) dealBlockSubscription(tx *commonPb.Transaction, server apiPb.RpcNode_SubscribeServer) error {
//	var (
//		err             error
//		errMsg          string
//		errCode         commonErr.ErrCode
//		payload         commonPb.SubscribeBlockPayload
//		db              protocol.BlockchainStore
//		lastBlockHeight int64
//	)
//
//	if err = proto.Unmarshal(tx.RequestPayload, &payload); err != nil {
//		errCode = commonErr.ERR_CODE_SYSTEM_CONTRACT_PB_UNMARSHAL
//		errMsg = s.getErrMsg(errCode, err)
//		s.log.Error(errMsg)
//		return status.Error(codes.InvalidArgument, errMsg)
//	}
//
//	if err = s.checkSubscribePayload(payload.StartBlock, payload.EndBlock); err != nil {
//		errCode = commonErr.ERR_CODE_CHECK_PAYLOAD_PARAM_SUBSCRIBE_BLOCK
//		errMsg = s.getErrMsg(errCode, err)
//		s.log.Error(errMsg)
//		return status.Error(codes.InvalidArgument, errMsg)
//	}
//
//	s.log.Infof("Recv block subscribe request: [start:%d]/[end:%d]/[withRWSet:%v]",
//		payload.StartBlock, payload.EndBlock, payload.WithRwSet)
//
//	chainId := tx.Payload.ChainId
//	if db, err = s.chainMakerServer.GetStore(chainId); err != nil {
//		errCode = commonErr.ERR_CODE_GET_STORE
//		errMsg = s.getErrMsg(errCode, err)
//		s.log.Error(errMsg)
//		return status.Error(codes.Internal, errMsg)
//	}
//
//	if lastBlockHeight, err = s.checkAndGetLastBlockHeight(db, payload.StartBlock); err != nil {
//		errCode = commonErr.ERR_CODE_GET_LAST_BLOCK
//		errMsg = s.getErrMsg(errCode, err)
//		s.log.Error(errMsg)
//		return status.Error(codes.Internal, errMsg)
//	}
//
//	var startBlockHeight int64
//	if payload.StartBlock > startBlockHeight {
//		startBlockHeight = payload.StartBlock
//	}
//
//	if payload.StartBlock == -1 && payload.EndBlock == -1 {
//		return s.sendNewBlock(db, tx, server, payload.EndBlock, payload.WithRwSet, -1)
//	}
//
//	if payload.EndBlock != -1 && payload.EndBlock <= lastBlockHeight {
//		err, _ := s.sendHistoryBlock(db, server, startBlockHeight, payload.EndBlock, payload.WithRwSet)
//		if err != nil {
//			s.log.Errorf("sendHistoryBlock failed, %s", err)
//			return err
//		}
//
//		return status.Error(codes.OK, "OK")
//	}
//
//	err, alreadySendHistoryBlockHeight := s.sendHistoryBlock(db, server, startBlockHeight, payload.EndBlock, payload.WithRwSet)
//	if err != nil {
//		s.log.Errorf("sendHistoryBlock failed, %s", err)
//		return err
//	}
//
//	s.log.Debugf("after sendHistoryBlock, alreadySendHistoryBlockHeight is %d", alreadySendHistoryBlockHeight)
//
//	return s.sendNewBlock(db, tx, server, payload.EndBlock, payload.WithRwSet, alreadySendHistoryBlockHeight)
//}
//
//// sendNewBlock - send new block to subscriber
//func (s *ApiService) sendNewBlock(store protocol.BlockchainStore, tx *commonPb.Transaction,
//	server apiPb.RpcNode_SubscribeServer,
//	endBlockHeight int64, withRWSet bool, alreadySendHistoryBlockHeight int64) error {
//
//	var (
//		errCode         commonErr.ErrCode
//		err             error
//		errMsg          string
//		eventSubscriber *subscriber.EventSubscriber
//		blockInfo       *commonPb.BlockInfo
//	)
//
//	blockCh := make(chan model.NewBlockEvent)
//
//	chainId := tx.Payload.ChainId
//	if eventSubscriber, err = s.chainMakerServer.GetEventSubscribe(chainId); err != nil {
//		errCode = commonErr.ERR_CODE_GET_SUBSCRIBER
//		errMsg = s.getErrMsg(errCode, err)
//		s.log.Error(errMsg)
//		return status.Error(codes.Internal, errMsg)
//	}
//
//	sub := eventSubscriber.SubscribeBlockEvent(blockCh)
//	defer sub.Unsubscribe()
//
//	for {
//		select {
//		case ev := <-blockCh:
//			blockInfo = ev.BlockInfo
//
//			if alreadySendHistoryBlockHeight != -1 && blockInfo.Block.Header.BlockHeight > alreadySendHistoryBlockHeight {
//				err, _ = s.sendHistoryBlock(store, server, alreadySendHistoryBlockHeight+1,
//					blockInfo.Block.Header.BlockHeight, withRWSet)
//				if err != nil {
//					s.log.Errorf("send history block failed, %s", err)
//					return err
//				}
//
//				alreadySendHistoryBlockHeight = -1
//				continue
//			}
//
//			if err = s.dealBlockSubscribeResult(server, blockInfo, endBlockHeight, withRWSet); err != nil {
//				s.log.Errorf(err.Error())
//				return status.Error(codes.Internal, err.Error())
//			}
//
//			if endBlockHeight != -1 && blockInfo.Block.Header.BlockHeight >= endBlockHeight {
//				return status.Error(codes.OK, "OK")
//			}
//
//		case <-server.Context().Done():
//			return nil
//		case <-s.ctx.Done():
//			return nil
//		}
//	}
//}
//
//func (s *ApiService) dealBlockSubscribeResult(server apiPb.RpcNode_SubscribeServer, blockInfo *commonPb.BlockInfo,
//	endBlockHeight int64, withRWSet bool) error {
//
//	var (
//		err    error
//		result *commonPb.SubscribeResult
//	)
//
//	if !withRWSet {
//		blockInfo = &commonPb.BlockInfo{
//			Block:     blockInfo.Block,
//			RwsetList: nil,
//		}
//	}
//	if result, err = s.getBlockSubscribeResult(blockInfo); err != nil {
//		return fmt.Errorf("get block subscribe result failed, %s", err)
//	}
//
//	if err := server.Send(result); err != nil {
//		return fmt.Errorf("send block info by realtime failed, %s", err)
//	}
//
//	return nil
//}
//
//// sendNewTx - send new tx to subscriber
//func (s *ApiService) sendNewTx(store protocol.BlockchainStore, tx *commonPb.Transaction,
//	server apiPb.RpcNode_SubscribeServer, payload commonPb.SubscribeTxPayload,
//	txIdsMap map[string]struct{}, alreadySendHistoryBlockHeight int64) error {
//
//	var (
//		errCode         commonErr.ErrCode
//		err             error
//		errMsg          string
//		eventSubscriber *subscriber.EventSubscriber
//		block           *commonPb.Block
//	)
//
//	blockCh := make(chan model.NewBlockEvent)
//
//	chainId := tx.Payload.ChainId
//	if eventSubscriber, err = s.chainMakerServer.GetEventSubscribe(chainId); err != nil {
//		errCode = commonErr.ERR_CODE_GET_SUBSCRIBER
//		errMsg = s.getErrMsg(errCode, err)
//		s.log.Error(errMsg)
//		return status.Error(codes.Internal, errMsg)
//	}
//
//	sub := eventSubscriber.SubscribeBlockEvent(blockCh)
//	defer sub.Unsubscribe()
//
//	for {
//		select {
//		case ev := <-blockCh:
//			block = ev.BlockInfo.Block
//
//			if alreadySendHistoryBlockHeight != -1 && block.Header.BlockHeight > alreadySendHistoryBlockHeight {
//				err, _ = s.sendHistoryTx(store, server, alreadySendHistoryBlockHeight+1,
//					block.Header.BlockHeight, payload.TxType, payload.TxIds, txIdsMap)
//				if err != nil {
//					s.log.Errorf("send history block failed, %s", err)
//					return err
//				}
//
//				alreadySendHistoryBlockHeight = -1
//				continue
//			}
//
//			if err := s.sendSubscribeTx(server, block.Txs, payload.TxType, payload.TxIds, txIdsMap); err != nil {
//				errMsg = fmt.Sprintf("send subscribe tx failed, %s", err)
//				s.log.Error(errMsg)
//				return status.Error(codes.Internal, errMsg)
//			}
//
//			if s.checkIsFinish(payload, txIdsMap, ev.BlockInfo) {
//				return status.Error(codes.OK, "OK")
//			}
//
//		case <-server.Context().Done():
//			return nil
//		case <-s.ctx.Done():
//			return nil
//		}
//	}
//}
//
//func (s *ApiService) checkIsFinish(payload commonPb.SubscribeTxPayload,
//	txIdsMap map[string]struct{}, blockInfo *commonPb.BlockInfo) bool {
//
//	if len(payload.TxIds) > 0 && len(txIdsMap) == 0 {
//		return true
//	}
//
//	if payload.EndBlock != -1 && blockInfo.Block.Header.BlockHeight >= payload.EndBlock {
//		return true
//	}
//
//	return false
//}
//
//func (s *ApiService) getRateLimitToken() error {
//	if s.subscriberRateLimiter != nil {
//		if err := s.subscriberRateLimiter.Wait(s.ctx); err != nil {
//			errMsg := fmt.Sprintf("subscriber rateLimiter wait token failed, %s", err.Error())
//			s.log.Error(errMsg)
//			return errors.New(errMsg)
//		}
//	}
//
//	return nil
//}
//
//// sendHistoryBlock - send history block to subscriber
//func (s *ApiService) sendHistoryBlock(store protocol.BlockchainStore, server apiPb.RpcNode_SubscribeServer,
//	startBlockHeight, endBlockHeight int64, withRWSet bool) (error, int64) {
//
//	var (
//		err    error
//		errMsg string
//		result *commonPb.SubscribeResult
//	)
//
//	i := startBlockHeight
//	for {
//		select {
//		case <-s.ctx.Done():
//			return status.Error(codes.Internal, "chainmaker is restarting, please retry later"), -1
//		default:
//			if err = s.getRateLimitToken(); err != nil {
//				return status.Error(codes.Internal, err.Error()), -1
//			}
//
//			if endBlockHeight != -1 && i > endBlockHeight {
//				return nil, i - 1
//			}
//
//			blockInfo, alreadySendHistoryBlockHeight, err := s.getBlockInfoFromStore(store, i, withRWSet)
//			if err != nil {
//				return status.Error(codes.Internal, errMsg), -1
//			}
//
//			if blockInfo == nil || alreadySendHistoryBlockHeight > 0 {
//				return nil, alreadySendHistoryBlockHeight
//			}
//
//			if result, err = s.getBlockSubscribeResult(blockInfo); err != nil {
//				errMsg = fmt.Sprintf("get block subscribe result failed, %s", err)
//				s.log.Error(errMsg)
//				return errors.New(errMsg), -1
//			}
//
//			if err := server.Send(result); err != nil {
//				errMsg = fmt.Sprintf("send block info by history failed, %s", err)
//				s.log.Error(errMsg)
//				return status.Error(codes.Internal, errMsg), -1
//			}
//
//			i++
//		}
//	}
//}
//
//func (s *ApiService) getBlockInfoFromStore(store protocol.BlockchainStore, curblockHeight int64, withRWSet bool) (
//	blockInfo *commonPb.BlockInfo, alreadySendHistoryBlockHeight int64, err error) {
//	var (
//		errMsg         string
//		block          *commonPb.Block
//		blockWithRWSet *storePb.BlockWithRWSet
//	)
//
//	if withRWSet {
//		blockWithRWSet, err = store.GetBlockWithRWSets(curblockHeight)
//	} else {
//		block, err = store.GetBlock(curblockHeight)
//	}
//
//	if err != nil {
//		if withRWSet {
//			errMsg = fmt.Sprintf("get block with rwset failed, at [height:%d], %s", curblockHeight, err)
//		} else {
//			errMsg = fmt.Sprintf("get block failed, at [height:%d], %s", curblockHeight, err)
//		}
//		s.log.Error(errMsg)
//		return nil, -1, errors.New(errMsg)
//	}
//
//	if withRWSet {
//		if blockWithRWSet == nil {
//			return nil, curblockHeight - 1, nil
//		}
//
//		blockInfo = &commonPb.BlockInfo{
//			Block:     blockWithRWSet.Block,
//			RwsetList: blockWithRWSet.TxRWSets,
//		}
//	} else {
//		if block == nil {
//			return nil, curblockHeight - 1, nil
//		}
//
//		blockInfo = &commonPb.BlockInfo{
//			Block:     block,
//			RwsetList: nil,
//		}
//	}
//
//	return blockInfo, -1, nil
//}
//
//// sendHistoryTx - send history tx to subscriber
//func (s *ApiService) sendHistoryTx(store protocol.BlockchainStore,
//	server apiPb.RpcNode_SubscribeServer,
//	startBlockHeight, endBlockHeight int64,
//	txType commonPb.TxType, txIds []string, txIdsMap map[string]struct{}) (error, int64) {
//
//	var (
//		err    error
//		errMsg string
//		block  *commonPb.Block
//	)
//
//	i := startBlockHeight
//	for {
//		select {
//		case <-s.ctx.Done():
//			return status.Error(codes.Internal, "chainmaker is restarting, please retry later"), -1
//		default:
//			if err = s.getRateLimitToken(); err != nil {
//				return status.Error(codes.Internal, err.Error()), -1
//			}
//
//			if endBlockHeight != -1 && i > endBlockHeight {
//				return nil, i - 1
//			}
//
//			if len(txIds) > 0 && len(txIdsMap) == 0 {
//				return nil, i - 1
//			}
//
//			block, err = store.GetBlock(i)
//
//			if err != nil {
//				errMsg = fmt.Sprintf("get block failed, at [height:%d], %s", i, err)
//				s.log.Error(errMsg)
//				return status.Error(codes.Internal, errMsg), -1
//			}
//
//			if block == nil {
//				return nil, i - 1
//			}
//
//			if err := s.sendSubscribeTx(server, block.Txs, txType, txIds, txIdsMap); err != nil {
//				errMsg = fmt.Sprintf("send subscribe tx failed, %s", err)
//				s.log.Error(errMsg)
//				return status.Error(codes.Internal, errMsg), -1
//			}
//
//			i++
//		}
//	}
//}
//
//// checkSubscribePayload - check subscriber payload info
//func (s *ApiService) checkSubscribePayload(startBlockHeight, endBlockHeight int64) error {
//	if startBlockHeight < -1 || endBlockHeight < -1 ||
//		(endBlockHeight != -1 && startBlockHeight > endBlockHeight) {
//
//		return errors.New("invalid start block height or end block height")
//	}
//
//	return nil
//}
//
//func (s *ApiService) getTxSubscribeResult(tx *commonPb.Transaction) (*commonPb.SubscribeResult, error) {
//	txBytes, err := proto.Marshal(tx)
//	if err != nil {
//		errMsg := fmt.Sprintf("marshal tx info failed, %s", err)
//		s.log.Error(errMsg)
//		return nil, errors.New(errMsg)
//	}
//
//	result := &commonPb.SubscribeResult{
//		Data: txBytes,
//	}
//
//	return result, nil
//}
//
//func (s *ApiService) getBlockSubscribeResult(blockInfo *commonPb.BlockInfo) (*commonPb.SubscribeResult, error) {
//
//	blockBytes, err := proto.Marshal(blockInfo)
//	if err != nil {
//		errMsg := fmt.Sprintf("marshal block info failed, %s", err)
//		s.log.Error(errMsg)
//		return nil, errors.New(errMsg)
//	}
//
//	result := &commonPb.SubscribeResult{
//		Data: blockBytes,
//	}
//
//	return result, nil
//}
//
//func (s *ApiService) getContractEventSubscribeResult(contractEventsInfoList *commonPb.ContractEventInfoList) (*commonPb.SubscribeResult, error) {
//
//	eventBytes, err := proto.Marshal(contractEventsInfoList)
//	if err != nil {
//		errMsg := fmt.Sprintf("marshal contract event info failed, %s", err)
//		s.log.Error(errMsg)
//		return nil, errors.New(errMsg)
//	}
//
//	result := &commonPb.SubscribeResult{
//		Data: eventBytes,
//	}
//
//	return result, nil
//}
//func (s *ApiService) sendSubscribeTx(server apiPb.RpcNode_SubscribeServer,
//	txs []*commonPb.Transaction, txType commonPb.TxType, txIds []string, txIdsMap map[string]struct{}) error {
//
//	var (
//		err error
//	)
//
//	for _, tx := range txs {
//		if txType == -1 && len(txIds) == 0 {
//			if err = s.doSendSubscribeTx(server, tx); err != nil {
//				return err
//			}
//			continue
//		}
//
//		if s.checkIsContinue(tx, txType, txIds, txIdsMap) {
//			continue
//		}
//
//		if err = s.doSendSubscribeTx(server, tx); err != nil {
//			return err
//		}
//	}
//
//	return nil
//}
//
//func (s *ApiService) checkIsContinue(tx *commonPb.Transaction, txType commonPb.TxType, txIds []string, txIdsMap map[string]struct{}) bool {
//	if txType != -1 && tx.Payload.TxType != txType {
//		return true
//	}
//
//	if len(txIds) > 0 {
//		_, ok := txIdsMap[tx.Payload.TxId]
//		if !ok {
//			return true
//		}
//
//		delete(txIdsMap, tx.Payload.TxId)
//	}
//
//	return false
//}
//
//func (s *ApiService) doSendSubscribeTx(server apiPb.RpcNode_SubscribeServer, tx *commonPb.Transaction) error {
//	var (
//		err    error
//		errMsg string
//		result *commonPb.SubscribeResult
//	)
//
//	if result, err = s.getTxSubscribeResult(tx); err != nil {
//		errMsg = fmt.Sprintf("get tx subscribe result failed, %s", err)
//		s.log.Error(errMsg)
//		return errors.New(errMsg)
//	}
//
//	if err := server.Send(result); err != nil {
//		errMsg = fmt.Sprintf("send subscribe tx result failed, %s", err)
//		s.log.Error(errMsg)
//		return errors.New(errMsg)
//	}
//
//	return nil
//}
//
//func (s *ApiService) checkAndGetLastBlockHeight(store protocol.BlockchainStore,
//	payloadStartBlockHeight int64) (int64, error) {
//
//	var (
//		err       error
//		errMsg    string
//		errCode   commonErr.ErrCode
//		lastBlock *commonPb.Block
//	)
//
//	if lastBlock, err = store.GetLastBlock(); err != nil {
//		errCode = commonErr.ERR_CODE_GET_LAST_BLOCK
//		errMsg = s.getErrMsg(errCode, err)
//		s.log.Error(errMsg)
//		return -1, status.Error(codes.Internal, errMsg)
//	}
//
//	if lastBlock.Header.BlockHeight < payloadStartBlockHeight {
//		errMsg = fmt.Sprintf("payload start block height > last block height")
//		s.log.Error(errMsg)
//		return -1, status.Error(codes.InvalidArgument, errMsg)
//	}
//
//	return lastBlock.Header.BlockHeight, nil
//}
