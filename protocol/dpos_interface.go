package protocol

import (
	"chainmaker.org/chainmaker-go/pb/protogo/common"
	consensuspb "chainmaker.org/chainmaker-go/pb/protogo/consensus"
)

type Dpos interface {
	CreateDposRWSets(preBlkHash []byte, proposedBlock *consensuspb.ProposalBlock) (*common.TxRWSet, error)
	VerifyConsensusArgs(block *common.Block, blockTxRwSet map[string]*common.TxRWSet) error
	GetValidators() ([]string, error)
	AddConsensusArgsToBlock(rwSet *common.TxRWSet, block *common.Block) (*common.Block, error)
}
