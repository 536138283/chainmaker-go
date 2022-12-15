/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package accesscontrol

import (
	"chainmaker.org/chainmaker/pb-go/v3/common"
	"chainmaker.org/chainmaker/protocol/v3"
)

var _ protocol.Principal = (*principal)(nil)

//  principal
//  @Description: pk principal
//
type principal struct {
	resourceName string
	endorsement  []*common.EndorsementEntry
	message      []byte

	targetOrg string
}

// GetResourceName returns principal resource name
//  @Description:
//  @receiver p
//  @return string
//
func (p *principal) GetResourceName() string {
	return p.resourceName
}

// GetEndorsement returns principal endorsement
//  @Description:
//  @receiver p
//  @return []*common.EndorsementEntry
//
func (p *principal) GetEndorsement() []*common.EndorsementEntry {
	return p.endorsement
}

// GetMessage returns principal message
//  @Description:
//  @receiver p
//  @return []byte
//
func (p *principal) GetMessage() []byte {
	return p.message
}

// GetTargetOrgId returns principal target orgId
//  @Description:
//  @receiver p
//  @return string
//
func (p *principal) GetTargetOrgId() string {
	return p.targetOrg
}
