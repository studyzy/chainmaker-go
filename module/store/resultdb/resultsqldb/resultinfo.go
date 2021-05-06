/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package resultsqldb

import (
	commonpb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"github.com/gogo/protobuf/proto"
)

type SavePoint struct {
	BlockHeight uint64 `gorm:"primarykey"`
}

// StateHistoryInfo defines mysql orm model, used to create mysql table 'state_history_infos'
type ResultInfo struct {
	TxId        string `gorm:"size:128;primaryKey"`
	BlockHeight int64
	TxIndex     int
	Rwset       []byte `gorm:"type:longblob"`
	Status      int    `gorm:"default:0"`
	Result      []byte `gorm:"type:blob"`
	Message     string `gorm:"type:longtext"`
}

// NewHistoryInfo construct a new HistoryInfo
func NewResultInfo(txid string, blockHeight int64, txIndex int, result *commonpb.ContractResult, rw *commonpb.TxRWSet) *ResultInfo {
	rwBytes, _ := proto.Marshal(rw)

	return &ResultInfo{
		TxId:        txid,
		BlockHeight: blockHeight,
		TxIndex:     txIndex,
		Status:      int(result.Code),
		Result:      result.Result,
		Message:     result.Message,
		Rwset:       rwBytes,
	}
}
