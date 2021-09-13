/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package proposer

//
//import (
//	"chainmaker.org/chainmaker/common/v2/random/uuid"
//	"chainmaker.org/chainmaker-go/core/cache"
//	"chainmaker.org/chainmaker/localconf/v2"
//	"chainmaker.org/chainmaker/logger/v2"
//	"chainmaker.org/chainmaker/protocol/v2/mock"
//	acpb "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
//	commonpb "chainmaker.org/chainmaker/pb-go/v2/common"
//	configpb "chainmaker.org/chainmaker/pb-go/v2/config"
//	"chainmaker.org/chainmaker/pb-go/v2/consensus"
//	txpoolpb "chainmaker.org/chainmaker/pb-go/v2/txpool"
//	"chainmaker.org/chainmaker/utils/v2"
//	"crypto/sha256"
//	"fmt"
//	gogo "github.com/gogo/protobuf/proto"
//	"github.com/golang/mock/gomock"
//	"github.com/stretchr/testify/require"
//	"testing"
//	"time"
//)
//
//var (
//	chainId      = "Chain1"
//	contractName = "contractName"
//)
//
//func TestProposeStatusChange(t *testing.T) {
//	ctl := gomock.NewController(t)
//	txPool := mock.NewMockTxPool(ctl)
//	snapshotMgr := mock.NewMockSnapshotManager(ctl)
//	msgBus := mock.NewMockMessageBus(ctl)
//	identity := mock.NewMockSigningMember(ctl)
//	ledgerCache := cache.NewLedgerCache(chainId)
//	//consensus := mock.NewMockConsensusEngine(ctl)
//	proposedCache := cache.NewProposalCache(nil, ledgerCache)
//	txScheduler := mock.NewMockTxScheduler(ctl)
//	blockChainStore := mock.NewMockBlockchainStore(ctl)
//	chainConf := mock.NewMockChainConf(ctl)
//
//	ledgerCache.SetLastCommittedBlock(createNewTestBlock(0))
//
//	txs := make([]*commonpb.Transaction, 0)
//
//	for i := 0; i < 5; i++ {
//		txs = append(txs, createNewTestTx())
//	}
//
//	txPool.EXPECT().FetchTxBatch(gomock.Any()).Return(txs).Times(10)
//	txPool.EXPECT().RetryAndRemoveTxs(gomock.Any(), gomock.Any())
//	identity.EXPECT().Serialize(true).Return([]byte("0123456789"), nil).Times(10)
//	msgBus.EXPECT().Publish(gomock.Any(), gomock.Any())
//	blockChainStore.EXPECT().TxExists("").AnyTimes()
//	consensus := configpb.ConsensusConfig{
//		Type: consensus.ConsensusType_TBFT,
//	}
//	block := configpb.BlockConfig{
//		TxTimestampVerify: false,
//		TxTimeout:         1000000000,
//		BlockTxCapacity:   100,
//		BlockSize:        100000,
//		BlockInterval:     1000,
//	}
//	crypro := configpb.CryptoConfig{Hash: "SHA256"}
//	contract := configpb.ContractConfig{EnableSqlSupport: false}
//	chainConfig := configpb.ChainConfig{Consensus: &consensus, Block: &block, Contract: &contract, Crypto: &crypro}
//	chainConf.EXPECT().ChainConfig().Return(&chainConfig).AnyTimes()
//
//	snapshotMgr.EXPECT().NewSnapshot(gomock.Any(), gomock.Any()).AnyTimes()
//	txScheduler.EXPECT().Schedule(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
//
//	blockProposer := &BlockProposerImpl{
//		chainId:         chainId,
//		isProposer:      false, // not proposer when initialized
//		idle:            true,
//		msgBus:          msgBus,
//		canProposeC:     make(chan bool),
//		txPoolSignalC:   make(chan *txpoolpb.TxPoolSignal),
//		proposeTimer:    nil,
//		exitC:           make(chan bool),
//		txPool:          txPool,
//		snapshotManager: snapshotMgr,
//		txScheduler:     txScheduler,
//		identity:        identity,
//		ledgerCache:     ledgerCache,
//		proposalCache:   proposedCache,
//		log:             logger.GetLoggerByChain(logger.MODULE_CORE, chainId),
//		finishProposeC:  make(chan bool),
//		blockchainStore: blockChainStore,
//		chainConf:       chainConf,
//	}
//	require.False(t, blockProposer.isProposer)
//	require.Nil(t, blockProposer.proposeTimer)
//
//	blockProposer.proposeBlock()
//	blockProposer.OnReceiveYieldProposeSignal(true)
//
//}
//
//func TestShouldPropose(t *testing.T) {
//	ctl := gomock.NewController(t)
//	txPool := mock.NewMockTxPool(ctl)
//	snapshotMgr := mock.NewMockSnapshotManager(ctl)
//	msgBus := mock.NewMockMessageBus(ctl)
//	identity := mock.NewMockSigningMember(ctl)
//	ledgerCache := cache.NewLedgerCache(chainId)
//	proposedCache := cache.NewProposalCache(nil, ledgerCache)
//	txScheduler := mock.NewMockTxScheduler(ctl)
//
//	ledgerCache.SetLastCommittedBlock(createNewTestBlock(0))
//	blockProposer := &BlockProposerImpl{
//		chainId:         chainId,
//		isProposer:      false, // not proposer when initialized
//		idle:            true,
//		msgBus:          msgBus,
//		canProposeC:     make(chan bool),
//		txPoolSignalC:   make(chan *txpoolpb.TxPoolSignal),
//		proposeTimer:    nil,
//		exitC:           make(chan bool),
//		txPool:          txPool,
//		snapshotManager: snapshotMgr,
//		txScheduler:     txScheduler,
//		identity:        identity,
//		ledgerCache:     ledgerCache,
//		proposalCache:   proposedCache,
//		log:             logger.GetLoggerByChain(logger.MODULE_CORE, chainId),
//	}
//
//	b0 := createNewTestBlock(0)
//	ledgerCache.SetLastCommittedBlock(b0)
//	require.True(t, blockProposer.shouldProposeByBFT(b0.Header.BlockHeight+1))
//
//	b := createNewTestBlock(1)
//	proposedCache.SetProposedBlock(b, nil, nil, false)
//	require.Nil(t, proposedCache.GetSelfProposedBlockAt(1))
//	b1, _, _ := proposedCache.GetProposedBlock(b)
//	require.NotNil(t, b1)
//
//	b2 := createNewTestBlock(1)
//	b2.Header.BlockHash = nil
//	proposedCache.SetProposedBlock(b2, nil, nil, true)
//	require.False(t, blockProposer.shouldProposeByBFT(b2.Header.BlockHeight))
//	require.NotNil(t, proposedCache.GetSelfProposedBlockAt(1))
//	ledgerCache.SetLastCommittedBlock(b2)
//	require.True(t, blockProposer.shouldProposeByBFT(b2.Header.BlockHeight+1))
//
//	b3, _, _ := proposedCache.GetProposedBlock(b2)
//	require.NotNil(t, b3)
//
//	proposedCache.SetProposedAt(b3.Header.BlockHeight)
//	require.False(t, blockProposer.shouldProposeByBFT(b3.Header.BlockHeight))
//
//}
//
//func TestShouldProposeChainedBFT(t *testing.T) {
//	ctl := gomock.NewController(t)
//	txPool := mock.NewMockTxPool(ctl)
//	snapshotMgr := mock.NewMockSnapshotManager(ctl)
//	msgBus := mock.NewMockMessageBus(ctl)
//	identity := mock.NewMockSigningMember(ctl)
//	ledgerCache := cache.NewLedgerCache(chainId)
//	proposedCache := cache.NewProposalCache(nil, ledgerCache)
//	txScheduler := mock.NewMockTxScheduler(ctl)
//
//	ledgerCache.SetLastCommittedBlock(createNewTestBlock(0))
//	blockProposer := &BlockProposerImpl{
//		chainId:         chainId,
//		isProposer:      false, // not proposer when initialized
//		idle:            true,
//		msgBus:          msgBus,
//		canProposeC:     make(chan bool),
//		txPoolSignalC:   make(chan *txpoolpb.TxPoolSignal),
//		proposeTimer:    nil,
//		exitC:           make(chan bool),
//		txPool:          txPool,
//		snapshotManager: snapshotMgr,
//		txScheduler:     txScheduler,
//		identity:        identity,
//		ledgerCache:     ledgerCache,
//		proposalCache:   proposedCache,
//		log:             logger.GetLoggerByChain(logger.MODULE_CORE, chainId),
//	}
//
//	b0 := createNewTestBlock(0)
//	ledgerCache.SetLastCommittedBlock(b0)
//	require.True(t, blockProposer.shouldProposeByChainedBFT(b0.Header.BlockHeight+1, b0.Header.BlockHash))
//	require.False(t, blockProposer.shouldProposeByChainedBFT(b0.Header.BlockHeight+1, []byte("xyz")))
//	require.False(t, blockProposer.shouldProposeByChainedBFT(b0.Header.BlockHeight, b0.Header.PreBlockHash))
//
//	b := createNewTestBlock(1)
//	proposedCache.SetProposedBlock(b, nil, nil, false)
//	require.Nil(t, proposedCache.GetSelfProposedBlockAt(1))
//	b1, _, _ := proposedCache.GetProposedBlock(b)
//	require.NotNil(t, b1)
//
//	b2 := createNewTestBlock(1)
//	b2.Header.BlockHash = nil
//	proposedCache.SetProposedBlock(b2, nil, nil, true)
//	require.NotNil(t, proposedCache.GetSelfProposedBlockAt(1))
//	require.True(t, blockProposer.shouldProposeByChainedBFT(b2.Header.BlockHeight, b0.Header.BlockHash))
//
//	b3, _, _ := proposedCache.GetProposedBlock(b2)
//	require.NotNil(t, b3)
//
//}
//
//func TestYieldGoRountine(t *testing.T) {
//	exitC := make(chan bool)
//	go func() {
//		time.Sleep(3 * time.Second)
//		exitC <- true
//	}()
//
//	sig := <-exitC
//	require.True(t, sig)
//	fmt.Println("exit1")
//}
//
//func TestHash(t *testing.T) {
//	txCount := 50000
//	txs := make([][]byte, 0)
//	for i := 0; i < txCount; i++ {
//		txId := uuid.GetUUID() + uuid.GetUUID()
//		txs = append(txs, []byte(txId))
//	}
//	require.Equal(t, txCount, len(txs))
//	hf := sha256.New()
//
//	start := utils.CurrentTimeMillisSeconds()
//	for _, txId := range txs {
//		hf.Write(txId)
//		hf.Sum(nil)
//		hf.Reset()
//	}
//	fmt.Println(utils.CurrentTimeMillisSeconds() - start)
//}
//
//func TestFinalize(t *testing.T) {
//	txCount := 50000
//	dag := &commonpb.DAG{Vertexes: make([]*commonpb.DAG_Neighbor, txCount)}
//	txRead := &commonpb.TxRead{
//		Key:          []byte("key"),
//		Value:        []byte("value"),
//		ContractName: contractName,
//		Version:      nil,
//	}
//	txReads := make([]*commonpb.TxRead, 5)
//	for i := 0; i < 5; i++ {
//		txReads[i] = txRead
//	}
//	block := &commonpb.Block{
//		Header: &commonpb.BlockHeader{
//			ChainId:        "chain1",
//			BlockHeight:    0,
//			PreBlockHash:   nil,
//			BlockHash:      nil,
//			PreConfHeight:  0,
//			BlockVersion:   []byte("v1.0.0"),
//			DagHash:        nil,
//			RwSetRoot:      nil,
//			TxRoot:         nil,
//			BlockTimestamp: 0,
//			Proposer:       []byte("proposer"),
//			ConsensusArgs:  nil,
//			TxCount:        int64(txCount),
//			Signature:      nil,
//		},
//		Dag:            nil,
//		Txs:            nil,
//		AdditionalData: nil,
//	}
//	txs := make([]*commonpb.Transaction, 0)
//	rwSetMap := make(map[string]*commonpb.TxRWSet)
//	for i := 0; i < txCount; i++ {
//		dag.Vertexes[i] = &commonpb.DAG_Neighbor{
//			Neighbors: nil,
//		}
//		txId := uuid.GetUUID() + uuid.GetUUID()
//		payload := parsePayload(txId)
//		payloadBytes, _ := gogo.Marshal(payload)
//		tx := parseTx(txId, payloadBytes)
//		txs = append(txs, tx)
//		txWrite := &commonpb.TxWrite{
//			Key:          []byte(txId),
//			Value:        payloadBytes,
//			ContractName: contractName,
//		}
//		txWrites := make([]*commonpb.TxWrite, 0)
//		txWrites = append(txWrites, txWrite)
//		rwSetMap[txId] = &commonpb.TxRWSet{
//			TxId:     txId,
//			TxReads:  txReads,
//			TxWrites: txWrites,
//		}
//	}
//	require.Equal(t, txCount, len(txs))
//	block.Txs = txs
//	block.Dag = dag
//	confKV := &commonpb.KeyValuePair{
//		Key:   "IsExtreme",
//		Value: []byte("true"),
//	}
//	kvs := make([]*commonpb.KeyValuePair, 1)
//	kvs[0] = confKV
//	localconf.UpdateDebugConfig(kvs)
//	for i := 0; i < 10; i++ {
//		timeUsed, err := finalizeBlockRoots()
//		if err != nil {
//			fmt.Println(err)
//			return
//		}
//		fmt.Println(fmt.Sprintf("%v", timeUsed))
//	}
//
//}
//
//func finalizeBlockRoots() (interface{}, interface{}) {
//	return nil, nil
//}
//
//func TestProto(t *testing.T) {
//	txs := parseTxs(20000)
//	require.Equal(t, 20000, len(txs))
//	startTick1 := utils.CurrentTimeMillisSeconds()
//	for _, tx := range txs {
//		_, err := gogo.Marshal(tx)
//		if err != nil {
//			return
//		}
//	}
//	fmt.Println(fmt.Sprintf("gogo.protobuf:%v", utils.CurrentTimeMillisSeconds()-startTick1))
//}
//
//func parseTxs(num int) []*commonpb.Transaction {
//	txs := make([]*commonpb.Transaction, 0)
//	for i := 0; i < num; i++ {
//		txId := uuid.GetUUID() + uuid.GetUUID()
//		payload := parsePayload(txId)
//		payloadBytes, _ := gogo.Marshal(payload)
//		txs = append(txs, parseTx(txId, payloadBytes))
//	}
//	return txs
//}
//
//func parsePayload(txId string) *commonpb.TransactPayload {
//	pairs := []*commonpb.KeyValuePair{
//		{
//			Key:   "file_hash",
//			Value: []byte(txId)[len(txId)/2:],
//		},
//	}
//	return &commonpb.TransactPayload{
//		ContractName: contractName,
//		Method:       "save",
//		Parameters:   pairs,
//	}
//}
//
//func parseTx(txId string, payloadBytes []byte) *commonpb.Transaction {
//	return &commonpb.Transaction{
//		Header: &commonpb.TxHeader{
//			ChainId: "chain1",
//			Sender: &acpb.Member{
//				OrgId:      "wx-org1.chainmaker.org",
//				MemberInfo: []byte("wx-org1.chainmaker.org"),
//				MemberType: acPb.MemberType_CERT_HASH,
//			},
//			TxType:         0,
//			TxId:           txId,
//			Timestamp:      0,
//			ExpirationTime: 0,
//		},
//		RequestPayload:   payloadBytes,
//		RequestSignature: []byte("proposer"),
//		Result: &commonpb.Result{
//			Code: 0,
//			ContractResult: &commonpb.ContractResult{
//				Code:    0,
//				Result:  payloadBytes,
//				Message: "SUCCESS",
//				GasUsed: 0,
//			},
//			RwSetHash: nil,
//		},
//	}
//}
//
//func createNewTestBlock(height uint64) *commonpb.Block {
//	var hash = []byte("0123456789")
//	var version = []byte("0")
//	var block = &commonpb.Block{
//		Header: &commonpb.BlockHeader{
//			ChainId:        "Chain1",
//			BlockHeight:    height,
//			PreBlockHash:   hash,
//			BlockHash:      hash,
//			PreConfHeight:  0,
//			BlockVersion:   version,
//			DagHash:        hash,
//			RwSetRoot:      hash,
//			TxRoot:         hash,
//			BlockTimestamp: 0,
//			Proposer:       hash,
//			ConsensusArgs:  nil,
//			TxCount:        1,
//			Signature:      []byte(""),
//		},
//		Dag: &commonpb.DAG{
//			Vertexes: nil,
//		},
//		Txs: nil,
//	}
//	tx := createNewTestTx()
//	txs := make([]*commonpb.Transaction, 1)
//	txs[0] = tx
//	block.Txs = txs
//	return block
//}
//
//func createNewTestTx() *commonpb.Transaction {
//	var hash = []byte("0123456789")
//	return &commonpb.Transaction{
//		Header: &commonpb.TxHeader{
//			ChainId:        "",
//			Sender:         nil,
//			TxType:         0,
//			TxId:           "",
//			Timestamp:      0,
//			ExpirationTime: 0,
//		},
//		RequestPayload:   hash,
//		RequestSignature: hash,
//		Result: &commonpb.Result{
//			Code:           commonpb.TxStatusCode_SUCCESS,
//			ContractResult: nil,
//			RwSetHash:      nil,
//		},
//	}
//}
