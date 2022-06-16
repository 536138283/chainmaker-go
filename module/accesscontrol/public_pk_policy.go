package accesscontrol

import (
	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"
	"chainmaker.org/chainmaker/protocol/v2"
)

var pkResourcePolicy = map[string]*policy{

	protocol.ResourceNameConsensusNode: pubPolicyConsensus,
	// for txtype
	common.TxType_QUERY_CONTRACT.String():  pubPolicyTransaction,
	common.TxType_INVOKE_CONTRACT.String(): pubPolicyTransaction,
	common.TxType_SUBSCRIBE.String():       pubPolicyTransaction,
	common.TxType_ARCHIVE.String():         pubPolicyManage,

	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_NODE_ID_ADD.String(): pubPolicyMajorityAdmin,
	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_NODE_ID_DELETE.String(): pubPolicyMajorityAdmin,
	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_NODE_ID_UPDATE.String(): pubPolicyMajorityAdmin,
	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_NODE_ORG_UPDATE.String(): pubPolicyMajorityAdmin,

	syscontract.SystemContract_CONTRACT_MANAGE.String() + "-" +
		syscontract.ContractManageFunction_INIT_CONTRACT.String(): pubPolicyManage,
	syscontract.SystemContract_CONTRACT_MANAGE.String() + "-" +
		syscontract.ContractManageFunction_UPGRADE_CONTRACT.String(): pubPolicyManage,
	syscontract.SystemContract_CONTRACT_MANAGE.String() + "-" +
		syscontract.ContractManageFunction_FREEZE_CONTRACT.String(): pubPolicyManage,
	syscontract.SystemContract_CONTRACT_MANAGE.String() + "-" +
		syscontract.ContractManageFunction_UNFREEZE_CONTRACT.String(): pubPolicyManage,
	syscontract.SystemContract_CONTRACT_MANAGE.String() + "-" +
		syscontract.ContractManageFunction_REVOKE_CONTRACT.String(): pubPolicyManage,

	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_CORE_UPDATE.String(): pubPolicyMajorityAdmin,
	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_BLOCK_UPDATE.String(): pubPolicyMajorityAdmin,
	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_ENABLE_OR_DISABLE_GAS.String(): pubPolicyMajorityAdmin,
	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_ALTER_ADDR_TYPE.String(): pubPolicyMajorityAdmin,

	// for admin management
	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_TRUST_ROOT_UPDATE.String(): pubPolicyMajorityAdmin,

	// for gas admin
	syscontract.SystemContract_ACCOUNT_MANAGER.String() + "-" +
		syscontract.GasAccountFunction_SET_ADMIN.String(): pubPolicyMajorityAdmin,
}

var pkExpResourcePolicy = map[string]*policy{

	// exceptional resourceName
	protocol.ResourceNamePrivateCompute: pubPolicyForbidden,
	syscontract.SystemContract_PRIVATE_COMPUTE.String() + "-" +
		syscontract.PrivateComputeFunction_SAVE_CA_CERT.String(): pubPolicyForbidden,
	syscontract.SystemContract_PRIVATE_COMPUTE.String() + "-" +
		syscontract.PrivateComputeFunction_SAVE_ENCLAVE_REPORT.String(): pubPolicyForbidden,

	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_TRUST_MEMBER_ADD.String(): pubPolicyForbidden,
	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_TRUST_MEMBER_DELETE.String(): pubPolicyForbidden,
	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_TRUST_MEMBER_UPDATE.String(): pubPolicyForbidden,

	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_NODE_ORG_ADD.String(): pubPolicyForbidden,
	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_NODE_ORG_DELETE.String(): pubPolicyForbidden,

	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_CONSENSUS_EXT_ADD.String(): pubPolicyMajorityAdmin,
	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_CONSENSUS_EXT_UPDATE.String(): pubPolicyMajorityAdmin,
	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_CONSENSUS_EXT_DELETE.String(): pubPolicyMajorityAdmin,

	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_PERMISSION_ADD.String(): pubPolicyForbidden,
	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_PERMISSION_UPDATE.String(): pubPolicyForbidden,
	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_PERMISSION_DELETE.String(): pubPolicyForbidden,

	syscontract.SystemContract_CERT_MANAGE.String() + "-" +
		syscontract.CertManageFunction_CERT_ADD.String(): pubPolicyForbidden,
	syscontract.SystemContract_CERT_MANAGE.String() + "-" +
		syscontract.CertManageFunction_CERTS_FREEZE.String(): pubPolicyForbidden,
	syscontract.SystemContract_CERT_MANAGE.String() + "-" +
		syscontract.CertManageFunction_CERTS_UNFREEZE.String(): pubPolicyForbidden,
	syscontract.SystemContract_CERT_MANAGE.String() + "-" +
		syscontract.CertManageFunction_CERTS_DELETE.String(): pubPolicyForbidden,
	syscontract.SystemContract_CERT_MANAGE.String() + "-" +
		syscontract.CertManageFunction_CERTS_REVOKE.String(): pubPolicyForbidden,

	syscontract.SystemContract_CERT_MANAGE.String() + "-" +
		syscontract.CertManageFunction_CERT_ALIAS_ADD.String(): pubPolicyForbidden,
	syscontract.SystemContract_CERT_MANAGE.String() + "-" +
		syscontract.CertManageFunction_CERT_ALIAS_UPDATE.String(): pubPolicyForbidden,
	syscontract.SystemContract_CERT_MANAGE.String() + "-" +
		syscontract.CertManageFunction_CERTS_ALIAS_DELETE.String(): pubPolicyForbidden,

	syscontract.SystemContract_PUBKEY_MANAGE.String() + "-" +
		syscontract.PubkeyManageFunction_PUBKEY_ADD.String(): pubPolicyForbidden,
	syscontract.SystemContract_PUBKEY_MANAGE.String() + "-" +
		syscontract.PubkeyManageFunction_PUBKEY_DELETE.String(): pubPolicyForbidden,

	// disable trust root add & delete for public mode
	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_TRUST_ROOT_ADD.String(): pubPolicyForbidden,
	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_TRUST_ROOT_DELETE.String(): pubPolicyForbidden,

	// disable contract access for public mode
	syscontract.SystemContract_CONTRACT_MANAGE.String() + "-" +
		syscontract.ContractManageFunction_GRANT_CONTRACT_ACCESS.String(): pubPolicyForbidden,
	syscontract.SystemContract_CONTRACT_MANAGE.String() + "-" +
		syscontract.ContractManageFunction_REVOKE_CONTRACT_ACCESS.String(): pubPolicyForbidden,
	syscontract.SystemContract_CONTRACT_MANAGE.String() + "-" +
		syscontract.ContractManageFunction_VERIFY_CONTRACT_ACCESS.String(): pubPolicyForbidden,
	syscontract.SystemContract_CONTRACT_MANAGE.String() + "-" +
		syscontract.ContractQueryFunction_GET_DISABLED_CONTRACT_LIST.String(): pubPolicyForbidden,

	// forbidden charge gas by go sdk
	syscontract.SystemContract_ACCOUNT_MANAGER.String() + "-" +
		syscontract.GasAccountFunction_CHARGE_GAS.String(): pubPolicyForbidden,

	// forbidden refund gas vm by go sdk
	syscontract.SystemContract_ACCOUNT_MANAGER.String() + "-" +
		syscontract.GasAccountFunction_REFUND_GAS_VM.String(): pubPolicyForbidden,
}

var pkResourcePolicyForDPOS = map[string]*policy{

	protocol.ResourceNameConsensusNode: pubPolicyConsensus,
	// for txtype
	common.TxType_QUERY_CONTRACT.String():  pubPolicyTransaction,
	common.TxType_INVOKE_CONTRACT.String(): pubPolicyTransaction,
	common.TxType_SUBSCRIBE.String():       pubPolicyTransaction,
	common.TxType_ARCHIVE.String():         pubPolicyManage,

	// for admin management
	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_TRUST_ROOT_UPDATE.String(): pubPolicyMajorityAdmin,
}

var pkExpResourcePolicyForDPOS = map[string]*policy{
	// exceptional resourceName
	protocol.ResourceNamePrivateCompute: pubPolicyForbidden,
	syscontract.SystemContract_PRIVATE_COMPUTE.String() + "-" +
		syscontract.PrivateComputeFunction_SAVE_CA_CERT.String(): pubPolicyForbidden,
	syscontract.SystemContract_PRIVATE_COMPUTE.String() + "-" +
		syscontract.PrivateComputeFunction_SAVE_ENCLAVE_REPORT.String(): pubPolicyForbidden,

	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_TRUST_MEMBER_ADD.String(): pubPolicyForbidden,
	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_TRUST_MEMBER_DELETE.String(): pubPolicyForbidden,
	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_TRUST_MEMBER_UPDATE.String(): pubPolicyForbidden,

	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_NODE_ID_ADD.String(): pubPolicyForbidden,
	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_NODE_ID_DELETE.String(): pubPolicyForbidden,
	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_NODE_ID_UPDATE.String(): pubPolicyForbidden,

	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_NODE_ORG_ADD.String(): pubPolicyForbidden,
	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_NODE_ORG_UPDATE.String(): pubPolicyForbidden,
	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_NODE_ORG_DELETE.String(): pubPolicyForbidden,

	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_CONSENSUS_EXT_ADD.String(): pubPolicyMajorityAdmin,
	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_CONSENSUS_EXT_UPDATE.String(): pubPolicyMajorityAdmin,
	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_CONSENSUS_EXT_DELETE.String(): pubPolicyMajorityAdmin,

	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_PERMISSION_ADD.String(): pubPolicyForbidden,
	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_PERMISSION_UPDATE.String(): pubPolicyForbidden,
	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_PERMISSION_DELETE.String(): pubPolicyForbidden,

	syscontract.SystemContract_CERT_MANAGE.String() + "-" +
		syscontract.CertManageFunction_CERT_ADD.String(): pubPolicyForbidden,
	syscontract.SystemContract_CERT_MANAGE.String() + "-" +
		syscontract.CertManageFunction_CERTS_FREEZE.String(): pubPolicyForbidden,
	syscontract.SystemContract_CERT_MANAGE.String() + "-" +
		syscontract.CertManageFunction_CERTS_UNFREEZE.String(): pubPolicyForbidden,
	syscontract.SystemContract_CERT_MANAGE.String() + "-" +
		syscontract.CertManageFunction_CERTS_DELETE.String(): pubPolicyForbidden,
	syscontract.SystemContract_CERT_MANAGE.String() + "-" +
		syscontract.CertManageFunction_CERTS_REVOKE.String(): pubPolicyForbidden,

	syscontract.SystemContract_CERT_MANAGE.String() + "-" +
		syscontract.CertManageFunction_CERT_ALIAS_ADD.String(): pubPolicyForbidden,
	syscontract.SystemContract_CERT_MANAGE.String() + "-" +
		syscontract.CertManageFunction_CERT_ALIAS_UPDATE.String(): pubPolicyForbidden,
	syscontract.SystemContract_CERT_MANAGE.String() + "-" +
		syscontract.CertManageFunction_CERTS_ALIAS_DELETE.String(): pubPolicyForbidden,

	syscontract.SystemContract_PUBKEY_MANAGE.String() + "-" +
		syscontract.PubkeyManageFunction_PUBKEY_ADD.String(): pubPolicyForbidden,
	syscontract.SystemContract_PUBKEY_MANAGE.String() + "-" +
		syscontract.PubkeyManageFunction_PUBKEY_DELETE.String(): pubPolicyForbidden,

	// disable trust root add & delete for public mode
	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_TRUST_ROOT_ADD.String(): pubPolicyForbidden,
	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_TRUST_ROOT_DELETE.String(): pubPolicyForbidden,

	// disable multisign for public mode
	syscontract.SystemContract_MULTI_SIGN.String() + "-" +
		syscontract.MultiSignFunction_REQ.String(): pubPolicyForbidden,
	syscontract.SystemContract_MULTI_SIGN.String() + "-" +
		syscontract.MultiSignFunction_VOTE.String(): pubPolicyForbidden,
	syscontract.SystemContract_MULTI_SIGN.String() + "-" +
		syscontract.MultiSignFunction_QUERY.String(): pubPolicyForbidden,

	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_CORE_UPDATE.String(): pubPolicyManage,
	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_BLOCK_UPDATE.String(): pubPolicyManage,

	syscontract.SystemContract_CONTRACT_MANAGE.String() + "-" +
		syscontract.ContractManageFunction_UPGRADE_CONTRACT.String(): pubPolicyManage,
	syscontract.SystemContract_CONTRACT_MANAGE.String() + "-" +
		syscontract.ContractManageFunction_FREEZE_CONTRACT.String(): pubPolicyManage,
	syscontract.SystemContract_CONTRACT_MANAGE.String() + "-" +
		syscontract.ContractManageFunction_UNFREEZE_CONTRACT.String(): pubPolicyManage,
	syscontract.SystemContract_CONTRACT_MANAGE.String() + "-" +
		syscontract.ContractManageFunction_REVOKE_CONTRACT.String(): pubPolicyManage,
	// disable contract access for public mode
	syscontract.SystemContract_CONTRACT_MANAGE.String() + "-" +
		syscontract.ContractManageFunction_GRANT_CONTRACT_ACCESS.String(): pubPolicyForbidden,
	syscontract.SystemContract_CONTRACT_MANAGE.String() + "-" +
		syscontract.ContractManageFunction_REVOKE_CONTRACT_ACCESS.String(): pubPolicyForbidden,
	syscontract.SystemContract_CONTRACT_MANAGE.String() + "-" +
		syscontract.ContractManageFunction_VERIFY_CONTRACT_ACCESS.String(): pubPolicyForbidden,
	syscontract.SystemContract_CONTRACT_MANAGE.String() + "-" +
		syscontract.ContractQueryFunction_GET_DISABLED_CONTRACT_LIST.String(): pubPolicyForbidden,

	// disable gas related native contract
	//syscontract.SystemContract_ACCOUNT_MANAGER.String()+"-"+
	//	syscontract.GasAccountFunction_CHARGE_GAS.String(): pubPolicyForbidden,
	syscontract.SystemContract_ACCOUNT_MANAGER.String() + "-" +
		syscontract.GasAccountFunction_REFUND_GAS_VM.String(): pubPolicyForbidden,
	syscontract.SystemContract_ACCOUNT_MANAGER.String() + "-" +
		syscontract.GasAccountFunction_SET_ADMIN.String(): pubPolicyForbidden,
	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_ENABLE_OR_DISABLE_GAS.String(): pubPolicyForbidden,
	syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_ALTER_ADDR_TYPE.String(): pubPolicyForbidden,
}
