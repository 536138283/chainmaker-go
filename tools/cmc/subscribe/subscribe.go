// Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package subscribe

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	sdkConfPath       string
	adminKeyFilePaths string
	adminCrtFilePaths string
	adminOrgIds       string
	resultToString    bool

	startBlock, endBlock int64
	contractName         string
	txIds                []string
	topic                string

	withRWSet, onlyHeader bool
)

const (
	flagSdkConfPath       = "sdk-conf-path"
	flagAdminKeyFilePaths = "admin-key-file-paths"
	flagAdminCrtFilePaths = "admin-crt-file-paths"
	flagAdminOrgIds       = "admin-org-ids"
	flagResultToString    = "result-to-string"

	flagStartBlock   = "start-block"
	flagEndBlock     = "end-block"
	flagContractName = "contract-name"
	flagTxIds        = "tx-ids"
	flagTopic        = "topic"

	flagWithRWSet  = "with-rw-set"
	flagOnlyHeader = "only-header"
)

// NewSubscribeCMD new subscribe tx/block/contract_event command
func NewSubscribeCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sub",
		Short: "subscribe tx/block/contract_event",
		Long:  "subscribe tx/block/contract_event",
	}

	cmd.AddCommand(newSubTxCMD())
	cmd.AddCommand(newSubContractEventCMD())
	cmd.AddCommand(newSubBlockCMD())

	return cmd
}

var flags *pflag.FlagSet

func init() {
	flags = &pflag.FlagSet{}

	flags.StringVar(&sdkConfPath, flagSdkConfPath, "", "specify sdk config path")
	flags.StringVar(&adminKeyFilePaths, flagAdminKeyFilePaths, "", "specify admin key file paths, use ',' to separate")
	flags.StringVar(&adminCrtFilePaths, flagAdminCrtFilePaths, "", "specify admin cert file paths, use ',' to separate")
	flags.StringVar(&adminOrgIds, flagAdminOrgIds, "", "specify admin org-ids, use ',' to separate")
	flags.BoolVar(&resultToString, flagResultToString, false,
		"enable convert Transaction.Result.ContractResult.Result to string for readable output")

	flags.Int64Var(&startBlock, flagStartBlock, -1, `The block number to start with when subscribing. 
Default -1 means start from the latest block (exclusive)`)
	flags.Int64Var(&endBlock, flagEndBlock, -1,
		"The block number to end when subscribing. Default -1 means never end subscription")
	flags.StringVar(&contractName, flagContractName, "",
		`Subscribe to the transaction or event of the specified contract name. 
Default empty string means that the contract name filtering rules are not applied`)
	flags.StringSliceVar(&txIds, flagTxIds, nil, `txids for subscribe txs, use ',' to separate. 
Default empty string means that the txid filtering rules are not applied`)
	flags.StringVar(&topic, flagTopic, "", `Subscribe to the contract event of the specified topic. 
Default empty string means that the topic filtering rules are not applied`)

	flags.BoolVar(&withRWSet, flagWithRWSet, false, "Whether the subscribed block contains read-write set")
	flags.BoolVar(&onlyHeader, flagOnlyHeader, false, "Whether to only subscribe to block headers")
}
