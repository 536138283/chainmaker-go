/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package accesscontrol

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"chainmaker.org/chainmaker/pb-go/v3/consensus"
	"chainmaker.org/chainmaker/pb-go/v3/consensus/maxbft"

	"chainmaker.org/chainmaker/common/v3/msgbus"

	"encoding/hex"

	"chainmaker.org/chainmaker/common/v3/crypto"
	"chainmaker.org/chainmaker/common/v3/crypto/asym"
	"chainmaker.org/chainmaker/localconf/v3"
	pbac "chainmaker.org/chainmaker/pb-go/v3/accesscontrol"

	"chainmaker.org/chainmaker/pb-go/v3/common"
	"chainmaker.org/chainmaker/pb-go/v3/config"
	"chainmaker.org/chainmaker/protocol/v3"
)

var _ protocol.AccessControlProvider = (*permissionedPkACProvider)(nil)

var nilPermissionedPkACProvider ACProvider = (*permissionedPkACProvider)(nil)

//  permissionedPkACProvider
//  @Description: ac provider for pwk mode
//
type permissionedPkACProvider struct {
	acService *accessControlService

	// local org id
	localOrg string

	// admin list in permissioned public key mode
	adminMember *sync.Map

	// consensus list in permissioned public key mode
	consensusMember *sync.Map

	//consensus type
	consensusType consensus.ConsensusType
}

type adminMemberModel struct {
	publicKey crypto.PublicKey
	pkBytes   []byte
	orgId     string
}

type consensusMemberModel struct {
	nodeId string
	orgId  string
}

//
// NewACProvider
//  @Description: instantiate pwk provider
//  @receiver pp
//  @param chainConf
//  @param localOrgId
//  @param store
//  @param log
//  @param msgBus
//  @return protocol.AccessControlProvider
//  @return error
//
func (pp *permissionedPkACProvider) NewACProvider(chainConf protocol.ChainConf, localOrgId string,
	store protocol.BlockchainStore, log protocol.Logger, msgBus msgbus.MessageBus) (
	protocol.AccessControlProvider, error) {
	pPkACProvider, err := newPermissionedPkACProvider(chainConf.ChainConfig(), localOrgId, store, log)
	if err != nil {
		return nil, err
	}
	msgBus.Register(msgbus.ChainConfig, pPkACProvider)
	msgBus.Register(msgbus.PubkeyManageDelete, pPkACProvider)
	msgBus.Register(msgbus.MaxbftEpochConf, pPkACProvider)
	// v220_compat Deprecated
	{
		chainConf.AddWatch(pPkACProvider)   //nolint: staticcheck
		chainConf.AddVmWatch(pPkACProvider) //nolint: staticcheck
	}
	return pPkACProvider, nil
}

func newPermissionedPkACProvider(chainConfig *config.ChainConfig, localOrgId string,
	store protocol.BlockchainStore, log protocol.Logger) (*permissionedPkACProvider, error) {
	ppacProvider := &permissionedPkACProvider{
		adminMember:     &sync.Map{},
		consensusMember: &sync.Map{},
		localOrg:        localOrgId,
	}

	var maxbftCfg *maxbft.GovernanceContract
	var err error
	ppacProvider.consensusType = chainConfig.Consensus.Type
	if ppacProvider.consensusType == consensus.ConsensusType_MAXBFT {
		maxbftCfg, err = loadChainConfigFromGovernance(store)
		if err != nil {
			return nil, err
		}
		//omit 1'st epoch, GovernanceContract don't save chainConfig in 1'st epoch
		if maxbftCfg != nil && maxbftCfg.ChainConfig != nil {
			chainConfig = maxbftCfg.ChainConfig
		}
	}

	chainConfig.AuthType = strings.ToLower(chainConfig.AuthType)
	ppacProvider.acService = initAccessControlService(chainConfig.GetCrypto().Hash,
		chainConfig.AuthType, store, log)
	ppacProvider.acService.pwkNewMember = ppacProvider.NewMemberFromAcs

	err = ppacProvider.initAdminMembers(chainConfig.TrustRoots)
	if err != nil {
		return nil, err
	}

	err = ppacProvider.initConsensusMember(chainConfig.Consensus.Nodes)
	if err != nil {
		return nil, err
	}

	ppacProvider.acService.initResourcePolicy(chainConfig.ResourcePolicies, localOrgId)
	ppacProvider.acService.initResourcePolicy_220(chainConfig.ResourcePolicies, localOrgId)

	return ppacProvider, nil
}

// initAdminMembers
//  @Description: init admin members of pwk mode
//  @receiver pp
//  @param trustRootList
//  @return error
//
func (pp *permissionedPkACProvider) initAdminMembers(trustRootList []*config.TrustRootConfig) error {
	var (
		tempSyncMap, orgList sync.Map
		orgNum               int32
	)

	// admin members in trustRoot, different from cert mode which is root certificate
	for _, trustRoot := range trustRootList {
		for _, root := range trustRoot.Root {
			pk, err := asym.PublicKeyFromPEM([]byte(root))
			if err != nil {
				return fmt.Errorf("init admin member failed: parse the public key from PEM failed")
			}

			pkBytes, err := pk.Bytes()
			if err != nil {
				return fmt.Errorf("init admin member failed: %s", err.Error())
			}

			adminMember := &adminMemberModel{
				publicKey: pk,
				pkBytes:   pkBytes,
				orgId:     trustRoot.OrgId,
			}
			adminKey := hex.EncodeToString(pkBytes)
			tempSyncMap.Store(adminKey, adminMember)
		}

		_, ok := orgList.Load(trustRoot.OrgId)
		if !ok {
			orgList.Store(trustRoot.OrgId, struct{}{})
			orgNum++
		}
	}
	atomic.StoreInt32(&pp.acService.orgNum, orgNum)
	pp.acService.orgList = &orgList
	pp.adminMember = &tempSyncMap
	return nil
}

// initConsensusMember
//  @Description: init consensus member of pwk mode
//  @receiver pp
//  @param consensusConf
//  @return error
//
func (pp *permissionedPkACProvider) initConsensusMember(consensusConf []*config.OrgConfig) error {
	var tempSyncMap sync.Map
	for _, conf := range consensusConf {
		for _, node := range conf.NodeId {

			consensusMember := &consensusMemberModel{
				nodeId: node,
				orgId:  conf.OrgId,
			}
			tempSyncMap.Store(node, consensusMember)
		}
	}
	pp.consensusMember = &tempSyncMap
	return nil
}

//
// refinePrincipal
//  @Description: // all-in-one validation for signing members: certificate chain/whitelist, signature, policies
//  @receiver pp
//  @param principal
//  @return protocol.Principal
//  @return error
func (pp *permissionedPkACProvider) refinePrincipal(principal protocol.Principal) (protocol.Principal, error) {
	endorsements := principal.GetEndorsement()
	msg := principal.GetMessage()
	refinedEndorsement := pp.RefineEndorsements(endorsements, msg)
	if len(refinedEndorsement) <= 0 {
		return nil, fmt.Errorf("refine endorsements failed, all endorsers have failed verification")
	}

	refinedPrincipal, err := pp.CreatePrincipal(principal.GetResourceName(), refinedEndorsement, msg)
	if err != nil {
		return nil, fmt.Errorf("create principal failed: [%s]", err.Error())
	}

	return refinedPrincipal, nil
}

// refineEndorsements
//  @Description: verify endorser's signatures
//  @receiver pp
//  @param endorsements
//  @param msg
//  @return []*common.EndorsementEntry
//
func (pp *permissionedPkACProvider) RefineEndorsements(endorsements []*common.EndorsementEntry,
	msg []byte) []*common.EndorsementEntry {

	refinedSigners := map[string]bool{}
	var refinedEndorsement []*common.EndorsementEntry

	for _, endorsementEntry := range endorsements {
		if endorsementEntry == nil || endorsementEntry.Signer == nil {
			continue
		}
		endorsement := &common.EndorsementEntry{
			Signer: &pbac.Member{
				OrgId:      endorsementEntry.Signer.OrgId,
				MemberInfo: endorsementEntry.Signer.MemberInfo,
				MemberType: endorsementEntry.Signer.MemberType,
			},
			Signature: endorsementEntry.Signature,
		}

		memInfo := string(endorsement.Signer.MemberInfo)

		remoteMember, err := pp.NewMember(endorsement.Signer)
		if err != nil {
			pp.acService.log.Infof("new member failed: [%s]", err.Error())
			continue
		}

		if err := remoteMember.Verify(pp.GetHashAlg(), msg, endorsement.Signature); err != nil {
			pp.acService.log.Infof("signer member verify signature failed: [%s]", err.Error())
			pp.acService.log.Debugf("information for invalid signature:\norganization: %s\npubkey: %s\nmessage: %s\n"+
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

// NewMember creates a member from pb Member
//  @Description:
//  @receiver pp
//  @param member
//  @return protocol.Member
//  @return error
//
func (pp *permissionedPkACProvider) NewMember(member *pbac.Member) (protocol.Member, error) {
	return pp.acService.newPkMember(member, pp.adminMember, pp.consensusMember)
}

// NewMemberFromAcs creates a member from pb Member
//  @Description:
//  @receiver pp
//  @param member
//  @return protocol.Member
//  @return error
//
func (pp *permissionedPkACProvider) NewMemberFromAcs(member *pbac.Member) (protocol.Member, error) {
	pkMember, err := newPkMemberFromAcs(member, pp.adminMember, pp.consensusMember, pp.acService)
	if err != nil {
		return nil, fmt.Errorf("new public key member failed: %s", err.Error())
	}
	if pkMember.GetOrgId() != member.OrgId && member.OrgId != "" {
		return nil, fmt.Errorf("new public key member failed: member orgId does not match on chain")
	}
	return pkMember, err
}

// GetHashAlg return hash algorithm the access control provider uses
//  @Description:
//  @receiver pp
//  @return string
//
func (pp *permissionedPkACProvider) GetHashAlg() string {
	return pp.acService.hashType
}

// ValidateResourcePolicy checks whether the given resource principal is valid
//  @Description:
//  @receiver pp
//  @param resourcePolicy
//  @return bool
//
func (pp *permissionedPkACProvider) ValidateResourcePolicy(resourcePolicy *config.ResourcePolicy) bool {
	return pp.acService.validateResourcePolicy(resourcePolicy)
}

// CreatePrincipalForTargetOrg creates a principal for "SELF" type principal,
// which needs to convert SELF to a sepecific organization id in one authentication
func (pp *permissionedPkACProvider) CreatePrincipalForTargetOrg(resourceName string,
	endorsements []*common.EndorsementEntry, message []byte,
	targetOrgId string) (protocol.Principal, error) {
	return pp.acService.createPrincipalForTargetOrg(resourceName, endorsements, message, targetOrgId)
}

// CreatePrincipal creates a principal for one time authentication
//  @Description:
//  @receiver pp
//  @param resourceName
//  @param endorsements
//  @param message
//  @return protocol.Principal
//  @return error
//
func (pp *permissionedPkACProvider) CreatePrincipal(resourceName string, endorsements []*common.EndorsementEntry,
	message []byte) (
	protocol.Principal, error) {
	return pp.acService.createPrincipal(resourceName, endorsements, message)
}

// LookUpPolicy returns corresponding policy configured for the given resource name
//  @Description:
//  @receiver pp
//  @param resourceName
//  @return *pbac.Policy
//  @return error
//
func (pp *permissionedPkACProvider) LookUpPolicy(resourceName string) (*pbac.Policy, error) {
	return pp.acService.lookUpPolicy(resourceName)
}

// LookUpExceptionalPolicy returns corresponding exceptional policy configured for the given resource name
//  @Description:
//  @receiver pp
//  @param resourceName
//  @return *pbac.Policy
//  @return error
//
func (pp *permissionedPkACProvider) LookUpExceptionalPolicy(resourceName string) (*pbac.Policy, error) {
	return pp.acService.lookUpExceptionalPolicy(resourceName)
}

// GetMemberStatus get the status information of the member
//  @Description:
//  @receiver pp
//  @param member
//  @return pbac.MemberStatus
//  @return error
//
func (pp *permissionedPkACProvider) GetMemberStatus(member *pbac.Member) (pbac.MemberStatus, error) {
	if _, err := pp.newNodeMember(member); err != nil {
		pp.acService.log.Infof("get member status: %s", err.Error())
		return pbac.MemberStatus_INVALID, err
	}
	return pbac.MemberStatus_NORMAL, nil
}

// VerifyRelatedMaterial verify the member's relevant identity material
//  @Description:
//  @receiver pp
//  @param verifyType
//  @param data
//  @return bool
//  @return error
//
func (pp *permissionedPkACProvider) VerifyRelatedMaterial(verifyType pbac.VerifyType, data []byte) (bool, error) {
	return true, nil
}

// VerifyPrincipal verifies if the principal for the resource is met
//  @Description:
//  @receiver pp
//  @param principal
//  @return bool
//  @return error
//
func (pp *permissionedPkACProvider) VerifyPrincipal(principal protocol.Principal) (bool, error) {

	if atomic.LoadInt32(&pp.acService.orgNum) <= 0 {
		return false, fmt.Errorf("authentication failed: empty organization list or trusted node list on this chain")
	}

	refinedPrincipal, err := pp.refinePrincipal(principal)
	if err != nil {
		return false, fmt.Errorf("authentication failed, [%s]", err.Error())
	}

	if localconf.ChainMakerConfig.DebugConfig.IsSkipAccessControl {
		return true, nil
	}

	p, err := pp.acService.lookUpPolicyByResourceName(principal.GetResourceName())
	if err != nil {
		return false, fmt.Errorf("authentication failed, [%s]", err.Error())
	}

	return pp.acService.verifyPrincipalPolicy(principal, refinedPrincipal, p)
}

// GetValidEndorsements filters all endorsement entries and returns all valid ones
//  @Description:
//  @receiver pp
//  @param principal
//  @return []*common.EndorsementEntry
//  @return error
//
func (pp *permissionedPkACProvider) GetValidEndorsements(principal protocol.Principal) (
	[]*common.EndorsementEntry, error) {
	if atomic.LoadInt32(&pp.acService.orgNum) <= 0 {
		return nil, fmt.Errorf("authentication fail: empty organization list or trusted node list on this chain")
	}
	refinedPolicy, err := pp.refinePrincipal(principal)
	if err != nil {
		return nil, fmt.Errorf("authentication fail, not a member on this chain: [%v]", err)
	}
	endorsements := refinedPolicy.GetEndorsement()

	p, err := pp.acService.lookUpPolicyByResourceName(principal.GetResourceName())
	if err != nil {
		return nil, fmt.Errorf("authentication fail: [%v]", err)
	}
	orgListRaw := p.GetOrgList()
	roleListRaw := p.GetRoleList()
	orgList := map[string]bool{}
	roleList := map[protocol.Role]bool{}
	for _, orgRaw := range orgListRaw {
		orgList[orgRaw] = true
	}
	for _, roleRaw := range roleListRaw {
		roleList[roleRaw] = true
	}
	return pp.acService.getValidEndorsements(orgList, roleList, endorsements), nil
}

// newNodeMember
//  @Description: init node member of pwk mode
//  @receiver pp
//  @param member
//  @return protocol.Member
//  @return error
//
func (pp *permissionedPkACProvider) newNodeMember(member *pbac.Member) (protocol.Member, error) {
	return pp.acService.newNodePkMember(member, pp.consensusMember)
}

// GetAllPolicy returns all default policies
//  @Description:
//  @receiver p
//  @return map[string]*pbac.Policy
//  @return error
//
func (p *permissionedPkACProvider) GetAllPolicy() (map[string]*pbac.Policy, error) {
	var policyMap = make(map[string]*pbac.Policy)
	p.acService.resourceNamePolicyMap.Range(func(key, value interface{}) bool {
		k, _ := key.(string)
		v, _ := value.(*policy)
		policyMap[k] = newPbPolicyFromPolicy(v)
		return true
	})
	p.acService.exceptionalPolicyMap.Range(func(key, value interface{}) bool {
		k, _ := key.(string)
		v, _ := value.(*policy)
		policyMap[k] = newPbPolicyFromPolicy(v)
		return true
	})
	return policyMap, nil
}
