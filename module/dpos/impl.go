package dpos

import (
	"chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/pb/protogo/dpos"
	"chainmaker.org/chainmaker-go/protocol"
)

type DposImpl struct {
	log     protocol.Logger
	stateDB protocol.BlockchainStore
}

func NewDposImpl(log protocol.Logger, blockChainStore protocol.BlockchainStore) *DposImpl {
	return &DposImpl{stateDB: blockChainStore, log: log}
}

func (impl *DposImpl) CreateDposRWSets(proposalHeight uint64) *common.TxRWSet {
	epoch, err := impl.getEpochInfo()
	if err != nil {
		return nil
	}
	if epoch.NextEpochCreateHeight != proposalHeight {
		return nil
	}

	newEpoch := impl.createNewEpoch(proposalHeight, epoch)
	txRwSet, err := impl.createEpochRwSet(newEpoch)
	if err != nil {
		return nil
	}
	return txRwSet
}

func (impl *DposImpl) createNewEpoch(proposalHeight uint64, oldEpoch *common.Epoch) *common.Epoch {
	candidates, err := impl.getAllCandidateInfo()
	if err != nil {
		return nil
	}
	if len(candidates) == 0 {
		impl.log.Errorf("not found candidates from contract")
		return nil
	}
	validators := impl.SelectValidators(candidates)
	proposer := make([]string, 0, len(validators))
	for _, val := range validators {
		proposer = append(proposer, val.PeerID)
	}
	return &common.Epoch{
		EpochID:               oldEpoch.EpochID + 1,
		NextEpochCreateHeight: proposalHeight + 0, // todo. may be query property
		ProposerVector:        proposer,
	}
}

func (impl *DposImpl) SelectValidators(candidates []*dpos.CandidateInfo) []*dpos.CandidateInfo {
	// todo. may be query property: valNum
	valNum := 4
	vals, err := ValidatorsElection(candidates, valNum, true)
	if err != nil {
		impl.log.Errorf("select validators from candidates failed, reason: %s", err)
		return nil
	}
	return vals
}
