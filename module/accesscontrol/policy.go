/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package accesscontrol

import (
	bcx509 "chainmaker.org/chainmaker/common/crypto/x509"
	pbac "chainmaker.org/chainmaker/pb-go/accesscontrol"
	"chainmaker.org/chainmaker-go/protocol"
	"strings"
)

type policy struct {
	rule     protocol.Rule
	orgList  []string
	roleList []protocol.Role
}

func (p *policy) GetRule() protocol.Rule {
	return p.rule
}

func (p *policy) GetOrgList() []string {
	return p.orgList
}

func (p *policy) GetRoleList() []protocol.Role {
	return p.roleList
}

func NewPolicy(rule protocol.Rule, orgList []string, roleList []protocol.Role) *policy {
	return &policy{
		rule:     rule,
		orgList:  orgList,
		roleList: roleList,
	}
}

func NewPolicyFromPb(input *pbac.Policy) *policy {
	p := &policy{
		rule:     protocol.Rule(input.Rule),
		orgList:  input.OrgList,
		roleList: nil,
	}
	for _, role := range input.RoleList {
		role = strings.ToUpper(role)
		p.roleList = append(p.roleList, protocol.Role(role))
	}

	return p
}

// Authentication validation policy
type policyWhiteList struct {
	policyType AuthMode
	policyList map[string]*bcx509.Certificate
}
