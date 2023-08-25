/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package accesscontrol

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"chainmaker.org/chainmaker/common/v3/concurrentlru"
	bcx509 "chainmaker.org/chainmaker/common/v3/crypto/x509"
	"chainmaker.org/chainmaker/localconf/v3"
	pbac "chainmaker.org/chainmaker/pb-go/v3/accesscontrol"
	"chainmaker.org/chainmaker/pb-go/v3/common"
	"chainmaker.org/chainmaker/pb-go/v3/config"
	"chainmaker.org/chainmaker/pb-go/v3/syscontract"
	"chainmaker.org/chainmaker/protocol/v3"
)

// Special characters allowed to define customized access rules
const (
	// LIMIT_DELIMITER for policy list 3/4
	LIMIT_DELIMITER = "/"

	// PARAM_CERTS param key
	PARAM_CERTS = "certs"

	// PARAM_CERTHASHES param key
	PARAM_CERTHASHES = "cert_hashes"

	// PARAM_ALIASES  param key
	PARAM_ALIASES = "aliases"

	// PARAM_ALIAS  param key
	PARAM_ALIAS = "alias"

	// PUBLIC_KEYS  param key
	PUBLIC_KEYS = "pubkey"

	unsupportedRuleErrorTemplate = "bad configuration: unsupported rule [%s]"
)

var notEnoughParticipantsSupportError = "authentication fail: not enough participants support this action"

// hsmHandleMap is a global handle map for pkcs11 or sdf hsm
var hsmHandleMap = map[string]interface{}{}

// restrainedResourceList List of access principals which should not be customized
var restrainedResourceList = map[string]bool{
	protocol.ResourceNameAllTest:       true,
	protocol.ResourceNameP2p:           true,
	protocol.ResourceNameConsensusNode: true,

	common.TxType_QUERY_CONTRACT.String():           true,
	common.TxType_INVOKE_CONTRACT.String():          true,
	common.TxType_SUBSCRIBE.String():                true,
	common.TxType_ARCHIVE.String():                  true,
	common.TxType_HOT_COLD_DATA_SEPARATION.String(): true,
	common.TxType_SNAPSHOT.String():                 true,
}

// predifined policies
var (
	policyRead = newPolicy(
		protocol.RuleAny,
		nil,
		[]protocol.Role{
			protocol.RoleConsensusNode,
			protocol.RoleCommonNode,
			protocol.RoleClient,
			protocol.RoleAdmin,
		},
	)

	policySpecialRead = newPolicy(
		protocol.RuleAny,
		nil,
		[]protocol.Role{
			protocol.RoleConsensusNode,
			protocol.RoleCommonNode,
			protocol.RoleClient,
			protocol.RoleAdmin,
			protocol.RoleLight,
		},
	)

	policySpecialWrite = newPolicy(
		protocol.RuleAny,
		nil,
		[]protocol.Role{
			protocol.RoleClient,
			protocol.RoleAdmin,
			protocol.RoleLight,
		},
	)

	policyWrite = newPolicy(
		protocol.RuleAny,
		nil,
		[]protocol.Role{
			protocol.RoleClient,
			protocol.RoleAdmin,
			protocol.RoleConsensusNode,
		},
	)

	policyConsensus = newPolicy(
		protocol.RuleAny,
		nil,
		[]protocol.Role{
			protocol.RoleConsensusNode,
		},
	)

	policyP2P = newPolicy(
		protocol.RuleAny,
		nil,
		[]protocol.Role{
			protocol.RoleConsensusNode,
			protocol.RoleCommonNode,
		},
	)
	policyAdmin = newPolicy(
		protocol.RuleAny,
		nil,
		[]protocol.Role{
			protocol.RoleAdmin,
		},
	)

	policySubscribe = newPolicy(
		protocol.RuleAny,
		nil,
		[]protocol.Role{
			protocol.RoleLight, protocol.RoleClient,
			protocol.RoleAdmin,
		},
	)

	policyArchive = newPolicy(
		protocol.RuleAny,
		nil,
		[]protocol.Role{
			protocol.RoleAdmin,
		},
	)

	policyHotColdDataSeparate = newPolicy(
		protocol.RuleAny,
		nil,
		[]protocol.Role{
			protocol.RoleAdmin,
		},
	)

	policySnapshot = newPolicy(
		protocol.RuleAny,
		nil,
		[]protocol.Role{
			protocol.RoleAdmin,
		},
	)

	policyConfig = newPolicy(
		protocol.RuleMajority,
		nil,
		[]protocol.Role{
			protocol.RoleAdmin,
		},
	)

	policySelfConfig = newPolicy(
		protocol.RuleSelf,
		nil,
		[]protocol.Role{
			protocol.RoleAdmin,
		},
	)

	policyForbidden = newPolicy(
		protocol.RuleForbidden,
		nil,
		nil)

	policyAllTest = newPolicy(
		protocol.RuleAll,
		nil,
		[]protocol.Role{
			protocol.RoleAdmin,
		},
	)

	policyLimitTestAny = newPolicy(
		"2",
		nil,
		nil,
	)

	policyLimitTestAdmin = newPolicy(
		"2",
		nil,
		[]protocol.Role{
			protocol.RoleAdmin,
		},
	)

	policyPortionTestAny = newPolicy(
		"3/4",
		nil,
		nil,
	)

	policyPortionTestAnyAdmin = newPolicy(
		"3/4",
		nil,
		[]protocol.Role{
			protocol.RoleAdmin,
		},
	)
)

//
//  accessControlService
//  @Description: ac provider's accessControl for cert and pwk mode
//
type accessControlService struct {
	orgNum int32

	// orgList map[string]interface{} , orgId -> interface{}
	orgList *sync.Map

	// resourceNamePolicyMap map[string]*policy , resourceName -> *policy
	resourceNamePolicyMap *sync.Map

	// exceptionalPolicyMap map[string]*policy , resourceName -> *policy
	exceptionalPolicyMap *sync.Map

	// lastestPolicyMap map[string]*policy , resourceName -> *policy
	lastestPolicyMap *sync.Map

	// resourceNamePolicyMap220	compatible to version 220
	resourceNamePolicyMap220 *sync.Map
	// exceptionalPolicyMap220	compatible to version 220
	exceptionalPolicyMap220 *sync.Map

	//memberCache local cache for member
	memberCache *concurrentlru.Cache

	// dataStore blockchain data store
	dataStore protocol.BlockchainStore

	log protocol.Logger

	// hashType hash algorithm for chains
	hashType string

	// authType ac authentication type
	authType string

	// pwkNewMember used to new member for pwk when memberCache is cleared concurrently
	pwkNewMember func(member *pbac.Member) (protocol.Member, error)

	// getCertVerifyOptions used to verify certificate chain when new member for cert mode
	getCertVerifyOptions func() *bcx509.VerifyOptions
}

//
//  memberCached
//  @Description: member cache struct
//
type memberCached struct {
	member    protocol.Member
	certChain []*bcx509.Certificate
}

//
// initAccessControlService
//  @Description: access control instantiate
//  @param hashType
//  @param authType
//  @param store
//  @param log
//  @return *accessControlService
//
func initAccessControlService(hashType, authType string,
	store protocol.BlockchainStore, log protocol.Logger) *accessControlService {
	acService := &accessControlService{
		orgNum:                   0,
		orgList:                  &sync.Map{},
		resourceNamePolicyMap:    &sync.Map{},
		exceptionalPolicyMap:     &sync.Map{},
		lastestPolicyMap:         &sync.Map{},
		resourceNamePolicyMap220: &sync.Map{},
		exceptionalPolicyMap220:  &sync.Map{},
		memberCache:              concurrentlru.New(localconf.ChainMakerConfig.NodeConfig.CertCacheSize),
		dataStore:                store,
		log:                      log,
		hashType:                 hashType,
		authType:                 authType,
	}
	return acService
}

//
// createDefaultResourcePolicy
//  @Description: create default resource policies for certificate mode!
//  @receiver acs
//  @param localOrgId
//
func (acs *accessControlService) createDefaultResourcePolicy(localOrgId string) {

	policyArchive.orgList = []string{localOrgId}

	acs.resourceNamePolicyMap.Store(protocol.ResourceNameReadData, policyRead)
	acs.resourceNamePolicyMap.Store(protocol.ResourceNameWriteData, policyWrite)
	acs.resourceNamePolicyMap.Store(protocol.ResourceNameUpdateSelfConfig, policySelfConfig)
	acs.resourceNamePolicyMap.Store(protocol.ResourceNameUpdateConfig, policyConfig)
	acs.resourceNamePolicyMap.Store(protocol.ResourceNameConsensusNode, policyConsensus)
	acs.resourceNamePolicyMap.Store(protocol.ResourceNameP2p, policyP2P)

	// for txtype
	acs.resourceNamePolicyMap.Store(common.TxType_QUERY_CONTRACT.String(), policyRead)
	acs.resourceNamePolicyMap.Store(common.TxType_INVOKE_CONTRACT.String(), policyWrite)
	acs.resourceNamePolicyMap.Store(common.TxType_SUBSCRIBE.String(), policySubscribe)
	acs.resourceNamePolicyMap.Store(common.TxType_ARCHIVE.String(), policyArchive)
	acs.resourceNamePolicyMap.Store(common.TxType_HOT_COLD_DATA_SEPARATION.String(), policyHotColdDataSeparate)
	acs.resourceNamePolicyMap.Store(common.TxType_SNAPSHOT.String(), policySnapshot)

	// exceptional resourceName opened for light user
	acs.exceptionalPolicyMap.Store(syscontract.SystemContract_CHAIN_QUERY.String()+"-"+
		syscontract.ChainQueryFunction_GET_BLOCK_BY_HEIGHT.String(), policySpecialRead)
	acs.exceptionalPolicyMap.Store(syscontract.SystemContract_CHAIN_QUERY.String()+"-"+
		syscontract.ChainQueryFunction_GET_BLOCK_WITH_TXRWSETS_BY_HEIGHT.String(), policySpecialRead)
	acs.exceptionalPolicyMap.Store(syscontract.SystemContract_CHAIN_QUERY.String()+"-"+
		syscontract.ChainQueryFunction_GET_BLOCK_BY_HASH.String(), policySpecialRead)
	acs.exceptionalPolicyMap.Store(syscontract.SystemContract_CHAIN_QUERY.String()+"-"+
		syscontract.ChainQueryFunction_GET_BLOCK_WITH_TXRWSETS_BY_HASH.String(), policySpecialRead)
	acs.exceptionalPolicyMap.Store(syscontract.SystemContract_CHAIN_QUERY.String()+"-"+
		syscontract.ChainQueryFunction_GET_BLOCK_BY_TX_ID.String(), policySpecialRead)
	acs.exceptionalPolicyMap.Store(syscontract.SystemContract_CHAIN_QUERY.String()+"-"+
		syscontract.ChainQueryFunction_GET_TX_BY_TX_ID.String(), policySpecialRead)
	acs.exceptionalPolicyMap.Store(syscontract.SystemContract_CHAIN_QUERY.String()+"-"+
		syscontract.ChainQueryFunction_GET_LAST_CONFIG_BLOCK.String(), policySpecialRead)
	acs.exceptionalPolicyMap.Store(syscontract.SystemContract_CHAIN_QUERY.String()+"-"+
		syscontract.ChainQueryFunction_GET_LAST_BLOCK.String(), policySpecialRead)
	acs.exceptionalPolicyMap.Store(syscontract.SystemContract_CHAIN_QUERY.String()+"-"+
		syscontract.ChainQueryFunction_GET_FULL_BLOCK_BY_HEIGHT.String(), policySpecialRead)
	acs.exceptionalPolicyMap.Store(syscontract.SystemContract_CHAIN_QUERY.String()+"-"+
		syscontract.ChainQueryFunction_GET_BLOCK_HEIGHT_BY_TX_ID.String(), policySpecialRead)
	acs.exceptionalPolicyMap.Store(syscontract.SystemContract_CHAIN_QUERY.String()+"-"+
		syscontract.ChainQueryFunction_GET_BLOCK_HEIGHT_BY_HASH.String(), policySpecialRead)
	acs.exceptionalPolicyMap.Store(syscontract.SystemContract_CHAIN_QUERY.String()+"-"+
		syscontract.ChainQueryFunction_GET_BLOCK_HEADER_BY_HEIGHT.String(), policySpecialRead)
	acs.exceptionalPolicyMap.Store(syscontract.SystemContract_CHAIN_QUERY.String()+"-"+
		syscontract.ChainQueryFunction_GET_ARCHIVED_BLOCK_HEIGHT.String(), policySpecialRead)
	acs.exceptionalPolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_GET_CHAIN_CONFIG.String(), policySpecialRead)
	acs.exceptionalPolicyMap.Store(syscontract.SystemContract_CERT_MANAGE.String()+"-"+
		syscontract.CertManageFunction_CERTS_QUERY.String(), policySpecialRead)
	acs.exceptionalPolicyMap.Store(syscontract.SystemContract_CERT_MANAGE.String()+"-"+
		syscontract.CertManageFunction_CERT_ADD.String(), policySpecialWrite)
	acs.exceptionalPolicyMap.Store(syscontract.SystemContract_CERT_MANAGE.String()+"-"+
		syscontract.CertManageFunction_CERTS_ALIAS_QUERY.String(), policySpecialRead)
	acs.exceptionalPolicyMap.Store(syscontract.SystemContract_CERT_MANAGE.String()+"-"+
		syscontract.CertManageFunction_CERT_ALIAS_ADD.String(), policySpecialWrite)

	// Disable pubkey management for cert mode
	acs.exceptionalPolicyMap.Store(syscontract.SystemContract_PUBKEY_MANAGE.String()+"-"+
		syscontract.PubkeyManageFunction_PUBKEY_ADD.String(), policyForbidden)
	acs.exceptionalPolicyMap.Store(syscontract.SystemContract_PUBKEY_MANAGE.String()+"-"+
		syscontract.PubkeyManageFunction_PUBKEY_DELETE.String(), policyForbidden)

	//for private compute
	acs.resourceNamePolicyMap.Store(protocol.ResourceNamePrivateCompute, policyWrite)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_PRIVATE_COMPUTE.String()+"-"+
		syscontract.PrivateComputeFunction_SAVE_CA_CERT.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_PRIVATE_COMPUTE.String()+"-"+
		syscontract.PrivateComputeFunction_SAVE_ENCLAVE_REPORT.String(), policyConfig)

	// system contract interface resource definitions
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_CORE_UPDATE.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_UPDATE_VERSION.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_BLOCK_UPDATE.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_UPDATE_VERSION.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_TRUST_ROOT_ADD.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_TRUST_ROOT_DELETE.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_TRUST_ROOT_UPDATE.String(), policySelfConfig)

	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_TRUST_MEMBER_ADD.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_TRUST_MEMBER_DELETE.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_TRUST_MEMBER_UPDATE.String(), policyConfig)

	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_NODE_ID_ADD.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_NODE_ID_DELETE.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_NODE_ID_UPDATE.String(), policySelfConfig)

	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_NODE_ORG_ADD.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_NODE_ORG_UPDATE.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_NODE_ORG_DELETE.String(), policyConfig)

	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_CONSENSUS_EXT_ADD.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_CONSENSUS_EXT_UPDATE.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_CONSENSUS_EXT_DELETE.String(), policyConfig)

	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_PERMISSION_ADD.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_PERMISSION_UPDATE.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_PERMISSION_DELETE.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_ALTER_ADDR_TYPE.String(), policyConfig)
	// add majority permission for gas enable/disable config under cert mode
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_ENABLE_OR_DISABLE_GAS.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_ACCOUNT_MANAGER.String()+"-"+
		syscontract.GasAccountFunction_SET_ADMIN.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_SET_ACCOUNT_MANAGER_ADMIN.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_SET_INVOKE_BASE_GAS.String(), policyConfig)
	// add majority permission for multi sign enable/disable enable_manual_run mode
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_MULTI_SIGN_ENABLE_MANUAL_RUN.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_CONSENSUS_SWITCH.String(), policyConfig)
	// for vm support list add new vm type
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_VM_SUPPORT_LIST_ADD.String(), policyConfig)

	acs.createDefaultContractManageResourcePolicy()

	// certificate management
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CERT_MANAGE.String()+"-"+
		syscontract.CertManageFunction_CERTS_FREEZE.String(), policyAdmin)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CERT_MANAGE.String()+"-"+
		syscontract.CertManageFunction_CERTS_UNFREEZE.String(), policyAdmin)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CERT_MANAGE.String()+"-"+
		syscontract.CertManageFunction_CERTS_DELETE.String(), policyAdmin)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CERT_MANAGE.String()+"-"+
		syscontract.CertManageFunction_CERTS_REVOKE.String(), policyAdmin)
	// for cert_alias
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CERT_MANAGE.String()+"-"+
		syscontract.CertManageFunction_CERT_ALIAS_UPDATE.String(), policyAdmin)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CERT_MANAGE.String()+"-"+
		syscontract.CertManageFunction_CERTS_ALIAS_DELETE.String(), policyAdmin)

	// for charge gas in optimize mode
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_ACCOUNT_MANAGER.String()+"-"+
		syscontract.GasAccountFunction_CHARGE_GAS_FOR_MULTI_ACCOUNT.String(), policyConsensus)
}

func (acs *accessControlService) createDefaultContractManageResourcePolicy() {
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CONTRACT_MANAGE.String()+"-"+
		syscontract.ContractManageFunction_INIT_CONTRACT.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CONTRACT_MANAGE.String()+"-"+
		syscontract.ContractManageFunction_UPGRADE_CONTRACT.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CONTRACT_MANAGE.String()+"-"+
		syscontract.ContractManageFunction_FREEZE_CONTRACT.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CONTRACT_MANAGE.String()+"-"+
		syscontract.ContractManageFunction_UNFREEZE_CONTRACT.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CONTRACT_MANAGE.String()+"-"+
		syscontract.ContractManageFunction_REVOKE_CONTRACT.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CONTRACT_MANAGE.String()+"-"+
		syscontract.ContractManageFunction_GRANT_CONTRACT_ACCESS.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CONTRACT_MANAGE.String()+"-"+
		syscontract.ContractManageFunction_REVOKE_CONTRACT_ACCESS.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CONTRACT_MANAGE.String()+"-"+
		syscontract.ContractManageFunction_VERIFY_CONTRACT_ACCESS.String(), policyConfig)
}

func (acs *accessControlService) createDefaultResourcePolicyForPK(localOrgId string) {

	policyArchive.orgList = []string{localOrgId}

	acs.resourceNamePolicyMap.Store(protocol.ResourceNameReadData, policyRead)
	acs.resourceNamePolicyMap.Store(protocol.ResourceNameWriteData, policyWrite)
	acs.resourceNamePolicyMap.Store(protocol.ResourceNameUpdateSelfConfig, policySelfConfig)
	acs.resourceNamePolicyMap.Store(protocol.ResourceNameUpdateConfig, policyConfig)
	acs.resourceNamePolicyMap.Store(protocol.ResourceNameConsensusNode, policyConsensus)
	acs.resourceNamePolicyMap.Store(protocol.ResourceNameP2p, policyP2P)

	// for txtype
	acs.resourceNamePolicyMap.Store(common.TxType_QUERY_CONTRACT.String(), policyRead)
	acs.resourceNamePolicyMap.Store(common.TxType_INVOKE_CONTRACT.String(), policyWrite)
	acs.resourceNamePolicyMap.Store(common.TxType_SUBSCRIBE.String(), policySubscribe)
	acs.resourceNamePolicyMap.Store(common.TxType_ARCHIVE.String(), policyArchive)
	acs.resourceNamePolicyMap.Store(common.TxType_HOT_COLD_DATA_SEPARATION.String(), policyHotColdDataSeparate)
	acs.resourceNamePolicyMap.Store(common.TxType_SNAPSHOT.String(), policySnapshot)

	// exceptional resourceName opened for light user
	acs.exceptionalPolicyMap.Store(syscontract.SystemContract_CHAIN_QUERY.String()+"-"+
		syscontract.ChainQueryFunction_GET_BLOCK_BY_HEIGHT.String(), policySpecialRead)
	acs.exceptionalPolicyMap.Store(syscontract.SystemContract_CHAIN_QUERY.String()+"-"+
		syscontract.ChainQueryFunction_GET_BLOCK_WITH_TXRWSETS_BY_HEIGHT.String(), policySpecialRead)
	acs.exceptionalPolicyMap.Store(syscontract.SystemContract_CHAIN_QUERY.String()+"-"+
		syscontract.ChainQueryFunction_GET_BLOCK_BY_HASH.String(), policySpecialRead)
	acs.exceptionalPolicyMap.Store(syscontract.SystemContract_CHAIN_QUERY.String()+"-"+
		syscontract.ChainQueryFunction_GET_BLOCK_WITH_TXRWSETS_BY_HASH.String(), policySpecialRead)
	acs.exceptionalPolicyMap.Store(syscontract.SystemContract_CHAIN_QUERY.String()+"-"+
		syscontract.ChainQueryFunction_GET_BLOCK_BY_TX_ID.String(), policySpecialRead)
	acs.exceptionalPolicyMap.Store(syscontract.SystemContract_CHAIN_QUERY.String()+"-"+
		syscontract.ChainQueryFunction_GET_TX_BY_TX_ID.String(), policySpecialRead)
	acs.exceptionalPolicyMap.Store(syscontract.SystemContract_CHAIN_QUERY.String()+"-"+
		syscontract.ChainQueryFunction_GET_LAST_CONFIG_BLOCK.String(), policySpecialRead)
	acs.exceptionalPolicyMap.Store(syscontract.SystemContract_CHAIN_QUERY.String()+"-"+
		syscontract.ChainQueryFunction_GET_LAST_BLOCK.String(), policySpecialRead)
	acs.exceptionalPolicyMap.Store(syscontract.SystemContract_CHAIN_QUERY.String()+"-"+
		syscontract.ChainQueryFunction_GET_FULL_BLOCK_BY_HEIGHT.String(), policySpecialRead)
	acs.exceptionalPolicyMap.Store(syscontract.SystemContract_CHAIN_QUERY.String()+"-"+
		syscontract.ChainQueryFunction_GET_BLOCK_HEIGHT_BY_TX_ID.String(), policySpecialRead)
	acs.exceptionalPolicyMap.Store(syscontract.SystemContract_CHAIN_QUERY.String()+"-"+
		syscontract.ChainQueryFunction_GET_BLOCK_HEIGHT_BY_HASH.String(), policySpecialRead)
	acs.exceptionalPolicyMap.Store(syscontract.SystemContract_CHAIN_QUERY.String()+"-"+
		syscontract.ChainQueryFunction_GET_BLOCK_HEADER_BY_HEIGHT.String(), policySpecialRead)
	acs.exceptionalPolicyMap.Store(syscontract.SystemContract_CHAIN_QUERY.String()+"-"+
		syscontract.ChainQueryFunction_GET_ARCHIVED_BLOCK_HEIGHT.String(), policySpecialRead)
	acs.exceptionalPolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_GET_CHAIN_CONFIG.String(), policySpecialRead)
	acs.exceptionalPolicyMap.Store(syscontract.SystemContract_CERT_MANAGE.String()+"-"+
		syscontract.CertManageFunction_CERTS_QUERY.String(), policySpecialRead)
	acs.exceptionalPolicyMap.Store(syscontract.SystemContract_CERT_MANAGE.String()+"-"+
		syscontract.CertManageFunction_CERT_ADD.String(), policySpecialWrite)

	// Disable certificate management for pk mode
	acs.exceptionalPolicyMap.Store(syscontract.SystemContract_CERT_MANAGE.String()+"-"+
		syscontract.CertManageFunction_CERT_ADD.String(), policyForbidden)
	acs.exceptionalPolicyMap.Store(syscontract.SystemContract_CERT_MANAGE.String()+"-"+
		syscontract.CertManageFunction_CERTS_FREEZE.String(), policyForbidden)
	acs.exceptionalPolicyMap.Store(syscontract.SystemContract_CERT_MANAGE.String()+"-"+
		syscontract.CertManageFunction_CERTS_UNFREEZE.String(), policyForbidden)
	acs.exceptionalPolicyMap.Store(syscontract.SystemContract_CERT_MANAGE.String()+"-"+
		syscontract.CertManageFunction_CERTS_DELETE.String(), policyForbidden)
	acs.exceptionalPolicyMap.Store(syscontract.SystemContract_CERT_MANAGE.String()+"-"+
		syscontract.CertManageFunction_CERTS_REVOKE.String(), policyForbidden)

	acs.exceptionalPolicyMap.Store(syscontract.SystemContract_CERT_MANAGE.String()+"-"+
		syscontract.CertManageFunction_CERT_ALIAS_ADD.String(), policyForbidden)
	acs.exceptionalPolicyMap.Store(syscontract.SystemContract_CERT_MANAGE.String()+"-"+
		syscontract.CertManageFunction_CERT_ALIAS_UPDATE.String(), policyForbidden)
	acs.exceptionalPolicyMap.Store(syscontract.SystemContract_CERT_MANAGE.String()+"-"+
		syscontract.CertManageFunction_CERTS_ALIAS_DELETE.String(), policyForbidden)

	// Disable trust member management for pk mode
	acs.exceptionalPolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_TRUST_MEMBER_ADD.String(), policyForbidden)
	acs.exceptionalPolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_TRUST_MEMBER_DELETE.String(), policyForbidden)
	acs.exceptionalPolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_TRUST_MEMBER_UPDATE.String(), policyForbidden)

	//for private compute
	acs.resourceNamePolicyMap.Store(protocol.ResourceNamePrivateCompute, policyWrite)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_PRIVATE_COMPUTE.String()+"-"+
		syscontract.PrivateComputeFunction_SAVE_CA_CERT.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_PRIVATE_COMPUTE.String()+"-"+
		syscontract.PrivateComputeFunction_SAVE_ENCLAVE_REPORT.String(), policyConfig)

	// system contract interface resource definitions
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_CORE_UPDATE.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_UPDATE_VERSION.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_BLOCK_UPDATE.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_UPDATE_VERSION.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_TRUST_ROOT_ADD.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_TRUST_ROOT_DELETE.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_TRUST_ROOT_UPDATE.String(), policySelfConfig)

	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_NODE_ID_ADD.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_NODE_ID_DELETE.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_NODE_ID_UPDATE.String(), policySelfConfig)

	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_NODE_ORG_ADD.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_NODE_ORG_UPDATE.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_NODE_ORG_DELETE.String(), policyConfig)

	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_CONSENSUS_EXT_ADD.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_CONSENSUS_EXT_UPDATE.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_CONSENSUS_EXT_DELETE.String(), policyConfig)

	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_PERMISSION_ADD.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_PERMISSION_UPDATE.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_PERMISSION_DELETE.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_ALTER_ADDR_TYPE.String(), policyConfig)
	// add majority permission for gas enable/disable config under pwk mode
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_ENABLE_OR_DISABLE_GAS.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_ACCOUNT_MANAGER.String()+"-"+
		syscontract.GasAccountFunction_SET_ADMIN.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_SET_ACCOUNT_MANAGER_ADMIN.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_SET_INVOKE_BASE_GAS.String(), policyConfig)
	// add majority permission for multi sign enable/disable enable_manual_run mode
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_MULTI_SIGN_ENABLE_MANUAL_RUN.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_CONSENSUS_SWITCH.String(), policyConfig)
	// for vm support list add new vm type
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_VM_SUPPORT_LIST_ADD.String(), policyConfig)

	acs.createDefaultContractManageResourcePolicyForPK()

	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_PUBKEY_MANAGE.String()+"-"+
		syscontract.PubkeyManageFunction_PUBKEY_ADD.String(), policySelfConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_PUBKEY_MANAGE.String()+"-"+
		syscontract.PubkeyManageFunction_PUBKEY_DELETE.String(), policySelfConfig)

	// for charging gas in optimize mode
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_ACCOUNT_MANAGER.String()+"-"+
		syscontract.GasAccountFunction_CHARGE_GAS_FOR_MULTI_ACCOUNT.String(), policyConsensus)
}

func (acs *accessControlService) createDefaultContractManageResourcePolicyForPK() {
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CONTRACT_MANAGE.String()+"-"+
		syscontract.ContractManageFunction_INIT_CONTRACT.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CONTRACT_MANAGE.String()+"-"+
		syscontract.ContractManageFunction_UPGRADE_CONTRACT.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CONTRACT_MANAGE.String()+"-"+
		syscontract.ContractManageFunction_FREEZE_CONTRACT.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CONTRACT_MANAGE.String()+"-"+
		syscontract.ContractManageFunction_UNFREEZE_CONTRACT.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CONTRACT_MANAGE.String()+"-"+
		syscontract.ContractManageFunction_REVOKE_CONTRACT.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CONTRACT_MANAGE.String()+"-"+
		syscontract.ContractManageFunction_GRANT_CONTRACT_ACCESS.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CONTRACT_MANAGE.String()+"-"+
		syscontract.ContractManageFunction_REVOKE_CONTRACT_ACCESS.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CONTRACT_MANAGE.String()+"-"+
		syscontract.ContractManageFunction_VERIFY_CONTRACT_ACCESS.String(), policyConfig) //
}

//
// initResourcePolicy
//  @Description: initialize resource policy when restart or handle chainconfig msg from msgbus
//  @receiver acs
//  @param resourcePolicies
//  @param localOrgId
//
func (acs *accessControlService) initResourcePolicy(resourcePolicies []*config.ResourcePolicy,
	localOrgId string) {
	authType := strings.ToLower(acs.authType)
	switch authType {
	case protocol.PermissionedWithCert, protocol.Identity:
		acs.createDefaultResourcePolicy(localOrgId)
	case protocol.PermissionedWithKey:
		acs.createDefaultResourcePolicyForPK(localOrgId)
	}
	lastestPolicyMap := &sync.Map{}
	for _, resourcePolicy := range resourcePolicies {
		if acs.validateResourcePolicy(resourcePolicy) {
			policy := newPolicyFromPb(resourcePolicy.Policy)
			lastestPolicyMap.Store(resourcePolicy.ResourceName, policy)
		}
	}
	acs.lastestPolicyMap = lastestPolicyMap
}

//
// checkResourcePolicyOrgList
//  @Description: check if resource policy valid
//  @receiver acs
//  @param policy
//  @return bool
//
func (acs *accessControlService) checkResourcePolicyOrgList(policy *pbac.Policy) bool {
	orgCheckList := map[string]bool{}
	for _, org := range policy.OrgList {
		if _, ok := acs.orgList.Load(org); !ok {
			acs.log.Errorf("bad configuration: configured organization list contains unknown organization [%s]", org)
			return false
		} else if _, alreadyIn := orgCheckList[org]; alreadyIn {
			acs.log.Errorf("bad configuration: duplicated entries [%s] in organization list", org)
			return false
		} else {
			orgCheckList[org] = true
		}
	}
	return true
}

//
// checkResourcePolicyRule
//  @Description: check resource policy roles
//  @receiver acs
//  @param resourcePolicy
//  @return bool
//
func (acs *accessControlService) checkResourcePolicyRule(resourcePolicy *config.ResourcePolicy) bool {
	switch resourcePolicy.Policy.Rule {
	case string(protocol.RuleAny), string(protocol.RuleAll), string(protocol.RuleForbidden):
		return true
	case string(protocol.RuleSelf):
		return acs.checkResourcePolicyRuleSelfCase(resourcePolicy)
	case string(protocol.RuleMajority):
		return acs.checkResourcePolicyRuleMajorityCase(resourcePolicy.Policy)
	case string(protocol.RuleDelete):
		acs.log.Debugf("delete policy configuration of %s", resourcePolicy.ResourceName)
		return true
	default:
		return acs.checkResourcePolicyRuleDefaultCase(resourcePolicy.Policy)
	}
}

// checkResourcePolicyRuleSelfCase
//  @Description: check resource policy of 'self'
//  @receiver acs
//  @param resourcePolicy
//  @return bool
//
func (acs *accessControlService) checkResourcePolicyRuleSelfCase(resourcePolicy *config.ResourcePolicy) bool {
	switch resourcePolicy.ResourceName {
	case syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_TRUST_ROOT_UPDATE.String(),
		syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
			syscontract.ChainConfigFunction_NODE_ID_UPDATE.String():
		return true
	default:
		acs.log.Errorf("bad configuration: the access rule of [%s] should not be [%s]", resourcePolicy.ResourceName,
			resourcePolicy.Policy.Rule)
		return false
	}
}

// checkResourcePolicyRuleMajorityCase
//  @Description: check resource policy for 'majority'
//  @receiver acs
//  @param policy
//  @return bool
//
func (acs *accessControlService) checkResourcePolicyRuleMajorityCase(policy *pbac.Policy) bool {
	if len(policy.OrgList) != int(atomic.LoadInt32(&acs.orgNum)) {
		acs.log.Warnf("[%s] rule considers all the organizations on the chain, any customized configuration for "+
			"organization list will be overridden, should use [Portion] rule for customized organization list",
			protocol.RuleMajority)
	}
	switch len(policy.RoleList) {
	case 0:
		acs.log.Warnf("role allowed in [%s] is [%s]", protocol.RuleMajority, protocol.RoleAdmin)
		return true
	case 1:
		if policy.RoleList[0] != string(protocol.RoleAdmin) {
			acs.log.Warnf("role allowed in [%s] is only [%s], [%s] will be overridden", protocol.RuleMajority,
				protocol.RoleAdmin, policy.RoleList[0])
		}
		return true
	default:
		acs.log.Warnf("role allowed in [%s] is only [%s], the other roles in the list will be ignored",
			protocol.RuleMajority, protocol.RoleAdmin)
		return true
	}
}

// checkResourcePolicyRuleDefaultCase
//  @Description: check resource policy of 'default' case
//  @receiver acs
//  @param policy
//  @return bool
//
func (acs *accessControlService) checkResourcePolicyRuleDefaultCase(policy *pbac.Policy) bool {
	nums := strings.Split(policy.Rule, LIMIT_DELIMITER)
	switch len(nums) {
	case 1:
		_, err := strconv.Atoi(nums[0])
		if err != nil {
			acs.log.Errorf(unsupportedRuleErrorTemplate, policy.Rule)
			return false
		}
		return true
	case 2:
		numerator, err := strconv.Atoi(nums[0])
		if err != nil {
			acs.log.Errorf(unsupportedRuleErrorTemplate, policy.Rule)
			return false
		}
		denominator, err := strconv.Atoi(nums[1])
		if err != nil {
			acs.log.Errorf(unsupportedRuleErrorTemplate, policy.Rule)
			return false
		}
		if numerator <= 0 || denominator <= 0 {
			acs.log.Errorf(unsupportedRuleErrorTemplate, policy.Rule)
			return false
		}
		return true
	default:
		acs.log.Errorf(unsupportedRuleErrorTemplate, policy.Rule)
		return false
	}
}

// lookUpMemberInCache
//  @Description: lookup member in cache by memberInfo
//  @receiver acs
//  @param memberInfo
//  @return *memberCached
//  @return bool
//
func (acs *accessControlService) lookUpMemberInCache(memberInfo string) (*memberCached, bool) {
	ret, ok := acs.memberCache.Get(memberInfo)
	if ok {
		return ret.(*memberCached), true
	}
	return nil, false
}

// addMemberToCache
//  @Description: add a member into cache
//  @receiver acs
//  @param memberInfo
//  @param member
//
func (acs *accessControlService) addMemberToCache(memberInfo string, member *memberCached) {
	acs.memberCache.Add(memberInfo, member)
}

// addOrg
//  @Description: add one org into orgList
//  @receiver acs
//  @param orgId
//  @param orgInfo
//
func (acs *accessControlService) addOrg(orgId string, orgInfo interface{}) {
	_, loaded := acs.orgList.LoadOrStore(orgId, orgInfo)
	if loaded {
		acs.orgList.Store(orgId, orgInfo)
	} else {
		acs.orgNum++
	}
}

// getOrgInfoByOrgId
//  @Description: get organization details by org id
//  @receiver acs
//  @param orgId
//  @return interface{}
//
func (acs *accessControlService) getOrgInfoByOrgId(orgId string) interface{} {
	orgInfo, ok := acs.orgList.Load(orgId)
	if !ok {
		return nil
	}
	return orgInfo
}

// getAllOrgInfos
//  @Description: get all organization details
//  @receiver acs
//  @return []interface{}
//
func (acs *accessControlService) getAllOrgInfos() []interface{} {
	orgInfos := make([]interface{}, 0)
	acs.orgList.Range(func(_, value interface{}) bool {
		orgInfos = append(orgInfos, value)
		return true
	})
	return orgInfos
}

// validateResourcePolicy
//  @Description: check if resource policy is valid
//  @receiver acs
//  @param resourcePolicy
//  @return bool
//
func (acs *accessControlService) validateResourcePolicy(resourcePolicy *config.ResourcePolicy) bool {
	if _, ok := restrainedResourceList[resourcePolicy.ResourceName]; ok {
		acs.log.Errorf("bad configuration: should not modify the access policy of the resource: %s",
			resourcePolicy.ResourceName)
		return false
	}

	if resourcePolicy.Policy == nil {
		acs.log.Errorf("bad configuration: access principle should not be nil when modifying access control configurations")
		return false
	}

	if !acs.checkResourcePolicyOrgList(resourcePolicy.Policy) {
		return false
	}

	return acs.checkResourcePolicyRule(resourcePolicy)
}

// createPrincipal
//  @Description: create a principal, used to valid hereafter
//  @receiver acs
//  @param resourceName
//  @param endorsements
//  @param message
//  @return protocol.Principal
//  @return error
//
func (acs *accessControlService) createPrincipal(resourceName string, endorsements []*common.EndorsementEntry,
	message []byte) (protocol.Principal, error) {

	if len(endorsements) == 0 || message == nil {
		return nil, fmt.Errorf("setup access control principal failed, a principal should contain valid (non-empty)" +
			" signer information, signature, and message")
	}
	if endorsements[0] == nil {
		return nil, fmt.Errorf("setup access control principal failed, signer-signature pair should not be nil")
	}
	return &principal{
		resourceName: resourceName,
		endorsement:  endorsements,
		message:      message,
		targetOrg:    "",
	}, nil
}

// createPrincipalForTargetOrg
//  @Description: create a principal for specific organization
//  @receiver acs
//  @param resourceName
//  @param endorsements
//  @param message
//  @param targetOrgId
//  @return protocol.Principal
//  @return error
//
func (acs *accessControlService) createPrincipalForTargetOrg(resourceName string,
	endorsements []*common.EndorsementEntry, message []byte, targetOrgId string) (protocol.Principal, error) {
	p, err := acs.createPrincipal(resourceName, endorsements, message)
	if err != nil {
		return nil, err
	}
	p.(*principal).targetOrg = targetOrgId
	return p, nil
}

// lookUpPolicyByResourceName
//  @Description: query resource policy by resource name
//  @receiver acs
//  @param resourceName
//  @return *policy
//  @return error
//
func (acs *accessControlService) lookUpPolicyByResourceName(resourceName string) (*policy, error) {
	blockVersion, policyResourceName := getBlockVersionAndResourceName(resourceName)
	resourceName = policyResourceName

	//  check block version
	// if blockerVersion <= 220, query policy from Policy220 map

	if blockVersion > 0 && blockVersion <= 220 {
		return acs.lookUpPolicyByResourceName220(resourceName)
	}

	// first, query from latest policyMap
	if p, ok := acs.lastestPolicyMap.Load(resourceName); ok {
		return p.(*policy), nil
	}
	// second, query from default resource policy map
	p, ok := acs.resourceNamePolicyMap.Load(resourceName)
	if !ok {

		// thirdly, query from default exceptional policy map
		if p, ok = acs.exceptionalPolicyMap.Load(resourceName); !ok {
			return nil, fmt.Errorf("look up access policy failed, did not configure access policy "+
				"for resource %s", resourceName)
		}
	}
	return p.(*policy), nil
}

// newCertMember
//  @Description: new member of certificate mode by protocol member
//  @receiver acs
//  @param pbMember
//  @return protocol.Member
//  @return error
//
func (acs *accessControlService) newCertMember(pbMember *pbac.Member) (protocol.Member, error) {
	return newCertMemberFromPb(pbMember, acs)
}

// newPkMember
//  @Description: new member of pwk mode by protocol member
//  @receiver acs
//  @param member
//  @param adminList
//  @param consensusList
//  @return protocol.Member
//  @return error
//
func (acs *accessControlService) newPkMember(member *pbac.Member, adminList,
	consensusList *sync.Map) (protocol.Member, error) {

	// check member cache
	memberCache := acs.getMemberFromCache(member)
	if memberCache != nil {
		return memberCache, nil
	}

	// new member if not found in cache
	pkMember, err := newPkMemberFromAcs(member, adminList, consensusList, acs)
	if err != nil {
		return nil, fmt.Errorf("new public key member failed: %s", err.Error())
	}

	// check member's org
	if pkMember.GetOrgId() != member.OrgId && member.OrgId != "" {
		return nil, fmt.Errorf("new public key member failed: member orgId does not match on chain")
	}
	cached := &memberCached{
		member:    pkMember,
		certChain: nil,
	}

	// add member to cache if a new one
	acs.addMemberToCache(string(member.MemberInfo), cached)
	return pkMember, nil
}

// newNodePkMember
//  @Description:  new node member of pwk mode by protocol member
//  @receiver acs
//  @param member
//  @param consensusList
//  @return protocol.Member
//  @return error
//
func (acs *accessControlService) newNodePkMember(member *pbac.Member,
	consensusList *sync.Map) (protocol.Member, error) {

	memberCache := acs.getMemberFromCache(member)
	if memberCache != nil {
		if memberCache.GetRole() != protocol.RoleConsensusNode &&
			memberCache.GetRole() != protocol.RoleCommonNode {
			return nil, fmt.Errorf("get member from cache, the public key is not a node member")
		}
		return memberCache, nil
	}
	pkMember, err := newPkNodeMemberFromAcs(member, consensusList, acs)
	if err != nil {
		return nil, err
	}
	if pkMember.GetOrgId() != member.OrgId && member.OrgId != "" {
		return nil, fmt.Errorf("new public key node member failed: member orgId does not match on chain")
	}
	cached := &memberCached{
		member:    pkMember,
		certChain: nil,
	}
	acs.addMemberToCache(string(member.MemberInfo), cached)
	return pkMember, nil
}

// getMemberFromCache
//  @Description: get member from cache, if not found, new it
//  @receiver acs
//  @param member
//  @return protocol.Member
//
func (acs *accessControlService) getMemberFromCache(member *pbac.Member) protocol.Member {
	// lookup in member cache
	cached, ok := acs.lookUpMemberInCache(string(member.MemberInfo))
	if ok {
		acs.log.Debugf("member found in local cache")
		if cached.member.GetOrgId() != member.OrgId {
			acs.log.Debugf("get member from cache failed: member orgId does not match on chain")
			return nil
		}
		return cached.member
	}

	//handle false positive when member cache is cleared
	var tmpMember protocol.Member
	var err error
	var certChains [][]*bcx509.Certificate
	// new member of cert mode
	if acs.authType == protocol.PermissionedWithCert {
		tmpMember, err = acs.newCertMember(member)
		certMember, ok := tmpMember.(*certificateMember)
		if !ok {
			return nil
		}
		certChains, err = certMember.cert.Verify(*acs.getCertVerifyOptions())
		if err != nil {
			acs.log.Debugf("certMember verify cert chain failed, err = %s", err.Error())
			return nil
		}
		if len(certChains) == 0 {
			acs.log.Debugf("certMember verify cert chain failed, len(certChains) = %d", len(certChains))
			return nil
		}

		// new member of pwk mode
	} else if acs.authType == protocol.PermissionedWithKey {
		tmpMember, err = acs.pwkNewMember(member)
	}
	if err != nil {
		acs.log.Debugf("new member failed, authType = %s, err = %s", acs.authType, err.Error())
		return nil
	}
	//add to cache
	if certChains != nil {
		cached = &memberCached{
			member:    tmpMember,
			certChain: certChains[0],
		}
	} else {
		cached = &memberCached{
			member:    tmpMember,
			certChain: nil,
		}
	}
	acs.addMemberToCache(string(member.MemberInfo), cached)

	return tmpMember
}

// verifyPrincipalPolicy
//  @Description: verify principal, not checking signature here
//  @receiver acs
//  @param principal
//  @param refinedPrincipal
//  @param p
//  @return bool
//  @return error
//
func (acs *accessControlService) verifyPrincipalPolicy(principal, refinedPrincipal protocol.Principal, p *policy) (
	bool, error) {
	endorsements := refinedPrincipal.GetEndorsement()
	rule := p.GetRule()

	switch rule {
	case protocol.RuleForbidden:
		return false, fmt.Errorf("authentication fail: [%s] is forbidden to access", refinedPrincipal.GetResourceName())
	case protocol.RuleMajority:
		return acs.verifyPrincipalPolicyRuleMajorityCase(p, endorsements)
	case protocol.RuleSelf:
		return acs.verifyPrincipalPolicyRuleSelfCase(principal.GetTargetOrgId(), endorsements)
	case protocol.RuleAny:
		return acs.verifyPrincipalPolicyRuleAnyCase(p, endorsements, principal.GetResourceName())
	case protocol.RuleAll:
		return acs.verifyPrincipalPolicyRuleAllCase(p, endorsements)
	default:
		return acs.verifyPrincipalPolicyRuleDefaultCase(p, endorsements)
	}
}

// verifyPrincipalPolicyRuleMajorityCase
//  @Description: verify  MajorityCase
//  @receiver acs
//  @param p
//  @param endorsements
//  @return bool
//  @return error
//
func (acs *accessControlService) verifyPrincipalPolicyRuleMajorityCase(p *policy,
	endorsements []*common.EndorsementEntry) (bool, error) {
	// notice: accept admin role only, and require majority of all the organizations on the chain
	role := protocol.RoleAdmin
	// orgList, _ := buildOrgListRoleListOfPolicyForVerifyPrincipal(p)

	// warning: majority keywork with non admin constraints
	/*
		if roleList[0] != protocol.protocol.RoleAdmin {
			return false, fmt.Errorf("authentication fail: MAJORITY keyword only allows admin role")
		}
	*/

	numOfValid := acs.countValidEndorsements(map[string]bool{}, map[protocol.Role]bool{role: true}, endorsements)

	if float64(numOfValid) > float64(acs.orgNum)/2.0 {
		return true, nil
	}
	return false, fmt.Errorf("%s: %d valid endorsements required, %d valid endorsements received",
		notEnoughParticipantsSupportError, int(float64(acs.orgNum)/2.0+1), numOfValid)
}

// verifyPrincipalPolicyRuleSelfCase
//  @Description: verify SelfCase
//  @receiver acs
//  @param targetOrg
//  @param endorsements
//  @return bool
//  @return error
//
func (acs *accessControlService) verifyPrincipalPolicyRuleSelfCase(targetOrg string,
	endorsements []*common.EndorsementEntry) (bool, error) {
	role := protocol.RoleAdmin
	if targetOrg == "" {
		return false, fmt.Errorf("authentication fail: SELF keyword requires the owner of the affected target")
	}
	for _, entry := range endorsements {
		if entry.Signer.OrgId != targetOrg {
			continue
		}

		member := acs.getMemberFromCache(entry.Signer)
		if member == nil {
			acs.log.Debugf(
				"authentication warning: the member is not in member cache, memberInfo[%s]",
				string(entry.Signer.MemberInfo))
			continue
		}

		if member.GetRole() == role {
			return true, nil
		}
	}
	return false, fmt.Errorf("authentication fail: target [%s] does not belong to the signer", targetOrg)
}

// verifyPrincipalPolicyRuleAnyCase
//  @Description: verify anycase
//  @receiver acs
//  @param p
//  @param endorsements
//  @param resourceName
//  @return bool
//  @return error
//
func (acs *accessControlService) verifyPrincipalPolicyRuleAnyCase(p *policy, endorsements []*common.EndorsementEntry,
	resourceName string) (bool, error) {
	orgList, roleList := buildOrgListRoleListOfPolicyForVerifyPrincipal(p)
	for _, endorsement := range endorsements {
		if len(orgList) > 0 {
			if _, ok := orgList[endorsement.Signer.OrgId]; !ok {
				acs.log.Debugf("authentication warning: signer's organization [%s] is not permitted, requires [%v]",
					endorsement.Signer.OrgId, p.GetOrgList())
				continue
			}
		}

		if len(roleList) == 0 {
			return true, nil
		}

		member := acs.getMemberFromCache(endorsement.Signer)
		if member == nil {
			acs.log.Debugf(
				"authentication warning: the member is not in member cache, memberInfo[%s]",
				string(endorsement.Signer.MemberInfo))
			continue
		}

		if _, ok := roleList[member.GetRole()]; ok {
			return true, nil
		}
		acs.log.Debugf("authentication warning: signer's role [%v] is not permitted, requires [%v]",
			member.GetRole(), p.GetRoleList())
	}

	return false, fmt.Errorf("authentication fail: signers do not meet the requirement (%s)",
		resourceName)
}

// verifyPrincipalPolicyRuleAllCase
//  @Description: verify allCase
//  @receiver acs
//  @param p
//  @param endorsements
//  @return bool
//  @return error
//
func (acs *accessControlService) verifyPrincipalPolicyRuleAllCase(p *policy, endorsements []*common.EndorsementEntry) (
	bool, error) {
	orgList, roleList := buildOrgListRoleListOfPolicyForVerifyPrincipal(p)
	numOfValid := acs.countValidEndorsements(orgList, roleList, endorsements)
	if len(orgList) <= 0 && numOfValid == int(atomic.LoadInt32(&acs.orgNum)) {
		return true, nil
	}
	if len(orgList) > 0 && numOfValid == len(orgList) {
		return true, nil
	}
	return false, fmt.Errorf("authentication fail: not all of the listed organtizations consend to this action")
}

// verifyPrincipalPolicyRuleDefaultCase
//  @Description: verify default case
//  @receiver acs
//  @param p
//  @param endorsements
//  @return bool
//  @return error
//
func (acs *accessControlService) verifyPrincipalPolicyRuleDefaultCase(p *policy,
	endorsements []*common.EndorsementEntry) (bool, error) {
	rule := p.GetRule()
	orgList, roleList := buildOrgListRoleListOfPolicyForVerifyPrincipal(p)
	nums := strings.Split(string(rule), LIMIT_DELIMITER)
	switch len(nums) {
	case 1:
		threshold, err := strconv.Atoi(nums[0])
		if err != nil {
			return false, fmt.Errorf("authentication fail: unrecognized rule, should be ANY, MAJORITY, ALL, " +
				"SELF, ac threshold (integer), or ac portion (fraction)")
		}

		numOfValid := acs.countValidEndorsements(orgList, roleList, endorsements)
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
			denominator = int(atomic.LoadInt32(&acs.orgNum))
		}

		numOfValid := acs.countValidEndorsements(orgList, roleList, endorsements)

		var numRequired float64
		if len(orgList) <= 0 {
			numRequired = float64(atomic.LoadInt32(&acs.orgNum)) * float64(numerator) / float64(denominator)
		} else {
			numRequired = float64(len(orgList)) * float64(numerator) / float64(denominator)
		}
		if float64(numOfValid) >= numRequired {
			return true, nil
		}
		return false, fmt.Errorf("%s: %f valid endorsements required, %d valid endorsements received",
			notEnoughParticipantsSupportError, numRequired, numOfValid)
	default:
		return false, fmt.Errorf("authentication fail: unrecognized principle type, should be ANY, MAJORITY, " +
			"ALL, SELF, an integer (Threshold), or ac fraction (Portion)")
	}
}

// countValidEndorsements
//  @Description: count valid endorsement or signature. omit invalid ones
//  @receiver acs
//  @param orgList
//  @param roleList
//  @param endorsements
//  @return int
//
func (acs *accessControlService) countValidEndorsements(orgList map[string]bool, roleList map[protocol.Role]bool,
	endorsements []*common.EndorsementEntry) int {
	refinedEndorsements := acs.getValidEndorsements(orgList, roleList, endorsements)
	return countOrgsFromEndorsements(refinedEndorsements)
}

// getValidEndorsements
//  @Description: get valid endorsers
//  @receiver acs
//  @param orgList
//  @param roleList
//  @param endorsements
//  @return []*common.EndorsementEntry
//
func (acs *accessControlService) getValidEndorsements(orgList map[string]bool, roleList map[protocol.Role]bool,
	endorsements []*common.EndorsementEntry) []*common.EndorsementEntry {
	var refinedEndorsements []*common.EndorsementEntry
	for _, endorsement := range endorsements {
		if len(orgList) > 0 {
			if _, ok := orgList[endorsement.Signer.OrgId]; !ok {
				acs.log.Debugf("authentication warning: signer's organization [%s] is not permitted, requires",
					endorsement.Signer.OrgId, orgList)
				continue
			}
		}

		if len(roleList) == 0 {
			refinedEndorsements = append(refinedEndorsements, endorsement)
			continue
		}

		member := acs.getMemberFromCache(endorsement.Signer)
		if member == nil {
			acs.log.Debugf(
				"authentication warning: the member is not in member cache, memberInfo[%s]",
				string(endorsement.Signer.MemberInfo))
			continue
		}

		isRoleMatching := isRoleMatching(member.GetRole(), roleList, &refinedEndorsements, endorsement)
		if !isRoleMatching {
			acs.log.Debugf(
				"authentication warning: signer's role [%v] is not permitted, requires [%v]",
				member.GetRole(),
				roleList,
			)
		}
	}

	return refinedEndorsements
}

// isRoleMatching
//  @Description: check if signer role is valid
//  @param signerRole
//  @param roleList
//  @param refinedEndorsements
//  @param endorsement
//  @return bool
//
func isRoleMatching(signerRole protocol.Role, roleList map[protocol.Role]bool,
	refinedEndorsements *[]*common.EndorsementEntry, endorsement *common.EndorsementEntry) bool {
	isRoleMatching := false
	if _, ok := roleList[signerRole]; ok {
		*refinedEndorsements = append(*refinedEndorsements, endorsement)
		isRoleMatching = true
	}
	return isRoleMatching
}

func countOrgsFromEndorsements(endorsements []*common.EndorsementEntry) int {
	mapOrg := map[string]int{}
	for _, endorsement := range endorsements {
		mapOrg[endorsement.Signer.OrgId]++
	}
	return len(mapOrg)
}

func buildOrgListRoleListOfPolicyForVerifyPrincipal(p *policy) (map[string]bool, map[protocol.Role]bool) {
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
	return orgList, roleList
}

// lookUpPolicy
//  @Description: looup latest policy from policy map
//  @receiver acs
//  @param resourceName
//  @return *pbac.Policy
//  @return error
//
func (acs *accessControlService) lookUpPolicy(resourceName string) (*pbac.Policy, error) {
	blockVersion, policyResourceName := getBlockVersionAndResourceName(resourceName)

	if blockVersion > 0 && blockVersion <= 220 {
		policy, err := acs.lookUpPolicy220(policyResourceName)
		if err != nil {
			return nil, err
		}
		return policy, nil
	}

	if p, ok := acs.lastestPolicyMap.Load(policyResourceName); ok {
		return p.(*policy).GetPbPolicy(), nil
	}
	if p, ok := acs.resourceNamePolicyMap.Load(policyResourceName); ok {
		return p.(*policy).GetPbPolicy(), nil
	}
	return nil, fmt.Errorf("policy not found for resource %s", policyResourceName)
}

// lookUpExceptionalPolicy
//  @Description: lookup policy from exceptional policy map
//  @receiver acs
//  @param resourceName
//  @return *pbac.Policy
//  @return error
//
func (acs *accessControlService) lookUpExceptionalPolicy(resourceName string) (*pbac.Policy, error) {
	blockVersion, policyResourceName := getBlockVersionAndResourceName(resourceName)

	if blockVersion > 0 && blockVersion <= 220 {
		return acs.lookUpExceptionalPolicy220(policyResourceName)
	}

	if p, ok := acs.exceptionalPolicyMap.Load(policyResourceName); ok {
		return p.(*policy).GetPbPolicy(), nil

	}
	return nil, fmt.Errorf("exceptional policy not found for resource %s", resourceName)
}

// setVerifyOptionsFunc used to set verifyOptionsFunc which will check if  certificate chain valid
func (acs *accessControlService) setVerifyOptionsFunc(verifyOptionsFunc func() *bcx509.VerifyOptions) {
	acs.getCertVerifyOptions = verifyOptionsFunc
}
