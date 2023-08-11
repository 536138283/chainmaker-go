package accesscontrol

import (
	"testing"

	acPb "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/protocol/v2/test"
	"github.com/stretchr/testify/require"
)

func TestGetMemberStatus(t *testing.T) {
	logger := &test.GoLogger{}
	certProvider, err := newCertACProvider(testChainConfig, testOrg1, nil, logger)
	require.Nil(t, err)
	require.NotNil(t, certProvider)

	pbMember := &acPb.Member{
		OrgId:      testOrg1,
		MemberType: acPb.MemberType_CERT,
		MemberInfo: []byte(testConsensusSignOrg1.cert),
	}

	memberStatus, err := certProvider.GetMemberStatus(pbMember)
	require.Nil(t, err)
	require.Equal(t, acPb.MemberStatus_NORMAL, memberStatus)
}

func testInitFunc(t *testing.T) map[string]*orgMember {
	_, cleanFunc, err := createTempDirWithCleanFunc()
	require.Nil(t, err)
	defer cleanFunc()

	var orgMemberMap = make(map[string]*orgMember, len(orgMemberInfoMap))
	for orgId, info := range orgMemberInfoMap {
		orgMemberMap[orgId] = initOrgMember(t, info)
	}

	return orgMemberMap
}

func testCreateEndorsementEntry(orgMember *orgMember, roleType protocol.Role, hashType, msg string) (*common.EndorsementEntry, error) {
	var (
		sigResource    []byte
		err            error
		signerResource *acPb.Member
	)
	switch roleType {
	case protocol.RoleConsensusNode:
		sigResource, err = orgMember.consensus.Sign(hashType, []byte(msg))
		if err != nil {
			return nil, err
		}

		signerResource, err = orgMember.consensus.GetMember()
		if err != nil {
			return nil, err
		}
	case protocol.RoleAdmin:
		sigResource, err = orgMember.admin.Sign(hashType, []byte(msg))
		if err != nil {
			return nil, err
		}

		signerResource, err = orgMember.admin.GetMember()
		if err != nil {
			return nil, err
		}
	default:
		sigResource, err = orgMember.client.Sign(hashType, []byte(msg))
		if err != nil {
			return nil, err
		}

		signerResource, err = orgMember.client.GetMember()
		if err != nil {
			return nil, err
		}
	}

	return &common.EndorsementEntry{
		Signer:    signerResource,
		Signature: sigResource,
	}, nil
}
