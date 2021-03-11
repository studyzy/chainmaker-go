/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package utils

import "time"

// CurrentTimeSeconds return current unix timestamp in seconds
func CurrentTimeSeconds() int64 {
	return time.Now().Unix()
}

// CurrentTimeMillisSeconds return current unix timestamp in milliseconds
func CurrentTimeMillisSeconds() int64 {
	return time.Now().UnixNano() / 1e6
}
