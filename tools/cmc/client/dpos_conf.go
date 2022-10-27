/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"fmt"
	"strings"

	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/protocol/v2"
	sdk "chainmaker.org/chainmaker/sdk-go/v2"
	sdkutils "chainmaker.org/chainmaker/sdk-go/v2/utils"

	"chainmaker.org/chainmaker-go/tools/cmc/util"
	"github.com/hokaccha/go-prettyjson"
	"github.com/spf13/cobra"
)

// dposConfigCMD dpos sub command
// @return *cobra.Command
func dposConfigCMD() *cobra.Command {
	chainConfigCmd := &cobra.Command{
		Use:   "dposconfig",
		Short: "dpos config command",
		Long:  "dpos config command",
	}
	//查询配置
	//chainConfigCmd.AddCommand(readMinSelfDelegationCMD())
	chainConfigCmd.AddCommand(readEpochBlocNumberCMD())
	chainConfigCmd.AddCommand(readEpochValidatorNumberCMD())
	chainConfigCmd.AddCommand(readDistributionPerBlockCMD())
	chainConfigCmd.AddCommand(readSlashingPerBlockCMD())
	chainConfigCmd.AddCommand(readGasExchangeRateCMD())

	//多签操作
	chainConfigCmd.AddCommand(setMinSelfDelegationCMD())
	chainConfigCmd.AddCommand(setEpochBlockNumberCMD())
	chainConfigCmd.AddCommand(setEpochValidatorNumberCMD())
	chainConfigCmd.AddCommand(setEpochBlockNumberAndValidatorNumberCMD())
	chainConfigCmd.AddCommand(setDistributionPerBlockCMD())
	chainConfigCmd.AddCommand(setSlashingPerBlockCMD())
	chainConfigCmd.AddCommand(setGasExchangeRateCMD())

	//erc20 管理员操作
	//不再使用，改用多签方式操作
	//chainConfigCmd.AddCommand(updateEpochBlocNumberCMD())
	chainConfigCmd.AddCommand(updateDistributionPerBlockCMD())
	chainConfigCmd.AddCommand(updateGasExchangeRateCMD())

	return chainConfigCmd
}

// nolint: unused
// readMinSelfDelegationCMD returns the minimum number of delegates,
// this method is not available for the time being
// @return *cobra.Command
//func readMinSelfDelegationCMD() *cobra.Command {
//	cmd := &cobra.Command{
//		Use:   "read-min-self-delegation",
//		Short: "read min self delegation",
//		Long:  "read min self delegation",
//		RunE: func(_ *cobra.Command, _ []string) error {
//			return getMinSelfDelegation()
//		},
//	}
//
//	attachFlags(cmd, []string{
//		flagUserSignKeyFilePath, flagUserSignCrtFilePath,
//		flagSdkConfPath, flagOrgId, flagEnableCertHash,
//		flagUserTlsCrtFilePath, flagUserTlsKeyFilePath,
//	})
//
//	return cmd
//}

// readEpochBlocNumberCMD returns the configuration of the number of blocks in an epoch
// @return *cobra.Command
func readEpochBlocNumberCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "read-epoch-block-number",
		Short: "read epoch block number",
		Long:  "read epoch block number",
		RunE: func(_ *cobra.Command, _ []string) error {
			return getEpochBlockNumber()
		},
	}

	attachFlags(cmd, []string{
		flagUserSignKeyFilePath, flagUserSignCrtFilePath,
		flagSdkConfPath, flagOrgId, flagEnableCertHash,
		flagUserTlsCrtFilePath, flagUserTlsKeyFilePath,
	})

	return cmd
}

// readEpochValidatorNumberCMD returns the configuration of the number of validators in an epoch
// @return *cobra.Command
func readEpochValidatorNumberCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "read-epoch-validator-number",
		Short: "read epoch validator number",
		Long:  "read epoch validator number",
		RunE: func(_ *cobra.Command, _ []string) error {
			return getEpochValidatorNumber()
		},
	}

	attachFlags(cmd, []string{
		flagUserSignKeyFilePath, flagUserSignCrtFilePath,
		flagSdkConfPath, flagOrgId, flagEnableCertHash,
		flagUserTlsCrtFilePath, flagUserTlsKeyFilePath,
	})

	return cmd
}

// readDistributionPerBlockCMD returns a configuration of block reward amount
// @return *cobra.Command
func readDistributionPerBlockCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "read-distribution-per-block",
		Short: "read distribution per block",
		Long:  "read distribution per block",
		RunE: func(_ *cobra.Command, _ []string) error {
			return getDistributionPerBlock()
		},
	}

	attachFlags(cmd, []string{
		flagUserSignKeyFilePath, flagUserSignCrtFilePath,
		flagSdkConfPath, flagOrgId, flagEnableCertHash,
		flagUserTlsCrtFilePath, flagUserTlsKeyFilePath,
	})

	return cmd
}

// readDistributionPerBlockCMD returns a configuration of block penalty amount
// @return *cobra.Command
func readSlashingPerBlockCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "read-slashing-per-block",
		Short: "read slashing per block",
		Long:  "read slashing per block",
		RunE: func(_ *cobra.Command, _ []string) error {
			return getSlashingPerBlock()
		},
	}

	attachFlags(cmd, []string{
		flagUserSignKeyFilePath, flagUserSignCrtFilePath,
		flagSdkConfPath, flagOrgId, flagEnableCertHash,
		flagUserTlsCrtFilePath, flagUserTlsKeyFilePath,
	})

	return cmd
}

// readGasExchangeRateCMD returns the gas replacement ratio
// @return *cobra.Command
func readGasExchangeRateCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "read-gas-exchange-rate",
		Short: "read gas exchange rate",
		Long:  "read gas exchange rate",
		RunE: func(_ *cobra.Command, _ []string) error {
			return getGasExchangeRate()
		},
	}

	attachFlags(cmd, []string{
		flagUserSignKeyFilePath, flagUserSignCrtFilePath,
		flagSdkConfPath, flagOrgId, flagEnableCertHash,
		flagUserTlsCrtFilePath, flagUserTlsKeyFilePath,
	})

	return cmd
}

//nolint: unused
// setMinSelfDelegationCMD set the minimum number of delegates,
// this method is not available for the time being
// @return *cobra.Command
func setMinSelfDelegationCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-min-self-delegation",
		Short: "set min self delegation",
		Long:  "set min self delegation",
		RunE: func(_ *cobra.Command, _ []string) error {
			return setMinSelfDelegation()
		},
	}

	attachFlags(cmd, []string{
		flagUserSignKeyFilePath, flagUserSignCrtFilePath, flagUserTlsCrtFilePath, flagUserTlsKeyFilePath, flagChainId,
		flagSdkConfPath, flagOrgId, flagAdminCrtFilePaths, flagAdminKeyFilePaths, flagAdminOrgIds,
		flagMinSelfDelegation,
	})

	cmd.MarkFlagRequired(flagMinSelfDelegation)
	cmd.MarkFlagRequired(flagSdkConfPath)

	return cmd
}

//nolint: unused
// setEpochBlockNumberCMD set the configuration of the number of blocks in an epoch
// this method is not available for the time being
// need to use setEpochBlockNumberAndValidatorNumberCMD, modify 2 configurations at the same time,
// require the number of blocks to be an integer multiple of the number of validators
// @return *cobra.Command
func setEpochBlockNumberCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-epoch-block-number",
		Short: "set epoch block number",
		Long:  "set epoch block number",
		RunE: func(_ *cobra.Command, _ []string) error {
			return setEpochBlocNumber()
		},
	}

	attachFlags(cmd, []string{
		flagUserSignKeyFilePath, flagUserSignCrtFilePath, flagUserTlsCrtFilePath, flagUserTlsKeyFilePath, flagChainId,
		flagSdkConfPath, flagOrgId, flagAdminCrtFilePaths, flagAdminKeyFilePaths, flagAdminOrgIds,
		flagEpochBlockNumber,
	})

	cmd.MarkFlagRequired(flagEpochBlockNumber)
	cmd.MarkFlagRequired(flagSdkConfPath)

	return cmd
}

//nolint: unused
// setEpochBlockNumberCMD set the configuration of the number of validators in an epoch
// this method is not available for the time being
// need to use setEpochBlockNumberAndValidatorNumberCMD, modify 2 configurations at the same time,
// require the number of blocks to be an integer multiple of the number of validators
// @return *cobra.Command
func setEpochValidatorNumberCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-epoch-validator-number",
		Short: "set epoch validator number",
		Long:  "set epoch validator number",
		RunE: func(_ *cobra.Command, _ []string) error {
			return setEpochValidatorNumber()
		},
	}

	attachFlags(cmd, []string{
		flagUserSignKeyFilePath, flagUserSignCrtFilePath, flagUserTlsCrtFilePath, flagUserTlsKeyFilePath, flagChainId,
		flagSdkConfPath, flagOrgId, flagAdminCrtFilePaths, flagAdminKeyFilePaths, flagAdminOrgIds,
		flagEpochValidatorNumber,
	})

	cmd.MarkFlagRequired(flagEpochValidatorNumber)
	cmd.MarkFlagRequired(flagSdkConfPath)

	return cmd
}

// setEpochBlockNumberCMD set the configuration of the number of validators and numbers in an epoch
// modify 2 configurations at the same time,
// require the number of blocks to be an integer multiple of the number of validators
// @return *cobra.Command
func setEpochBlockNumberAndValidatorNumberCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-epoch-block-number-and-validator-number",
		Short: "set epoch block number and validator number",
		Long:  "set epoch block number and validator number",
		RunE: func(_ *cobra.Command, _ []string) error {
			return setEpochBlockNumberAndValidatorNumber()
		},
	}

	attachFlags(cmd, []string{
		flagUserSignKeyFilePath, flagUserSignCrtFilePath, flagUserTlsCrtFilePath, flagUserTlsKeyFilePath, flagChainId,
		flagSdkConfPath, flagOrgId, flagAdminCrtFilePaths, flagAdminKeyFilePaths, flagAdminOrgIds,
		flagEpochBlockNumber, flagEpochValidatorNumber,
	})

	cmd.MarkFlagRequired(flagEpochBlockNumber)
	cmd.MarkFlagRequired(flagEpochValidatorNumber)
	cmd.MarkFlagRequired(flagSdkConfPath)

	return cmd
}

// setDistributionPerBlockCMD set a configuration of block reward amount
// @return *cobra.Command
func setDistributionPerBlockCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-distribution-per-block",
		Short: "set distribution per block",
		Long:  "set distribution per block",
		RunE: func(_ *cobra.Command, _ []string) error {
			return setDistributionPerBlock()
		},
	}

	attachFlags(cmd, []string{
		flagUserSignKeyFilePath, flagUserSignCrtFilePath, flagUserTlsCrtFilePath, flagUserTlsKeyFilePath, flagChainId,
		flagSdkConfPath, flagOrgId, flagAdminCrtFilePaths, flagAdminKeyFilePaths, flagAdminOrgIds,
		flagDistributionPerBlock,
	})

	cmd.MarkFlagRequired(flagDistributionPerBlock)
	cmd.MarkFlagRequired(flagSdkConfPath)

	return cmd
}

// setSlashingPerBlockCMD set a configuration of block penalty amount
// @return *cobra.Command
func setSlashingPerBlockCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-slashing-per-block",
		Short: "set slashing per block",
		Long:  "set slashing per block",
		RunE: func(_ *cobra.Command, _ []string) error {
			return setSlashingPerBlock()
		},
	}

	attachFlags(cmd, []string{
		flagUserSignKeyFilePath, flagUserSignCrtFilePath, flagUserTlsCrtFilePath, flagUserTlsKeyFilePath, flagChainId,
		flagSdkConfPath, flagOrgId, flagAdminCrtFilePaths, flagAdminKeyFilePaths, flagAdminOrgIds,
		flagSlashingPerBlock,
	})

	cmd.MarkFlagRequired(flagSlashingPerBlock)
	cmd.MarkFlagRequired(flagSdkConfPath)

	return cmd
}

// setGasExchangeRateCMD set the gas replacement ratio
// @return *cobra.Command
func setGasExchangeRateCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-gas-exchange-rate",
		Short: "set gas exchange rate",
		Long:  "set gas exchange rate",
		RunE: func(_ *cobra.Command, _ []string) error {
			return setGasExchangeRate()
		},
	}

	attachFlags(cmd, []string{
		flagUserSignKeyFilePath, flagUserSignCrtFilePath, flagUserTlsCrtFilePath, flagUserTlsKeyFilePath, flagChainId,
		flagSdkConfPath, flagOrgId, flagAdminCrtFilePaths, flagAdminKeyFilePaths, flagAdminOrgIds,
		flagGasExchangeRate,
	})

	cmd.MarkFlagRequired(flagGasExchangeRate)
	cmd.MarkFlagRequired(flagSdkConfPath)

	return cmd
}

// updateDistributionPerBlockCMD
// Realize that the erc20 contract administrator can perform this operation.
// This time, the multi-signature method is used to perform this operation.
// @return *cobra.Command
func updateDistributionPerBlockCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-distribution-per-block",
		Short: "update distribution per block",
		Long:  "update distribution per block",
		RunE: func(_ *cobra.Command, _ []string) error {
			return updateDistributionPerBlock(distributionPerBlock)
		},
	}

	attachFlags(cmd, []string{
		flagUserSignKeyFilePath, flagUserSignCrtFilePath,
		flagSdkConfPath, flagOrgId, flagEnableCertHash,
		flagUserTlsCrtFilePath, flagUserTlsKeyFilePath,
		flagDistributionPerBlock,
	})

	cmd.MarkFlagRequired(flagDistributionPerBlock)

	return cmd
}

// updateGasExchangeRateCMD
// Realize that the erc20 contract administrator can perform this operation.
// This time, the multi-signature method is used to perform this operation.
// @return *cobra.Command
func updateGasExchangeRateCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-gas-exchange-rate",
		Short: "update gas exchange rate",
		Long:  "update gas exchange rate",
		RunE: func(_ *cobra.Command, _ []string) error {
			return updateGasExchangeRate(gasExchangeRate)
		},
	}

	attachFlags(cmd, []string{
		flagUserSignKeyFilePath, flagUserSignCrtFilePath,
		flagSdkConfPath, flagOrgId, flagEnableCertHash,
		flagUserTlsCrtFilePath, flagUserTlsKeyFilePath,
		flagGasExchangeRate,
	})

	cmd.MarkFlagRequired(flagGasExchangeRate)

	return cmd
}

// updateEpochBlocNumber
// Realize that the erc20 contract administrator can perform this operation.
// This time, the multi-signature method is used to perform this operation.
// @return *cobra.Command
//func updateEpochBlocNumber(epochBlockNumber int) error {
//	client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
//		userSignCrtFilePath, userSignKeyFilePath)
//	if err != nil {
//		return fmt.Errorf("create user client failed, %s", err.Error())
//	}
//	defer client.Stop()
//	resp, err := client.SetEpochBlockNumber(epochBlockNumber)
//	if err != nil {
//		return fmt.Errorf("update epoch bloc number failed, %s", err.Error())
//	}
//
//	fmt.Printf("response %+v\n", resp)
//	return nil
//}

// getMinSelfDelegation returns the minimum number of delegates,
// this method is not available for the time being
//func getMinSelfDelegation() error {
//	client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
//		userSignCrtFilePath, userSignKeyFilePath)
//	if err != nil {
//		return fmt.Errorf("create user client failed, %s", err.Error())
//	}
//	defer client.Stop()
//	chainConfig, err := client.GetMinSelfDelegation()
//	if err != nil {
//		return fmt.Errorf("get chain config failed, %s", err.Error())
//	}
//
//	output, err := prettyjson.Marshal(chainConfig)
//	if err != nil {
//		return err
//	}
//	fmt.Println(string(output))
//	return nil
//}

// getEpochBlockNumber returns the configuration of the number of blocks in an epoch
func getEpochBlockNumber() error {
	client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
		userSignCrtFilePath, userSignKeyFilePath)
	if err != nil {
		return fmt.Errorf("create user client failed, %s", err.Error())
	}
	defer client.Stop()
	chainConfig, err := client.GetEpochBlockNumber()
	if err != nil {
		return fmt.Errorf("get chain config failed, %s", err.Error())
	}

	output, err := prettyjson.Marshal(chainConfig)
	if err != nil {
		return err
	}
	fmt.Println(string(output))
	return nil
}

// getEpochValidatorNumber returns the configuration of the number of validators in an epoch
func getEpochValidatorNumber() error {
	client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
		userSignCrtFilePath, userSignKeyFilePath)
	if err != nil {
		return fmt.Errorf("create user client failed, %s", err.Error())
	}
	defer client.Stop()
	chainConfig, err := client.GetEpochValidatorNumber()
	if err != nil {
		return fmt.Errorf("get chain config failed, %s", err.Error())
	}

	output, err := prettyjson.Marshal(chainConfig)
	if err != nil {
		return err
	}
	fmt.Println(string(output))
	return nil
}

// getDistributionPerBlock returns a configuration of block reward amount
func getDistributionPerBlock() error {
	client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
		userSignCrtFilePath, userSignKeyFilePath)
	if err != nil {
		return fmt.Errorf("create user client failed, %s", err.Error())
	}
	defer client.Stop()
	chainConfig, err := client.GetDistributionPerBlock()
	if err != nil {
		return fmt.Errorf("get chain config failed, %s", err.Error())
	}

	output, err := prettyjson.Marshal(chainConfig)
	if err != nil {
		return err
	}
	fmt.Println(string(output))
	return nil
}

// getSlashingPerBlock returns a configuration of block penalty amount
func getSlashingPerBlock() error {
	client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
		userSignCrtFilePath, userSignKeyFilePath)
	if err != nil {
		return fmt.Errorf("create user client failed, %s", err.Error())
	}
	defer client.Stop()
	chainConfig, err := client.GetSlashingPerBlock()
	if err != nil {
		return fmt.Errorf("get chain config failed, %s", err.Error())
	}

	output, err := prettyjson.Marshal(chainConfig)
	if err != nil {
		return err
	}
	fmt.Println(string(output))
	return nil
}

// getGasExchangeRate returns the gas replacement ratio
func getGasExchangeRate() error {
	client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
		userSignCrtFilePath, userSignKeyFilePath)
	if err != nil {
		return fmt.Errorf("create user client failed, %s", err.Error())
	}
	defer client.Stop()
	chainConfig, err := client.GetGasExchangeRate()
	if err != nil {
		return fmt.Errorf("get chain config failed, %s", err.Error())
	}

	output, err := prettyjson.Marshal(chainConfig)
	if err != nil {
		return err
	}
	fmt.Println(string(output))
	return nil
}

// getClientAndAdmin create a client and return different admin data according to the certificate type
func getClientAndAdmin(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
	userSignCrtFilePath, userSignKeyFilePath string) (*sdk.ChainClient, []string,
	[]string, []string, error) {
	var adminKeys []string
	var adminCrts []string
	var adminOrgs []string
	client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
		userSignCrtFilePath, userSignKeyFilePath)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	defer client.Stop()

	if sdk.AuthTypeToStringMap[client.GetAuthType()] == protocol.PermissionedWithCert {
		if adminKeyFilePaths != "" {
			adminKeys = strings.Split(adminKeyFilePaths, ",")
		}
		if adminCrtFilePaths != "" {
			adminCrts = strings.Split(adminCrtFilePaths, ",")
		}
		if len(adminKeys) != len(adminCrts) {
			return nil, nil, nil, nil, fmt.Errorf(ADMIN_ORGID_KEY_CERT_LENGTH_NOT_EQUAL_FORMAT, len(adminKeys), len(adminCrts))
		}
	} else if sdk.AuthTypeToStringMap[client.GetAuthType()] == protocol.PermissionedWithKey {
		if adminKeyFilePaths != "" {
			adminKeys = strings.Split(adminKeyFilePaths, ",")
		}
		if adminOrgIds != "" {
			adminOrgs = strings.Split(adminOrgIds, ",")
		}
		if len(adminKeys) != len(adminOrgs) {
			return nil, nil, nil, nil, fmt.Errorf(ADMIN_ORGID_KEY_LENGTH_NOT_EQUAL_FORMAT, len(adminKeys), len(adminOrgs))
		}
	} else {
		if adminKeyFilePaths != "" {
			adminKeys = strings.Split(adminKeyFilePaths, ",")
		}
	}

	return client, adminKeys, adminCrts, adminOrgs, nil
}

// getEndorsementEntrys returns endorsement information entries, including signers and their signatures
func getEndorsementEntrys(client *sdk.ChainClient,
	adminKeys []string, adminCrts []string, adminOrgs []string,
	payload *common.Payload) ([]*common.EndorsementEntry, error) {
	endorsementEntrys := make([]*common.EndorsementEntry, len(adminKeys))
	for i := range adminKeys {
		if sdk.AuthTypeToStringMap[client.GetAuthType()] == protocol.PermissionedWithCert {
			e, err := sdkutils.MakeEndorserWithPath(adminKeys[i], adminCrts[i], payload)
			if err != nil {
				return nil, err
			}

			endorsementEntrys[i] = e
		} else if sdk.AuthTypeToStringMap[client.GetAuthType()] == protocol.PermissionedWithKey {
			e, err := sdkutils.MakePkEndorserWithPath(
				adminKeys[i],
				client.GetHashType(),
				adminOrgs[i],
				payload,
			)
			if err != nil {
				return nil, err
			}

			endorsementEntrys[i] = e
		} else {
			e, err := sdkutils.MakePkEndorserWithPath(
				adminKeys[i],
				client.GetHashType(),
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

// setMinSelfDelegation set the minimum number of delegates,
// this method is not available for the time being
//nolint: unused
func setMinSelfDelegation() error {
	client, adminKeys, adminCrts, adminOrgs, err := getClientAndAdmin(
		sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
		userSignCrtFilePath, userSignKeyFilePath)
	if err != nil {
		return err
	}
	defer client.Stop()

	payload, err := client.SetMinSelfDelegation(minSelfDelegation)
	if err != nil {
		return fmt.Errorf("create chain config block update payload failed, %s", err.Error())
	}

	endorsementEntrys, err := getEndorsementEntrys(client,
		adminKeys, adminCrts, adminOrgs,
		payload)
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

// setEpochBlocNumber set the configuration of the number of blocks in an epoch
// this method is not available for the time being
func setEpochBlocNumber() error {
	client, adminKeys, adminCrts, adminOrgs, err := getClientAndAdmin(
		sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
		userSignCrtFilePath, userSignKeyFilePath)
	if err != nil {
		return err
	}
	defer client.Stop()

	payload, err := client.SetEpochBlockNumber(epochBlockNumber)
	if err != nil {
		return fmt.Errorf("create chain config block update payload failed, %s", err.Error())
	}

	endorsementEntrys, err := getEndorsementEntrys(client,
		adminKeys, adminCrts, adminOrgs,
		payload)
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

//nolint: unused
// setEpochValidatorNumber set the configuration of the number of validators in an epoch
// this method is not available for the time being
func setEpochValidatorNumber() error {
	client, adminKeys, adminCrts, adminOrgs, err := getClientAndAdmin(
		sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
		userSignCrtFilePath, userSignKeyFilePath)
	if err != nil {
		return err
	}
	defer client.Stop()

	payload, err := client.SetEpochValidatorNumber(epochValidatorNumber)
	if err != nil {
		return fmt.Errorf("create chain config block update payload failed, %s", err.Error())
	}

	endorsementEntrys, err := getEndorsementEntrys(client,
		adminKeys, adminCrts, adminOrgs,
		payload)
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

// setEpochBlockNumberAndValidatorNumber set the configuration of the number of validators and numbers in an epoch
// modify 2 configurations at the same time,
// require the number of blocks to be an integer multiple of the number of validators
func setEpochBlockNumberAndValidatorNumber() error {
	client, adminKeys, adminCrts, adminOrgs, err := getClientAndAdmin(
		sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
		userSignCrtFilePath, userSignKeyFilePath)
	if err != nil {
		return err
	}
	defer client.Stop()

	payload, err := client.SetEpochBlockNumberAndValidatorNumber(epochBlockNumber, epochValidatorNumber)
	if err != nil {
		return fmt.Errorf("create chain config block update payload failed, %s", err.Error())
	}

	endorsementEntrys, err := getEndorsementEntrys(client,
		adminKeys, adminCrts, adminOrgs,
		payload)
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

// setDistributionPerBlock set a configuration of block reward amount
func setDistributionPerBlock() error {
	client, adminKeys, adminCrts, adminOrgs, err := getClientAndAdmin(
		sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
		userSignCrtFilePath, userSignKeyFilePath)
	if err != nil {
		return err
	}
	defer client.Stop()

	payload, err := client.SetDistributionPerBlock(distributionPerBlock)
	if err != nil {
		return fmt.Errorf("create chain config block update payload failed, %s", err.Error())
	}

	endorsementEntrys, err := getEndorsementEntrys(client,
		adminKeys, adminCrts, adminOrgs,
		payload)
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

// setSlashingPerBlock set a configuration of block penalty amount
func setSlashingPerBlock() error {
	client, adminKeys, adminCrts, adminOrgs, err := getClientAndAdmin(
		sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
		userSignCrtFilePath, userSignKeyFilePath)
	if err != nil {
		return err
	}
	defer client.Stop()

	payload, err := client.SetSlashingPerBlock(slashingPerBlock)
	if err != nil {
		return fmt.Errorf("create chain config block update payload failed, %s", err.Error())
	}

	endorsementEntrys, err := getEndorsementEntrys(client,
		adminKeys, adminCrts, adminOrgs,
		payload)
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

// setGasExchangeRate set the gas replacement ratio
func setGasExchangeRate() error {
	client, adminKeys, adminCrts, adminOrgs, err := getClientAndAdmin(
		sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
		userSignCrtFilePath, userSignKeyFilePath)
	if err != nil {
		return err
	}
	defer client.Stop()

	payload, err := client.SetGasExchangeRate(gasExchangeRate)
	if err != nil {
		return fmt.Errorf("create chain config block update payload failed, %s", err.Error())
	}

	endorsementEntrys, err := getEndorsementEntrys(client,
		adminKeys, adminCrts, adminOrgs,
		payload)
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

// updateDistributionPerBlock
// Realize that the erc20 contract administrator can perform this operation.
// This time, the multi-signature method is used to perform this operation.
//nolint: unused
func updateDistributionPerBlock(distributionPerBlock int) error {
	client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
		userSignCrtFilePath, userSignKeyFilePath)
	if err != nil {
		return fmt.Errorf("create user client failed, %s", err.Error())
	}
	defer client.Stop()
	resp, err := client.SetDistributionPerBlock(distributionPerBlock)
	if err != nil {
		return fmt.Errorf("update distribution per block failed, %s", err.Error())
	}

	fmt.Printf("response %+v\n", resp)
	return nil
}

// updateGasExchangeRate
// Realize that the erc20 contract administrator can perform this operation.
// This time, the multi-signature method is used to perform this operation.
//nolint: unused
func updateGasExchangeRate(gasExchangeRate int) error {
	client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
		userSignCrtFilePath, userSignKeyFilePath)
	if err != nil {
		return fmt.Errorf("create user client failed, %s", err.Error())
	}
	defer client.Stop()
	resp, err := client.SetGasExchangeRate(gasExchangeRate)
	if err != nil {
		return fmt.Errorf("update gas exchange rate failed, %s", err.Error())
	}

	fmt.Printf("response %+v\n", resp)
	return nil
}
