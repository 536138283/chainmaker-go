/*
Copyright (C) Beijing Advanced Innovation Center for Future Blockchain and Privacy Computing (未来区块链与隐
私计算⾼精尖创新中⼼). All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"errors"
	"fmt"
	"io/ioutil"

	"chainmaker.org/chainmaker-go/tools/cmc/util"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/protocol/v2"
	sdk "chainmaker.org/chainmaker/sdk-go/v2"
	sdkutils "chainmaker.org/chainmaker/sdk-go/v2/utils"
	"github.com/spf13/cobra"
)

const (
	addMasterKey = iota
	removeMasterKey
	updateMasterKey
)

// configTrustRootCMD trust root command
// @return *cobra.Command
func configMasterKeyCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "masterkey",
		Short: "master key command",
		Long:  "master key command",
	}
	cmd.AddCommand(addMasterKeyCMD())
	cmd.AddCommand(removeMasterKeyCMD())
	cmd.AddCommand(updateMasterKeyCMD())

	return cmd
}

// addTrustRootCMD add trust root ca cert
// @return *cobra.Command
func addMasterKeyCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "add master key",
		Long:  "add master key",
		RunE: func(_ *cobra.Command, _ []string) error {
			return configMasterKey(addMasterKey)
		},
	}

	attachFlags(cmd, []string{
		flagUserSignKeyFilePath, flagUserSignCrtFilePath,
		flagSdkConfPath, flagOrgId, flagMasterKeyPath, flagMasterKeyOrgId,
		flagAdminCrtFilePaths, flagAdminKeyFilePaths, flagAdminOrgIds, flagUserTlsCrtFilePath, flagUserTlsKeyFilePath,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagMasterKeyOrgId)
	cmd.MarkFlagRequired(flagMasterKeyPath)

	return cmd
}

func configMasterKey(op int) error {
	client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
		userSignCrtFilePath, userSignKeyFilePath)
	if err != nil {
		return err
	}
	defer client.Stop()

	adminKeys, adminCrts, _, err := util.MakeAdminInfo(client, adminKeyFilePaths, adminCrtFilePaths, adminOrgIds)
	if err != nil {
		return err
	}

	var masterKeyBytes []string
	if op == addMasterKey || op == updateMasterKey {

		if len(masterKeyPaths) == 0 {
			return fmt.Errorf("please specify master key path")
		}
		for _, masterKeyPath := range masterKeyPaths {
			masterKey, err := ioutil.ReadFile(masterKeyPath)
			if err != nil {
				return err
			}
			masterKeyBytes = append(masterKeyBytes, string(masterKey))
		}
	}

	var payload *common.Payload
	switch op {
	case addMasterKey:
		payload, err = client.CreateChainConfigMasterKeyAddPayload(masterKeyOrgId, masterKeyBytes)
	case removeMasterKey:
		payload, err = client.CreateChainConfigMasterKeyDeletePayload(masterKeyOrgId)
	case updateMasterKey:
		payload, err = client.CreateChainConfigMasterKeyUpdatePayload(masterKeyOrgId, masterKeyBytes)
	default:
		err = errors.New("invalid master key operation")
	}
	if err != nil {
		return err
	}

	if sdk.AuthTypeToStringMap[client.GetAuthType()] != protocol.PermissionedWithIBC {
		return errors.New("auth type must be PermissionedWithIBC")
	}

	endorsementEntrys := make([]*common.EndorsementEntry, len(adminKeys))
	for i := range adminKeys {
		e, err := sdkutils.MakeIBCEndorserWithPath(adminKeys[i], adminCrts[i], payload)
		if err != nil {
			return err
		}
		endorsementEntrys[i] = e
	}

	resp, err := client.SendChainConfigUpdateRequest(payload, endorsementEntrys, -1, syncResult)
	if err != nil {
		return err
	}
	err = util.CheckProposalRequestResp(resp, false)
	if err != nil {
		return err
	}
	fmt.Printf("trustroot response %+v\n", resp)
	return nil
}

// removeTrustRootCMD remove IBC master key
func removeMasterKeyCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove",
		Short: "remove IBC master key",
		Long:  "remove IBC master key",
		RunE: func(_ *cobra.Command, _ []string) error {
			return configMasterKey(removeMasterKey)
		},
	}

	attachFlags(cmd, []string{
		flagUserSignKeyFilePath, flagUserSignCrtFilePath,
		flagSdkConfPath, flagOrgId, flagEnableCertHash, flagMasterKeyPath, flagMasterKeyOrgId,
		flagAdminCrtFilePaths, flagAdminKeyFilePaths, flagAdminOrgIds, flagUserTlsCrtFilePath, flagUserTlsKeyFilePath,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagMasterKeyOrgId)

	return cmd
}

// updateTrustRootCMD update IBC master key
// @return *cobra.Command
func updateMasterKeyCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "update trust root ca cert",
		Long:  "update trust root ca cert",
		RunE: func(_ *cobra.Command, _ []string) error {
			return configMasterKey(updateMasterKey)
		},
	}

	attachFlags(cmd, []string{
		flagUserSignKeyFilePath, flagUserSignCrtFilePath,
		flagSdkConfPath, flagOrgId, flagEnableCertHash, flagMasterKeyPath, flagMasterKeyOrgId,
		flagAdminCrtFilePaths, flagAdminKeyFilePaths, flagAdminOrgIds, flagUserTlsCrtFilePath, flagUserTlsKeyFilePath,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagMasterKeyOrgId)
	cmd.MarkFlagRequired(flagMasterKeyPath)

	return cmd
}
