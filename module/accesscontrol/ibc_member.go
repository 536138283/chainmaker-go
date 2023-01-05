/*
Copyright (C) Beijing Advanced Innovation Center for Future Blockchain and Privacy Computing (未来区块链与隐
私计算⾼精尖创新中⼼). All rights reserved.
SPDX-License-Identifier: Apache-2.0
*/

package accesscontrol

import (
	"errors"
	"fmt"

	bccrypto "chainmaker.org/chainmaker/common/v2/crypto"
	"chainmaker.org/chainmaker/common/v2/crypto/asym"
	"chainmaker.org/chainmaker/common/v2/crypto/asym/sm9"
	pbac "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	"chainmaker.org/chainmaker/protocol/v2"
)

var _ protocol.Member = (*ibcMember)(nil)

// an instance whose member type is a ibc
type ibcMember struct {

	// id Logical networking of nodes compatible with tls certificates
	id string

	// organization identity who owns this member
	orgId string

	// role of this member
	role protocol.Role

	// hashType hash algorithm for chains (It's not the hash algorithm that the certificate uses)
	hashType string

	// isCompressed the certificate is compressed or not
	isCompressed bool

	// ibcPK public Key used for authentication
	ibcPK *sm9.PublicKey
}

// GetPk returns public Key used for authentication
func (im *ibcMember) GetPk() bccrypto.PublicKey {
	return im.ibcPK
}

// GetMemberId returns the identity of this member
func (im *ibcMember) GetMemberId() string {
	return im.id
}

// GetOrgId returns the organization id which this member belongs to
func (im *ibcMember) GetOrgId() string {
	return im.orgId
}

// GetRole returns roles of this member
func (im *ibcMember) GetRole() protocol.Role {
	return im.role
}

// GetUid returns the identity of this member
func (im *ibcMember) GetUid() string {
	return string(im.ibcPK.K)
}

// Verify verifies a signature over some message using this member
func (im *ibcMember) Verify(hashType string, msg []byte, sig []byte) error {
	ok, err := im.ibcPK.VerifyWithOpts(msg, sig, nil)
	if err != nil {
		return fmt.Errorf("IBC member verify signature failed: [%s]", err.Error())
	}
	if !ok {
		return fmt.Errorf("IBC member verify signature failed: invalid signature")
	}
	return nil
}

// GetMember returns Member
func (im *ibcMember) GetMember() (*pbac.Member, error) {
	memInfo, err := im.ibcPK.String()
	if err != nil {
		return nil, errors.New("GetMember faild: " + err.Error())
	}
	return &pbac.Member{
		OrgId:      im.orgId,
		MemberInfo: []byte(memInfo),
		MemberType: pbac.MemberType_IBC,
	}, nil
}

func newMemberFromIBCInfo(orgId, hashType string, ibcInfoBytes []byte, isCompressed bool) (*ibcMember, error) {
	var member ibcMember
	var err error

	pk, err := sm9.PublicKeyFromPEM(ibcInfoBytes)
	if err != nil {
		return nil, errors.New("fail to parse key: " + err.Error())
	}
	ibcInfo, err := sm9.ParsePKInfo(pk.K)
	if err != nil {
		return nil, errors.New("invalid ibc key: " + err.Error())
	}
	if ibcInfo.OrgId != orgId {
		return nil, fmt.Errorf("org not match: %s - %s", ibcInfo.OrgId, orgId)
	}

	member.ibcPK = pk
	member.id = ibcInfo.Id + ".sign." + ibcInfo.OrgId // Logical networking of nodes compatible with tls certificates
	member.role = protocol.Role(ibcInfo.Role)
	member.orgId = ibcInfo.OrgId
	member.hashType = hashType
	member.isCompressed = isCompressed

	return &member, nil
}

func newIBCMemberFromPb(member *pbac.Member, acs *accessControlService) (*ibcMember, error) {
	if member.MemberType == pbac.MemberType_IBC {
		return newMemberFromIBCInfo(member.OrgId, acs.hashType, member.MemberInfo, false)
	}

	return nil, fmt.Errorf("newIBCMemberFromPb  failed")
}

type signingIBCMember struct {
	ibcMember
	// Sign the message
	sk bccrypto.PrivateKey
}

func (sim *signingIBCMember) Sign(hashType string, msg []byte) ([]byte, error) {
	hash, ok := bccrypto.HashAlgoMap[hashType]
	if !ok {
		return nil, fmt.Errorf("sign failed: unsupport hash type")
	}
	return sim.sk.SignWithOpts(msg, &bccrypto.SignOpts{
		Hash: hash,
		UID:  bccrypto.CRYPTO_DEFAULT_UID,
	})
}

// NewIBCSigningMember 基于传入的参数新建一个SigningMember
// @param hashType
// @param member
// @param privateKeyPem
// @param password
// @return protocol.SigningMember
// @return error
func NewIBCSigningMember(hashType string, member *pbac.Member, privateKeyPem,
	password string) (protocol.SigningMember, error) {

	ibcMember, err := newMemberFromIBCInfo(member.OrgId, hashType, member.MemberInfo, false)
	if err != nil {
		return nil, err
	}

	var sk bccrypto.PrivateKey
	sk, err = asym.PrivateKeyFromPEM([]byte(privateKeyPem), []byte(password))
	if err != nil {
		return nil, err
	}

	return &signingIBCMember{
		*ibcMember,
		sk,
	}, nil
}
