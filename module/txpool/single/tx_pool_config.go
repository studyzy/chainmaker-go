/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package single

import (
	"time"

	"chainmaker.org/chainmaker-go/localconf"
)

const (
	DefaultChannelSize         = 10000        // The channel size to add the txs
	DefaultMaxTxCount          = 1000         // Maximum number of transactions in a block
	DefaultMaxTxPoolSize       = 5120         // Maximum number of common transaction in the pool
	DefaultMaxConfigTxPoolSize = 100          // Maximum number of config transaction in the pool
	DefaultFullNotifyAgainTime = int64(30)    // The unit is in seconds
	DefaultMaxTxTimeTimeout    = float64(600) // The unit is in seconds
	DefaultCacheThreshold      = 1
	DefaultFlushTimeOut        = 2 * time.Second
	DefaultFlushTicker         = 2
)

// ===========config in the blockchain============
// isTxTimeVerify Whether transactions require validation
func (pool *txPoolImpl) isTxTimeVerify() bool {
	chainConf := pool.chainConf
	if chainConf != nil {
		config := chainConf.ChainConfig()
		if config != nil {
			return config.Block.TxTimestampVerify
		}
	}
	return false
}

// maxTxTimeTimeout The maximum timeout for a transaction
func (pool *txPoolImpl) maxTxTimeTimeout() float64 {
	chainConf := pool.chainConf
	if chainConf != nil {
		config := chainConf.ChainConfig()
		if config != nil && config.Block.TxTimeout > 0 {
			return float64(config.Block.TxTimeout)
		}
	}
	return DefaultMaxTxTimeTimeout
}

// maxTxCount Maximum number of transactions in a block
func (pool *txPoolImpl) maxTxCount() int {
	chainConf := pool.chainConf
	if chainConf != nil {
		config := chainConf.ChainConfig()
		if config != nil && config.Block.BlockTxCapacity > 0 {
			return int(config.Block.BlockTxCapacity)
		}
	}
	return DefaultMaxTxCount
}

// ===========config in the local============
// maxCommonTxPoolSize Maximum number of common transaction in the pool
func (pool *txPoolImpl) maxCommonTxPoolSize() int {
	config := localconf.ChainMakerConfig.TxPoolConfig
	if config.MaxTxPoolSize != 0 {
		return int(config.MaxTxPoolSize)
	}
	return DefaultMaxTxPoolSize
}

// maxConfigTxPoolSize The maximum number of configure transaction in the pool
func (pool *txPoolImpl) maxConfigTxPoolSize() int {
	config := localconf.ChainMakerConfig.TxPoolConfig
	if config.MaxConfigTxPoolSize != 0 {
		return int(config.MaxConfigTxPoolSize)
	}
	return DefaultMaxConfigTxPoolSize
}

// notifyCycle The time to notify again when the trading pool is full
func (pool *txPoolImpl) notifyCycle() int64 {
	config := localconf.ChainMakerConfig.TxPoolConfig
	if config.FullNotifyAgainTime != 0 {
		return int64(config.FullNotifyAgainTime)
	}
	return DefaultFullNotifyAgainTime
}

// isMetrics Whether to log operation time
func isMetrics() bool {
	config := localconf.ChainMakerConfig.TxPoolConfig
	return config.IsMetrics
}
