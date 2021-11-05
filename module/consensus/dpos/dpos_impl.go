/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dpos

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"chainmaker.org/chainmaker/vm-native/v2/dposmgr"

	"chainmaker.org/chainmaker/logger/v2"

	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/consensus"
	"chainmaker.org/chainmaker/pb-go/v2/consensus/dpos"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"
	"chainmaker.org/chainmaker/protocol/v2"
	"github.com/gogo/protobuf/proto"
)

type DPoSImpl struct {
	log       protocol.Logger
	chainConf protocol.ChainConf
	stateDB   protocol.BlockchainStore
}

func NewDPoSImpl(chainConf protocol.ChainConf, blockChainStore protocol.BlockchainStore) *DPoSImpl {
	log := logger.GetLoggerByChain(logger.MODULE_DPOS, chainConf.ChainConfig().ChainId)
	return &DPoSImpl{stateDB: blockChainStore, log: log, chainConf: chainConf}
}

func (impl *DPoSImpl) CreateDPoSRWSet(preBlkHash []byte, proposedBlock *consensus.ProposalBlock) error {
	consensusRwSets, err := impl.createDPoSRWSet(preBlkHash, proposedBlock)
	if err != nil {
		return err
	}
	if consensusRwSets != nil {
		err = impl.addConsensusArgsToBlock(consensusRwSets, proposedBlock.Block)
	}
	return err
}

func (impl *DPoSImpl) createDPoSRWSet(
	preBlkHash []byte, proposedBlock *consensus.ProposalBlock) (*common.TxRWSet, error) {
	impl.log.Debugf("begin createDPoS rwSet, blockInfo: %d:%x ",
		proposedBlock.Block.Header.BlockHeight, proposedBlock.Block.Header.BlockHash)
	// 1. judge consensus: DPoS
	if !impl.isDPoSConsensus() {
		impl.log.Debugf("no dpos consensus")
		return nil, nil
	}
	var (
		block        = proposedBlock.Block
		blockTxRwSet = proposedBlock.TxsRwSet
	)
	blockHeight := uint64(block.Header.BlockHeight)
	// 2. get epoch info from stateDB
	epoch, err := impl.getEpochInfo()
	if err != nil {
		return nil, err
	}
	if epoch.NextEpochCreateHeight != blockHeight {
		return nil, nil
	}
	// 3. create unbounding rwset
	unboundingRwSet, err := impl.completeUnbounding(epoch, block, blockTxRwSet)
	if err != nil {
		impl.log.Errorf("create complete unbonding error, reason: %s", err)
		return nil, err
	}
	// 4. create newEpoch
	newEpoch, err := impl.createNewEpoch(blockHeight, epoch, preBlkHash)
	if err != nil {
		impl.log.Errorf("create new epoch error, reason: %s", err)
		return nil, err
	}
	epochRwSet, err := impl.createEpochRwSet(newEpoch)
	if err != nil {
		impl.log.Errorf("create epoch rwSet error, reason: %s", err)
		return nil, err
	}
	validatorsRwSet, err := impl.createValidatorsRwSet(newEpoch)
	if err != nil {
		impl.log.Errorf("create validators rwSet error, reason: %s", err)
		return nil, err
	}
	// 5. Aggregate read-write set
	unboundingRwSet.TxWrites = append(unboundingRwSet.TxWrites, epochRwSet.TxWrites...)
	unboundingRwSet.TxWrites = append(unboundingRwSet.TxWrites, validatorsRwSet.TxWrites...)
	impl.log.Debugf("end createDPoS rwSet: %v ", unboundingRwSet)
	return unboundingRwSet, nil
}

func (impl *DPoSImpl) isDPoSConsensus() bool {
	return impl.chainConf.ChainConfig().Consensus.Type == consensus.ConsensusType_DPOS
}

func (impl *DPoSImpl) createNewEpoch(
	proposalHeight uint64, oldEpoch *syscontract.Epoch, seed []byte) (*syscontract.Epoch, error) {
	impl.log.Debugf("begin create new epoch in blockHeight: %d", proposalHeight)
	// 1. get property: epochBlockNum
	epochBlockNumBz, err := impl.stateDB.ReadObject(
		syscontract.SystemContract_DPOS_STAKE.String(), []byte(dposmgr.KeyEpochBlockNumber))
	if err != nil {
		impl.log.Errorf("load epochBlockNum from db failed, reason: %s", err)
		return nil, err
	}
	epochBlockNum := binary.BigEndian.Uint64(epochBlockNumBz)
	impl.log.Debugf("epoch blockNum: %d", epochBlockNum)

	// 2. get all candidates
	candidates, err := impl.getAllCandidateInfo()
	if err != nil {
		return nil, err
	}
	if len(candidates) == 0 {
		impl.log.Errorf("not found candidates from contract")
		return nil, fmt.Errorf("not found candidates from contract")
	}

	// 3. select validators from candidates
	validators, err := impl.selectValidators(candidates, seed)
	if err != nil {
		return nil, err
	}
	proposer := make([]string, 0, len(validators))
	for _, val := range validators {
		proposer = append(proposer, val.PeerId)
	}

	// 4. create NewEpoch
	newEpoch := &syscontract.Epoch{
		EpochId:               oldEpoch.EpochId + 1,
		NextEpochCreateHeight: proposalHeight + epochBlockNum,
		ProposerVector:        proposer,
	}
	impl.log.Debugf("new epoch: %s", newEpoch.String())
	return newEpoch, nil
}

func (impl *DPoSImpl) selectValidators(candidates []*dpos.CandidateInfo, seed []byte) ([]*dpos.CandidateInfo, error) {
	valNumBz, err := impl.stateDB.ReadObject(
		syscontract.SystemContract_DPOS_STAKE.String(), []byte(dposmgr.KeyEpochValidatorNumber))
	if err != nil {
		impl.log.Errorf("load epochBlockNum from db failed, reason: %s", err)
		return nil, err
	}
	valNum := binary.BigEndian.Uint64(valNumBz)
	vals, err := ValidatorsElection(candidates, int(valNum), seed, true)
	if err != nil {
		impl.log.Errorf("select validators from candidates failed, reason: %s", err)
		return nil, err
	}
	impl.log.Debugf("select validators: %v from candidates: %v by seed: %x", vals, candidates, seed)
	return vals, nil
}

func (impl *DPoSImpl) addConsensusArgsToBlock(rwSet *common.TxRWSet, block *common.Block) error {
	impl.log.Debugf("begin add consensus args to block ")
	if !impl.isDPoSConsensus() {
		return nil
	}
	consensusArgs := &consensus.BlockHeaderConsensusArgs{
		ConsensusType: int64(consensus.ConsensusType_DPOS),
		ConsensusData: rwSet,
	}
	argBytes, err := proto.Marshal(consensusArgs)
	if err != nil {
		impl.log.Errorf("marshal BlockHeaderConsensusArgs failed, reason: %s", err)
		return err
	}
	block.Header.ConsensusArgs = argBytes
	impl.log.Debugf("end add consensus args ")
	return nil
}

func (impl *DPoSImpl) VerifyConsensusArgs(block *common.Block, blockTxRwSet map[string]*common.TxRWSet) (err error) {
	impl.log.Debugf(
		"begin VerifyConsensusArgs, blockHeight: %d, blockHash: %x",
		block.Header.BlockHeight, block.Header.BlockHash)
	if !impl.isDPoSConsensus() {
		return nil
	}

	localConsensus, err := impl.createDPoSRWSet(
		block.Header.PreBlockHash, &consensus.ProposalBlock{Block: block, TxsRwSet: blockTxRwSet})
	if err != nil {
		impl.log.Errorf("get DPoS txRwSets failed, reason: %s", err)
		return err
	}

	var localBz []byte
	if localConsensus != nil {
		localBz, err = proto.Marshal(&consensus.BlockHeaderConsensusArgs{
			ConsensusType: int64(consensus.ConsensusType_DPOS),
			ConsensusData: localConsensus,
		})
		if err != nil {
			impl.log.Errorf("marshal BlockHeaderConsensusArgs failed, reason: %s", err)
			return err
		}
	}
	if bytes.Equal(block.Header.ConsensusArgs, localBz) {
		impl.log.Debugf("end VerifyConsensusArgs")
		return nil
	}
	consensusArgs := &consensus.BlockHeaderConsensusArgs{}
	if err = proto.Unmarshal(block.Header.ConsensusArgs, consensusArgs); err != nil {
		return fmt.Errorf("unmarshal dpos consensusArgs from blockHeader failed,reason: %s ", err)
	}
	return fmt.Errorf("consensus args verify mismatch, blockConsensus: %v, "+
		"localConsensus: %v by seed: %x", consensusArgs, localConsensus, block.Header.PreBlockHash)
}

func (impl *DPoSImpl) GetValidators() ([]string, error) {
	if !impl.isDPoSConsensus() {
		return nil, nil
	}
	epoch, err := impl.getEpochInfo()
	if err != nil {
		return nil, err
	}
	nodeIDs, err := impl.getNodeIDsFromValidators(epoch)
	return nodeIDs, err
}
