package protocol

import (
	"chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/pb/protogo/dpos"
)

type Dpos interface {
	CreateDposRWSets(proposalHeight int64) []*common.TxRWSet
	SelectValidators(candidates []dpos.CandidateInfo) (validators []string)
}
