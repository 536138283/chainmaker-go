/*
Copyright (C) Beijing Advanced Innovation Center for Future Blockchain and Privacy Computing (未来区块链与隐
私计算⾼精尖创新中⼼). All rights reserved.
SPDX-License-Identifier: Apache-2.0
*/

package accesscontrol

import (
	"testing"

	pbac "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/protocol/v2/test"
	"github.com/stretchr/testify/require"
)

var (
	test1IBCACProvider protocol.AccessControlProvider
	test2IBCACProvider protocol.AccessControlProvider
)

func TestIBCGetMemberStatus(t *testing.T) {
	logger := &test.GoLogger{}
	ibcProvider, err := newIBCACProvider(testIBCChainConfig, testOrg1, nil, logger)
	require.Nil(t, err)
	require.NotNil(t, ibcProvider)

	ibcMember := &pbac.Member{
		OrgId:      testIBCOrg1,
		MemberType: pbac.MemberType_IBC,
		MemberInfo: []byte(testIBCConsensusPK1),
	}

	memberStatus, err := ibcProvider.GetMemberStatus(ibcMember)
	require.Nil(t, err)
	require.Equal(t, pbac.MemberStatus_NORMAL, memberStatus)
}

func TestIBCNewCertMember(t *testing.T) {
	logger := &test.GoLogger{}
	ibcProvider, err := newIBCACProvider(testIBCChainConfig, testOrg1, nil, logger)
	require.Nil(t, err)

	pbMember := &pbac.Member{OrgId: testIBCOrg1, MemberType: pbac.MemberType_CERT,
		MemberInfo: []byte(testIBCConsensusTLSCert1)}
	_, err = ibcProvider.NewMember(pbMember)
	require.Nil(t, err)

	pbMember.MemberType = pbac.MemberType_CERT_HASH
	_, err = ibcProvider.NewMember(pbMember)
	require.NotNil(t, err)

}

func testIBCInitFunc(t *testing.T) map[string]*orgMember {
	_, cleanFunc, err := createTempDirWithCleanFunc()
	require.Nil(t, err)
	defer cleanFunc()

	var orgMemberMap = make(map[string]*orgMember, len(ibcOrgMemberInfoMap))
	for orgId, info := range ibcOrgMemberInfoMap {
		orgMemberMap[orgId] = initIBCOrgMember(t, info)
	}
	test1IBCACProvider = orgMemberMap[testIBCOrg1].acProvider
	test2IBCACProvider = orgMemberMap[testIBCOrg2].acProvider
	return orgMemberMap
}

func TestIBCVerifyReadPrincipal(t *testing.T) {
	orgMemberMap := testIBCInitFunc(t)
	//read
	orgMemberInfo := orgMemberMap[testIBCOrg2]
	endorsement, err := testCreateEndorsementEntry(orgMemberInfo, protocol.RoleClient, testHashType, testMsg)
	require.Nil(t, err)
	require.NotNil(t, endorsement)
	ok, err := testVerifyPrincipal(test1IBCACProvider, common.TxType_QUERY_CONTRACT.String(), []*common.EndorsementEntry{endorsement})
	require.Nil(t, err)
	require.Equal(t, true, ok)

	//read invalid
	orgMemberInfo = orgMemberMap[testIBCOrg5]
	endorsement, err = testCreateEndorsementEntry(orgMemberInfo, protocol.RoleClient, testHashType, testMsg)
	require.Nil(t, err)
	require.NotNil(t, endorsement)
	ok, err = testVerifyPrincipal(test1IBCACProvider, common.TxType_QUERY_CONTRACT.String(), []*common.EndorsementEntry{endorsement})
	require.NotNil(t, err)
	require.Equal(t, false, ok)
}

func TestIBCVerifyP2PPrincipal(t *testing.T) {
	orgMemberMap := testIBCInitFunc(t)
	//P2P
	orgMemberInfo := orgMemberMap[testIBCOrg1]
	endorsement, err := testCreateEndorsementEntry(orgMemberInfo, protocol.RoleConsensusNode, testHashType, testMsg)
	require.Nil(t, err)
	require.NotNil(t, endorsement)
	ok, err := testVerifyPrincipal(test1IBCACProvider, protocol.ResourceNameP2p, []*common.EndorsementEntry{endorsement})
	require.Nil(t, err)
	require.Equal(t, true, ok)

	//P2P invalid
	orgMemberInfo = orgMemberMap[testIBCOrg1]
	endorsement, err = testCreateEndorsementEntry(orgMemberInfo, protocol.RoleClient, testHashType, testMsg)
	require.Nil(t, err)
	require.NotNil(t, endorsement)
	ok, err = testVerifyPrincipal(test1IBCACProvider, protocol.ResourceNameP2p, []*common.EndorsementEntry{endorsement})
	require.NotNil(t, err)
	require.Equal(t, false, ok)

	orgMemberInfo = orgMemberMap[testIBCOrg5]
	endorsement, err = testCreateEndorsementEntry(orgMemberInfo, protocol.RoleConsensusNode, testHashType, testMsg)
	require.Nil(t, err)
	require.NotNil(t, endorsement)
	ok, err = testVerifyPrincipal(test1IBCACProvider, protocol.ResourceNameP2p, []*common.EndorsementEntry{endorsement})
	require.NotNil(t, err)
	require.Equal(t, false, ok)
}

func TestIBCVerifyConsensusPrincipal(t *testing.T) {
	orgMemberMap := testIBCInitFunc(t)
	//consensus
	orgMemberInfo := orgMemberMap[testIBCOrg1]
	endorsement, err := testCreateEndorsementEntry(orgMemberInfo, protocol.RoleConsensusNode, testHashType, testMsg)
	require.Nil(t, err)
	require.NotNil(t, endorsement)
	ok, err := testVerifyPrincipal(test1IBCACProvider, protocol.ResourceNameConsensusNode, []*common.EndorsementEntry{endorsement})
	require.Nil(t, err)
	require.Equal(t, true, ok)

	//consensus invalid
	orgMemberInfo = orgMemberMap[testIBCOrg1]
	endorsement, err = testCreateEndorsementEntry(orgMemberInfo, protocol.RoleAdmin, testHashType, testMsg)
	require.Nil(t, err)
	require.NotNil(t, endorsement)
	ok, err = testVerifyPrincipal(test1IBCACProvider, protocol.ResourceNameConsensusNode, []*common.EndorsementEntry{endorsement})
	require.NotNil(t, err)
	require.Equal(t, false, ok)

	orgMemberInfo = orgMemberMap[testIBCOrg5]
	endorsement, err = testCreateEndorsementEntry(orgMemberInfo, protocol.RoleConsensusNode, testHashType, testMsg)
	require.Nil(t, err)
	require.NotNil(t, endorsement)
	ok, err = testVerifyPrincipal(test1IBCACProvider, protocol.ResourceNameConsensusNode, []*common.EndorsementEntry{endorsement})
	require.NotNil(t, err)
	require.Equal(t, false, ok)
}

func TestIBCVerifySelfPrincipal(t *testing.T) {
	orgMemberMap := testIBCInitFunc(t)
	//self
	orgMemberInfo := orgMemberMap[testIBCOrg1]
	endorsement, err := testCreateEndorsementEntry(orgMemberInfo, protocol.RoleAdmin, testHashType, testMsg)
	require.Nil(t, err)
	require.NotNil(t, endorsement)
	principal, err := test1IBCACProvider.CreatePrincipalForTargetOrg(protocol.ResourceNameUpdateSelfConfig,
		[]*common.EndorsementEntry{endorsement}, []byte(testMsg), testIBCOrg1)
	require.Nil(t, err)
	ok, err := test1IBCACProvider.VerifyPrincipal(principal)
	require.Nil(t, err)
	require.Equal(t, true, ok)

	//self invalid
	orgMemberInfo = orgMemberMap[testIBCOrg1]
	endorsement, err = testCreateEndorsementEntry(orgMemberInfo, protocol.RoleAdmin, testHashType, testMsg)
	require.Nil(t, err)
	require.NotNil(t, endorsement)
	ok, err = testVerifyPrincipal(test1IBCACProvider, protocol.ResourceNameUpdateSelfConfig, []*common.EndorsementEntry{endorsement})
	require.NotNil(t, err)
	require.Equal(t, false, ok)

}

func TestIBCVerifyMajorityPrincipal(t *testing.T) {
	orgMemberMap := testIBCInitFunc(t)
	//majority
	orgMemberInfo1 := orgMemberMap[testIBCOrg1]
	endorsement1, err := testCreateEndorsementEntry(orgMemberInfo1, protocol.RoleAdmin, testHashType, testMsg)
	require.Nil(t, err)
	require.NotNil(t, endorsement1)

	orgMemberInfo2 := orgMemberMap[testIBCOrg2]
	endorsement2, err := testCreateEndorsementEntry(orgMemberInfo2, protocol.RoleAdmin, testHashType, testMsg)
	require.Nil(t, err)
	require.NotNil(t, endorsement2)

	orgMemberInfo3 := orgMemberMap[testIBCOrg3]
	endorsement3, err := testCreateEndorsementEntry(orgMemberInfo3, protocol.RoleAdmin, testHashType, testMsg)
	require.Nil(t, err)
	require.NotNil(t, endorsement3)

	ok, err := testVerifyPrincipal(test1IBCACProvider, protocol.ResourceNameUpdateConfig,
		[]*common.EndorsementEntry{endorsement1, endorsement2, endorsement3})
	require.Nil(t, err)
	require.Equal(t, true, ok)

	validEndorsements, err := testIBCGetValidEndorsements(test1IBCACProvider, protocol.ResourceNameUpdateConfig,
		[]*common.EndorsementEntry{endorsement1, endorsement2, endorsement3})

	require.Nil(t, err)
	require.Equal(t, len(validEndorsements), 3)

	//majority invalid

	ok, err = testVerifyPrincipal(test1IBCACProvider, protocol.ResourceNameUpdateConfig,
		[]*common.EndorsementEntry{endorsement1, endorsement2})
	require.NotNil(t, err)
	require.Equal(t, false, ok)

	validEndorsements, err = testIBCGetValidEndorsements(test1IBCACProvider, protocol.ResourceNameUpdateConfig,
		[]*common.EndorsementEntry{endorsement1, endorsement2})

	require.Nil(t, err)
	require.Equal(t, len(validEndorsements), 2)

}

func testIBCGetValidEndorsements(provider protocol.AccessControlProvider,
	resourceName string, endorsements []*common.EndorsementEntry) ([]*common.EndorsementEntry, error) {
	principal, err := provider.CreatePrincipal(resourceName, endorsements, []byte(testMsg))
	if err != nil {
		return nil, err
	}
	return provider.GetValidEndorsements(principal)
}

func TestIBCVerifyAllPrincipal(t *testing.T) {
	orgMemberMap := testIBCInitFunc(t)
	//all
	orgMemberInfo1 := orgMemberMap[testIBCOrg1]
	endorsement1, err := testCreateEndorsementEntry(orgMemberInfo1, protocol.RoleAdmin, testHashType, testMsg)
	require.Nil(t, err)
	require.NotNil(t, endorsement1)

	orgMemberInfo2 := orgMemberMap[testIBCOrg2]
	endorsement2, err := testCreateEndorsementEntry(orgMemberInfo2, protocol.RoleAdmin, testHashType, testMsg)
	require.Nil(t, err)
	require.NotNil(t, endorsement2)

	orgMemberInfo3 := orgMemberMap[testIBCOrg3]
	endorsement3, err := testCreateEndorsementEntry(orgMemberInfo3, protocol.RoleAdmin, testHashType, testMsg)
	require.Nil(t, err)
	require.NotNil(t, endorsement3)

	orgMemberInfo4 := orgMemberMap[testIBCOrg4]
	endorsement4, err := testCreateEndorsementEntry(orgMemberInfo4, protocol.RoleAdmin, testHashType, testMsg)
	require.Nil(t, err)
	require.NotNil(t, endorsement4)

	validEndorsements, err := testIBCGetValidEndorsements(test1IBCACProvider, protocol.ResourceNameUpdateConfig,
		[]*common.EndorsementEntry{endorsement1, endorsement2, endorsement3, endorsement4})

	require.Nil(t, err)
	require.Equal(t, len(validEndorsements), 4)

	//all invalid

	ok, err := testVerifyPrincipal(test1IBCACProvider, protocol.ResourceNameUpdateConfig,
		[]*common.EndorsementEntry{endorsement1, endorsement2})
	require.NotNil(t, err)
	require.Equal(t, false, ok)

	validEndorsements, err = testIBCGetValidEndorsements(test1IBCACProvider, protocol.ResourceNameUpdateConfig,
		[]*common.EndorsementEntry{endorsement1, endorsement2})

	require.Nil(t, err)
	require.Equal(t, len(validEndorsements), 2)

}

func TestIBCVerifyRelatedMaterial(t *testing.T) {
	logger := &test.GoLogger{}
	ibcProvider, err := newIBCACProvider(testIBCChainConfig, testIBCOrg1, nil, logger)
	require.Nil(t, err)
	require.NotNil(t, ibcProvider)
	isRevoked, err := ibcProvider.VerifyRelatedMaterial(pbac.VerifyType_CRL, []byte(""))
	require.NotNil(t, err)
	require.Equal(t, false, isRevoked)
	ibcProvider.VerifyRelatedMaterial(pbac.VerifyType_CRL, []byte(testCRL))
	require.NotNil(t, err)
	require.Equal(t, false, isRevoked)
}
