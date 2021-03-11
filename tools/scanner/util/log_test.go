/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package util

import (
	"fmt"
	"testing"
)

func TestLog(t *testing.T) {
	log := GetLog("2020-12-09 15:36:50.028	[INFO]	[Core] @chain1	proposer/block_proposer_impl.go:115	block proposer starts")
	fmt.Printf("%+v\n", log)
	log = GetLog("2020-12-09 15:36:50.028	[INFO]	[Blockchain]	blockchain/chainmaker_server.go:125	[Core] blockchain [chain1] start success")
	fmt.Printf("%+v\n", log)
}

func TestReplace(t *testing.T) {
	log := GetLog("2020-12-09 16:08:48.029	[INFO]	[Blockchain]	blockchain/chainmaker_server.go:125	[Core] blockchain [chain1] start success")
	fmt.Println(log.Replace("链${1}启动成功"))
}
