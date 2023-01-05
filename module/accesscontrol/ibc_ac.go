/*
Copyright (C) Beijing Advanced Innovation Center for Future Blockchain and Privacy Computing (未来区块链与隐
私计算⾼精尖创新中⼼). All rights reserved.
SPDX-License-Identifier: Apache-2.0
*/

package accesscontrol

import (
	"bytes"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"chainmaker.org/chainmaker/common/v2/crypto/asym/sm9"
	bcx509 "chainmaker.org/chainmaker/common/v2/crypto/x509"
	"chainmaker.org/chainmaker/common/v2/json"
	"chainmaker.org/chainmaker/common/v2/msgbus"
	"chainmaker.org/chainmaker/localconf/v2"
	pbac "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/config"
	"chainmaker.org/chainmaker/pb-go/v2/consensus"
	"chainmaker.org/chainmaker/pb-go/v2/consensus/maxbft"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"
	"chainmaker.org/chainmaker/protocol/v2"
	esm9 "github.com/emmansun/gmsm/sm9"
)

type ibcACProvider struct {
	acService *accessControlService

	// local cache for certificate revocation list and frozen list
	crl        sync.Map
	frozenList sync.Map

	// verification options for organization members
	opts bcx509.VerifyOptions

	localOrg *organization

	store protocol.BlockchainStore

	//consensus type
	consensusType consensus.ConsensusType

	//master public key
	ibcOrg *sync.Map
}

var _ protocol.AccessControlProvider = (*ibcACProvider)(nil)

var nilIBCACProvider ACProvider = (*ibcACProvider)(nil)

// NewACProvider 构造一个AccessControlProvider
// @param chainConf
// @param localOrgId
// @param store
// @param log
// @param msgBus
// @return protocol.AccessControlProvider
// @return error
func (ip *ibcACProvider) NewACProvider(chainConf protocol.ChainConf, localOrgId string,
	store protocol.BlockchainStore, log protocol.Logger, msgBus msgbus.MessageBus) (
	protocol.AccessControlProvider, error) {
	ibcACProvider, err := newIBCACProvider(chainConf.ChainConfig(), localOrgId, store, log)
	if err != nil {
		return nil, err
	}

	msgBus.Register(msgbus.ChainConfig, ibcACProvider)
	msgBus.Register(msgbus.CertManageCertsDelete, ibcACProvider)
	msgBus.Register(msgbus.CertManageCertsUnfreeze, ibcACProvider)
	msgBus.Register(msgbus.CertManageCertsFreeze, ibcACProvider)
	msgBus.Register(msgbus.CertManageCertsRevoke, ibcACProvider)
	msgBus.Register(msgbus.CertManageCertsAliasDelete, ibcACProvider)
	msgBus.Register(msgbus.CertManageCertsAliasUpdate, ibcACProvider)
	msgBus.Register(msgbus.MaxbftEpochConf, ibcACProvider)

	return ibcACProvider, nil
}

func newIBCACProvider(chainConfig *config.ChainConfig, localOrgId string,
	store protocol.BlockchainStore, log protocol.Logger) (*ibcACProvider, error) {
	ibcACProvider := &ibcACProvider{
		crl:        sync.Map{},
		frozenList: sync.Map{},
		opts: bcx509.VerifyOptions{
			Intermediates: bcx509.NewCertPool(),
			Roots:         bcx509.NewCertPool(),
		},
		localOrg: nil,
		store:    store,
		ibcOrg:   &sync.Map{},
	}

	var maxbftCfg *maxbft.GovernanceContract
	var err error
	ibcACProvider.consensusType = chainConfig.Consensus.Type
	if ibcACProvider.consensusType == consensus.ConsensusType_MAXBFT {
		maxbftCfg, err = ibcACProvider.loadChainConfigFromGovernance()
		if err != nil {
			return nil, err
		}
		//omit 1'st epoch, GovernanceContract don't save chainConfig in 1'st epoch
		if maxbftCfg != nil && maxbftCfg.ChainConfig != nil {
			chainConfig = maxbftCfg.ChainConfig
		}
	}
	log.DebugDynamic(func() string {
		return fmt.Sprintf("init ac from chainconfig: %+v", chainConfig)
	})

	ibcACProvider.acService = initAccessControlService(chainConfig.GetCrypto().Hash,
		chainConfig.AuthType, store, log)
	ibcACProvider.acService.setVerifyOptionsFunc(ibcACProvider.getVerifyOptions)

	err = ibcACProvider.initIBCMasterKey(chainConfig.IbcMasterKeys)
	if err != nil {
		return nil, err
	}

	err = ibcACProvider.initTrustRoots(chainConfig.TrustRoots, localOrgId)
	if err != nil {
		return nil, err
	}

	ibcACProvider.acService.initResourcePolicy(chainConfig.ResourcePolicies, localOrgId)
	ibcACProvider.acService.initResourcePolicy_220(chainConfig.ResourcePolicies, localOrgId)

	ibcACProvider.opts.KeyUsages = make([]x509.ExtKeyUsage, 1)
	ibcACProvider.opts.KeyUsages[0] = x509.ExtKeyUsageAny

	if ibcACProvider.consensusType == consensus.ConsensusType_MAXBFT && maxbftCfg != nil {
		err = ibcACProvider.updateFrozenAndCRL(maxbftCfg)
		if err != nil {
			return nil, err
		}
	} else {
		if err := ibcACProvider.loadCRL(); err != nil {
			return nil, err
		}
		if err := ibcACProvider.loadCertFrozenList(); err != nil {
			return nil, err
		}
	}
	return ibcACProvider, nil
}

func (ip *ibcACProvider) getVerifyOptions() *bcx509.VerifyOptions {
	return &ip.opts
}

func (ip *ibcACProvider) initTrustRoots(roots []*config.TrustRootConfig, localOrgId string) error {

	for _, orgRoot := range roots {
		org := &organization{
			id:                       orgRoot.OrgId,
			trustedRootCerts:         map[string]*bcx509.Certificate{},
			trustedIntermediateCerts: map[string]*bcx509.Certificate{},
		}
		for _, root := range orgRoot.Root {
			certificateChain, err := ip.buildCertificateChain(root, orgRoot.OrgId, org)
			if err != nil {
				return err
			}
			if certificateChain == nil || !certificateChain[len(certificateChain)-1].IsCA {
				return fmt.Errorf("the certificate configured as root for organization %s is not a CA certificate", orgRoot.OrgId)
			}
			org.trustedRootCerts[string(certificateChain[len(certificateChain)-1].Raw)] =
				certificateChain[len(certificateChain)-1]
			ip.opts.Roots.AddCert(certificateChain[len(certificateChain)-1])
			for i := 0; i < len(certificateChain); i++ {
				org.trustedIntermediateCerts[string(certificateChain[i].Raw)] = certificateChain[i]
				ip.opts.Intermediates.AddCert(certificateChain[i])
			}

			if len(org.trustedRootCerts) <= 0 {
				return fmt.Errorf(
					"setup organization failed, no trusted root (for %s): "+
						"please configure trusted root certificate or trusted public key whitelist",
					orgRoot.OrgId,
				)
			}
		}
		ip.acService.addOrg(orgRoot.OrgId, org)
	}

	localOrg := ip.acService.getOrgInfoByOrgId(localOrgId)
	if localOrg == nil {
		localOrg = &organization{
			id:                       localOrgId,
			trustedRootCerts:         map[string]*bcx509.Certificate{},
			trustedIntermediateCerts: map[string]*bcx509.Certificate{},
		}
	}
	ip.localOrg, _ = localOrg.(*organization)
	return nil
}

func (ip *ibcACProvider) buildCertificateChain(root, orgId string, org *organization) ([]*bcx509.Certificate, error) {

	var certificates, certificateChain []*bcx509.Certificate
	pemBlock, rest := pem.Decode([]byte(root))
	for pemBlock != nil {
		cert, errCert := bcx509.ParseCertificate(pemBlock.Bytes)
		if errCert != nil || cert == nil {
			return nil, fmt.Errorf("invalid entry int trusted root cert list")
		}
		if len(cert.Signature) == 0 {
			return nil, fmt.Errorf("invalid certificate [SN: %s]", cert.SerialNumber)
		}
		certificates = append(certificates, cert)
		pemBlock, rest = pem.Decode(rest)
	}
	certificateChain = bcx509.BuildCertificateChain(certificates)
	return certificateChain, nil
}

type ibcOrg struct {
	id   string
	mpks []*esm9.SignMasterPublicKey
}

func (ip *ibcACProvider) initIBCMasterKey(keyInOrgs []*config.IBCMasterKeyConfig) error {
	var newIBCOrg sync.Map
	for _, keyInOrg := range keyInOrgs {
		mpks := make([]*esm9.SignMasterPublicKey, 0, len(keyInOrg.MasterKeys))
		for _, mpkPEM := range keyInOrg.MasterKeys {
			mpk, err := sm9.MasterSignPubKeyFromPEM([]byte(mpkPEM))
			if err != nil {
				return err
			}
			mpks = append(mpks, mpk)
		}
		newIBCOrg.Store(keyInOrg.OrgId, &ibcOrg{
			id:   keyInOrg.OrgId,
			mpks: mpks,
		})
	}
	ip.ibcOrg = &newIBCOrg
	return nil
}

func (ip *ibcACProvider) loadCRL() error {
	if ip.acService.dataStore == nil {
		return nil
	}

	crlAKIList, err := ip.acService.dataStore.ReadObject(syscontract.SystemContract_CERT_MANAGE.String(),
		[]byte(protocol.CertRevokeKey))
	if err != nil {
		return fmt.Errorf("fail to update CRL list: %v", err)
	}
	if crlAKIList == nil {
		ip.acService.log.Debugf("empty CRL")
		return nil
	}

	var crlAKIs []string
	err = json.Unmarshal(crlAKIList, &crlAKIs)
	if err != nil {
		return fmt.Errorf("fail to update CRL list: %v", err)
	}

	err = ip.storeCrls(crlAKIs)
	return err
}

func (ip *ibcACProvider) storeCrls(crlAKIs []string) error {
	for _, crlAKI := range crlAKIs {
		crlbytes, err := ip.acService.dataStore.ReadObject(syscontract.SystemContract_CERT_MANAGE.String(), []byte(crlAKI))
		if err != nil {
			return fmt.Errorf("fail to load CRL [%s]: %v", hex.EncodeToString([]byte(crlAKI)), err)
		}
		if crlbytes == nil {
			return fmt.Errorf("fail to load CRL [%s]: CRL is nil", hex.EncodeToString([]byte(crlAKI)))
		}
		crls, err := ip.ValidateCRL(crlbytes)
		if err != nil {
			return err
		}
		if crls == nil {
			return fmt.Errorf("empty CRL")
		}

		for _, crl := range crls {
			aki, _, err := bcx509.GetAKIFromExtensions(crl.TBSCertList.Extensions)
			if err != nil {
				return fmt.Errorf("fail to load CRL, fail to get AKI from CRL: %v", err)
			}
			ip.crl.Store(string(aki), crl)
		}
	}
	return nil
}

//ValidateCRL validates whether the CRL is issued by a trusted CA
func (ip *ibcACProvider) ValidateCRL(crlBytes []byte) ([]*pkix.CertificateList, error) {
	crlPEM, rest := pem.Decode(crlBytes)
	if crlPEM == nil {
		return nil, fmt.Errorf("empty CRL")
	}
	var crls []*pkix.CertificateList
	orgInfos := ip.acService.getAllOrgInfos()
	for crlPEM != nil {
		crl, err := x509.ParseCRL(crlPEM.Bytes)
		if err != nil {
			return nil, fmt.Errorf("invalid CRL: %v\n[%s]", err, hex.EncodeToString(crlPEM.Bytes))
		}

		err = ip.validateCrlVersion(crlPEM.Bytes, crl)
		if err != nil {
			return nil, err
		}
		orgs := make([]*organization, 0)
		for _, org := range orgInfos {
			orgs = append(orgs, org.(*organization))
		}
		err1 := ip.checkCRLAgainstTrustedCerts(crl, orgs, false)
		err2 := ip.checkCRLAgainstTrustedCerts(crl, orgs, true)
		if err1 != nil && err2 != nil {
			return nil, fmt.Errorf("invalid CRL: \n\t[verification against trusted root certs: %v], \n\t["+
				"verification against trusted intermediate certs: %v]", err1, err2)
		}

		crls = append(crls, crl)
		crlPEM, rest = pem.Decode(rest)
	}
	return crls, nil
}

func (ip *ibcACProvider) validateCrlVersion(crlPemBytes []byte, crl *pkix.CertificateList) error {
	if ip.acService.dataStore != nil {
		aki, isASN1Encoded, err := bcx509.GetAKIFromExtensions(crl.TBSCertList.Extensions)
		if err != nil {
			return fmt.Errorf("invalid CRL: %v\n[%s]", err, hex.EncodeToString(crlPemBytes))
		}
		ip.acService.log.Debugf("AKI is ASN1 encoded: %v", isASN1Encoded)
		crlOldBytes, err := ip.acService.dataStore.ReadObject(syscontract.SystemContract_CERT_MANAGE.String(), aki)
		if err != nil {
			return fmt.Errorf("lookup CRL [%s] failed: %v", hex.EncodeToString(aki), err)
		}
		if crlOldBytes != nil {
			crlOldBlock, _ := pem.Decode(crlOldBytes)
			crlOld, err := x509.ParseCRL(crlOldBlock.Bytes)
			if err != nil {
				return fmt.Errorf("parse old CRL failed: %v", err)
			}
			if crlOld.TBSCertList.Version > crl.TBSCertList.Version {
				return fmt.Errorf("validate CRL failed: version of new CRL should be greater than the old one")
			}
		}
	}
	return nil
}

//check CRL against trusted certs
func (ip *ibcACProvider) checkCRLAgainstTrustedCerts(crl *pkix.CertificateList,
	orgList []*organization, isIntermediate bool) error {
	aki, isASN1Encoded, err := bcx509.GetAKIFromExtensions(crl.TBSCertList.Extensions)
	if err != nil {
		return fmt.Errorf("fail to get AKI of CRL [%s]: %v", crl.TBSCertList.Issuer.String(), err)
	}
	ip.acService.log.Debugf("AKI is ASN1 encoded: %v", isASN1Encoded)
	for _, org := range orgList {
		var targetCerts map[string]*bcx509.Certificate
		if !isIntermediate {
			targetCerts = org.trustedRootCerts
		} else {
			targetCerts = org.trustedIntermediateCerts
		}
		for _, cert := range targetCerts {
			if bytes.Equal(aki, cert.SubjectKeyId) {
				if err := cert.CheckCRLSignature(crl); err != nil {
					return fmt.Errorf("CRL [AKI: %s] is not signed by CA it claims: %v", hex.EncodeToString(aki), err)
				}
				return nil
			}
		}
	}
	return fmt.Errorf("CRL [AKI: %s] is not signed by ac trusted CA", hex.EncodeToString(aki))
}

var errCertRevoked = errors.New("certificate is revoked")

func (ip *ibcACProvider) checkCRL(certChain []*bcx509.Certificate) error {
	if len(certChain) < 1 {
		return fmt.Errorf("given certificate chain is empty")
	}

	for _, cert := range certChain {
		akiCert := cert.AuthorityKeyId

		crl, ok := ip.crl.Load(string(akiCert))
		if ok {
			// we have ac CRL, check whether the serial number is revoked
			for _, rc := range crl.(*pkix.CertificateList).TBSCertList.RevokedCertificates {
				if rc.SerialNumber.Cmp(cert.SerialNumber) == 0 {
					return errCertRevoked
				}
			}
		}
	}

	return nil
}

func (ip *ibcACProvider) loadCertFrozenList() error {
	if ip.acService.dataStore == nil {
		return nil
	}

	certList, err := ip.acService.dataStore.
		ReadObject(syscontract.SystemContract_CERT_MANAGE.String(),
			[]byte(protocol.CertFreezeKey))
	if err != nil {
		return fmt.Errorf("update frozen certificate list failed: %v", err)
	}
	if certList == nil {
		return nil
	}

	var certIDs []string
	err = json.Unmarshal(certList, &certIDs)
	if err != nil {
		return fmt.Errorf("update frozen certificate list failed: %v", err)
	}

	for _, certID := range certIDs {
		certBytes, err := ip.acService.dataStore.
			ReadObject(syscontract.SystemContract_CERT_MANAGE.String(), []byte(certID))
		if err != nil {
			return fmt.Errorf("load frozen certificate failed: %s", certID)
		}
		if certBytes == nil {
			return fmt.Errorf("load frozen certificate failed: empty certificate [%s]", certID)
		}

		certBlock, _ := pem.Decode(certBytes)
		ip.frozenList.Store(string(certBlock.Bytes), true)
	}
	return nil
}

var errCertFrozen = errors.New("certificate is frozen")

func (ip *ibcACProvider) checkCertFrozenList(certChain []*bcx509.Certificate) error {
	if len(certChain) < 1 {
		return fmt.Errorf("given certificate chain is empty")
	}
	if _, ok := ip.frozenList.Load(string(certChain[0].Raw)); ok {
		return errCertFrozen
	}
	return nil
}

// GetHashAlg return hash algorithm the access control provider uses
func (ip *ibcACProvider) GetHashAlg() string {
	return ip.acService.hashType
}

// NewMember 基于参数Member构建Member接口的实例
// @param pbMember
// @return protocol.Member
// @return error
func (ip *ibcACProvider) NewMember(pbMember *pbac.Member) (protocol.Member, error) {
	//New IBC Member
	if pbMember.MemberType == pbac.MemberType_IBC {

		tmpMember := ip.acService.getMemberFromCache(pbMember)
		ibcMem, ok := tmpMember.(*ibcMember)
		if !ok || ibcMem == nil {
			return nil, errors.New("new member failed")
		}

		if err := ip.verifyIBCMember(ibcMem); err != nil {
			return nil, errors.New("verify member failed: " + err.Error())
		}
		return ibcMem, nil
	}
	//New Cert Member
	var memberTmp *pbac.Member
	if pbMember.MemberType != pbac.MemberType_CERT {
		return nil, fmt.Errorf("new member failed: the member type does not match")
	}
	memberTmp = pbMember

	memberCache, ok := ip.acService.lookUpMemberInCache(string(memberTmp.MemberInfo))
	if !ok {
		remoteMember, isTrustMember, err := ip.newNoCacheMember(memberTmp)
		if err != nil {
			return nil, fmt.Errorf("new member failed: %s", err.Error())
		}

		var certChain []*bcx509.Certificate
		if !isTrustMember {
			certChain, err = ip.verifyMember(remoteMember)
			if err != nil {
				return nil, fmt.Errorf("new member failed: %s", err.Error())
			}
		}

		ip.acService.memberCache.Add(string(memberTmp.MemberInfo), &memberCached{
			member:    remoteMember,
			certChain: certChain,
		})
		return remoteMember, nil
	}
	return memberCache.member, nil

}

func (ip *ibcACProvider) newNoCacheMember(pbMember *pbac.Member) (member protocol.Member,
	isTrustMember bool, err error) {
	member, err = ip.acService.newCertMember(pbMember)
	if err != nil {
		return nil, isTrustMember, fmt.Errorf("new member failed: %s", err.Error())
	}
	return member, isTrustMember, nil
}

// ValidateResourcePolicy checks whether the given resource principal is valid
func (ip *ibcACProvider) ValidateResourcePolicy(resourcePolicy *config.ResourcePolicy) bool {
	return ip.acService.validateResourcePolicy(resourcePolicy)
}

// CreatePrincipalForTargetOrg creates a principal for "SELF" type principal,
// which needs to convert SELF to a specific organization id in one authentication
func (ip *ibcACProvider) CreatePrincipalForTargetOrg(resourceName string,
	endorsements []*common.EndorsementEntry, message []byte,
	targetOrgId string) (protocol.Principal, error) {
	return ip.acService.createPrincipalForTargetOrg(resourceName, endorsements, message, targetOrgId)
}

// CreatePrincipal creates a principal for one time authentication
func (ip *ibcACProvider) CreatePrincipal(resourceName string, endorsements []*common.EndorsementEntry,
	message []byte) (
	protocol.Principal, error) {
	return ip.acService.createPrincipal(resourceName, endorsements, message)
}

func (ip *ibcACProvider) LookUpPolicy(resourceName string) (*pbac.Policy, error) {
	return ip.acService.lookUpPolicy(resourceName)
}

func (ip *ibcACProvider) LookUpExceptionalPolicy(resourceName string) (*pbac.Policy, error) {
	return ip.acService.lookUpExceptionalPolicy(resourceName)
}

func (ip *ibcACProvider) GetMemberStatus(pbMember *pbac.Member) (pbac.MemberStatus, error) {

	_, err := ip.NewMember(pbMember)
	if err != nil {
		ip.acService.log.Infof("get member status: %s", err.Error())
		return pbac.MemberStatus_INVALID, err
	}

	return pbac.MemberStatus_NORMAL, nil
}

func (ip *ibcACProvider) VerifyRelatedMaterial(verifyType pbac.VerifyType, data []byte) (bool, error) {

	if verifyType != pbac.VerifyType_CRL {
		return false, fmt.Errorf("verify related material failed: only CRL allowed in permissionedWithCert mode")
	}

	crlPEM, rest := pem.Decode(data)
	if crlPEM == nil {
		ip.acService.log.Debug("verify member's related material failed: empty CRL")
		return false, fmt.Errorf("empty CRL")
	}
	orgInfos := ip.acService.getAllOrgInfos()

	var err1, err2 error

	for crlPEM != nil {
		crl, err := x509.ParseCRL(crlPEM.Bytes)
		if err != nil {
			return false, fmt.Errorf("invalid CRL: %v\n[%s]", err, hex.EncodeToString(crlPEM.Bytes))
		}

		err = ip.validateCrlVersion(crlPEM.Bytes, crl)
		if err != nil {
			return false, err
		}
		orgs := make([]*organization, 0)
		for _, org := range orgInfos {
			orgs = append(orgs, org.(*organization))
		}
		err1 = ip.checkCRLAgainstTrustedCerts(crl, orgs, false)
		err2 = ip.checkCRLAgainstTrustedCerts(crl, orgs, true)
		if err1 != nil && err2 != nil {
			return false, fmt.Errorf(
				"invalid CRL: \n\t[verification against trusted root certs: %v], "+
					"\n\t[verification against trusted intermediate certs: %v]",
				err1,
				err2,
			)
		}
		crlPEM, rest = pem.Decode(rest)
	}
	return true, nil
}

// VerifyPrincipal verifies if the principal for the resource is met
func (ip *ibcACProvider) VerifyPrincipal(principal protocol.Principal) (bool, error) {

	if atomic.LoadInt32(&ip.acService.orgNum) <= 0 {
		return false, fmt.Errorf("authentication failed: empty organization list or trusted node list on this chain")
	}

	refinedPrincipal, err := ip.refinePrincipal(principal)
	if err != nil {
		return false, fmt.Errorf("authentication failed, [%s]", err.Error())
	}

	if localconf.ChainMakerConfig.DebugConfig.IsSkipAccessControl {
		return true, nil
	}

	p, err := ip.acService.lookUpPolicyByResourceName(principal.GetResourceName())
	if err != nil {
		return false, fmt.Errorf("authentication failed, [%s]", err.Error())
	}

	return ip.acService.verifyPrincipalPolicy(principal, refinedPrincipal, p)
}

// all-in-one validation for signing members: certificate chain/whitelist, signature, policies
func (ip *ibcACProvider) refinePrincipal(principal protocol.Principal) (protocol.Principal, error) {
	endorsements := principal.GetEndorsement()
	msg := principal.GetMessage()
	refinedEndorsement := ip.refineEndorsements(endorsements, msg)
	if len(refinedEndorsement) <= 0 {
		return nil, fmt.Errorf("refine endorsements failed, all endorsers have failed verification")
	}

	refinedPrincipal, err := ip.CreatePrincipal(principal.GetResourceName(), refinedEndorsement, msg)
	if err != nil {
		return nil, fmt.Errorf("create principal failed: [%s]", err.Error())
	}

	return refinedPrincipal, nil
}

func (ip *ibcACProvider) refineEndorsements(endorsements []*common.EndorsementEntry,
	msg []byte) []*common.EndorsementEntry {

	refinedSigners := map[string]bool{}
	var refinedEndorsement []*common.EndorsementEntry
	var memInfo string

	for _, endorsementEntry := range endorsements {
		endorsement := &common.EndorsementEntry{
			Signer: &pbac.Member{
				OrgId:      endorsementEntry.Signer.OrgId,
				MemberInfo: endorsementEntry.Signer.MemberInfo,
				MemberType: endorsementEntry.Signer.MemberType,
			},
			Signature: endorsementEntry.Signature,
		}
		ip.acService.log.Debugf("target endorser uses IbcInfo")
		memInfo = string(endorsement.Signer.MemberInfo)

		member, err := newMemberFromIBCInfo(endorsementEntry.Signer.OrgId,
			ip.acService.hashType, endorsement.Signer.MemberInfo, false)
		if err != nil {
			ip.acService.log.Errorf("fail to new Member From IbcInfo [%s]", err)
			continue
		}
		if err = ip.verifyIBCMember(member); err != nil {
			ip.acService.log.Warnf("verify member [%s] failed: %+v", member.id, err)
			continue
		}
		err = member.Verify(ip.acService.hashType, msg, endorsementEntry.Signature)
		if err != nil {
			ip.acService.log.Warnf("verify principal signer failed [%s]", err)
			continue
		}
		if _, ok := refinedSigners[memInfo]; !ok {
			refinedSigners[memInfo] = true
			refinedEndorsement = append(refinedEndorsement, endorsement)
		}
	}
	return refinedEndorsement
}

// Check whether the provided member is a valid member of this group
func (ip *ibcACProvider) verifyMember(mem protocol.Member) ([]*bcx509.Certificate, error) {
	if mem == nil {
		return nil, fmt.Errorf("invalid member: member should not be nil")
	}
	certMember, ok := mem.(*certificateMember)
	if !ok {
		return nil, fmt.Errorf("invalid member: member type err")
	}

	orgIdFromCert := certMember.cert.Subject.Organization[0]
	org := ip.acService.getOrgInfoByOrgId(orgIdFromCert)

	// the Third-party CA
	if certMember.cert.IsCA && org == nil {
		ip.acService.log.Info("the Third-party CA verify the member")
		certChain := []*bcx509.Certificate{certMember.cert}
		err := ip.checkCRL(certChain)
		if err != nil {
			return nil, err
		}

		err = ip.checkCertFrozenList(certChain)
		if err != nil {
			return nil, err
		}

		return certChain, nil
	}

	if mem.GetOrgId() != orgIdFromCert {
		return nil, fmt.Errorf(
			"signer does not belong to the organization it claims [claim: %s, certificate: %s]",
			mem.GetOrgId(),
			orgIdFromCert,
		)
	}

	if org == nil {
		return nil, fmt.Errorf("no orgnization found")
	}

	certChains, err := certMember.cert.Verify(ip.opts)
	if err != nil {
		return nil, fmt.Errorf("not ac valid certificate from trusted CAs: %v", err)
	}

	if len(org.(*organization).trustedRootCerts) <= 0 {
		return nil, fmt.Errorf("no trusted root: please configure trusted root certificate")
	}

	certChain := ip.findCertChain(org.(*organization), certChains)
	if certChain != nil {
		return certChain, nil
	}
	return nil, fmt.Errorf("authentication failed, signer does not belong to the organization it claims"+
		" [claim: %s]", mem.GetOrgId())
}

func (ip *ibcACProvider) findCertChain(org *organization, certChains [][]*bcx509.Certificate) []*bcx509.Certificate {
	for _, chain := range certChains {
		rootCert := chain[len(chain)-1]
		_, ok := org.trustedRootCerts[string(rootCert.Raw)]
		if ok {
			var err error
			// check CRL and frozen list
			err = ip.checkCRL(chain)
			if err != nil {
				ip.acService.log.Warnf("authentication failed, CRL: %v", err)
				continue
			}
			err = ip.checkCertFrozenList(chain)
			if err != nil {
				ip.acService.log.Warnf("authentication failed, certificate frozen list: %v", err)
				continue
			}
			return chain
		}
	}
	return nil
}

func (ip *ibcACProvider) initTrustRootsForUpdatingChainConfig(chainConfig *config.ChainConfig,
	localOrgId string) error {

	var orgNum int32
	orgList := sync.Map{}
	opts := bcx509.VerifyOptions{
		Intermediates: bcx509.NewCertPool(),
		Roots:         bcx509.NewCertPool(),
	}
	for _, orgRoot := range chainConfig.TrustRoots {
		org := &organization{
			id:                       orgRoot.OrgId,
			trustedRootCerts:         map[string]*bcx509.Certificate{},
			trustedIntermediateCerts: map[string]*bcx509.Certificate{},
		}

		for _, root := range orgRoot.Root {
			certificateChain, err := ip.buildCertificateChain(root, orgRoot.OrgId, org)
			if err != nil {
				return err
			}
			for _, certificate := range certificateChain {
				if certificate.IsCA {
					org.trustedRootCerts[string(certificate.Raw)] = certificate
					opts.Roots.AddCert(certificate)
				} else {
					org.trustedIntermediateCerts[string(certificate.Raw)] = certificate
					opts.Intermediates.AddCert(certificate)
				}
			}

			if len(org.trustedRootCerts) <= 0 {
				return fmt.Errorf(
					"update configuration failed, no trusted root (for %s): "+
						"please configure trusted root certificate or trusted public key whitelist",
					orgRoot.OrgId,
				)
			}
		}
		orgList.Store(org.id, org)
		orgNum++
	}
	atomic.StoreInt32(&ip.acService.orgNum, orgNum)
	ip.acService.orgList = &orgList
	ip.opts = opts
	localOrg := ip.acService.getOrgInfoByOrgId(localOrgId)
	if localOrg == nil {
		localOrg = &organization{
			id:                       localOrgId,
			trustedRootCerts:         map[string]*bcx509.Certificate{},
			trustedIntermediateCerts: map[string]*bcx509.Certificate{},
		}
	}
	ip.localOrg, _ = localOrg.(*organization)
	return nil
}

//GetValidEndorsements filters all endorsement entries and returns all valid ones
func (ip *ibcACProvider) GetValidEndorsements(principal protocol.Principal) ([]*common.EndorsementEntry, error) {
	if atomic.LoadInt32(&ip.acService.orgNum) <= 0 {
		return nil, fmt.Errorf("authentication fail: empty organization list or trusted node list on this chain")
	}
	refinedPolicy, err := ip.refinePrincipal(principal)
	if err != nil {
		return nil, fmt.Errorf("authentication fail, not a member on this chain: [%v]", err)
	}
	endorsements := refinedPolicy.GetEndorsement()

	p, err := ip.acService.lookUpPolicyByResourceName(principal.GetResourceName())
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
	return ip.acService.getValidEndorsements(orgList, roleList, endorsements), nil
}

//GetAllPolicy returns all default policies
func (ip *ibcACProvider) GetAllPolicy() (map[string]*pbac.Policy, error) {
	var policyMap = make(map[string]*pbac.Policy)
	ip.acService.resourceNamePolicyMap.Range(func(key, value interface{}) bool {
		k, _ := key.(string)
		v, _ := value.(*policy)
		policyMap[k] = newPbPolicyFromPolicy(v)
		return true
	})
	ip.acService.exceptionalPolicyMap.Range(func(key, value interface{}) bool {
		k, _ := key.(string)
		v, _ := value.(*policy)
		policyMap[k] = newPbPolicyFromPolicy(v)
		return true
	})
	return policyMap, nil
}

func (ip *ibcACProvider) verifyIBCMember(mem *ibcMember) error {
	if mem == nil {
		return errors.New("member is nil")
	}
	ibcOrgV, ok := ip.ibcOrg.Load(mem.orgId)
	if !ok {
		return errors.New("org [" + mem.orgId + "] not found")
	}
	memIBCOrg, _ := ibcOrgV.(*ibcOrg)
	for _, mpk := range memIBCOrg.mpks {
		if mpk.MasterPublicKey.Equal(mem.ibcPK.MK.MasterPublicKey) {
			return nil
		}
	}
	return errors.New("member's organization not found")
}
