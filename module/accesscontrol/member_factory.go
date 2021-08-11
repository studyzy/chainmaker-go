/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package accesscontrol

import (
	"sync"

	pbac "chainmaker.org/chainmaker/pb-go/accesscontrol"
	"chainmaker.org/chainmaker/protocol"
)

type memberFactory struct {
}

var once sync.Once
var memInstance *memberFactory

func MemberFactory() *memberFactory {
	once.Do(func() { memInstance = new(memberFactory) })
	return memInstance
}

func (mf *memberFactory) NewMember(pbMember *pbac.Member, acs *accessControlService) (protocol.Member, error) {
	p := NewMemberByMemberType(pbMember.MemberType)
	return p.NewMember(pbMember, acs)
}
