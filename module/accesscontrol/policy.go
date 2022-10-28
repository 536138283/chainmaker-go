/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package accesscontrol

import (
	"strings"

	pbac "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	"chainmaker.org/chainmaker/protocol/v2"
)

type policy struct {
	rule     protocol.Rule
	orgList  []string
	roleList []protocol.Role
}

// GetRule
//  @Description: return policy rule
//  @receiver p
//  @return protocol.Rule
//
func (p *policy) GetRule() protocol.Rule {
	return p.rule
}

// GetPbPolicy return protocol policy
//  @Description:
//  @receiver p
//  @return *pbac.Policy
//
func (p *policy) GetPbPolicy() *pbac.Policy {
	var pbRoleList []string
	for _, role := range p.roleList {
		var roleStr = string(role)
		pbRoleList = append(pbRoleList, roleStr)
	}
	return &pbac.Policy{
		Rule:     string(p.rule),
		OrgList:  p.orgList,
		RoleList: pbRoleList,
	}
}

// GetOrgList return org list
//  @Description:
//  @receiver p
//  @return []string
//
func (p *policy) GetOrgList() []string {
	return p.orgList
}

// GetRoleList return role list
//  @Description:
//  @receiver p
//  @return []protocol.Role
//
func (p *policy) GetRoleList() []protocol.Role {
	return p.roleList
}

// newPolicy
//  @Description: returns a policy
//  @param rule
//  @param orgList
//  @param roleList
//  @return *policy
//
func newPolicy(rule protocol.Rule, orgList []string, roleList []protocol.Role) *policy {
	return &policy{
		rule:     rule,
		orgList:  orgList,
		roleList: roleList,
	}
}

// newPolicyFromPb
//  @Description: convert to ac policy from pb policy
//  @param input
//  @return *policy
//
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

// newPbPolicyFromPolicy
//  @Description: convert ac policy to pb policy
//  @param input
//  @return *pbac.Policy
//
func newPbPolicyFromPolicy(input *policy) *pbac.Policy {
	p := &pbac.Policy{
		Rule:     string(input.rule),
		OrgList:  input.orgList,
		RoleList: nil,
	}

	for _, role := range input.roleList {
		p.RoleList = append(p.RoleList, string(role))
	}
	return p
}
