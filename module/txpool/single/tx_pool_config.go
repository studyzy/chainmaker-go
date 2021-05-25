/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package single

import (
	"time"
)

const (
	DefaultChannelSize    = 10000 // The channel size to add the txs
	DefaultCacheThreshold = 1
	DefaultFlushTimeOut   = 2 * time.Second
	DefaultFlushTicker    = 2
)
