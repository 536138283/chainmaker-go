package accesscontrol

import (
	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"
	"chainmaker.org/chainmaker/protocol/v2"
)

var defaultResourcePolicy = map[string]*policy{
	protocol.ResourceNameReadData:         policyRead,
	protocol.ResourceNameWriteData:        policyWrite,
	protocol.ResourceNameUpdateSelfConfig: policySelfConfig,
	protocol.ResourceNameUpdateConfig:     policyConfig,
	protocol.ResourceNameConsensusNode:    policyConsensus,
	protocol.ResourceNameP2p:              policyP2P,

	// only used for test
	protocol.ResourceNameAllTest: policyAllTest,
	"test_2":                     policyLimitTestAny,
	"test_2_admin":               policyLimitTestAdmin,
	"test_3/4":                   policyPortionTestAny,
	"test_3/4_admin":             policyPortionTestAnyAdmin,

	// for txtype
	common.TxType_QUERY_CONTRACT.String():  policyRead,
	common.TxType_INVOKE_CONTRACT.String(): policyWrite,
	common.TxType_SUBSCRIBE.String():       policySubscribe,
	common.TxType_ARCHIVE.String():         policyArchive,

	//for private compute
	protocol.ResourceNamePrivateCompute: policyWrite,
	syscontract.SystemContract_PRIVATE_COMPUTE.String() + "-" +
		syscontract.PrivateComputeFunction_SAVE_CA_CERT.String(): policyConfig,
	syscontract.SystemContract_PRIVATE_COMPUTE.String() + "-" +
		syscontract.PrivateComputeFunction_SAVE_ENCLAVE_REPORT.String(): policyConfig,

	// system contract interface resource definitions
	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_CORE_UPDATE.String(): policyConfig,

	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_BLOCK_UPDATE.String(): policyConfig,

	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_TRUST_ROOT_ADD.String(): policyConfig,
	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_TRUST_ROOT_DELETE.String(): policyConfig,
	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_TRUST_ROOT_UPDATE.String(): policySelfConfig,

	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_TRUST_MEMBER_ADD.String(): policyConfig,
	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_TRUST_MEMBER_DELETE.String(): policyConfig,
	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_TRUST_MEMBER_UPDATE.String(): policyConfig,

	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_NODE_ID_ADD.String(): policyConfig,
	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_NODE_ID_DELETE.String(): policyConfig,
	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_NODE_ID_UPDATE.String(): policySelfConfig,

	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_NODE_ORG_ADD.String(): policyConfig,
	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_NODE_ORG_UPDATE.String(): policyConfig,
	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_NODE_ORG_DELETE.String(): policyConfig,

	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_CONSENSUS_EXT_ADD.String(): policyConfig,
	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_CONSENSUS_EXT_UPDATE.String(): policyConfig,
	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_CONSENSUS_EXT_DELETE.String(): policyConfig,

	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_PERMISSION_ADD.String(): policyConfig,
	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_PERMISSION_UPDATE.String(): policyConfig,
	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_PERMISSION_DELETE.String(): policyConfig,
	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_ALTER_ADDR_TYPE.String(): policyConfig,
	// add majority permission for gas enable/disable config under cert mode
	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_ENABLE_OR_DISABLE_GAS.String(): policyConfig,
	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_UPDATE_VERSION.String(): policyConfig,
	syscontract.SystemContract_CONTRACT_MANAGE.String() + "-" +
		syscontract.ContractManageFunction_INIT_CONTRACT.String(): policyConfig,
	syscontract.SystemContract_CONTRACT_MANAGE.String() + "-" +
		syscontract.ContractManageFunction_UPGRADE_CONTRACT.String(): policyConfig,
	syscontract.SystemContract_CONTRACT_MANAGE.String() + "-" +
		syscontract.ContractManageFunction_FREEZE_CONTRACT.String(): policyConfig,
	syscontract.SystemContract_CONTRACT_MANAGE.String() + "-" +
		syscontract.ContractManageFunction_UNFREEZE_CONTRACT.String(): policyConfig,
	syscontract.SystemContract_CONTRACT_MANAGE.String() + "-" +
		syscontract.ContractManageFunction_REVOKE_CONTRACT.String(): policyConfig,
	syscontract.SystemContract_CONTRACT_MANAGE.String() + "-" +
		syscontract.ContractManageFunction_GRANT_CONTRACT_ACCESS.String(): policyConfig,
	syscontract.SystemContract_CONTRACT_MANAGE.String() + "-" +
		syscontract.ContractManageFunction_REVOKE_CONTRACT_ACCESS.String(): policyConfig,
	syscontract.SystemContract_CONTRACT_MANAGE.String() + "-" +
		syscontract.ContractManageFunction_VERIFY_CONTRACT_ACCESS.String(): policyConfig,

	// certificate management
	syscontract.SystemContract_CERT_MANAGE.String() + "-" +
		syscontract.CertManageFunction_CERTS_FREEZE.String(): policyAdmin,
	syscontract.SystemContract_CERT_MANAGE.String() + "-" +
		syscontract.CertManageFunction_CERTS_UNFREEZE.String(): policyAdmin,
	syscontract.SystemContract_CERT_MANAGE.String() + "-" +
		syscontract.CertManageFunction_CERTS_DELETE.String(): policyAdmin,
	syscontract.SystemContract_CERT_MANAGE.String() + "-" +
		syscontract.CertManageFunction_CERTS_REVOKE.String(): policyAdmin,
	// for cert_alias
	syscontract.SystemContract_CERT_MANAGE.String() + "-" +
		syscontract.CertManageFunction_CERT_ALIAS_UPDATE.String(): policyAdmin,
	syscontract.SystemContract_CERT_MANAGE.String() + "-" +
		syscontract.CertManageFunction_CERTS_ALIAS_DELETE.String(): policyAdmin,

	// for charge gas in optimize mode
	syscontract.SystemContract_ACCOUNT_MANAGER.String() + "-" +
		syscontract.GasAccountFunction_CHARGE_GAS_FOR_MULTI_ACCOUNT.String(): policyConsensus,
	// for gas admin
	syscontract.SystemContract_ACCOUNT_MANAGER.String() + "-" +
		syscontract.GasAccountFunction_SET_ADMIN.String(): pubPolicyMajorityAdmin,
}

var defaultExpResourcePolicy = map[string]*policy{
	// exceptional resourceName opened for light user
	syscontract.SystemContract_CHAIN_QUERY.String() + "-" +
		syscontract.ChainQueryFunction_GET_BLOCK_BY_HEIGHT.String(): policySpecialRead,
	syscontract.SystemContract_CHAIN_QUERY.String() + "-" +
		syscontract.ChainQueryFunction_GET_BLOCK_WITH_TXRWSETS_BY_HEIGHT.String(): policySpecialRead,
	syscontract.SystemContract_CHAIN_QUERY.String() + "-" +
		syscontract.ChainQueryFunction_GET_BLOCK_BY_HASH.String(): policySpecialRead,
	syscontract.SystemContract_CHAIN_QUERY.String() + "-" +
		syscontract.ChainQueryFunction_GET_BLOCK_WITH_TXRWSETS_BY_HASH.String(): policySpecialRead,
	syscontract.SystemContract_CHAIN_QUERY.String() + "-" +
		syscontract.ChainQueryFunction_GET_BLOCK_BY_TX_ID.String(): policySpecialRead,
	syscontract.SystemContract_CHAIN_QUERY.String() + "-" +
		syscontract.ChainQueryFunction_GET_TX_BY_TX_ID.String(): policySpecialRead,
	syscontract.SystemContract_CHAIN_QUERY.String() + "-" +
		syscontract.ChainQueryFunction_GET_LAST_CONFIG_BLOCK.String(): policySpecialRead,
	syscontract.SystemContract_CHAIN_QUERY.String() + "-" +
		syscontract.ChainQueryFunction_GET_LAST_BLOCK.String(): policySpecialRead,
	syscontract.SystemContract_CHAIN_QUERY.String() + "-" +
		syscontract.ChainQueryFunction_GET_FULL_BLOCK_BY_HEIGHT.String(): policySpecialRead,
	syscontract.SystemContract_CHAIN_QUERY.String() + "-" +
		syscontract.ChainQueryFunction_GET_BLOCK_HEIGHT_BY_TX_ID.String(): policySpecialRead,
	syscontract.SystemContract_CHAIN_QUERY.String() + "-" +
		syscontract.ChainQueryFunction_GET_BLOCK_HEIGHT_BY_HASH.String(): policySpecialRead,
	syscontract.SystemContract_CHAIN_QUERY.String() + "-" +
		syscontract.ChainQueryFunction_GET_BLOCK_HEADER_BY_HEIGHT.String(): policySpecialRead,
	syscontract.SystemContract_CHAIN_QUERY.String() + "-" +
		syscontract.ChainQueryFunction_GET_ARCHIVED_BLOCK_HEIGHT.String(): policySpecialRead,
	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_GET_CHAIN_CONFIG.String(): policySpecialRead,
	syscontract.SystemContract_CERT_MANAGE.String() + "-" +
		syscontract.CertManageFunction_CERTS_QUERY.String(): policySpecialRead,
	syscontract.SystemContract_CERT_MANAGE.String() + "-" +
		syscontract.CertManageFunction_CERT_ADD.String(): policySpecialWrite,
	syscontract.SystemContract_CERT_MANAGE.String() + "-" +
		syscontract.CertManageFunction_CERTS_ALIAS_QUERY.String(): policySpecialRead,
	syscontract.SystemContract_CERT_MANAGE.String() + "-" +
		syscontract.CertManageFunction_CERT_ALIAS_ADD.String(): policySpecialWrite,

	// Disable pubkey management for cert mode
	syscontract.SystemContract_PUBKEY_MANAGE.String() + "-" +
		syscontract.PubkeyManageFunction_PUBKEY_ADD.String(): policyForbidden,
	syscontract.SystemContract_PUBKEY_MANAGE.String() + "-" +
		syscontract.PubkeyManageFunction_PUBKEY_DELETE.String(): policyForbidden,
}
