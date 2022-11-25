package util

import (
	"chainmaker.org/chainmaker/common/v2/crypto"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/protocol/v2"
	sdk "chainmaker.org/chainmaker/sdk-go/v2"
	sdkutils "chainmaker.org/chainmaker/sdk-go/v2/utils"
	"fmt"
	"strings"
)

const (
	// AdminOrgidKeyCertLengthNotEqualFormat define AdminOrgidKeyCertLengthNotEqualFormat error fmt
	AdminOrgidKeyCertLengthNotEqualFormat = "admin orgId & key & cert list length not equal, [keys len: %d]/[certs len:%d]"
	// AdminOrgidKeyLengthNotEqualFormat define AdminOrgidKeyLengthNotEqualFormat error fmt
	AdminOrgidKeyLengthNotEqualFormat = "admin orgId & key list length not equal, [keys len: %d]/[org-ids len:%d]"
)

// MakeEndorsement user admins to sign payload, make an endorsement lit
// @param adminKeys
// @param adminCrts
// @param adminOrgs
// @param client
// @param payload
// @return []*common.EndorsementEntry
// @return error
func MakeEndorsement(adminKeys, adminCrts, adminOrgs []string, client *sdk.ChainClient, payload *common.Payload) (
	[]*common.EndorsementEntry, error) {
	endorsementEntrys := make([]*common.EndorsementEntry, len(adminKeys))
	for i := range adminKeys {
		if sdk.AuthTypeToStringMap[client.GetAuthType()] == protocol.PermissionedWithCert {
			e, err := sdkutils.MakeEndorserWithPath(adminKeys[i], adminCrts[i], payload)
			if err != nil {
				return nil, err
			}
			endorsementEntrys[i] = e
		} else if sdk.AuthTypeToStringMap[client.GetAuthType()] == protocol.PermissionedWithKey {
			hashType := crypto.HashAlgoMap[client.GetHashType()]
			e, err := sdkutils.MakePkEndorserWithPath(
				adminKeys[i],
				hashType,
				adminOrgs[i],
				payload,
			)
			if err != nil {
				return nil, err
			}

			endorsementEntrys[i] = e
		} else {
			hashType := crypto.HashAlgoMap[client.GetHashType()]
			e, err := sdkutils.MakePkEndorserWithPath(
				adminKeys[i],
				hashType,
				"",
				payload,
			)
			if err != nil {
				return nil, err
			}

			endorsementEntrys[i] = e
		}
	}
	return endorsementEntrys, nil
}

// MakeAdminInfo get admin keys, certs and orges from client
// @param client
// @param adminKeyFilePaths
// @param adminCrtFilePaths
// @param adminOrgIds
// @return adminKeys
// @return adminCrts
// @return adminOrgs
// @return err
func MakeAdminInfo(client *sdk.ChainClient, adminKeyFilePaths, adminCrtFilePaths, adminOrgIds string) (
	adminKeys, adminCrts, adminOrgs []string, err error) {
	if sdk.AuthTypeToStringMap[client.GetAuthType()] == protocol.PermissionedWithCert {
		if adminKeyFilePaths != "" {
			adminKeys = strings.Split(adminKeyFilePaths, ",")
		}
		if adminCrtFilePaths != "" {
			adminCrts = strings.Split(adminCrtFilePaths, ",")
		}
		if len(adminKeys) != len(adminCrts) {
			err = fmt.Errorf(AdminOrgidKeyCertLengthNotEqualFormat, len(adminKeys), len(adminCrts))
		}
	} else if sdk.AuthTypeToStringMap[client.GetAuthType()] == protocol.PermissionedWithKey {
		if adminKeyFilePaths != "" {
			adminKeys = strings.Split(adminKeyFilePaths, ",")
		}
		if adminOrgIds != "" {
			adminOrgs = strings.Split(adminOrgIds, ",")
		}
		if len(adminKeys) != len(adminOrgs) {
			err = fmt.Errorf(AdminOrgidKeyLengthNotEqualFormat, len(adminKeys), len(adminOrgs))
		}
	} else {
		if adminKeyFilePaths != "" {
			adminKeys = strings.Split(adminKeyFilePaths, ",")
		}
	}
	return
}
