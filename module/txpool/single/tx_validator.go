/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package single

import (
	"fmt"
	"math"

	commonErrors "chainmaker.org/chainmaker-go/common/errors"
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/txpool/poolconf"
	"chainmaker.org/chainmaker-go/utils"
)

// validate verify the validity of the transaction
// when the source type is P2P, additional certificate and tx header checks will be performed
func (pool *txPoolImpl) validate(tx *commonPb.Transaction, source protocol.TxSource) error {
	startTime := utils.CurrentTimeMillisSeconds()
	msg := fmt.Sprintf("tx_validator validate txId= %s,validateTxCertAndHeader= %v", tx.Header.TxId, source == protocol.P2P)
	pool.metrics("[start]"+msg, startTime, startTime)
	defer pool.metrics("[end]"+msg, startTime, utils.CurrentTimeMillisSeconds())

	if source == protocol.P2P {
		if err := utils.VerifyTxWithoutPayload(tx, pool.chainId, pool.ac); err != nil {
			pool.log.Error("validate tx err", "failed reason", err, "txId", tx.Header.GetTxId())
			return err
		}
		pool.log.Debugf("validate tx success", "txId", tx.Header.GetTxId())
	}
	if err := pool.validateTxTime(tx); err != nil {
		return err
	}

	if pool.isTxExistInDB(tx) {
		pool.log.Warnf("transaction exists in DB", "txId", tx.Header.GetTxId())
		return commonErrors.ErrTxIdExistDB
	}
	return nil
}

func (pool *txPoolImpl) validateTxTime(tx *commonPb.Transaction) error {
	if poolconf.IsTxTimeVerify(pool.chainConf) {
		txTimestamp := tx.Header.Timestamp
		chainTime := utils.CurrentTimeSeconds()
		if math.Abs(float64(chainTime-txTimestamp)) > poolconf.MaxTxTimeTimeout(pool.chainConf) {
			pool.log.Errorw("the txId timestamp is error", "txId", tx.Header.GetTxId(), "txTimestamp", txTimestamp, "chainTimestamp", chainTime)
			return commonErrors.ErrTxTimeout
		}
	}
	return nil
}

// isTxExistInDB verifies whether the transaction exists in the db
func (pool *txPoolImpl) isTxExistInDB(tx *commonPb.Transaction) (exist bool) {
	if pool.blockchainStore != nil {
		exist, _ = pool.blockchainStore.TxExists(tx.Header.GetTxId())
	}
	return
}
