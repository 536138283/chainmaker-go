/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"encoding/json"
	"fmt"

	sdkutils "chainmaker.org/chainmaker/sdk-go/v3/utils"

	"github.com/spf13/cobra"

	"chainmaker.org/chainmaker-go/tools/cmc/util"
	"chainmaker.org/chainmaker/pb-go/v3/common"
	"chainmaker.org/chainmaker/pb-go/v3/syscontract"
	sdk "chainmaker.org/chainmaker/sdk-go/v3"
)

// erc20Mint DPos ERC20合约中的Mint发行Token
// @return *cobra.Command
func erc20Mint() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mint",
		Short: "mint feature of the erc20",
		RunE: func(_ *cobra.Command, _ []string) error {
			return mint()
		},
	}

	attachFlags(cmd, []string{
		flagAddress, flagAmount,
		flagSdkConfPath,
		flagOrgId, flagChainId,
		flagUserTlsCrtFilePath, flagUserTlsKeyFilePath,
		flagUserSignCrtFilePath, flagUserSignKeyFilePath,
		flagSyncResult, flagAdminKeyFilePaths, flagAdminOrgIds,
	})

	cmd.MarkFlagRequired(flagAddress)
	cmd.MarkFlagRequired(flagAmount)

	return cmd
}

// erc20TransferOwnership DPos ERC20合约中的转移owner权限
// @return *cobra.Command
func erc20TransferOwnership() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transfer-owner-ship",
		Short: "transfer owner ship of the erc20",
		RunE: func(_ *cobra.Command, _ []string) error {
			return transferOwnership()
		},
	}

	attachFlags(cmd, []string{
		flagAddress,
		flagSdkConfPath,
		flagOrgId, flagChainId,
		flagUserTlsCrtFilePath, flagUserTlsKeyFilePath,
		flagUserSignCrtFilePath, flagUserSignKeyFilePath,
		flagSyncResult, flagAdminKeyFilePaths, flagAdminOrgIds,
	})

	cmd.MarkFlagRequired(flagAddress)

	return cmd
}

// erc20Transfer DPoS ERC20的transfer操作
// @return *cobra.Command
func erc20Transfer() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transfer",
		Short: "transfer feature of the erc20",
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
			txId = sdkutils.GetTimestampTxId()
			resp, err := transfer(client, address, amount, txId, DEFAULT_TIMEOUT, false)
			if err != nil {
				return fmt.Errorf("transfer failed, %s", err.Error())
			}

			fmt.Printf("resp: %+v\n", resp)

			return nil
		},
	}

	attachFlags(cmd, []string{
		flagAddress, flagAmount,
		flagSdkConfPath,
		flagOrgId, flagChainId,
		flagUserTlsCrtFilePath, flagUserTlsKeyFilePath, flagUserSignCrtFilePath, flagUserSignKeyFilePath,
	})

	cmd.MarkFlagRequired(flagAddress)
	cmd.MarkFlagRequired(flagAmount)

	return cmd
}

// erc20BalanceOf query balance-of feature of the DPoS erc20
// @return *cobra.Command
func erc20BalanceOf() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "balance-of",
		Short: "balance-of feature of the erc20",
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

			resp, err := balanceOf(client, address, DEFAULT_TIMEOUT)
			if err != nil {
				return fmt.Errorf("balance of failed, %s", err.Error())
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
	})

	cmd.MarkFlagRequired(flagAddress)

	return cmd
}

// erc20Owner query DPoS erc20 owner(admin)
// @return *cobra.Command
func erc20Owner() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "owner",
		Short: "owner feature of the erc20",
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

			resp, err := owner(client, DEFAULT_TIMEOUT)
			if err != nil {
				return fmt.Errorf("owner failed, %s", err.Error())
			}

			fmt.Printf("resp: %+v\n", resp)

			return nil
		},
	}

	attachFlags(cmd, []string{
		flagSdkConfPath,
		flagOrgId, flagChainId,
		flagUserTlsCrtFilePath, flagUserTlsKeyFilePath, flagUserSignCrtFilePath, flagUserSignKeyFilePath,
	})

	return cmd
}

// erc20Decimals query DPoS erc20 decimal
// @return *cobra.Command
func erc20Decimals() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "decimals",
		Short: "decimals feature of the erc20",
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

			resp, err := decimals(client, DEFAULT_TIMEOUT)
			if err != nil {
				return fmt.Errorf("decimals failed, %s", err.Error())
			}

			fmt.Printf("resp: %+v\n", resp)

			return nil
		},
	}

	attachFlags(cmd, []string{
		flagSdkConfPath,
		flagOrgId, flagChainId,
		flagUserTlsCrtFilePath, flagUserTlsKeyFilePath, flagUserSignCrtFilePath, flagUserSignKeyFilePath,
	})

	return cmd
}

// erc20Total query total amount of DPoS erc20 token
// @return *cobra.Command
func erc20Total() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "total",
		Short: "total feature of the erc20",
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

			resp, err := total(client, DEFAULT_TIMEOUT)
			if err != nil {
				return fmt.Errorf("total failed, %s", err.Error())
			}

			fmt.Printf("resp: %+v\n", resp)

			return nil
		},
	}

	attachFlags(cmd, []string{
		flagSdkConfPath,
		flagOrgId, flagChainId,
		flagUserTlsCrtFilePath, flagUserTlsKeyFilePath, flagUserSignCrtFilePath, flagUserSignKeyFilePath,
	})

	return cmd
}

func mint() error {

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

	payload, err := client.Mint(address, amount)
	if err != nil {
		return fmt.Errorf("create chain config block update payload failed, %s", err.Error())
	}

	endorsementEntrys, err := util.MakeEndorsement(adminKeys, adminCrts, adminOrgs, client, payload)
	if err != nil {
		return err
	}
	resp, err := client.SendChainConfigUpdateRequest(payload, endorsementEntrys, -1, true)
	if err != nil {
		return fmt.Errorf("send chain config update request failed, %s", err.Error())
	}
	err = util.CheckProposalRequestResp(resp, false)
	if err != nil {
		return fmt.Errorf("check proposal request resp failed, %s", err.Error())
	}
	fmt.Printf("response %+v\n", resp)
	return nil
}

func transferOwnership() error {

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

	payload, err := client.TransferOwnership(address)
	if err != nil {
		return fmt.Errorf("create chain config block update payload failed, %s", err.Error())
	}

	endorsementEntrys, err := util.MakeEndorsement(adminKeys, adminCrts, adminOrgs, client, payload)
	if err != nil {
		return err
	}
	resp, err := client.SendChainConfigUpdateRequest(payload, endorsementEntrys, -1, true)
	if err != nil {
		return fmt.Errorf("send chain config update request failed, %s", err.Error())
	}
	err = util.CheckProposalRequestResp(resp, false)
	if err != nil {
		return fmt.Errorf("check proposal request resp failed, %s", err.Error())
	}
	fmt.Printf("response %+v\n", resp)
	return nil
}

// transfer ERC20的transfer操作
func transfer(cc *sdk.ChainClient, address, amount string, txId string, timeout int64,
	withSyncResult bool) (*common.TxResponse, error) {
	pairs := []*common.KeyValuePair{
		{
			Key:   "to",
			Value: []byte(address),
		},
		{
			Key:   "value",
			Value: []byte(amount),
		},
	}
	if txId == "" {
		txId = sdkutils.GetTimestampTxId()
	}
	resp, err := cc.InvokeSystemContract(
		syscontract.SystemContract_DPOS_ERC20.String(),
		syscontract.DPoSERC20Function_TRANSFER.String(),
		txId,
		pairs,
		timeout,
		withSyncResult,
	)
	if err != nil {
		return nil, fmt.Errorf("%s failed, %s", common.TxType_INVOKE_CONTRACT.String(), err.Error())
	}

	return resp, nil
}

// balanceOf query balance-of feature of the DPoS erc20
func balanceOf(cc *sdk.ChainClient, address string, timeout int64) (*common.TxResponse, error) {
	pairs := []*common.KeyValuePair{
		{
			Key:   "owner",
			Value: []byte(address),
		},
	}
	resp, err := cc.QuerySystemContract(
		syscontract.SystemContract_DPOS_ERC20.String(),
		syscontract.DPoSERC20Function_GET_BALANCEOF.String(),
		pairs,
		timeout,
	)
	if err != nil {
		return nil, fmt.Errorf("%s failed, %s", common.TxType_QUERY_CONTRACT.String(), err.Error())
	}

	return resp, nil
}

// owner query DPoS erc20 owner(admin)
func owner(cc *sdk.ChainClient, timeout int64) (*common.TxResponse, error) {
	resp, err := cc.QuerySystemContract(
		syscontract.SystemContract_DPOS_ERC20.String(),
		syscontract.DPoSERC20Function_GET_OWNER.String(),
		nil,
		timeout,
	)
	if err != nil {
		return nil, fmt.Errorf("%s failed, %s", common.TxType_QUERY_CONTRACT.String(), err.Error())
	}

	return resp, nil
}

// decimals query DPoS erc20 decimal
func decimals(cc *sdk.ChainClient, timeout int64) (*common.TxResponse, error) {
	resp, err := cc.QuerySystemContract(
		syscontract.SystemContract_DPOS_ERC20.String(),
		syscontract.DPoSERC20Function_GET_DECIMALS.String(),
		nil,
		timeout,
	)
	if err != nil {
		return nil, fmt.Errorf("%s failed, %s", common.TxType_QUERY_CONTRACT.String(), err.Error())
	}

	return resp, nil
}

// total query total amount of DPoS erc20 token
func total(cc *sdk.ChainClient, timeout int64) (*common.TxResponse, error) {
	resp, err := cc.QuerySystemContract(
		syscontract.SystemContract_DPOS_ERC20.String(),
		syscontract.DPoSERC20Function_GET_TOTAL_SUPPLY.String(),
		nil,
		timeout,
	)
	if err != nil {
		return nil, fmt.Errorf("%s failed, %s", common.TxType_QUERY_CONTRACT.String(), err.Error())
	}

	return resp, nil
}
