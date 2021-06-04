package dpos

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/pb/protogo/consensus"
	"chainmaker.org/chainmaker-go/pb/protogo/dpos"
	"chainmaker.org/chainmaker-go/protocol"
	"github.com/golang/protobuf/proto"
)

type DposImpl struct {
	log       protocol.Logger
	chainConf protocol.ChainConf
	stateDB   protocol.BlockchainStore
}

func NewDposImpl(log protocol.Logger, chainConf protocol.ChainConf, blockChainStore protocol.BlockchainStore) *DposImpl {
	return &DposImpl{stateDB: blockChainStore, log: log, chainConf: chainConf}
}

func (impl *DposImpl) CreateDposRWSets(proposalHeight uint64) (*common.TxRWSet, error) {
	// 1. judge consensus: dpos
	if !impl.isDposConsensus() {
		return nil, nil
	}
	// 2. get epoch info from stateDB
	epoch, err := impl.getEpochInfo()
	if err != nil {
		return nil, err
	}
	if epoch.NextEpochCreateHeight != proposalHeight {
		return nil, nil
	}

	// 3. create newEpoch
	newEpoch, err := impl.createNewEpoch(proposalHeight, epoch)
	if err != nil {
		return nil, err
	}
	txRwSet, err := impl.createEpochRwSet(newEpoch)
	if err != nil {
		return nil, err
	}
	return txRwSet, nil
}

func (impl *DposImpl) isDposConsensus() bool {
	return impl.chainConf.ChainConfig().Consensus.Type == consensus.ConsensusType_DPOS
}

func (impl *DposImpl) createNewEpoch(proposalHeight uint64, oldEpoch *common.Epoch) (*common.Epoch, error) {
	// 1. get property: epochBlockNum
	epochBlockNumBz, err := impl.stateDB.ReadObject(common.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String(), []byte(common.StakePrefix_Prefix_MinSelfDelegation.String()))
	if err != nil {
		impl.log.Errorf("load epochBlockNum from db failed, reason: %s", err)
		return nil, err
	}
	epochBlockNum := binary.BigEndian.Uint64(epochBlockNumBz)

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
	validators, err := impl.selectValidators(candidates)
	if err != nil {
		return nil, err
	}
	proposer := make([]string, 0, len(validators))
	for _, val := range validators {
		proposer = append(proposer, val.PeerID)
	}

	// 4. create NewEpoch
	return &common.Epoch{
		EpochID:               oldEpoch.EpochID + 1,
		NextEpochCreateHeight: proposalHeight + epochBlockNum, // todo. may be query property
		ProposerVector:        proposer,
	}, nil
}

func (impl *DposImpl) selectValidators(candidates []*dpos.CandidateInfo) ([]*dpos.CandidateInfo, error) {
	// todo. may be query property: valNum
	valNumBz, err := impl.stateDB.ReadObject(common.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String(), []byte(common.StakePrefix_Prefix_MinSelfDelegation.String()))
	if err != nil {
		impl.log.Errorf("load epochBlockNum from db failed, reason: %s", err)
		return nil, err
	}
	valNum := binary.BigEndian.Uint64(valNumBz)
	vals, err := ValidatorsElection(candidates, int(valNum), true)
	if err != nil {
		impl.log.Errorf("select validators from candidates failed, reason: %s", err)
		return nil, err
	}
	return vals, nil
}

func (impl *DposImpl) AddConsensusArgsToBlock(rwSet *common.TxRWSet, block *common.Block) (*common.Block, error) {
	if !impl.isDposConsensus() {
		return block, nil
	}
	consensusArgs := &consensus.BlockHeaderConsensusArgs{
		ConsensusType: int64(consensus.ConsensusType_DPOS),
		ConsensusData: rwSet,
	}
	argBytes, err := proto.Marshal(consensusArgs)
	if err != nil {
		impl.log.Errorf("marshal BlockHeaderConsensusArgs failed, reason: %s", err)
		return nil, err
	}
	block.Header.ConsensusArgs = argBytes
	return block, nil
}

func (impl *DposImpl) getConsensusArgsFromBlock(block *common.Block) *consensus.BlockHeaderConsensusArgs {
	if !impl.isDposConsensus() {
		return nil
	}

	consensusArgs := consensus.BlockHeaderConsensusArgs{}
	if len(block.Header.ConsensusArgs) == 0 {
		return nil
	}
	if err := proto.Unmarshal(block.Header.ConsensusArgs, &consensusArgs); err != nil {
		impl.log.Errorf("proto unmarshal consensus args failed, reason: %s", err)
		return nil
	}
	return &consensusArgs
}

func (impl *DposImpl) VerifyConsensusArgs(block *common.Block) error {
	if !impl.isDposConsensus() {
		return nil
	}
	localConsensus, err := impl.CreateDposRWSets(uint64(block.Header.BlockHeight))
	if err != nil {
		impl.log.Errorf("get dpos txRwSets failed, reason: %s", err)
		return err
	}
	localBz, err := proto.Marshal(&consensus.BlockHeaderConsensusArgs{
		ConsensusType: int64(consensus.ConsensusType_DPOS),
		ConsensusData: localConsensus,
	})
	if err != nil {
		impl.log.Errorf("marshal BlockHeaderConsensusArgs failed, reason: %s", err)
		return err
	}
	if bytes.Equal(block.Header.ConsensusArgs, localBz) {
		return nil
	}
	return fmt.Errorf("consensus args verify mismatch, blockConsensus: %v, localConsensus: %v", block.Header.ConsensusArgs, localConsensus)
}

func (impl *DposImpl) GetValidators() ([]string, error) {
	if !impl.isDposConsensus() {
		return nil, nil
	}
	epochBz, err := impl.stateDB.ReadObject(common.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String(), []byte(common.StakePrefix_Prefix_Curr_Epoch.String()))
	if err != nil {
		impl.log.Errorf("read epochInfo from stateDB failed, reason: %s", err)
		return nil, err
	}
	epoch := common.Epoch{}
	if err := proto.Unmarshal(epochBz, &epoch); err != nil {
		impl.log.Errorf("proto unmarshal epoch failed, reason: %s", err)
		return nil, err
	}
	return epoch.ProposerVector, nil
}
