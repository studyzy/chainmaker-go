/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"chainmaker.org/chainmaker-go/common/msgbus"
	"chainmaker.org/chainmaker-go/core/cache"
	"chainmaker.org/chainmaker-go/logger"
	"chainmaker.org/chainmaker-go/mock"
	commonpb "chainmaker.org/chainmaker-go/pb/protogo/common"
	configpb "chainmaker.org/chainmaker-go/pb/protogo/config"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/utils"
	"encoding/hex"
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/google/martian/log"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestAddBlock(t *testing.T) {
	ctl := gomock.NewController(t)
	blockchainStoreImpl := mock.NewMockBlockchainStore(ctl)
	txPool := mock.NewMockTxPool(ctl)
	snapshotManager := mock.NewMockSnapshotManager(ctl)
	ledgerCache := cache.NewLedgerCache("Chain1")
	chainConf := mock.NewMockChainConf(ctl)
	proposedCache := cache.NewProposalCache(chainConf, ledgerCache)

	lastBlock := createNewTestBlock(0)
	ledgerCache.SetLastCommittedBlock(lastBlock)
	rwSetMap := make(map[string]*commonpb.TxRWSet)
	contractEventMap := make(map[string][]*commonpb.ContractEvent)
	msgbus := mock.NewMockMessageBus(ctl)
	msgbus.EXPECT().Publish(gomock.Any(), gomock.Any()).Return().Times(2)

	blockCommitterImpl := initCommitter(blockchainStoreImpl, txPool, snapshotManager, ledgerCache, proposedCache, chainConf, msgbus)
	require.NotNil(t, blockCommitterImpl)

	crypto := configpb.CryptoConfig{
		Hash: "SHA256",
	}
	contractConf := configpb.ContractConfig{EnableSqlSupport: false}
	chainConfig := configpb.ChainConfig{Crypto: &crypto, Contract: &contractConf}
	chainConf.EXPECT().ChainConfig().Return(&chainConfig).Times(3)

	block := createNewBlock(lastBlock)
	proposedCache.SetProposedBlock(&block, rwSetMap, contractEventMap, true)

	log.Infof("init block(%d,%s)", block.Header.BlockHeight, hex.EncodeToString(block.Header.BlockHash))
	blockchainStoreImpl.EXPECT().PutBlock(&block, make([]*commonpb.TxRWSet, 0)).Return(nil)
	txPool.EXPECT().RetryAndRemoveTxs(gomock.Any(), gomock.Any()).Return()
	snapshotManager.EXPECT().NotifyBlockCommitted(&block).Return(nil)
	err := blockCommitterImpl.AddBlock(&block)
	require.Empty(t, err)

	//ledgerCache.SetLastCommittedBlock(lastBlock)
	block.Header.BlockHeight++
	log.Infof("wrong block height(%d,%d)", block.Header.BlockHeight, ledgerCache.GetLastCommittedBlock().Header.BlockHeight)
	err = blockCommitterImpl.AddBlock(&block)
	require.NotEmpty(t, err)

	ledgerCache.SetLastCommittedBlock(lastBlock)
	log.Infof("wrong block height(%d,%d)", block.Header.BlockHeight, ledgerCache.GetLastCommittedBlock().Header.BlockHeight)
	block.Header.BlockHeight--
	block.Header.PreBlockHash = []byte("wrong")
	err = blockCommitterImpl.AddBlock(&block)
	require.NotEmpty(t, err)

}

func TestBlockSerialize(t *testing.T) {
	lastBlock := createNewTestBlock(0)
	require.NotNil(t, lastBlock)
	fmt.Printf(utils.FormatBlock(lastBlock))
}

func initCommitter(
	blockchainStoreImpl protocol.BlockchainStore,
	txPool protocol.TxPool,
	snapshotManager protocol.SnapshotManager,
	ledgerCache protocol.LedgerCache,
	proposedCache protocol.ProposalCache,
	chainConf protocol.ChainConf,
	msgbus msgbus.MessageBus,
) protocol.BlockCommitter {

	chainId := "Chain1"
	blockCommitterImpl := &BlockCommitterImpl{
		chainId:         chainId,
		blockchainStore: blockchainStoreImpl,
		snapshotManager: snapshotManager,
		txPool:          txPool,
		ledgerCache:     ledgerCache,
		proposalCache:   proposedCache,
		log:             logger.GetLoggerByChain(logger.MODULE_CORE, chainId),
		chainConf:       chainConf,
		msgBus:          msgbus,
	}
	return blockCommitterImpl
}

func createNewBlock(last *commonpb.Block) commonpb.Block {
	var block commonpb.Block = commonpb.Block{
		Header: &commonpb.BlockHeader{
			BlockHeight:    0,
			PreBlockHash:   nil,
			BlockHash:      nil,
			PreConfHeight:  0,
			BlockVersion:   nil,
			DagHash:        nil,
			RwSetRoot:      nil,
			BlockTimestamp: 0,
			ConsensusArgs:  nil,
			TxCount:        0,
			Signature:      nil,
		},
		Dag: &commonpb.DAG{
			Vertexes: nil,
		},
		Txs: nil,
	}
	lastHash := last.Header.BlockHash //返回数组
	block.Header.PreBlockHash = lastHash[:]
	block.Header.BlockHeight = last.Header.BlockHeight + 1
	block.Header.BlockTimestamp = time.Now().Unix()
	block.Header.BlockHash, _ = utils.CalcBlockHash("SHA256", &block)
	return block
}

func createNewTestBlock(height int64) *commonpb.Block {
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
	tx := createNewTestTx()
	txs := make([]*commonpb.Transaction, 1)
	txs[0] = tx
	block.Txs = txs
	return block
}

func createNewTestTx() *commonpb.Transaction {
	var hash = []byte("0123456789")
	return &commonpb.Transaction{
		Header: &commonpb.TxHeader{
			ChainId:        "",
			Sender:         nil,
			TxType:         0,
			TxId:           "",
			Timestamp:      0,
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