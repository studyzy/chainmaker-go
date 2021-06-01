package dpos

import (
	"chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/pb/protogo/dpos"
	"chainmaker.org/chainmaker-go/protocol"
)

type DposImpl struct {
	stateDB protocol.BlockchainStore
}

func NewDposImpl(blockChainStore protocol.BlockchainStore) *DposImpl {
	return &DposImpl{stateDB: blockChainStore}
}

func (impl *DposImpl) CreateDposRWSets(proposalHeight int64) []*common.TxRWSet {
	return nil
}

func (impl *DposImpl) SelectValidators(candidates []dpos.CandidateInfo) (validators []string) {

	return nil
}
