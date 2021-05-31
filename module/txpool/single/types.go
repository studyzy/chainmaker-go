/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package single

import (
	commonPb "chainmaker.org/chainmaker/pb-go/common"
	"chainmaker.org/chainmaker-go/protocol"
)

type mempoolTxs struct {
	isConfigTxs bool
	txs         []*commonPb.Transaction
	source      protocol.TxSource
}

type valInPendingCache struct {
	inBlockHeight int64
	tx            *commonPb.Transaction
}
