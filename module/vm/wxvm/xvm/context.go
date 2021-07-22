package xvm

import (
	"chainmaker.org/chainmaker/common/serialize"
	commonPb "chainmaker.org/chainmaker/pb-go/common"
	"chainmaker.org/chainmaker/protocol"
)

type Context struct {
	ID             int64
	Parameters     map[string][]byte
	TxSimContext   protocol.TxSimContext
	ContractId     *commonPb.Contract
	ContractResult *commonPb.ContractResult

	callArgs      []*serialize.EasyCodecItem
	ContractEvent []*commonPb.ContractEvent

	gasUsed     uint64
	requestBody []byte
	in          []*serialize.EasyCodecItem
	resp        []*serialize.EasyCodecItem
	err         error
}
