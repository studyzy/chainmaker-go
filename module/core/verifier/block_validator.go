/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package verifier

import (
	"bytes"
	"chainmaker.org/chainmaker/common/crypto/hash"
	commonpb "chainmaker.org/chainmaker/pb-go/common"
	"chainmaker.org/chainmaker-go/utils"
	"fmt"
)

type BlockValidator struct {
	chainId  string
	hashType string
}

func NewBlockValidator(chainId, hashType string) *BlockValidator {
	return &BlockValidator{
		chainId:  chainId,
		hashType: hashType,
	}
}

// IsTxCountValid, to check if txcount in block is valid
func (bv *BlockValidator) IsTxCountValid(block *commonpb.Block) error {
	if block.Header.TxCount != int64(len(block.Txs)) {
		return fmt.Errorf("txcount expect %d, got %d", block.Header.TxCount, len(block.Txs))
	}
	return nil
}

// IsHeightValid, to check if block height is valid
func (bv *BlockValidator) IsHeightValid(block *commonpb.Block, currentHeight int64) error {
	if currentHeight+1 != block.Header.BlockHeight {
		return fmt.Errorf("height expect %d, got %d", currentHeight+1, block.Header.BlockHeight)
	}
	return nil
}

// IsPreHashValid, to check if block.preHash equals with last block hash
func (bv *BlockValidator) IsPreHashValid(block *commonpb.Block, preHash []byte) error {
	if !bytes.Equal(preHash, block.Header.PreBlockHash) {
		return fmt.Errorf("prehash expect %x, got %x", preHash, block.Header.PreBlockHash)
	}
	return nil
}

// IsBlockHashValid, to check if block hash equals with result calculated from block
func (bv *BlockValidator) IsBlockHashValid(block *commonpb.Block) error {
	hash, err := utils.CalcBlockHash(bv.hashType, block)
	if err != nil {
		return fmt.Errorf("calc block hash error")
	}
	if !bytes.Equal(hash, block.Header.BlockHash) {
		return fmt.Errorf("block hash expect %x, got %x", block.Header.BlockHash, hash)
	}
	return nil
}

// IsTxDuplicate, to check if there is duplicated transactions in one block
func (bv *BlockValidator) IsTxDuplicate(txs []*commonpb.Transaction) bool {
	txSet := make(map[string]struct{})
	exist := struct{}{}
	for _, tx := range txs {
		if tx == nil || tx.Header == nil {
			return true
		}
		txSet[tx.Header.TxId] = exist
	}
	// length of set < length of txs, means txs have duplicate tx
	return len(txSet) < len(txs)
}

// IsMerkleRootValid, to check if block merkle root equals with simulated merkle root
func (bv *BlockValidator) IsMerkleRootValid(block *commonpb.Block, txHashes [][]byte) error {
	txRoot, err := hash.GetMerkleRoot(bv.hashType, txHashes)
	if err != nil || !bytes.Equal(txRoot, block.Header.TxRoot) {
		return fmt.Errorf("txroot expect %x, got %x", block.Header.TxRoot, txRoot)
	}
	return nil
}

// IsDagHashValid, to check if block dag equals with simulated block dag
func (bv *BlockValidator) IsDagHashValid(block *commonpb.Block) error {
	dagHash, err := utils.CalcDagHash(bv.hashType, block.Dag)
	if err != nil || !bytes.Equal(dagHash, block.Header.DagHash) {
		return fmt.Errorf("dag expect %x, got %x", block.Header.DagHash, dagHash)
	}
	return nil
}

// IsRWSetHashValid, to check if read write set is valid
func (bv *BlockValidator) IsRWSetHashValid(block *commonpb.Block) error {
	rwSetRoot, err := utils.CalcRWSetRoot(bv.hashType, block.Txs)
	if err != nil {
		return fmt.Errorf("calc rwset error, %s", err)
	}
	if !bytes.Equal(rwSetRoot, block.Header.RwSetRoot) {
		return fmt.Errorf("rwset expect %x, got %x", block.Header.RwSetRoot, rwSetRoot)
	}
	return nil
}
