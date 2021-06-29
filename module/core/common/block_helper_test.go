/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"chainmaker.org/chainmaker-go/common/crypto/hash"
	"chainmaker.org/chainmaker-go/logger"
	commonpb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/utils"
	"fmt"
	"github.com/stretchr/testify/require"
	_ "net/http/pprof"
	"testing"
	"time"
)

//  statistic the time consuming of finalizeBlock between sync and async
// TxNum: 1000000; async:29718 ; sync: 36641
func TestFinalizeBlock_Async(t *testing.T) {

	log := logger.GetLogger("core")
	block := createBlock(10)
	txs := make([]*commonpb.Transaction, 0)
	txRWSetMap := make(map[string]*commonpb.TxRWSet)
	for i := 0; i < 1000; i++ {
		txId := "0x123456789" + fmt.Sprint(i)
		tx := createNewTestTx(txId)
		txs = append(txs, tx)
		txRWSetMap[txId] = &commonpb.TxRWSet{
			TxId:     txId,
			TxReads:  nil,
			TxWrites: nil,
		}
	}
	block.Txs = txs

	// 开启pprof
	//go func() {
	//	_ = http.ListenAndServe("localhost:8080", nil)
	//}()
	//
	//time.Sleep(time.Second * 5)
	var err error

	asyncTimeStart := CurrentTimeMillisSeconds()
	err = FinalizeBlock(block, txRWSetMap, nil, "SHA256", log)
	asyncTimeEnd := CurrentTimeMillisSeconds()
	require.Equal(t, nil, err)
	log.Infof("async mode cost:[%d]", asyncTimeEnd-asyncTimeStart)
	rwSetRoot := block.Header.RwSetRoot
	blockHash := block.Header.BlockHash
	dagHash := block.Header.DagHash

	syncTimeStart := CurrentTimeMillisSeconds()
	err = FinalizeBlockSync(block, txRWSetMap, nil, "SHA256", log)
	syncTimeEnd := CurrentTimeMillisSeconds()
	require.Equal(t, nil, err)
	log.Infof(fmt.Sprintf("sync mode cost:[%d]", syncTimeEnd-syncTimeStart))

	require.Equal(t, rwSetRoot, block.Header.RwSetRoot)
	require.Equal(t, blockHash, block.Header.BlockHash)
	require.Equal(t, dagHash, block.Header.DagHash)

	log.Infof(fmt.Sprintf("async mode cost:[%d], sync mode cost:[%d]", asyncTimeEnd-asyncTimeStart, syncTimeEnd-syncTimeStart))

}

func createBlock(height int64) *commonpb.Block {
	var hash = []byte("0123456789")
	var version = []byte("0")
	var block = &commonpb.Block{
		Header: &commonpb.BlockHeader{
			ChainId:        "Chain1",
			BlockHeight:    height,
			PreBlockHash:   hash,
			BlockHash:      hash,
			PreConfHeight:  0,
			BlockVersion:   version,
			DagHash:        hash,
			RwSetRoot:      hash,
			TxRoot:         hash,
			BlockTimestamp: 0,
			Proposer:       hash,
			ConsensusArgs:  nil,
			TxCount:        1,
			Signature:      []byte(""),
		},
		Dag: &commonpb.DAG{
			Vertexes: nil,
		},
		Txs: nil,
	}

	return block
}

func createNewTestTx(txID string) *commonpb.Transaction {
	var hash = []byte("0123456789")
	return &commonpb.Transaction{
		Header: &commonpb.TxHeader{
			ChainId:        "Chain1",
			Sender:         nil,
			TxType:         0,
			TxId:           txID,
			Timestamp:      CurrentTimeMillisSeconds(),
			ExpirationTime: 0,
		},
		RequestPayload:   hash,
		RequestSignature: hash,
		Result: &commonpb.Result{
			Code:           commonpb.TxStatusCode_SUCCESS,
			ContractResult: nil,
			RwSetHash:      nil,
		},
	}
}

func CurrentTimeMillisSeconds() int64 {
	return time.Now().UnixNano() / 1e6
}

// the sync way fo finalize block
func FinalizeBlockSync(
	block *commonpb.Block,
	txRWSetMap map[string]*commonpb.TxRWSet,
	aclFailTxs []*commonpb.Transaction,
	hashType string,
	logger protocol.Logger) error {

	if aclFailTxs != nil && len(aclFailTxs) > 0 {
		// append acl check failed txs to the end of block.Txs
		block.Txs = append(block.Txs, aclFailTxs...)
	}

	// TxCount contains acl verify failed txs and invoked contract txs
	txCount := len(block.Txs)
	block.Header.TxCount = int64(txCount)

	// TxRoot/RwSetRoot
	var err error
	txHashes := make([][]byte, txCount)
	for i, tx := range block.Txs {
		// finalize tx, put rwsethash into tx.Result
		rwSet := txRWSetMap[tx.Header.TxId]
		if rwSet == nil {
			rwSet = &commonpb.TxRWSet{
				TxId:     tx.Header.TxId,
				TxReads:  nil,
				TxWrites: nil,
			}
		}
		rwSetHash, err := utils.CalcRWSetHash(hashType, rwSet)
		logger.DebugDynamic(func() string {
			return fmt.Sprintf("CalcRWSetHash rwset: %+v ,hash: %x", rwSet, rwSetHash)
		})
		if err != nil {
			return err
		}
		if tx.Result == nil {
			// in case tx.Result is nil, avoid panic
			e := fmt.Errorf("tx(%s) result == nil", tx.Header.TxId)
			logger.Error(e.Error())
			return e
		}
		tx.Result.RwSetHash = rwSetHash
		// calculate complete tx hash, include tx.Header, tx.Payload, tx.Result
		txHash, err := utils.CalcTxHash(hashType, tx)
		if err != nil {
			return err
		}
		txHashes[i] = txHash
	}

	block.Header.TxRoot, err = hash.GetMerkleRoot(hashType, txHashes)
	if err != nil {
		logger.Warnf("get tx merkle root error %s", err)
		return err
	}
	block.Header.RwSetRoot, err = utils.CalcRWSetRoot(hashType, block.Txs)
	if err != nil {
		logger.Warnf("get rwset merkle root error %s", err)
		return err
	}

	// DagDigest
	dagHash, err := utils.CalcDagHash(hashType, block.Dag)
	if err != nil {
		logger.Warnf("get dag hash error %s", err)
		return err
	}
	block.Header.DagHash = dagHash

	return nil
}

//func TestAddBlock(t *testing.T) {
//	ctl := gomock.NewController(t)
//	blockchainStoreImpl := mock.NewMockBlockchainStore(ctl)
//	txPool := mock.NewMockTxPool(ctl)
//	snapshotManager := mock.NewMockSnapshotManager(ctl)
//	ledgerCache := cache.NewLedgerCache("Chain1")
//	chainConf := mock.NewMockChainConf(ctl)
//	proposedCache := cache.NewProposalCache(chainConf, ledgerCache)
//
//	lastBlock := createNewTestBlock(0)
//	ledgerCache.SetLastCommittedBlock(lastBlock)
//	rwSetMap := make(map[string]*commonpb.TxRWSet)
//	contractEventMap := make(map[string][]*commonpb.ContractEvent)
//	msgbus := mock.NewMockMessageBus(ctl)
//	msgbus.EXPECT().Publish(gomock.Any(), gomock.Any()).Return().Times(2)
//
//	blockCommitterImpl := initCommitter(blockchainStoreImpl, txPool, snapshotManager, ledgerCache, proposedCache, chainConf, msgbus)
//	require.NotNil(t, blockCommitterImpl)
//
//	crypto := configpb.CryptoConfig{
//		Hash: "SHA256",
//	}
//	contractConf := configpb.ContractConfig{EnableSqlSupport: false}
//	chainConfig := configpb.ChainConfig{Crypto: &crypto, Contract: &contractConf}
//	chainConf.EXPECT().ChainConfig().Return(&chainConfig).Times(3)
//
//	block := createNewBlock(lastBlock)
//	proposedCache.SetProposedBlock(&block, rwSetMap, contractEventMap, true)
//
//	log.Infof("init block(%d,%s)", block.Header.BlockHeight, hex.EncodeToString(block.Header.BlockHash))
//	blockchainStoreImpl.EXPECT().PutBlock(&block, make([]*commonpb.TxRWSet, 0)).Return(nil)
//	txPool.EXPECT().RetryAndRemoveTxs(gomock.Any(), gomock.Any()).Return()
//	snapshotManager.EXPECT().NotifyBlockCommitted(&block).Return(nil)
//	err := blockCommitterImpl.AddBlock(&block)
//	require.Empty(t, err)
//
//	//ledgerCache.SetLastCommittedBlock(lastBlock)
//	block.Header.BlockHeight++
//	log.Infof("wrong block height(%d,%d)", block.Header.BlockHeight, ledgerCache.GetLastCommittedBlock().Header.BlockHeight)
//	err = blockCommitterImpl.AddBlock(&block)
//	require.NotEmpty(t, err)
//
//	ledgerCache.SetLastCommittedBlock(lastBlock)
//	log.Infof("wrong block height(%d,%d)", block.Header.BlockHeight, ledgerCache.GetLastCommittedBlock().Header.BlockHeight)
//	block.Header.BlockHeight--
//	block.Header.PreBlockHash = []byte("wrong")
//	err = blockCommitterImpl.AddBlock(&block)
//	require.NotEmpty(t, err)
//
//}
//
//func TestBlockSerialize(t *testing.T) {
//	lastBlock := createNewTestBlock(0)
//	require.NotNil(t, lastBlock)
//	fmt.Printf(utils.FormatBlock(lastBlock))
//}
//
//func initCommitter(
//	blockchainStoreImpl protocol.BlockchainStore,
//	txPool protocol.TxPool,
//	snapshotManager protocol.SnapshotManager,
//	ledgerCache protocol.LedgerCache,
//	proposedCache protocol.ProposalCache,
//	chainConf protocol.ChainConf,
//	msgbus msgbus.MessageBus,
//) protocol.BlockCommitter {
//
//	chainId := "Chain1"
//	blockCommitterImpl := &BlockCommitterImpl{
//		chainId:         chainId,
//		blockchainStore: blockchainStoreImpl,
//		snapshotManager: snapshotManager,
//		txPool:          txPool,
//		ledgerCache:     ledgerCache,
//		proposalCache:   proposedCache,
//		log:             logger.GetLoggerByChain(logger.MODULE_CORE, chainId),
//		chainConf:       chainConf,
//		msgBus:          msgbus,
//	}
//	return blockCommitterImpl
//}
//
//func createNewBlock(last *commonpb.Block) commonpb.Block {
//	var block commonpb.Block = commonpb.Block{
//		Header: &commonpb.BlockHeader{
//			BlockHeight:    0,
//			PreBlockHash:   nil,
//			BlockHash:      nil,
//			PreConfHeight:  0,
//			BlockVersion:   nil,
//			DagHash:        nil,
//			RwSetRoot:      nil,
//			BlockTimestamp: 0,
//			ConsensusArgs:  nil,
//			TxCount:        0,
//			Signature:      nil,
//		},
//		Dag: &commonpb.DAG{
//			Vertexes: nil,
//		},
//		Txs: nil,
//	}
//	lastHash := last.Header.BlockHash //返回数组
//	block.Header.PreBlockHash = lastHash[:]
//	block.Header.BlockHeight = last.Header.BlockHeight + 1
//	block.Header.BlockTimestamp = time.Now().Unix()
//	block.Header.BlockHash, _ = utils.CalcBlockHash("SHA256", &block)
//	return block
//}
//
//func createNewTestBlock(height int64) *commonpb.Block {
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
