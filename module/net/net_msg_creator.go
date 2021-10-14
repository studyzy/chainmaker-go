/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package net

import (
	netPb "chainmaker.org/chainmaker/pb-go/v2/net"
)

// NewNetMsg create a new netPb.NetMsg .
func NewNetMsg(msg []byte, msgType netPb.NetMsg_MsgType, to string) *netPb.NetMsg {
	return &netPb.NetMsg{Payload: msg, Type: msgType, To: to}
}
