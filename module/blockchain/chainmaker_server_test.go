/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package blockchain

import (
	"testing"
	"time"
)

func TestInitAndStart(t *testing.T) {
	chainmakerServer := ChainMakerServer{}
	chainmakerServer.Init()
	timer := time.NewTimer(5 * time.Second)
	<-timer.C
}
