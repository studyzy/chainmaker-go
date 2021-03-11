package xvm

import (
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/protocol"
)

type Context struct {
	ID             int64
	Parameters     map[string]string
	TxSimContext   protocol.TxSimContext
	ContractId     *commonPb.ContractId
	ContractResult *commonPb.ContractResult

	callArgs *CallArgs
}
