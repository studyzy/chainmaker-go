/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package common

import (
	"bytes"
	"fmt"
	"sync"

	commonpb "chainmaker.org/chainmaker/pb-go/v2/common"
	consensuspb "chainmaker.org/chainmaker/pb-go/v2/consensus"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/utils/v2"
)

type VerifyBlockBatch struct {
	txs       []*commonpb.Transaction
	newAddTxs []*commonpb.Transaction
	txHash    [][]byte
}

func NewVerifyBlockBatch(txs, newAddTxs []*commonpb.Transaction, txHash [][]byte) VerifyBlockBatch {
	return VerifyBlockBatch{
		txs:       txs,
		newAddTxs: newAddTxs,
		txHash:    txHash,
	}
}

// verifyStat, statistic for verify steps
type VerifyStat struct {
	TotalCount  int
	DBLasts     int64
	SigLasts    int64
	OthersLasts int64
	SigCount    int
}

func ValidateTx(txsRet map[string]*commonpb.Transaction, tx *commonpb.Transaction, blockHeight uint64,
	stat *VerifyStat, newAddTxs []*commonpb.Transaction, block *commonpb.Block,
	consensusType consensuspb.ConsensusType, hashType string, store protocol.BlockchainStore,
	chainId string, ac protocol.AccessControlProvider) error {
	txInPool, existTx := txsRet[tx.Payload.TxId]
	if existTx {
		if consensuspb.ConsensusType_HOTSTUFF == consensusType &&
			blockHeight != block.Header.BlockHeight && blockHeight > 0 {

			err := fmt.Errorf("tx duplicate in pending (tx:%s), txInPoolHeight:%d, txInBlockHeight:%d",
				tx.Payload.TxId, blockHeight, block.Header.BlockHeight)
			return err
		}

		return IsTxHashValid(tx, txInPool, hashType)
	}
	startDBTicker := utils.CurrentTimeMillisSeconds()
	isExist, err := store.TxExists(tx.Payload.TxId)
	stat.DBLasts += utils.CurrentTimeMillisSeconds() - startDBTicker
	if err != nil || isExist {
		err = fmt.Errorf("tx duplicate in DB (tx:%s)", tx.Payload.TxId)
		return err
	}
	stat.SigCount++
	startSigTicker := utils.CurrentTimeMillisSeconds()
	// if tx in txpool, means tx has already validated. tx noIt in txpool, need to validate.
	if err = utils.VerifyTxWithoutPayload(tx, chainId, ac); err != nil {
		err = fmt.Errorf("acl error (tx:%s), %s", tx.Payload.TxId, err.Error())
		return err
	}
	stat.SigLasts += utils.CurrentTimeMillisSeconds() - startSigTicker
	// tx valid and put into txpool
	newAddTxs = append(newAddTxs, tx) //nolint

	return nil
}

func TxVerifyResultsMerge(resultTasks map[int]VerifyBlockBatch,
	verifyBatchs map[int][]*commonpb.Transaction, errTxs []*commonpb.Transaction, txHashes [][]byte,
	txNewAdd []*commonpb.Transaction) ([][]byte, []*commonpb.Transaction, []*commonpb.Transaction, error) {
	if len(resultTasks) < len(verifyBatchs) {
		return nil, nil, errTxs, fmt.Errorf("tx verify error, batch num mismatch, received: %d,expected:%d",
			len(resultTasks), len(verifyBatchs))
	}
	for i := 0; i < len(resultTasks); i++ {
		batch := resultTasks[i]
		if len(batch.txs) != len(batch.txHash) {
			return nil, nil, errTxs,
				fmt.Errorf("tx verify error, txs in batch mismatch, received: %d, expected:%d",
					len(batch.txHash), len(batch.txs))
		}
		txHashes = append(txHashes, batch.txHash...)
		txNewAdd = append(txNewAdd, batch.newAddTxs...)

	}
	return txHashes, txNewAdd, nil, nil
}

func RearrangeRWSet(block *commonpb.Block, rwSetMap map[string]*commonpb.TxRWSet) []*commonpb.TxRWSet {
	rwSet := make([]*commonpb.TxRWSet, 0)
	if rwSetMap == nil {
		return rwSet
	}
	for _, tx := range block.Txs {
		if set, ok := rwSetMap[tx.Payload.TxId]; ok {
			rwSet = append(rwSet, set)
		}
	}
	return rwSet
}

// IsTxHashValid, to check if transaction hash is valid
func IsTxHashValid(tx *commonpb.Transaction, txInPool *commonpb.Transaction, hashType string) error {
	poolTxRawHash, err := utils.CalcTxRequestHash(hashType, txInPool)
	if err != nil {
		return fmt.Errorf("calc pool txhash error (tx:%s), %s", tx.Payload.TxId, err.Error())
	}
	txRawHash, err := utils.CalcTxRequestHash(hashType, tx)
	if err != nil {
		return fmt.Errorf("calc req txhash error (tx:%s), %s", tx.Payload.TxId, err.Error())
	}
	// check if tx equals with tx in pool
	if !bytes.Equal(txRawHash, poolTxRawHash) {
		return fmt.Errorf("txhash (tx:%s) expect %x, got %x", tx.Payload.TxId, poolTxRawHash, txRawHash)
	}
	return nil
}

// VerifyTxResult, to check if transaction result is valid,
// compare result simulate in this node with executed in other node
func VerifyTxResult(tx *commonpb.Transaction, result *commonpb.Result, hashType string) error {
	// verify if result is equal
	txResultHash, err := utils.CalcTxResultHash(hashType, tx.Result)
	if err != nil {
		return fmt.Errorf("calc tx result (tx:%s), %s)", tx.Payload.TxId, err.Error())
	}
	resultHash, err := utils.CalcTxResultHash(hashType, result)
	if err != nil {
		return fmt.Errorf("calc tx result (tx:%s), %s)", tx.Payload.TxId, err.Error())
	}
	if !bytes.Equal(txResultHash, resultHash) {
		return fmt.Errorf("tx result (tx:%s) expect %x, got %x", tx.Payload.TxId, txResultHash, resultHash)
	}
	return nil
}

// IsTxRWSetValid, to check if transaction read write set is valid
func IsTxRWSetValid(block *commonpb.Block, tx *commonpb.Transaction, rwSet *commonpb.TxRWSet, result *commonpb.Result,
	rwsetHash []byte) error {
	if rwSet == nil || result == nil {
		return fmt.Errorf("txresult, rwset == nil (blockHeight: %d) (blockHash: %s) (tx:%s)",
			block.Header.BlockHeight, block.Header.BlockHash, tx.Payload.TxId)
	}
	if !bytes.Equal(tx.Result.RwSetHash, rwsetHash) {
		return fmt.Errorf("tx rwset (tx:%s) expect %x, got %x", tx.Payload.TxId, tx.Result.RwSetHash, rwsetHash)
	}
	return nil
}

type VerifierTx struct {
	block       *commonpb.Block
	txRWSetMap  map[string]*commonpb.TxRWSet
	txResultMap map[string]*commonpb.Result
	log         protocol.Logger
	store       protocol.BlockchainStore
	txPool      protocol.TxPool
	ac          protocol.AccessControlProvider
	chainConf   protocol.ChainConf
}

type VerifierTxConfig struct {
	Block       *commonpb.Block
	TxRWSetMap  map[string]*commonpb.TxRWSet
	TxResultMap map[string]*commonpb.Result
	Log         protocol.Logger
	Store       protocol.BlockchainStore
	TxPool      protocol.TxPool
	Ac          protocol.AccessControlProvider
	ChainConf   protocol.ChainConf
}

func NewVerifierTx(conf *VerifierTxConfig) *VerifierTx {
	return &VerifierTx{
		block:       conf.Block,
		txRWSetMap:  conf.TxRWSetMap,
		txResultMap: conf.TxResultMap,
		log:         conf.Log,
		store:       conf.Store,
		txPool:      conf.TxPool,
		ac:          conf.Ac,
		chainConf:   conf.ChainConf,
	}
}

// VerifyTxs verify transactions in block
// include if transaction is double spent, transaction signature
func (vt *VerifierTx) verifierTxs(block *commonpb.Block) (txHashes [][]byte, txNewAdd []*commonpb.Transaction,
	errTxs []*commonpb.Transaction, err error) {

	verifyBatchs := utils.DispatchTxVerifyTask(block.Txs)
	resultTasks := make(map[int]VerifyBlockBatch)
	stats := make(map[int]*VerifyStat)
	var resultMu sync.Mutex
	var wg sync.WaitGroup
	waitCount := len(verifyBatchs)
	wg.Add(waitCount)
	txIds := utils.GetTxIds(block.Txs)
	txsRet, txsHeightRet := vt.txPool.GetTxsByTxIds(txIds)

	startTicker := utils.CurrentTimeMillisSeconds()
	for i := 0; i < waitCount; i++ {
		index := i
		go func() {
			defer wg.Done()
			txs := verifyBatchs[index]
			stat := &VerifyStat{
				TotalCount: len(txs),
			}
			txHashes1, newAddTxs, err1 := vt.verifyTx(txs, txsRet, txsHeightRet, stat, block)
			if err1 != nil {
				vt.log.Error(err1)
				err = err1
				return
			}
			resultMu.Lock()
			defer resultMu.Unlock()
			resultTasks[index] = VerifyBlockBatch{
				txs:       txs,
				txHash:    txHashes1,
				newAddTxs: newAddTxs,
			}
			stats[index] = stat
		}()
	}
	wg.Wait()
	concurrentLasts := utils.CurrentTimeMillisSeconds() - startTicker
	txHashes, txNewAdd, errTxs, err = TxVerifyResultsMerge(resultTasks, verifyBatchs, errTxs, txHashes, txNewAdd)

	if err != nil {
		return txHashes, txNewAdd, errTxs, err
	}

	for i, stat := range stats {
		if stat != nil {
			vt.log.Debugf("verify stat (index:%d,sigcount:%d/%d,db:%d,sig:%d,other:%d,total:%d)",
				i, stat.SigLasts, stat.TotalCount, stat.DBLasts, stat.SigLasts, stat.OthersLasts, concurrentLasts)
		}
	}

	return txHashes, txNewAdd, nil, nil
}

func (vt *VerifierTx) verifyTx(txs []*commonpb.Transaction, txsRet map[string]*commonpb.Transaction,
	txsHeightRet map[string]uint64, stat *VerifyStat, block *commonpb.Block) (
	[][]byte, []*commonpb.Transaction, error) {
	txHashes := make([][]byte, 0)
	newAddTxs := make([]*commonpb.Transaction, 0) // tx that verified and not in txpool, need to be added to txpool
	for _, tx := range txs {
		blockHeight := txsHeightRet[tx.Payload.TxId]
		if err := ValidateTx(txsRet, tx, blockHeight, stat, newAddTxs, block,
			vt.chainConf.ChainConfig().Consensus.Type, vt.chainConf.ChainConfig().Crypto.Hash, vt.store,
			vt.chainConf.ChainConfig().ChainId, vt.ac); err != nil {
			return nil, nil, err
		}
		startOthersTicker := utils.CurrentTimeMillisSeconds()
		rwSet := vt.txRWSetMap[tx.Payload.TxId]
		result := vt.txResultMap[tx.Payload.TxId]
		rwsetHash, err := utils.CalcRWSetHash(vt.chainConf.ChainConfig().Crypto.Hash, rwSet)
		if err != nil {
			vt.log.Warnf("calc rwset hash error (tx:%s), %s", tx.Payload.TxId, err)
			return nil, nil, err
		}
		if err = IsTxRWSetValid(vt.block, tx, rwSet, result, rwsetHash); err != nil {
			return nil, nil, err
		}
		result.RwSetHash = rwsetHash
		// verify if rwset hash is equal
		if err = VerifyTxResult(tx, result, vt.chainConf.ChainConfig().Crypto.Hash); err != nil {
			return nil, nil, err
		}
		hash, err := utils.CalcTxHash(vt.chainConf.ChainConfig().Crypto.Hash, tx)
		if err != nil {
			vt.log.Warnf("calc txhash error (tx:%s), %s", tx.Payload.TxId, err)
			return nil, nil, err
		}
		txHashes = append(txHashes, hash)
		stat.OthersLasts += utils.CurrentTimeMillisSeconds() - startOthersTicker
	}
	return txHashes, newAddTxs, nil
}
