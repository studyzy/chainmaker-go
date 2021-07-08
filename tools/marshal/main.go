/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"fmt"
	"time"

	acPb "chainmaker.org/chainmaker/pb-go/accesscontrol"
	commonPb "chainmaker.org/chainmaker/pb-go/common"
	"github.com/gogo/protobuf/proto"
)

func main() {
	payload := &commonPb.Payload{
		ChainId:        "chain1",
		Sender:         &acPb.SerializedMember{OrgId: "wx-org1.chainmaker.com", MemberType: acPb.MemberType_CERT_HASH, MemberInfo: []byte{'a', 'b', 'c', 'd'}},
		TxType:         commonPb.TxType_INVOKE_CONTRACT,
		TxId:           "iiuowerytqwerewrwetretweryqooooereuy",
		Timestamp:      time.Now().Unix(),
		ExpirationTime: time.Now().Unix() + 20,
	}
	result, _ := proto.Marshal(payload)
	fmt.Print(result)
}
