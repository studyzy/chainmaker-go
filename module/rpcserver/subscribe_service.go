/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package rpcserver

import (
    "chainmaker.org/chainmaker-go/utils"
    "chainmaker.org/chainmaker/pb-go/syscontract"
    "errors"
    "fmt"
    "strings"

    "chainmaker.org/chainmaker-go/subscriber"
    "chainmaker.org/chainmaker-go/subscriber/model"
    commonErr "chainmaker.org/chainmaker/common/errors"
    apiPb "chainmaker.org/chainmaker/pb-go/api"
    commonPb "chainmaker.org/chainmaker/pb-go/common"
    storePb "chainmaker.org/chainmaker/pb-go/store"
    "chainmaker.org/chainmaker/protocol"
    "github.com/gogo/protobuf/proto"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
)

// Subscribe - deal block/tx subscribe request
func (s *ApiService) Subscribe(req *commonPb.TxRequest, server apiPb.RpcNode_SubscribeServer) error {
    var (
        errCode commonErr.ErrCode
        errMsg  string
    )

    tx := &commonPb.Transaction{
        Payload:    req.Payload,
        Sender:     req.Sender,
        Endorsers:  req.Endorsers,
        Result:     nil}

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
    )

    for _, kv := range payload.Parameters {
        if kv.Key == syscontract.SubscribeBlock_START_BLOCK.String() {
            startBlock, err = utils.BytesToInt64(kv.Value)
        } else if kv.Key == syscontract.SubscribeBlock_END_BLOCK.String() {
            endBlock, err = utils.BytesToInt64(kv.Value)
        } else if kv.Key == syscontract.SubscribeBlock_WITH_RWSET.String() {
            if string(kv.Value) == "true" {
                withRWSet = true
            }
        } else if kv.Key == syscontract.SubscribeBlock_ONLY_HEADER.String() {
            if string(kv.Value) == "true" {
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

    var startBlockHeight int64
    if startBlock > startBlockHeight {
        startBlockHeight = startBlock
    }

    if startBlock == -1 && endBlock == -1 {
        return s.sendNewBlock(db, tx, server, endBlock, withRWSet, onlyHeader, -1)
    }

    if endBlock != -1 && endBlock <= lastBlockHeight {
        err, _ := s.sendHistoryBlock(db, server, startBlockHeight, endBlock, withRWSet, onlyHeader)
        if err != nil {
            s.log.Errorf("sendHistoryBlock failed, %s", err)
            return err
        }

        return status.Error(codes.OK, "OK")
    }

    err, alreadySendHistoryBlockHeight := s.sendHistoryBlock(db, server, startBlockHeight, endBlock, withRWSet, onlyHeader)
    if err != nil {
        s.log.Errorf("sendHistoryBlock failed, %s", err)
        return err
    }

    s.log.Debugf("after sendHistoryBlock, alreadySendHistoryBlockHeight is %d", alreadySendHistoryBlockHeight)

    return s.sendNewBlock(db, tx, server, endBlock, withRWSet, onlyHeader, alreadySendHistoryBlockHeight)
}

// dealTxSubscription - deal tx subscribe request
func (s *ApiService) dealTxSubscription(tx *commonPb.Transaction, server apiPb.RpcNode_SubscribeServer) error {
    var (
        err             error
        errMsg          string
        errCode         commonErr.ErrCode
        db              protocol.BlockchainStore
        payload         = tx.Payload
        startBlock      int64
        endBlock        int64
        contractName    string
        txIds           []string
    )

    for _, kv := range payload.Parameters {
        if kv.Key == syscontract.SubscribeTx_START_BLOCK.String() {
            startBlock, err = utils.BytesToInt64(kv.Value)
        } else if kv.Key == syscontract.SubscribeTx_END_BLOCK.String() {
            endBlock, err = utils.BytesToInt64(kv.Value)
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

    return s.doSendTx(tx, db, server, startBlock, endBlock, contractName, txIds)
}

//dealContractEventSubscription - deal contract event subscribe request
func (s *ApiService) dealContractEventSubscription(tx *commonPb.Transaction, server apiPb.RpcNode_SubscribeServer) error {
    var (
        err             error
        errMsg          string
        errCode         commonErr.ErrCode
        payload         = tx.Payload
        topic           string
        contractName    string
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

func (s *ApiService) doSendContractEvent(tx *commonPb.Transaction, server apiPb.RpcNode_SubscribeServer, topic, contractName string) error {

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
    server apiPb.RpcNode_SubscribeServer, startBlock, endBlock int64, contractName string, txIds []string) error {

    var (
        txIdsMap                      = make(map[string]struct{})
        alreadySendHistoryBlockHeight int64
        err                           error
    )

    for _, txId := range txIds {
        txIdsMap[txId] = struct{}{}
    }

    if startBlock == -1 && endBlock == -1 {
        return s.sendNewTx(db, tx, server, startBlock, endBlock, contractName, txIds, txIdsMap, -1)
    }

    if alreadySendHistoryBlockHeight, err = s.doSendHistoryTx(db, server, startBlock, endBlock, contractName, txIds, txIdsMap); err != nil {
        return err
    }

    if alreadySendHistoryBlockHeight == 0 {
        return status.Error(codes.OK, "OK")
    }

    return s.sendNewTx(db, tx, server, startBlock, endBlock, contractName, txIds, txIdsMap, alreadySendHistoryBlockHeight)
}

func (s *ApiService) doSendHistoryTx(db protocol.BlockchainStore, server apiPb.RpcNode_SubscribeServer,
    startBlock, endBlock int64, contractName string, txIds []string, txIdsMap map[string]struct{}) (int64, error) {

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
        err, _ := s.sendHistoryTx(db, server, startBlockHeight, endBlock, contractName, txIds, txIdsMap)
        if err != nil {
            s.log.Errorf("sendHistoryTx failed, %s", err)
            return -1, err
        }

        return 0, status.Error(codes.OK, "OK")
    }

    if len(txIds) > 0 && len(txIdsMap) == 0 {
        return 0, status.Error(codes.OK, "OK")
    }

    err, alreadySendHistoryBlockHeight := s.sendHistoryTx(db, server, startBlockHeight, endBlock, contractName, txIds, txIdsMap)
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
    endBlockHeight int64, withRWSet, onlyHeader bool, alreadySendHistoryBlockHeight int64) error {

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
                err, _ = s.sendHistoryBlock(store, server, alreadySendHistoryBlockHeight+1,
                    int64(blockInfo.Block.Header.BlockHeight), withRWSet, onlyHeader)
                if err != nil {
                    s.log.Errorf("send history block failed, %s", err)
                    return err
                }

                alreadySendHistoryBlockHeight = -1
                continue
            }

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
        err         error
        result      *commonPb.SubscribeResult
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
    server apiPb.RpcNode_SubscribeServer, startBlock, endBlock int64, contractName string, txIds []string,
    txIdsMap map[string]struct{}, alreadySendHistoryBlockHeight int64) error {

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
                err, _ = s.sendHistoryTx(store, server, alreadySendHistoryBlockHeight+1,
                    int64(block.Header.BlockHeight), contractName, txIds, txIdsMap)
                if err != nil {
                    s.log.Errorf("send history block failed, %s", err)
                    return err
                }

                alreadySendHistoryBlockHeight = -1
                continue
            }

            if err := s.sendSubscribeTx(server, block.Txs, contractName, txIds, txIdsMap); err != nil {
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
    startBlockHeight, endBlockHeight int64, withRWSet, onlyHeader bool) (error, int64) {

    var (
        err    error
        errMsg string
        result *commonPb.SubscribeResult
    )

    i := startBlockHeight
    for {
        select {
        case <-s.ctx.Done():
            return status.Error(codes.Internal, "chainmaker is restarting, please retry later"), -1
        default:
            if err = s.getRateLimitToken(); err != nil {
                return status.Error(codes.Internal, err.Error()), -1
            }

            if endBlockHeight != -1 && i > endBlockHeight {
                return nil, i - 1
            }

            blockInfo, alreadySendHistoryBlockHeight, err := s.getBlockInfoFromStore(store, i, withRWSet)
            if err != nil {
                return status.Error(codes.Internal, errMsg), -1
            }

            if blockInfo == nil || alreadySendHistoryBlockHeight > 0 {
                return nil, alreadySendHistoryBlockHeight
            }

            if result, err = s.getBlockSubscribeResult(blockInfo, onlyHeader); err != nil {
                errMsg = fmt.Sprintf("get block subscribe result failed, %s", err)
                s.log.Error(errMsg)
                return errors.New(errMsg), -1
            }

            if err := server.Send(result); err != nil {
                errMsg = fmt.Sprintf("send block info by history failed, %s", err)
                s.log.Error(errMsg)
                return status.Error(codes.Internal, errMsg), -1
            }

            i++
        }
    }
}

func (s *ApiService) getBlockInfoFromStore(store protocol.BlockchainStore, curblockHeight int64, withRWSet bool) (
    blockInfo *commonPb.BlockInfo, alreadySendHistoryBlockHeight int64, err error) {
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
    } else {
        if block == nil {
            return nil, curblockHeight - 1, nil
        }

        blockInfo = &commonPb.BlockInfo{
            Block:     block,
            RwsetList: nil,
        }
    }

    return blockInfo, -1, nil
}

// sendHistoryTx - send history tx to subscriber
func (s *ApiService) sendHistoryTx(store protocol.BlockchainStore,
    server apiPb.RpcNode_SubscribeServer,
    startBlockHeight, endBlockHeight int64,
    contractName string, txIds []string, txIdsMap map[string]struct{}) (error, int64) {

    var (
        err    error
        errMsg string
        block  *commonPb.Block
    )

    i := startBlockHeight
    for {
        select {
        case <-s.ctx.Done():
            return status.Error(codes.Internal, "chainmaker is restarting, please retry later"), -1
        default:
            if err = s.getRateLimitToken(); err != nil {
                return status.Error(codes.Internal, err.Error()), -1
            }

            if endBlockHeight != -1 && i > endBlockHeight {
                return nil, i - 1
            }

            if len(txIds) > 0 && len(txIdsMap) == 0 {
                return nil, i - 1
            }

            block, err = store.GetBlock(uint64(i))

            if err != nil {
                errMsg = fmt.Sprintf("get block failed, at [height:%d], %s", i, err)
                s.log.Error(errMsg)
                return status.Error(codes.Internal, errMsg), -1
            }

            if block == nil {
                return nil, i - 1
            }

            if err := s.sendSubscribeTx(server, block.Txs, contractName, txIds, txIdsMap); err != nil {
                errMsg = fmt.Sprintf("send subscribe tx failed, %s", err)
                s.log.Error(errMsg)
                return status.Error(codes.Internal, errMsg), -1
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

func (s *ApiService) getBlockSubscribeResult(blockInfo *commonPb.BlockInfo, onlyHeader bool) (*commonPb.SubscribeResult, error) {
    var (
        resultBytes []byte
        err error
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

func (s *ApiService) getContractEventSubscribeResult(contractEventsInfoList *commonPb.ContractEventInfoList) (*commonPb.SubscribeResult, error) {

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
    txs []*commonPb.Transaction, contractName string, txIds []string, txIdsMap map[string]struct{}) error {

    var (
        err error
    )

    for _, tx := range txs {
        if contractName == "" && len(txIds) == 0 {
            if err = s.doSendSubscribeTx(server, tx); err != nil {
                return err
            }
            continue
        }

        if s.checkIsContinue(tx, contractName, txIds, txIdsMap) {
            continue
        }

        if err = s.doSendSubscribeTx(server, tx); err != nil {
            return err
        }
    }

    return nil
}

func (s *ApiService) checkIsContinue(tx *commonPb.Transaction, contractName string, txIds []string, txIdsMap map[string]struct{}) bool {
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

func (s *ApiService) doSendSubscribeTx(server apiPb.RpcNode_SubscribeServer, tx *commonPb.Transaction) error {
    var (
        err    error
        errMsg string
        result *commonPb.SubscribeResult
    )

    if result, err = s.getTxSubscribeResult(tx); err != nil {
        errMsg = fmt.Sprintf("get tx subscribe result failed, %s", err)
        s.log.Error(errMsg)
        return errors.New(errMsg)
    }

    if err := server.Send(result); err != nil {
        errMsg = fmt.Sprintf("send subscribe tx result failed, %s", err)
        s.log.Error(errMsg)
        return errors.New(errMsg)
    }

    return nil
}

func (s *ApiService) checkAndGetLastBlockHeight(store protocol.BlockchainStore,
    payloadStartBlockHeight int64) (int64, error) {

    var (
        err       error
        errMsg    string
        errCode   commonErr.ErrCode
        lastBlock *commonPb.Block
    )

    if lastBlock, err = store.GetLastBlock(); err != nil {
        errCode = commonErr.ERR_CODE_GET_LAST_BLOCK
        errMsg = s.getErrMsg(errCode, err)
        s.log.Error(errMsg)
        return -1, status.Error(codes.Internal, errMsg)
    }

    if int64(lastBlock.Header.BlockHeight) < payloadStartBlockHeight {
        errMsg = fmt.Sprintf("payload start block height > last block height")
        s.log.Error(errMsg)
        return -1, status.Error(codes.InvalidArgument, errMsg)
    }

    return int64(lastBlock.Header.BlockHeight), nil
}