/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package accesscontrol

import (
	"fmt"
	"sync"

	pbac "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	"chainmaker.org/chainmaker/protocol/v2"
)

type MemFactory struct {
}

var once sync.Once
var memInstance *MemFactory

func MemberFactory() *MemFactory {
	once.Do(func() { memInstance = new(MemFactory) })
	return memInstance
}

func (mf *MemFactory) NewMember(pbMember *pbac.Member, acs *accessControlService) (protocol.Member, error) {
	switch pbMember.MemberType {
	case pbac.MemberType_CERT, pbac.MemberType_CERT_HASH:
		return newCertMemberFromPb(pbMember, acs)
	}
	return nil, fmt.Errorf("new member failed: the member type is not supported")
}
