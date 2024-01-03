/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package accesscontrol

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"chainmaker.org/chainmaker/pb-go/v2/syscontract"

	"chainmaker.org/chainmaker/common/v2/msgbus"

	"chainmaker.org/chainmaker/common/v2/concurrentlru"
	"chainmaker.org/chainmaker/common/v2/crypto"
	"chainmaker.org/chainmaker/common/v2/crypto/asym"
	"chainmaker.org/chainmaker/localconf/v2"
	pbac "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	"chainmaker.org/chainmaker/pb-go/v2/consensus"

	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/config"
	"chainmaker.org/chainmaker/protocol/v2"
)

var _ protocol.AccessControlProvider = (*pkACProvider)(nil)

var nilPkACProvider ACProvider = (*pkACProvider)(nil)

const (
	//AdminPublicKey admin trust orgId
	AdminPublicKey = "public"
	//DposOrgId chainconfig the DPoS of orgId
	DposOrgId = "dpos_org_id"

	//PermissionConsensusOrgId chainconfig orgId for permission consensus, such as tbft
	PermissionConsensusOrgId = "public"
)

var (
	pubPolicyConsensus = newPolicy(
		protocol.RuleAny,
		nil,
		[]protocol.Role{
			protocol.RoleConsensusNode,
		},
	)
	pubPolicyManage = newPolicy(
		protocol.RuleAny,
		nil,
		[]protocol.Role{
			protocol.RoleAdmin,
		},
	)
	pubPolicyMajorityAdmin = newPolicy(
		protocol.RuleMajority,
		nil,
		[]protocol.Role{
			protocol.RoleAdmin,
		},
	)
	pubPolicyTransaction = newPolicy(
		protocol.RuleAny,
		nil,
		nil,
	)
	pubPolicyForbidden = newPolicy(
		protocol.RuleForbidden,
		nil,
		nil,
	)
)

type pkACProvider struct {

	//chainconfig authType
	authType string

	hashType string

	adminNum int32

	log protocol.Logger

	adminMember *sync.Map

	consensusMember *sync.Map

	memberCache *concurrentlru.Cache

	dataStore protocol.BlockchainStore

	txTypePolicyMap           *sync.Map
	msgTypePolicyMap          *sync.Map
	senderPolicyMap           *sync.Map
	resourceNamePolicyMap     *sync.Map
	resourceNamePolicyMap220  *sync.Map
	exceptionalPolicyMap220   *sync.Map
	resourceNamePolicyMap2320 *sync.Map
	exceptionalPolicyMap2320  *sync.Map
	latestPolicyMap           *sync.Map // map[string]*policy , resourceName -> *policy
}

type publicAdminMemberModel struct {
	publicKey crypto.PublicKey
	pkBytes   []byte
}

func (p *pkACProvider) NewACProvider(chainConf protocol.ChainConf, localOrgId string,
	store protocol.BlockchainStore, log protocol.Logger, msgBus msgbus.MessageBus) (
	protocol.AccessControlProvider, error) {
	pkAcProvider, err := newPkACProvider(chainConf.ChainConfig(), store, log)
	if err != nil {
		return nil, err
	}

	msgBus.Register(msgbus.ChainConfig, pkAcProvider)
	//v220_compat Deprecated
	chainConf.AddWatch(pkAcProvider) //nolint: staticcheck
	return pkAcProvider, nil
}

func newPkACProvider(chainConfig *config.ChainConfig,
	store protocol.BlockchainStore, log protocol.Logger) (*pkACProvider, error) {
	pkAcProvider := &pkACProvider{
		adminNum:                  0,
		hashType:                  chainConfig.Crypto.Hash,
		authType:                  chainConfig.AuthType,
		adminMember:               &sync.Map{},
		consensusMember:           &sync.Map{},
		memberCache:               concurrentlru.New(localconf.ChainMakerConfig.NodeConfig.CertCacheSize),
		log:                       log,
		dataStore:                 store,
		txTypePolicyMap:           &sync.Map{},
		msgTypePolicyMap:          &sync.Map{},
		senderPolicyMap:           &sync.Map{},
		resourceNamePolicyMap:     &sync.Map{},
		resourceNamePolicyMap220:  &sync.Map{},
		exceptionalPolicyMap220:   &sync.Map{},
		resourceNamePolicyMap2320: &sync.Map{},
		exceptionalPolicyMap2320:  &sync.Map{},
		latestPolicyMap:           &sync.Map{},
	}

	if chainConfig.Consensus.Type == consensus.ConsensusType_DPOS {

		pkAcProvider.createDefaultResourcePolicyForDPoS_220()
		pkAcProvider.createDefaultResourcePolicyForDPoS_2320()
		pkAcProvider.createDefaultResourcePolicyForDPoS()
	} else {
		pkAcProvider.createDefaultResourcePolicyForCommon_220()
		pkAcProvider.createDefaultResourcePolicyForCommon_2320()
		pkAcProvider.createDefaultResourcePolicyForCommon()
	}

	lastestPolicyMap := &sync.Map{}
	for _, resourcePolicy := range chainConfig.ResourcePolicies {
		if pkAcProvider.ValidateResourcePolicy(resourcePolicy) {
			policy := newPolicyFromPb(resourcePolicy.Policy)
			lastestPolicyMap.Store(resourcePolicy.ResourceName, policy)
		}
	}
	pkAcProvider.latestPolicyMap = lastestPolicyMap

	err := pkAcProvider.initAdminMembers(chainConfig.TrustRoots)
	if err != nil {
		return nil, fmt.Errorf("new public AC provider failed: %s", err.Error())
	}
	err = pkAcProvider.initConsensusMember(chainConfig)
	if err != nil {
		return nil, fmt.Errorf("new public AC provider failed: %s", err.Error())
	}
	return pkAcProvider, nil
}

func (p *pkACProvider) initAdminMembers(trustRootList []*config.TrustRootConfig) error {
	var (
		tempSyncMap sync.Map
	)

	if len(trustRootList) == 0 {
		p.log.Debugf("no super administrator is configured")
		return nil
	}

	var adminNum int32

	for _, trustRoot := range trustRootList {
		if strings.ToLower(trustRoot.OrgId) == AdminPublicKey {
			for _, root := range trustRoot.Root {
				pk, err := asym.PublicKeyFromPEM([]byte(root))
				if err != nil {
					return fmt.Errorf("init admin member failed: parse the public key from PEM failed")
				}
				pkBytes, err := pk.Bytes()
				if err != nil {
					return fmt.Errorf("init admin member failed: %s", err.Error())
				}
				adminMember := &publicAdminMemberModel{
					publicKey: pk,
					pkBytes:   pkBytes,
				}
				adminKey := hex.EncodeToString(pkBytes)
				tempSyncMap.Store(adminKey, adminMember)
				adminNum++
			}
		}
	}
	p.adminMember = &tempSyncMap
	atomic.StoreInt32(&p.adminNum, adminNum)
	return nil
}

func (p *pkACProvider) initConsensusMember(chainConfig *config.ChainConfig) error {
	if chainConfig.Consensus.Type == consensus.ConsensusType_DPOS {
		return p.initDPoSMember(chainConfig.Consensus.Nodes)
	} else if chainConfig.Consensus.Type == consensus.ConsensusType_TBFT {
		return p.initPermissionMember(chainConfig.Consensus.Nodes)
	}
	return fmt.Errorf("public chain mode does not support other consensus")
}

func (p *pkACProvider) initDPoSMember(consensusConf []*config.OrgConfig) error {
	if len(consensusConf) == 0 {
		return fmt.Errorf("update dpos consensus member failed: DPoS config can't be empty in chain config")
	}

	var consensusMember sync.Map
	if consensusConf[0].OrgId != DposOrgId {
		return fmt.Errorf("update dpos consensus member failed: DPoS node config orgId do not match")
	}
	for _, nodeId := range consensusConf[0].NodeId {
		consensusMember.Store(nodeId, struct{}{})
	}
	p.consensusMember = &consensusMember
	p.log.Infof("update consensus list: [%v]", p.consensusMember)
	return nil
}

func (p *pkACProvider) initPermissionMember(consensusConf []*config.OrgConfig) error {
	if len(consensusConf) == 0 {
		return fmt.Errorf("update permission consensus member failed: consensus node config can't be empty in chain config")
	}

	var consensusMember sync.Map
	if consensusConf[0].OrgId != PermissionConsensusOrgId {
		return fmt.Errorf("update permission consensus member failed: node config orgId do not match")
	}
	for _, nodeId := range consensusConf[0].NodeId {
		consensusMember.Store(nodeId, struct{}{})
	}
	p.consensusMember = &consensusMember
	p.log.Infof("update permission consensus list: [%v]", p.consensusMember)
	return nil
}

func (p *pkACProvider) lookUpMemberInCache(memberInfo string) (*memberCached, bool) {
	ret, ok := p.memberCache.Get(memberInfo)
	if ok {
		return ret.(*memberCached), true
	}
	return nil, false
}

func (p *pkACProvider) getMemberFromCache(member *pbac.Member) protocol.Member {
	cached, ok := p.lookUpMemberInCache(string(member.MemberInfo))
	if ok {
		p.log.Debugf("member found in local cache")
		return cached.member
	}
	// handle false positive when member cache is cleared
	if p.authType == protocol.Public {
		tmpMember, err := p.NewMemberFromAcs(member)
		if err != nil {
			p.log.Debugf("new member failed, authType = %s, err = %s", p.authType, err.Error())
			return nil
		}
		p.memberCache.Add(string(member.MemberInfo), &memberCached{
			member:    tmpMember,
			certChain: nil,
		})
		return tmpMember
	}
	return nil
}

//func (p *pkACProvider) Module() string {
//	return ModuleNameAccessControl
//}
//
//
//func (p *pkACProvider) Watch(chainConfig *config.ChainConfig) error {
//
//	p.hashType = chainConfig.GetCrypto().GetHash()
//	err := p.initAdminMembers(chainConfig.TrustRoots)
//	if err != nil {
//		return fmt.Errorf("new public AC provider failed: %s", err.Error())
//	}
//
//	err = p.initConsensusMember(chainConfig)
//	if err != nil {
//		return fmt.Errorf("new public AC provider failed: %s", err.Error())
//	}
//	p.memberCache.Clear()
//	return nil
//}

// NewMember creates a member from pb Member
func (p *pkACProvider) NewMember(pbMember *pbac.Member) (protocol.Member, error) {
	cache := p.getMemberFromCache(pbMember)
	if cache != nil {
		return cache, nil
	}
	member, err := publicNewPkMemberFromAcs(pbMember, p.adminMember, p.consensusMember, p.hashType)
	if err != nil {
		return nil, fmt.Errorf("new member failed: %s", err.Error())
	}
	p.memberCache.Add(string(pbMember.MemberInfo), &memberCached{
		member:    member,
		certChain: nil,
	})
	return member, nil
}

// NewMember creates a member from pb Member
func (p *pkACProvider) NewMemberFromAcs(pbMember *pbac.Member) (protocol.Member, error) {
	member, err := publicNewPkMemberFromAcs(pbMember, p.adminMember, p.consensusMember, p.hashType)
	if err != nil {
		return nil, fmt.Errorf("new member failed: %s", err.Error())
	}
	return member, nil
}

func (p *pkACProvider) verifyRuleAnyCase(pol *policy, endorsements []*common.EndorsementEntry) (bool, error) {
	roleList := p.buildRoleListForVerifyPrincipal(pol)
	for _, endorsement := range endorsements {
		if len(roleList) == 0 {
			return true, nil
		}
		member := p.getMemberFromCache(endorsement.Signer)
		if member == nil {
			p.log.Infof(
				"authentication warning: the member is not in member cache, memberInfo[%s]",
				string(endorsement.Signer.MemberInfo))
			continue
		}
		//In PK mode, the client obtains an empty string through getrole()
		role := member.GetRole()
		if role == "" {
			role = protocol.RoleClient
		}
		if _, ok := roleList[role]; ok {
			return true, nil
		}
		p.log.Infof("authentication warning, the member role is not in roleList, role: [%s]",
			member.GetRole())
	}

	err := fmt.Errorf("authentication fail for any rule, policy: rule: [%v],roleList: [%v]",
		pol.rule, pol.roleList)
	return false, err
}

func (p *pkACProvider) verifyRuleAllCase(pol *policy, endorsements []*common.EndorsementEntry) (bool, error) {
	role := protocol.RoleAdmin
	refinedEndorsements := p.getValidEndorsementsInner(
		map[string]bool{}, map[protocol.Role]bool{role: true}, endorsements)
	numOfValid := len(refinedEndorsements)
	p.log.Debugf("verifyRuleMajorityAdminCase: numOfValid=[%d], p.adminNum=[%d]", numOfValid, p.adminNum)
	if numOfValid >= int(p.adminNum) {
		return true, nil
	}
	return false, fmt.Errorf("%s: %d valid endorsements required, %d valid endorsements received",
		notEnoughParticipantsSupportError, p.adminNum, numOfValid)

}

func (p *pkACProvider) verifyRuleMajorityCase(pol *policy, endorsements []*common.EndorsementEntry) (bool, error) {
	role := protocol.RoleAdmin
	refinedEndorsements := p.getValidEndorsementsInner(
		map[string]bool{}, map[protocol.Role]bool{role: true}, endorsements)
	numOfValid := len(refinedEndorsements)
	p.log.Debugf("verifyRuleMajorityAdminCase: numOfValid=[%d], p.adminNum=[%d]", numOfValid, p.adminNum)
	if float64(numOfValid) > float64(p.adminNum)/2.0 {
		return true, nil
	}
	return false, fmt.Errorf("%s: %d valid endorsements required, %d valid endorsements received",
		notEnoughParticipantsSupportError, int(float64(p.adminNum)/2.0+1), numOfValid)
}

func (p *pkACProvider) verifyRuleDefaultCase(pol *policy, endorsements []*common.EndorsementEntry) (bool, error) {
	rule := pol.GetRule()
	nums := strings.Split(string(rule), LIMIT_DELIMITER)

	refinedEndorsements := p.getValidEndorsementsInner(
		map[string]bool{}, map[protocol.Role]bool{protocol.RoleAdmin: true}, endorsements)
	numOfValid := len(refinedEndorsements)

	switch len(nums) {
	case 1:
		threshold, err := strconv.Atoi(nums[0])
		if err != nil {
			return false, fmt.Errorf("authentication fail: unrecognized rule, should be ANY, MAJORITY, ALL, " +
				"SELF, ac threshold (integer), or ac portion (fraction)")
		}

		if numOfValid >= threshold {
			return true, nil
		}
		return false, fmt.Errorf("%s: %d valid endorsements required, %d valid endorsements received",
			notEnoughParticipantsSupportError, threshold, numOfValid)

	case 2:
		numerator, err := strconv.Atoi(nums[0])
		denominator, err2 := strconv.Atoi(nums[1])
		if err != nil || err2 != nil {
			return false, fmt.Errorf("authentication fail: unrecognized rule, should be ANY, MAJORITY, ALL, " +
				"SELF, an integer, or ac fraction")
		}

		if denominator <= 0 {
			denominator = int(p.adminNum)
		}

		var numRequired float64
		numRequired = float64(p.adminNum) * float64(numerator) / float64(denominator)

		if float64(numOfValid) >= numRequired {
			return true, nil
		}
		return false, fmt.Errorf("%s: %f valid endorsements required, %d valid endorsements received",
			notEnoughParticipantsSupportError, numRequired, numOfValid)
	default:
		return false, fmt.Errorf("authentication fail: unrecognized principle type, should be ANY, MAJORITY, " +
			"ALL, an integer (Threshold), or ac fraction (Portion)")
	}
}

func (p *pkACProvider) buildRoleListForVerifyPrincipal(pol *policy) map[protocol.Role]bool {
	roleListRaw := pol.GetRoleList()
	roleList := map[protocol.Role]bool{}
	for _, roleRaw := range roleListRaw {
		roleList[roleRaw] = true
	}
	return roleList
}

// all-in-one validation for signing members: signature, policies
func (p *pkACProvider) refinePrincipal(principal protocol.Principal) (protocol.Principal, error) {
	endorsements := principal.GetEndorsement()
	msg := principal.GetMessage()
	refinedEndorsement := p.RefineEndorsements(endorsements, msg)
	if len(refinedEndorsement) <= 0 {
		return nil, fmt.Errorf("refine endorsements failed, all endorsers have failed verification")
	}

	refinedPrincipal, err := p.CreatePrincipal(principal.GetResourceName(), refinedEndorsement, msg)
	if err != nil {
		return nil, fmt.Errorf("create principal failed: [%s]", err.Error())
	}

	return refinedPrincipal, nil
}

func (p *pkACProvider) RefineEndorsements(endorsements []*common.EndorsementEntry,
	msg []byte) []*common.EndorsementEntry {

	refinedSigners := map[string]bool{}
	var refinedEndorsement []*common.EndorsementEntry

	for _, endorsementEntry := range endorsements {
		endorsement := &common.EndorsementEntry{
			Signer: &pbac.Member{
				OrgId:      endorsementEntry.Signer.OrgId,
				MemberInfo: endorsementEntry.Signer.MemberInfo,
				MemberType: endorsementEntry.Signer.MemberType,
			},
			Signature: endorsementEntry.Signature,
		}
		memInfo := string(endorsement.Signer.MemberInfo)

		remoteMember, err := p.NewMember(endorsement.Signer)
		if err != nil {
			p.log.Infof("new member failed: [%s]", err.Error())
			continue
		}

		if err := remoteMember.Verify(p.hashType, msg, endorsement.Signature); err != nil {
			p.log.Infof("signer member verify signature failed: [%s]", err.Error())
			p.log.Debugf("information for invalid signature:\norganization: %s\npubkey: %s\nmessage: %s\n"+
				"signature: %s", endorsement.Signer.OrgId, memInfo, hex.Dump(msg), hex.Dump(endorsement.Signature))
			continue
		}
		if _, ok := refinedSigners[memInfo]; !ok {
			refinedSigners[memInfo] = true
			refinedEndorsement = append(refinedEndorsement, endorsement)
		}
	}
	return refinedEndorsement
}

func (p *pkACProvider) getValidEndorsementsInner(orgList map[string]bool, roleList map[protocol.Role]bool,
	endorsements []*common.EndorsementEntry) []*common.EndorsementEntry {
	var refinedEndorsements []*common.EndorsementEntry
	for _, endorsement := range endorsements {
		if len(roleList) == 0 {
			refinedEndorsements = append(refinedEndorsements, endorsement)
			continue
		}

		member := p.getMemberFromCache(endorsement.Signer)
		if member == nil {
			p.log.Debugf(
				"authentication warning: the member is not in member cache, memberInfo[%s]",
				string(endorsement.Signer.MemberInfo))
			continue
		}

		p.log.Debugf("getValidEndorsements: signer's role [%v]", member.GetRole())

		if _, ok := roleList[member.GetRole()]; ok {
			refinedEndorsements = append(refinedEndorsements, endorsement)
		} else {
			p.log.Debugf("authentication warning: signer's role [%v] is not permitted, requires [%v]",
				member.GetRole(), roleList)
		}
	}

	return refinedEndorsements
}

// GetHashAlg return hash algorithm the access control provider uses
func (p *pkACProvider) GetHashAlg() string {
	return p.hashType
}

// ValidateResourcePolicy checks whether the given resource principal is valid
func (p *pkACProvider) ValidateResourcePolicy(resourcePolicy *config.ResourcePolicy) bool {
	return true
}

// LookUpPolicy returns corresponding policy configured for the given resource name
func (p *pkACProvider) LookUpPolicy(resourceName string) (*pbac.Policy, error) {
	blockVersion, policyResourceName := getBlockVersionAndResourceName(resourceName)

	if blockVersion > 0 && blockVersion <= blockVersion220 {
		return p.lookUpPolicy220(policyResourceName)
	}

	if p, ok := p.latestPolicyMap.Load(policyResourceName); ok {
		return p.(*policy).GetPbPolicy(), nil
	}

	pol, ok := p.resourceNamePolicyMap.Load(policyResourceName)
	if !ok {
		return nil, fmt.Errorf("policy not found for resource %s", resourceName)
	}
	pbPolicy := pol.(*policy).GetPbPolicy()
	return pbPolicy, nil
}

//GetMemberStatus get the status information of the member
func (p *pkACProvider) GetMemberStatus(member *pbac.Member) (pbac.MemberStatus, error) {
	return pbac.MemberStatus_NORMAL, nil
}

//VerifyRelatedMaterial verify the member's relevant identity material
func (p *pkACProvider) VerifyRelatedMaterial(verifyType pbac.VerifyType, data []byte) (bool, error) {
	return true, nil
}

//GetAllPolicy returns all default policies
func (p *pkACProvider) GetAllPolicy() (map[string]*pbac.Policy, error) {
	var policyMap = make(map[string]*pbac.Policy)
	p.resourceNamePolicyMap.Range(func(key, value interface{}) bool {
		k, _ := key.(string)
		v, _ := value.(*policy)
		policyMap[k] = newPbPolicyFromPolicy(v)
		return true
	})
	p.senderPolicyMap.Range(func(key, value interface{}) bool {
		k, _ := key.(string)
		v, _ := value.(*policy)
		policyMap[k] = newPbPolicyFromPolicy(v)
		return true
	})
	return policyMap, nil
}

// VerifyPrincipalLT2330 verifies if the principal for the resource is met
func (pk *pkACProvider) VerifyPrincipalLT2330(principal protocol.Principal, blockVersion uint32) (bool, error) {

	if blockVersion <= blockVersion220 {
		return verifyPrincipal220(pk, principal)

	} else if blockVersion < blockVersion2330 {
		return verifyPrincipal2320(pk, principal)
	}

	return false, fmt.Errorf("`VerifyPrincipalLT2330` should not used by blockVersion(%d)", blockVersion)
}

//GetValidEndorsements filters all endorsement entries and returns all valid ones
func (pk *pkACProvider) GetValidEndorsements(
	principal protocol.Principal, blockVersion uint32) ([]*common.EndorsementEntry, error) {

	if blockVersion <= blockVersion220 {
		return pk.getValidEndorsements220(principal)
	}

	if blockVersion < blockVersion2330 {
		return pk.getValidEndorsements2320(principal)
	}
	return pk.getValidEndorsements(principal, blockVersion)
}

// VerifyMsgPrincipal verifies if the principal for the resource is met
func (p *pkACProvider) VerifyMsgPrincipal(principal protocol.Principal, blockVersion uint32) (bool, error) {
	if blockVersion <= blockVersion220 {
		return verifyPrincipal220(p, principal)
	}

	if blockVersion < blockVersion2330 {
		return verifyPrincipal2320(p, principal)
	}

	return verifyMsgTypePrincipal(p, principal, blockVersion)
}

// VerifyTxPrincipal verifies if the principal for the resource is met
func (p *pkACProvider) VerifyTxPrincipal(tx *common.Transaction,
	resourceName string, blockVersion uint32) (bool, error) {
	if blockVersion <= blockVersion220 {
		if err := verifyTxPrincipal220(tx, p); err != nil {
			return false, err
		}
		return true, nil
	}

	if blockVersion < blockVersion2330 {
		if err := verifyTxPrincipal2320(tx, resourceName, p); err != nil {
			return false, err
		}
		return true, nil
	}

	return verifyTxPrincipal(tx, resourceName, p, blockVersion)
}

// VerifyMultiSignTxPrincipal verify if the multi-sign tx should be finished
func (p *pkACProvider) VerifyMultiSignTxPrincipal(
	mInfo *syscontract.MultiSignInfo,
	blockVersion uint32) (syscontract.MultiSignStatus, error) {

	if blockVersion < blockVersion2330 {
		return mInfo.Status, fmt.Errorf(
			"func `verifyMultiSignTxPrincipal` cannot be used in blockVersion(%v)", blockVersion)
	}
	return verifyMultiSignTxPrincipal(p, mInfo, blockVersion, p.log)
}

// IsRuleSupportedByMultiSign verify the policy of resourceName is supported by multi-sign
// it's implements must be the same with vm-native/supportRule
func (p *pkACProvider) IsRuleSupportedByMultiSign(resourceName string, blockVersion uint32) error {
	if blockVersion < blockVersion220 {
		return isRuleSupportedByMultiSign220(p, resourceName, p.log)
	}

	if blockVersion < blockVersion2330 {
		return isRuleSupportedByMultiSign2320(resourceName, p, p.log)
	}

	return isRuleSupportedByMultiSign(p, resourceName, blockVersion, p.log)
}
