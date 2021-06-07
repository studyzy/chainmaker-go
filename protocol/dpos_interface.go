package protocol

import (
	"chainmaker.org/chainmaker-go/pb/protogo/common"
)

type Dpos interface {
	CreateDposRWSets(preBlkHash []byte, proposalHeight uint64) (*common.TxRWSet, error)
	VerifyConsensusArgs(block *common.Block) error
	GetValidators() ([]string, error)
	AddConsensusArgsToBlock(rwSet *common.TxRWSet, block *common.Block) (*common.Block, error)
}
