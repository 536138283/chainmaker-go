/*
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */
package client

import (
	"encoding/json"
	"fmt"

	"chainmaker.org/chainmaker-go/tools/cmc/util"
	sdkutils "chainmaker.org/chainmaker/sdk-go/v2/utils"
	"github.com/spf13/cobra"
)

// syncAddRule 同步模块添加规则
// @return *cobra.Command
func syncAddRule() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-rule",
		Short: "add-rule to the node",
		RunE: func(_ *cobra.Command, _ []string) error {
			var (
				err error
			)
			client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
				userSignCrtFilePath, userSignKeyFilePath, enableCertHash)
			if err != nil {
				return err
			}
			defer client.Stop()
			pairs := make(map[string]string)
			if params != "" {
				err := json.Unmarshal([]byte(params), &pairs)
				if err != nil {
					return err
				}
			}
			txId = sdkutils.GetTimestampTxId()
			payload, err := client.CreateAddSyncRulePayload(nodeId, rule, int(beginHeight), int(endHeight))
			if err != nil {
				return fmt.Errorf("get add rule payload failed, %s", err.Error())
			}
			resp, err := client.SendSyncRuleRequest(payload, nil, DEFAULT_TIMEOUT, syncResult)
			if err != nil {
				return fmt.Errorf("send request failed, %s", err.Error())
			}
			fmt.Printf("resp: %+v\n", resp)

			return nil
		},
	}

	attachFlags(cmd, []string{
		flagNodeId, flagRule, flagBeginHeight, flagEndHeight,
		flagSdkConfPath,
		flagOrgId, flagChainId,
		flagUserTlsCrtFilePath, flagUserTlsKeyFilePath,
		flagUserSignCrtFilePath, flagUserSignKeyFilePath,
		flagSyncResult,
	})

	cmd.MarkFlagRequired(flagNodeId)
	cmd.MarkFlagRequired(flagRule)

	return cmd
}

// syncGetRule 同步模块获取规则
// @return *cobra.Command
func syncGetRule() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-rule",
		Short: "get-rule of the node",
		RunE: func(_ *cobra.Command, _ []string) error {
			var (
				err error
			)
			client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
				userSignCrtFilePath, userSignKeyFilePath, enableCertHash)
			if err != nil {
				return err
			}
			defer client.Stop()
			pairs := make(map[string]string)
			if params != "" {
				err := json.Unmarshal([]byte(params), &pairs)
				if err != nil {
					return err
				}
			}
			txId = sdkutils.GetTimestampTxId()
			resp, err := client.GetNodeRule(nodeId)
			if err != nil {
				return fmt.Errorf("get rule failed, %s", err.Error())
			}

			fmt.Printf("resp: %+v\n", resp)

			return nil
		},
	}

	attachFlags(cmd, []string{
		flagNodeId,
		flagSdkConfPath,
		flagOrgId, flagChainId,
		flagUserTlsCrtFilePath, flagUserTlsKeyFilePath,
		flagUserSignCrtFilePath, flagUserSignKeyFilePath,
		flagSyncResult,
	})

	cmd.MarkFlagRequired(flagNodeId)

	return cmd
}

// syncClearRule 同步模块清除节点规则
// @return *cobra.Command
func syncClearRule() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clear-rule",
		Short: "clear-rule of the node",
		RunE: func(_ *cobra.Command, _ []string) error {
			var (
				err error
			)
			client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
				userSignCrtFilePath, userSignKeyFilePath, enableCertHash)
			if err != nil {
				return err
			}
			defer client.Stop()
			pairs := make(map[string]string)
			if params != "" {
				err := json.Unmarshal([]byte(params), &pairs)
				if err != nil {
					return err
				}
			}
			txId = sdkutils.GetTimestampTxId()
			payload := client.CreateClearNodeRulePayload(nodeId)
			resp, err := client.SendSyncRuleRequest(payload, nil, DEFAULT_TIMEOUT, syncResult)
			if err != nil {
				return fmt.Errorf("send request failed, %s", err.Error())
			}
			fmt.Printf("resp: %+v\n", resp)

			return nil
		},
	}

	attachFlags(cmd, []string{
		flagNodeId,
		flagSdkConfPath,
		flagOrgId, flagChainId,
		flagUserTlsCrtFilePath, flagUserTlsKeyFilePath,
		flagUserSignCrtFilePath, flagUserSignKeyFilePath,
		flagSyncResult,
	})

	cmd.MarkFlagRequired(flagNodeId)

	return cmd
}
