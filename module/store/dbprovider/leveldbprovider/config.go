/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package leveldbprovider

type LevelDbConfig struct {
	StorePath            string `mapstructure:"store_path"`
	WriteBufferSize      int    `mapstructure:"write_buffer_size"`
	BloomFilterBits      int    `mapstructure:"bloom_filter_bits"`
	BlockWriteBufferSize int    `mapstructure:"block_write_buffer_size"`
}
