/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package utils

import (
	"bytes"
	"crypto/sha256"
	"fmt"

	"chainmaker.org/chainmaker/common/v2/crypto/hash"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	consensusPb "chainmaker.org/chainmaker/pb-go/v2/consensus"
	"chainmaker.org/chainmaker/protocol/v2"
	"github.com/gogo/protobuf/proto"
)

// CalcBlockHash calculate block hash
func CalcBlockHash(hashType string, b *commonPb.Block) ([]byte, error) {
	if b == nil {
		return nil, fmt.Errorf("calc hash block == nil")
	}
	blockBytes, err := calcUnsignedBlockBytes(b)
	if err != nil {
		return nil, err
	}
	return hash.GetByStrType(hashType, blockBytes)
}

// SignBlock sign the block (in fact, here we sign block hash...) with signing member
// return hash bytes and signature bytes
func SignBlock(hashType string, singer protocol.SigningMember, b *commonPb.Block) ([]byte, []byte, error) {
	if singer == nil {
		return nil, nil, fmt.Errorf("sign block signer == nil")
	}
	blockHash, err := CalcBlockHash(hashType, b)
	if err != nil {
		return []byte{}, nil, err
	}
	sig, err := singer.Sign(hashType, blockHash)
	if err != nil {
		return nil, nil, err
	}
	return blockHash, sig, nil
}

// FormatBlock format block into string
func FormatBlock(b *commonPb.Block) string {
	serializedBlock := bytes.Buffer{}
	serializedBlock.WriteString("-------------------block begins-----------------\n")
	serializedBlock.WriteString(fmt.Sprintf("ChainId:\t%s\n", b.Header.ChainId))
	serializedBlock.WriteString(fmt.Sprintf("BlockHeight:\t%d\n", b.Header.BlockHeight))
	serializedBlock.WriteString(fmt.Sprintf("PreBlockHash:\t%x\n", b.Header.PreBlockHash))
	serializedBlock.WriteString(fmt.Sprintf("PreConfHeight:\t%d\n", b.Header.PreConfHeight))
	serializedBlock.WriteString(fmt.Sprintf("BlockVersion:\t%x\n", b.Header.BlockVersion))
	serializedBlock.WriteString(fmt.Sprintf("DagHash:\t%x\n", b.Header.DagHash))
	serializedBlock.WriteString(fmt.Sprintf("RwSetRoot:\t%x\n", b.Header.RwSetRoot))
	serializedBlock.WriteString(fmt.Sprintf("TxRoot:\t%x\n", b.Header.TxRoot))
	serializedBlock.WriteString(fmt.Sprintf("BlockTimestamp:\t%d\n", b.Header.BlockTimestamp))
	serializedBlock.WriteString(fmt.Sprintf("Proposer:\t%x\n", b.Header.Proposer))
	serializedBlock.WriteString(fmt.Sprintf("ConsensusArgs:\t%x\n", b.Header.ConsensusArgs))
	serializedBlock.WriteString(fmt.Sprintf("TxCount:\t%d\n", b.Header.TxCount))
	serializedBlock.WriteString("------------block signed part ends-------------\n")
	serializedBlock.WriteString(fmt.Sprintf("BlockHash:\t%x\n", b.Header.BlockHash))
	serializedBlock.WriteString(fmt.Sprintf("Signature:\t%x\n", b.Header.Signature))
	serializedBlock.WriteString("------------block unsigned part ends-----------\n")
	return serializedBlock.String()
}

// calcUnsignedBlockBytes calculate unsigned block bytes
// since dag & txs are already included in block header, we can safely set this two field to nil
func calcUnsignedBlockBytes(b *commonPb.Block) ([]byte, error) {
	//block := &commonPb.Block{
	//	Header: &commonPb.BlockHeader{
	//		ChainId:        b.Header.ChainId,
	//		BlockHeight:    b.Header.BlockHeight,
	//		PreBlockHash:   b.Header.PreBlockHash,
	//		BlockHash:      nil,
	//		PreConfHeight:  b.Header.PreConfHeight,
	//		BlockVersion:   b.Header.BlockVersion,
	//		DagHash:        b.Header.DagHash,
	//		RwSetRoot:      b.Header.RwSetRoot,
	//		TxRoot:         b.Header.TxRoot,
	//		BlockTimestamp: b.Header.BlockTimestamp,
	//		Proposer:       b.Header.Proposer,
	//		ConsensusArgs:  b.Header.ConsensusArgs,
	//		TxCount:        b.Header.TxCount,
	//		Signature:      nil,
	//	},
	//	Dag: nil,
	//	Txs: nil,
	//}
	//BlockHash就是HeaderHash，所以这里只需要把Header的Signature和BlockHash字段去掉，再序列化计算Hash即可。
	header := *b.Header
	header.Signature = nil
	header.BlockHash = nil
	blockBytes, err := proto.Marshal(&header)
	if err != nil {
		return nil, err
	}
	return blockBytes, nil
}

type BlockFingerPrint string

// CalcBlockFingerPrint since the block has not yet formed,
//snapshot uses fingerprint as the possible unique value of the block
func CalcBlockFingerPrint(block *commonPb.Block) BlockFingerPrint {
	if block == nil {
		return ""
	}
	chainId := block.Header.ChainId
	blockHeight := block.Header.BlockHeight
	blockTimestamp := block.Header.BlockTimestamp
	var blockProposer []byte
	if block.Header.Proposer != nil {
		blockProposer, _ = block.Header.Proposer.Marshal()
	}
	preBlockHash := block.Header.PreBlockHash

	return CalcFingerPrint(chainId, blockHeight, blockTimestamp, blockProposer, preBlockHash)
}

// CalcFingerPrint calculate finger print
func CalcFingerPrint(chainId string, height uint64, timestamp int64, proposer []byte, preHash []byte) BlockFingerPrint {
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%s-%v-%v-%v-%v", chainId, height, timestamp, proposer, preHash)))
	return BlockFingerPrint(fmt.Sprintf("%x", h.Sum(nil)))
}

// CalcPartialBlockHash calculate partial block bytes
// hash contains Header without BlockHash, ConsensusArgs, Signature
func CalcPartialBlockHash(hashType string, b *commonPb.Block) ([]byte, error) {
	if b == nil {
		return nil, fmt.Errorf("calc partial hash block == nil")
	}
	block := &commonPb.Block{
		Header: &commonPb.BlockHeader{
			ChainId:        b.Header.ChainId,
			BlockHeight:    b.Header.BlockHeight,
			PreBlockHash:   b.Header.PreBlockHash,
			BlockHash:      nil,
			PreConfHeight:  b.Header.PreConfHeight,
			BlockVersion:   b.Header.BlockVersion,
			DagHash:        b.Header.DagHash,
			RwSetRoot:      b.Header.RwSetRoot,
			TxRoot:         b.Header.TxRoot,
			BlockTimestamp: b.Header.BlockTimestamp,
			Proposer:       b.Header.Proposer,
			ConsensusArgs:  nil,
			TxCount:        b.Header.TxCount,
			Signature:      nil,
		},
		Dag: nil,
		Txs: nil,
	}

	blockBytes, err := proto.Marshal(block)
	if err != nil {
		return nil, err
	}
	return hash.GetByStrType(hashType, blockBytes)
}

// IsConfBlock is it a configuration block
func IsConfBlock(block *commonPb.Block) bool {
	if block == nil || len(block.Txs) == 0 {
		return false
	}
	tx := block.Txs[0]
	return IsValidConfigTx(tx)
}

// GetConsensusArgsFromBlock get args from block
func GetConsensusArgsFromBlock(block *commonPb.Block) (*consensusPb.BlockHeaderConsensusArgs, error) {
	if block == nil {
		return nil, fmt.Errorf("block is nil")
	}
	args := new(consensusPb.BlockHeaderConsensusArgs)
	if block.Header.ConsensusArgs == nil {
		return nil, fmt.Errorf("ConsensusArgs is nil")
	}
	err := proto.Unmarshal(block.Header.ConsensusArgs, args)
	if err != nil {
		return nil, fmt.Errorf("Unmarshal err:%s", err)
	}
	return args, nil
}

// IsEmptyBlock is it a empty block
func IsEmptyBlock(block *commonPb.Block) error {
	if block == nil || block.Header == nil || block.Header.BlockHash == nil ||
		block.Header.ChainId == "" || block.Header.PreBlockHash == nil || block.Header.Signature == nil {
		return fmt.Errorf("invalid block, yield verify")
	}
	return nil
}

// CanProposeEmptyBlock can empty blocks be packed
func CanProposeEmptyBlock(consensusType consensusPb.ConsensusType) bool {
	return consensusPb.ConsensusType_HOTSTUFF == consensusType || consensusPb.ConsensusType_POW == consensusType
}

func VerifyBlockSig(hashType string, b *commonPb.Block, ac protocol.AccessControlProvider) (bool, error) {
	hashedBlock, err := CalcBlockHash(hashType, b)
	if err != nil {
		return false, fmt.Errorf("fail to hash block: %v", err)
	}
	var member = b.Header.Proposer
	if err != nil {
		return false, fmt.Errorf("signer is unknown: %v", err)
	}
	endorsements := []*commonPb.EndorsementEntry{{
		Signer:    member,
		Signature: b.Header.Signature,
	}}
	principal, err := ac.CreatePrincipal(protocol.ResourceNameConsensusNode, endorsements, hashedBlock)
	if err != nil {
		return false, fmt.Errorf("fail to construct authentication principal: %v", err)
	}
	ok, err := ac.VerifyPrincipal(principal)
	if err != nil {
		return false, fmt.Errorf("authentication fail: %v", err)
	}
	if !ok {
		return false, fmt.Errorf("authentication fail")
	}
	return true, nil
}
func IsContractMgmtBlock(b *commonPb.Block) bool {
	if len(b.Txs) == 0 {
		return false
	}
	return IsContractMgmtTx(b.Txs[0])
}

func FilterBlockTxs(reqSenderOrgId string, block *commonPb.Block) *commonPb.Block {

	txs := block.GetTxs()
	results := make([]*commonPb.Transaction, 0, len(txs))

	newBlock := &commonPb.Block{
		Header:         block.Header,
		Dag:            block.Dag,
		AdditionalData: block.AdditionalData,
	}
	for i, tx := range txs {
		if block.Header.BlockHeight != 0 && tx.Sender.Signer.OrgId == reqSenderOrgId {
			results = append(results, txs[i])
		}
	}
	newBlock.Txs = results
	return newBlock
}
