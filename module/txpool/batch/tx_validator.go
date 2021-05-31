/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package batch

import (
	"math"

	"chainmaker.org/chainmaker-go/txpool/poolconf"

	commonErrors "chainmaker.org/chainmaker/common/errors"
	commonPb "chainmaker.org/chainmaker/pb-go/common"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/utils"
)

func (p *BatchTxPool) validate(tx *commonPb.Transaction, source protocol.TxSource) error {
	if source == protocol.P2P {
		if err := utils.VerifyTxWithoutPayload(tx, p.chainId, p.ac); err != nil {
			p.log.Error("validate tx err", "failed reason", err, "txId", tx.Header.GetTxId())
			return err
		}
		p.log.Debugf("validate tx success", "txId", tx.Header.GetTxId())
	}
	if err := p.validateTxTime(tx); err != nil {
		return err
	}
	if _, _, ok := p.batchTxIdRecorder.FindBatchIdWithTxId(tx.Header.TxId); ok {
		p.log.Errorw("transaction exists", "txId", tx.Header.GetTxId())
		return commonErrors.ErrTxIdExist
	}
	if p.isTxExistInDB(tx) {
		p.log.Errorw("transaction exists in DB", "txId", tx.Header.GetTxId())
		return commonErrors.ErrTxIdExistDB
	}
	return nil
}

func (p *BatchTxPool) validateTxTime(tx *commonPb.Transaction) error {
	txTimestamp := tx.Header.Timestamp
	chainTime := utils.CurrentTimeSeconds()
	if math.Abs(float64(chainTime-txTimestamp)) > poolconf.MaxTxTimeTimeout(p.chainConf) {
		p.log.Errorw("the txId timestamp is error", "txId", tx.Header.GetTxId(), "txTimestamp", txTimestamp, "chainTimestamp", chainTime)
		return commonErrors.ErrTxTimeout
	}
	return nil
}

func (p *BatchTxPool) isTxExistInDB(tx *commonPb.Transaction) bool {
	if p.chainStore != nil {
		if txInDB, err := p.chainStore.GetTx(tx.Header.TxId); err != nil {
			return txInDB != nil
		}
	}
	return false
}
