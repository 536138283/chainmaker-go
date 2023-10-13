package accesscontrol

import (
	"encoding/hex"
	"fmt"
	"sync"

	"chainmaker.org/chainmaker/common/v3/crypto"
	"chainmaker.org/chainmaker/common/v3/crypto/asym"
	"chainmaker.org/chainmaker/pb-go/v3/accesscontrol"
	acPb "chainmaker.org/chainmaker/pb-go/v3/accesscontrol"
	commonPb "chainmaker.org/chainmaker/pb-go/v3/common"
	configPb "chainmaker.org/chainmaker/pb-go/v3/config"
	"chainmaker.org/chainmaker/pb-go/v3/syscontract"
	"chainmaker.org/chainmaker/protocol/v3"
	"chainmaker.org/chainmaker/utils/v3"
)

var memberInfo2AddressCache *sync.Map
var memberInfo2PkCache *sync.Map

func init() {
	memberInfo2AddressCache = &sync.Map{}
	memberInfo2PkCache = &sync.Map{}
}

// ClearCache remove all the data in cache
func ClearCache() {
	memberInfo2AddressCache = &sync.Map{}
	memberInfo2PkCache = &sync.Map{}
}

// GetMemberPkAndAddress return public key and address of member
func GetMemberPkAndAddress(
	member *acPb.Member, snapshot protocol.Snapshot) (crypto.PublicKey, string, error) {

	var pk crypto.PublicKey
	var pkPem []byte
	var address string
	var ok, exist bool
	var err error
	memberInfoStr := string(member.GetMemberInfo())

	// load pk
	pkValue, exist := memberInfo2PkCache.Load(memberInfoStr)
	if exist {
		pk, ok = pkValue.(crypto.PublicKey)
		if !ok {
			memberInfo2PkCache.Delete(memberInfoStr)
		}
	}

	// reset memberInfo => pk
	if pk == nil {
		pkPem, err = getMemberPkPem(member, snapshot)
		if err != nil {
			return nil, "", fmt.Errorf("get member pk failed, err = %v", err)
		}
		pk, err = asym.PublicKeyFromPEM(pkPem)
		if err != nil {
			return nil, "", fmt.Errorf("publicKeyFromPEM failed, err = %v", err)
		}
		memberInfo2PkCache.Store(memberInfoStr, pk)
	}

	// load address
	addressValue, exist := memberInfo2AddressCache.Load(memberInfoStr)
	if exist {
		address, ok = addressValue.(string)
		if !ok {
			memberInfo2AddressCache.Delete(memberInfoStr)
		}
	}

	// reset memberInfo => address
	if len(address) == 0 {
		address, err = publicKeyToAddress(pk, snapshot.GetLastChainConfig())
		if err != nil {
			return nil, "", fmt.Errorf("publicKeyToAddress failed, err = %v", err)
		}
		memberInfo2AddressCache.Store(memberInfoStr, address)
	}

	return pk, address, nil
}

// publicKeyToAddress: generate address from public key, according to chainconfig parameter
func publicKeyToAddress(pk crypto.PublicKey, chainCfg *configPb.ChainConfig) (string, error) {

	publicKeyString, err := utils.PkToAddrStr(pk, chainCfg.Vm.AddrType, crypto.HashAlgoMap[chainCfg.Crypto.Hash])
	if err != nil {
		return "", err
	}

	if chainCfg.Vm.AddrType == configPb.AddrType_ZXL {
		publicKeyString = "ZX" + publicKeyString
	}
	return publicKeyString, nil
}

func getMemberPkPem(member *accesscontrol.Member, snapshot protocol.Snapshot) ([]byte, error) {

	var err error
	var pkPem []byte

	switch member.MemberType {
	case accesscontrol.MemberType_CERT:
		pkPem, err = publicKeyFromCert(member.MemberInfo)
		if err != nil {
			return nil, err
		}

	case accesscontrol.MemberType_CERT_HASH:
		var certInfo *commonPb.CertInfo
		infoHex := hex.EncodeToString(member.MemberInfo)
		if certInfo, err = wholeCertInfoFromSnapshot(snapshot, infoHex); err != nil {
			return nil, fmt.Errorf(" can not load the whole cert info,member[%s],reason: %s", infoHex, err)
		}

		pkPem, err = publicKeyFromCert(certInfo.Cert)
		if err != nil {
			return nil, err
		}

	case accesscontrol.MemberType_PUBLIC_KEY:
		pkPem = member.MemberInfo

	default:
		err = fmt.Errorf("invalid member type: %s", member.MemberType)
		return nil, err
	}

	return pkPem, nil
}

// extract public key from cert
func publicKeyFromCert(member []byte) ([]byte, error) {
	certificate, err := utils.ParseCert(member)
	if err != nil {
		return nil, err
	}
	pubKeyStr, err := certificate.PublicKey.String()
	if err != nil {
		return nil, err
	}
	return []byte(pubKeyStr), nil
}

func wholeCertInfoFromSnapshot(snapshot protocol.Snapshot, certHash string) (*commonPb.CertInfo, error) {
	certBytes, err := snapshot.GetKey(-1, syscontract.SystemContract_CERT_MANAGE.String(), []byte(certHash))
	if err != nil {
		return nil, err
	}

	return &commonPb.CertInfo{
		Hash: certHash,
		Cert: certBytes,
	}, nil
}
