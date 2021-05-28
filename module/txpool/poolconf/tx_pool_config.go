/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package poolconf

import (
	"chainmaker.org/chainmaker-go/localconf"
	"chainmaker.org/chainmaker-go/protocol"
)

const (
	DefaultMaxTxCount          = 1000         // Maximum number of transactions in a block
	DefaultMaxTxPoolSize       = 5120         // Maximum number of common transaction in the pool
	DefaultMaxConfigTxPoolSize = 100          // Maximum number of config transaction in the pool
	DefaultMaxTxTimeTimeout    = float64(600) // The unit is in seconds
)

// ===========config in the blockchain============
// IsTxTimeVerify Whether transactions require validation
func IsTxTimeVerify(chainConf protocol.ChainConf) bool {
	if chainConf != nil {
		config := chainConf.ChainConfig()
		if config != nil {
			return config.Block.TxTimestampVerify
		}
	}
	return false
}

// MaxTxTimeTimeout The maximum timeout for a transaction
func MaxTxTimeTimeout(chainConf protocol.ChainConf) float64 {
	if chainConf != nil {
		config := chainConf.ChainConfig()
		if config != nil && config.Block.TxTimeout > 0 {
			return float64(config.Block.TxTimeout)
		}
	}
	return DefaultMaxTxTimeTimeout
}

// MaxTxCount Maximum number of transactions in a block
func MaxTxCount(chainConf protocol.ChainConf) int {
	if chainConf != nil {
		config := chainConf.ChainConfig()
		if config != nil && config.Block.BlockTxCapacity > 0 {
			return int(config.Block.BlockTxCapacity)
		}
	}
	return DefaultMaxTxCount
}

// ===========config in the local============
// MaxCommonTxPoolSize Maximum number of common transaction in the pool
func MaxCommonTxPoolSize() int {
	config := localconf.ChainMakerConfig.TxPoolConfig
	if config.MaxTxPoolSize != 0 {
		return int(config.MaxTxPoolSize)
	}
	return DefaultMaxTxPoolSize
}

// MaxConfigTxPoolSize The maximum number of configure transaction in the pool
func MaxConfigTxPoolSize() int {
	config := localconf.ChainMakerConfig.TxPoolConfig
	if config.MaxConfigTxPoolSize != 0 {
		return int(config.MaxConfigTxPoolSize)
	}
	return DefaultMaxConfigTxPoolSize
}

// IsMetrics Whether to log operation time
func IsMetrics() bool {
	config := localconf.ChainMakerConfig.TxPoolConfig
	return config.IsMetrics
}
