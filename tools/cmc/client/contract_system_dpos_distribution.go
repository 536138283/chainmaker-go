/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"fmt"

	"chainmaker.org/chainmaker-go/tools/cmc/util"
	"chainmaker.org/chainmaker/common/v3/json"
	"chainmaker.org/chainmaker/pb-go/v3/common"
	"chainmaker.org/chainmaker/pb-go/v3/syscontract"
	sdk "chainmaker.org/chainmaker/sdk-go/v3"
	"github.com/spf13/cobra"
)

// distributionGetDetail get distribution detail
// @return *cobra.Command
func distributionGetDetail() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "read-distribution",
		Short: "read distribution detail in epoch",
		RunE: func(_ *cobra.Command, _ []string) error {
			var (
				err error
			)

			client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
				userSignCrtFilePath, userSignKeyFilePath)
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

			resp, err := getDistributionDetail(client, epochID, DEFAULT_TIMEOUT)
			if err != nil {
				return fmt.Errorf("get-node-id failed, %s", err.Error())
			}

			fmt.Printf("resp: %+v\n", resp)

			return nil
		},
	}

	attachFlags(cmd, []string{
		flagAddress,
		flagSdkConfPath,
		flagOrgId, flagChainId,
		flagUserTlsCrtFilePath, flagUserTlsKeyFilePath, flagUserSignCrtFilePath, flagUserSignKeyFilePath,
		flagEpochID,
	})

	cmd.MarkFlagRequired(flagEpochID)

	return cmd
}

// getDistributionDetail get distribution detail
func getDistributionDetail(cc *sdk.ChainClient, epochId string, timeout int64) (*common.TxResponse, error) {
	pairs := []*common.KeyValuePair{
		{
			Key:   "epoch_id",
			Value: []byte(epochId),
		},
	}
	resp, err := cc.QuerySystemContract(
		syscontract.SystemContract_DPOS_DISTRIBUTION.String(),
		syscontract.DPoSDistributionFunction_GET_DISTRIBUTION_DETAIL.String(),
		pairs,
		timeout,
	)
	if err != nil {
		return nil, fmt.Errorf("%s failed, %s", common.TxType_QUERY_CONTRACT.String(), err.Error())
	}

	return resp, nil
}
