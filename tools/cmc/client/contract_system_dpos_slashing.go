package client

import (
	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"
	sdk "chainmaker.org/chainmaker/sdk-go/v2"
	"fmt"

	"chainmaker.org/chainmaker-go/tools/cmc/util"
	"chainmaker.org/chainmaker/common/v2/json"
	"github.com/spf13/cobra"
)

// slashingGetDetail get slashing detail
// @return *cobra.Command
func slashingGetDetail() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "read-slashing",
		Short: "read slashing detail in epoch",
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

			resp, err := getSlashingDetail(client, epochID, DEFAULT_TIMEOUT)
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

// getSlashingDetail get slashing detail
func getSlashingDetail(cc *sdk.ChainClient, epochId string, timeout int64) (*common.TxResponse, error) {
	pairs := []*common.KeyValuePair{
		{
			Key:   "epoch_id",
			Value: []byte(epochId),
		},
	}
	resp, err := cc.QuerySystemContract(
		syscontract.SystemContract_DPOS_SLASHING.String(),
		syscontract.DPoSSlashingFunction_GET_SLASHING_DETAIL.String(),
		pairs,
		timeout,
	)
	if err != nil {
		return nil, fmt.Errorf("%s failed, %s", common.TxType_QUERY_CONTRACT.String(), err.Error())
	}

	//fmt.Printf("%s\n", resp.ContractResult.Result)
	return resp, nil
}
