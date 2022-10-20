// Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package subscribe

import (
	"context"
	"fmt"

	"chainmaker.org/chainmaker-go/tools/cmc/util"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	"github.com/spf13/cobra"
)

// newSubBlockCMD subscribe real-time/history blocks
// @return *cobra.Command
func newSubBlockCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "block",
		Short: "subscribe real-time/history blocks",
		Long:  "subscribe real-time/history blocks",
		RunE: func(cmd *cobra.Command, args []string) error {
			// chain client
			cc, err := util.CreateChainClientWithConfPath(sdkConfPath, false)
			if err != nil {
				return err
			}
			defer cc.Stop()

			// make subscribe payload
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			payload := cc.CreateSubscribeBlockPayload(startBlock, endBlock, withRWSet, onlyHeader)

			// make endorsers
			//adminKeys, adminCrts, adminOrgs, err := util.MakeAdminInfo(cc, adminKeyFilePaths, adminCrtFilePaths, adminOrgIds)
			//if err != nil {
			//	return err
			//}
			//endorsers, err := util.MakeEndorsement(adminKeys, adminCrts, adminOrgs, cc, payload)
			//if err != nil {
			//	return err
			//}

			// subscribing data
			dataC, err := cc.Subscribe(ctx, payload)
			if err != nil {
				return err
			}
			fmt.Println("Subscribe started!")

			for {
				select {
				case item, ok := <-dataC:
					if !ok {
						fmt.Println("Subscribe ended!")
						return nil
					}

					var blockHeight uint64
					if onlyHeader {
						blockHeader := item.(*common.BlockHeader)
						blockHeight = blockHeader.BlockHeight
					} else {
						blockInfo := item.(*common.BlockInfo)
						blockHeight = blockInfo.Block.Header.BlockHeight
					}

					fmt.Printf("⬇⬇⬇⬇ %d ⬇⬇⬇⬇\n", blockHeight)
					util.PrintPrettyJson(item)
				case <-ctx.Done():
					return nil
				}
			}
		},
	}

	util.AttachFlags(cmd, flags, []string{
		//flagAdminKeyFilePaths, flagAdminCrtFilePaths, flagAdminOrgIds,
		flagStartBlock, flagEndBlock, flagWithRWSet, flagOnlyHeader,
	})
	util.AttachAndRequiredFlags(cmd, flags, []string{
		flagSdkConfPath,
	})
	return cmd
}
