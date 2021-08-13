/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package accesscontrol

import (
	"fmt"
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
	switch pbMember.MemberType {
	case pbac.MemberType_CERT, pbac.MemberType_CERT_HASH:
		return newCertMemberFromPb(pbMember, acs)
	}
	return nil, fmt.Errorf("new member failed: the member type is not supported")
}
