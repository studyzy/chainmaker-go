/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package accesscontrol

import (
	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/protocol/v2"
)

var _ protocol.Principal = (*principal)(nil)

type principal struct {
	resourceName string
	endorsement  []*common.EndorsementEntry
	message      []byte

	targetOrg string
}

func (p *principal) GetResourceName() string {
	return p.resourceName
}

func (p *principal) GetEndorsement() []*common.EndorsementEntry {
	return p.endorsement
}

func (p *principal) GetMessage() []byte {
	return p.message
}

func (p *principal) GetTargetOrgId() string {
	return p.targetOrg
}
