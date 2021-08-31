/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package xvm

import (
	"chainmaker.org/chainmaker/common/v2/serialize"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/protocol/v2"
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
