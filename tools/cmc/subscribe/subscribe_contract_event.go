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

// newSubContractEventCMD subscribe real-time/history contract events
// @return *cobra.Command
func newSubContractEventCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "event",
		Short: "subscribe real-time/history contract events",
		Long:  "subscribe real-time/history contract events",
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
			payload := cc.CreateSubscribeContractEventPayload(startBlock, endBlock, contractName, topic)

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

					event := item.(*common.ContractEventInfo)
					fmt.Printf("⬇⬇⬇⬇ %s ⬇⬇⬇⬇\n", event.TxId)
					util.PrintPrettyJson(item)
				case <-ctx.Done():
					return nil
				}
			}
		},
	}

	util.AttachFlags(cmd, flags, []string{
		flagStartBlock, flagEndBlock, flagContractName, flagTopic,
	})
	util.AttachAndRequiredFlags(cmd, flags, []string{
		flagSdkConfPath,
	})
	return cmd
}
