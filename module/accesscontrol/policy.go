/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package accesscontrol

import (
	"strings"

	pbac "chainmaker.org/chainmaker/pb-go/accesscontrol"
	"chainmaker.org/chainmaker/protocol"
)

type policy struct {
	rule     protocol.Rule
	orgList  []string
	roleList []protocol.Role
}

func (p *policy) GetRule() protocol.Rule {
	return p.rule
}

func (p *policy) GetPbPolicy() *pbac.Policy {
	var pbRoleList []string
	for _, role := range p.roleList {
		var roleStr string
		roleStr = string(role)
		pbRoleList = append(pbRoleList, roleStr)
	}
	return &pbac.Policy{
		Rule:     string(p.rule),
		OrgList:  p.orgList,
		RoleList: pbRoleList,
	}
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

func NewPolicy(rule protocol.Rule, orgList []string, roleList []protocol.Role) *policy {
	return newPolicy(rule, orgList, roleList)
}

func newPolicyFromPb(input *pbac.Policy) *policy {
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
