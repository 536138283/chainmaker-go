package accesscontrol

import (
	"testing"

	"chainmaker.org/chainmaker/pb-go/v2/syscontract"
	"chainmaker.org/chainmaker/utils/v2"

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

func testInitCertFunc(t *testing.T) map[string]*orgMember {
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

func TestNative_GetChainConfig(t *testing.T) {
	testNative_GetChainConfig(blockVersion220, t)

	testNative_GetChainConfig(blockVersion2320, t)

	testNative_GetChainConfig(blockVersion2320, t)
}

func testNative_GetChainConfig(blockVersion uint32, t *testing.T) {
	// initialize
	testPkOrgMember := testInitCertFunc(t)
	orgMemberInfo1 := testPkOrgMember[testOrg1]

	var (
		tx  *common.Transaction
		err error
		ok  bool
	)

	//【valid】test case
	tx = testCreateTx(
		syscontract.SystemContract_CHAIN_CONFIG.String(),
		syscontract.ChainConfigFunction_GET_CHAIN_CONFIG.String(),
		"test-txid-12345")

	err = testAppendSender2Tx(tx, testPKHashType, orgMemberInfo1.admin)
	require.Nil(t, err)

	resourceName := utils.GetTxResourceName(tx)
	ok, err = orgMemberInfo1.acProvider.VerifyTxPrincipal(tx, resourceName, blockVersion)
	require.Nil(t, err)
	require.Equal(t, true, ok)
}
