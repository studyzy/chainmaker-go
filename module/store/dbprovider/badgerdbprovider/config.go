/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package badgerdbprovider

type BadgerDbConfig struct {
	StorePath      string `mapstructure:"store_path"`
	Compression    uint8  `mapstructure:"compression"`
	ValueThreshold int64  `mapstructure:"value_threshold"`
}
