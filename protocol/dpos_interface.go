package protocol

import (
	"chainmaker.org/chainmaker-go/pb/protogo/common"
	consensuspb "chainmaker.org/chainmaker-go/pb/protogo/consensus"
)

type DPoS interface {
	// CreateDPoSRWSet Creates a RwSet for DPoS for the proposed block
	CreateDPoSRWSet(preBlkHash []byte, proposedBlock *consensuspb.ProposalBlock) error
	// VerifyConsensusArgs Verify the contents of the DPoS RwSet contained within the block
	VerifyConsensusArgs(block *common.Block, blockTxRwSet map[string]*common.TxRWSet) error
	// GetValidators Gets the validators for the current epoch
	GetValidators() ([]string, error)
}
