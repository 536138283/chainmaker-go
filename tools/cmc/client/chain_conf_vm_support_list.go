/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"chainmaker.org/chainmaker-go/tools/cmc/util"
	"github.com/spf13/cobra"
)

// vmSupportListCMD update vm support list.
// @return *cobra.Command
func vmSupportListCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vm-support-list",
		Short: "update vm support list.",
		Long:  "update vm support list.",
	}
	cmd.AddCommand(addVmSupportCMD())
	return cmd
}

// addVmSupportCMD add a vm type to vm support list.
// @return *cobra.Command
func addVmSupportCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "add a vm type to vm support list.",
		Long:  "add a vm type to vm support list.",
		RunE: func(_ *cobra.Command, _ []string) error {
			client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
				userSignCrtFilePath, userSignKeyFilePath)
			if err != nil {
				return err
			}
			defer client.Stop()

			adminKeys, adminCrts, adminOrgs, err := util.MakeAdminInfo(client, adminKeyFilePaths, adminCrtFilePaths, adminOrgIds)
			if err != nil {
				return err
			}

			payload, err := client.CreateChainConfigVMSupportListAddPayload(vmType)
			if err != nil {
				return err
			}

			endorsementEntrys, err := util.MakeEndorsement(adminKeys, adminCrts, adminOrgs, client, payload)
			if err != nil {
				return err
			}

			// send
			resp, err := client.SendChainConfigUpdateRequest(payload, endorsementEntrys, timeout, syncResult)
			if err != nil {
				return err
			}

			util.PrintPrettyJson(resp)
			return nil
		},
	}

	util.AttachFlags(cmd, flags, []string{
		flagUserSignKeyFilePath, flagUserSignCrtFilePath, flagUserTlsKeyFilePath, flagUserTlsCrtFilePath, flagOrgId,
		flagChainId, flagTimeout, flagSyncResult, flagAdminCrtFilePaths, flagAdminKeyFilePaths, flagAdminOrgIds,
		flagSdkConfPath,
	})
	util.AttachAndRequiredFlags(cmd, flags, []string{
		flagVmType,
	})

	return cmd
}
