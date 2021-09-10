/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package batch

import (
	"math"

	"chainmaker.org/chainmaker-go/txpool/poolconf"

	commonErrors "chainmaker.org/chainmaker/common/v2/errors"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/utils/v2"
)

func (p *BatchTxPool) validate(tx *commonPb.Transaction, source protocol.TxSource) error {
	if source == protocol.P2P {
		if err := utils.VerifyTxWithoutPayload(tx, p.chainId, p.ac); err != nil {
			p.log.Error("validate tx err", "failed reason", err, "txId", tx.Payload.GetTxId())
			return err
		}
		p.log.Debugf("validate tx success", "txId", tx.Payload.GetTxId())
	}
	if err := p.validateTxTime(tx); err != nil {
		return err
	}
	if _, _, ok := p.batchTxIdRecorder.FindBatchIdWithTxId(tx.Payload.TxId); ok {
		p.log.Errorw("transaction exists", "txId", tx.Payload.GetTxId())
		return commonErrors.ErrTxIdExist
	}
	if p.isTxExistInDB(tx) {
		p.log.Errorw("transaction exists in DB", "txId", tx.Payload.GetTxId())
		return commonErrors.ErrTxIdExistDB
	}
	return nil
}

func (p *BatchTxPool) validateTxTime(tx *commonPb.Transaction) error {
	txTimestamp := tx.Payload.Timestamp
	chainTime := utils.CurrentTimeSeconds()
	if math.Abs(float64(chainTime-txTimestamp)) > poolconf.MaxTxTimeTimeout(p.chainConf) {
		p.log.Errorw("the txId timestamp is error", "txId", tx.Payload.GetTxId(), "txTimestamp",
			txTimestamp, "chainTimestamp", chainTime)
		return commonErrors.ErrTxTimeout
	}
	return nil
}

func (p *BatchTxPool) isTxExistInDB(tx *commonPb.Transaction) bool {
	if p.chainStore != nil {
		if txInDB, err := p.chainStore.GetTx(tx.Payload.TxId); err != nil {
			return txInDB != nil
		}
	}
	return false
}
