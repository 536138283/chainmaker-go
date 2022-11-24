// Copyright (C) BABEC. All rights reserved.
// Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

// Package query query block chain
package query

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	sdkConfPath    string
	chainId        string
	enableCertHash bool
	withRWSet      bool
	truncateValue  bool
	format         string
	pageSize       int
	pageIndex      int
)

const (
	flagSdkConfPath    = "sdk-conf-path"
	flagChainId        = "chain-id"
	flagEnableCertHash = "enable-cert-hash"
	flagWithRWSet      = "with-rw-set"
	flagTruncateValue  = "truncate-value"
	flagFormat         = "format"
	flagPageSize       = "page-size"
	flagPageIndex      = "page-index"
)

// NewQueryOnChainCMD new query on-chain blockchain data command
func NewQueryOnChainCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query",
		Short: "query on-chain blockchain data",
		Long:  "query on-chain blockchain data",
	}

	cmd.AddCommand(newQueryTxOnChainCMD())
	cmd.AddCommand(newQueryBlockByHeightOnChainCMD())
	cmd.AddCommand(newQueryBlockByHashOnChainCMD())
	cmd.AddCommand(newQueryBlockByTxIdOnChainCMD())
	cmd.AddCommand(newQueryArchivedHeightOnChainCMD())
	cmd.AddCommand(newQueryContractOnChainCMD())
	cmd.AddCommand(newQueryStateByKeyOnChainCMD())
	cmd.AddCommand(newQueryStateByPrefixOnChainCMD())

	return cmd
}

var flags *pflag.FlagSet

func init() {
	flags = &pflag.FlagSet{}

	flags.StringVar(&chainId, flagChainId, "", "Chain ID")
	flags.StringVar(&sdkConfPath, flagSdkConfPath, "", "specify sdk config path")
	flags.StringVar(&format, flagFormat, "", "specify result format:string/hex/json/raw/hex0x/pb")
	flags.BoolVar(&enableCertHash, flagEnableCertHash, true, "whether enable cert hash")
	flags.BoolVar(&withRWSet, flagWithRWSet, true, "whether with RWSet")
	flags.BoolVar(&truncateValue, flagTruncateValue, true, "enable truncate value, default true")
	flags.IntVar(&pageIndex, flagPageIndex, 0, "page index")
	flags.IntVar(&pageSize, flagPageSize, 0, "page size")
}
