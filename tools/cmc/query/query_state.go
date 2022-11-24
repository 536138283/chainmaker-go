// Copyright (C) BABEC. All rights reserved.
// Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package query

import (
	"fmt"

	"chainmaker.org/chainmaker-go/tools/cmc/util"
	sdk "chainmaker.org/chainmaker/sdk-go/v2"
	"github.com/hokaccha/go-prettyjson"
	"github.com/spf13/cobra"
)

// newQueryStateByKeyOnChainCMD `query tx` command implementation
func newQueryStateByKeyOnChainCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "state-by-key [contractName] [key]",
		Short: "query on-chain state by contract name and key",
		Long:  "query on-chain state by contract name and key",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			//// 1.Chain Client
			cc, err := sdk.NewChainClient(
				sdk.WithConfPath(sdkConfPath),
				sdk.WithChainClientChainId(chainId),
			)
			if err != nil {
				return err
			}
			defer cc.Stop()
			if err := util.DealChainClientCertHash(cc, enableCertHash); err != nil {
				return err
			}

			value, err := cc.GetStateByKey(args[0], args[1], sdk.Format(format))
			if err != nil {
				return err
			}
			fmt.Println(string(value))
			return nil
		},
	}

	util.AttachAndRequiredFlags(cmd, flags, []string{
		flagSdkConfPath,
	})
	util.AttachFlags(cmd, flags, []string{
		flagEnableCertHash, flagFormat,
	})
	return cmd
}
func newQueryStateByPrefixOnChainCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "state-by-prefix [contractName] [prefix]",
		Short: "query on-chain state by contract name and prefix",
		Long:  "query on-chain state by contract name and prefix",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			//// 1.Chain Client
			cc, err := sdk.NewChainClient(
				sdk.WithConfPath(sdkConfPath),
				sdk.WithChainClientChainId(chainId),
			)
			if err != nil {
				return err
			}
			defer cc.Stop()
			if err := util.DealChainClientCertHash(cc, enableCertHash); err != nil {
				return err
			}

			kvdata, err := cc.GetStateByPrefix(args[0], args[1], sdk.Paging(pageSize, pageIndex))
			if err != nil {
				return err
			}
			output, err := prettyjson.Marshal(kvdata)
			if err != nil {
				return err
			}
			fmt.Println(string(output))
			return nil
		},
	}

	util.AttachAndRequiredFlags(cmd, flags, []string{
		flagSdkConfPath,
	})
	util.AttachFlags(cmd, flags, []string{
		flagEnableCertHash, flagPageIndex, flagPageSize,
	})
	return cmd
}
