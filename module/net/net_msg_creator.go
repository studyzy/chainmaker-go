/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package net

import (
	netPb "chainmaker.org/chainmaker/pb-go/v2/net"
	"github.com/gogo/protobuf/proto"
)

// NewNetMsg create a new netPb.NetMsg .
func NewNetMsg(msg []byte, msgType netPb.NetMsg_MsgType, to string) *netPb.NetMsg {
	return &netPb.NetMsg{Payload: msg, Type: msgType, To: to}
}

// createMsgWithBytes create netPb.Msg with []byte.
func createMsgWithBytes(msg []byte) (*netPb.Msg, error) { //nolint: deadcode,unused
	var pbMsg netPb.Msg
	if err := proto.Unmarshal(msg, &pbMsg); err != nil {
		return nil, err
	}
	return &pbMsg, nil
}
