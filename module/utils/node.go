/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package utils

import (
	"errors"
	"fmt"
	"strings"
)

// GetNodeUidFromAddr get protocol.NodeUId from node's address
func GetNodeUidFromAddr(addr string) (string, error) {
	if addr == "" {
		return "", fmt.Errorf("get uid addr == nil")
	}
	addrInfo := strings.Split(addr, "/")
	l := len(addrInfo)
	if l < 2 {
		return "", errors.New("incorrect address format")
	}
	return addrInfo[l-1], nil
}
