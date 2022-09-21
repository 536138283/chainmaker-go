/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"fmt"
	"strings"

	"chainmaker.org/chainmaker-go/tools/cmc/util"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/consensus"
	sdk "chainmaker.org/chainmaker/sdk-go/v2"
	"github.com/spf13/cobra"
)

type (
	switchAction func(cc *sdk.ChainClient, extConfig, specificConf []*common.KeyValuePair) (*common.Payload, error)
	actionIdx    int16
)

var (
	//switchActions mapping actionIdx which based on consens from and to type and switchAction
	switchActions = map[actionIdx]switchAction{
		actionIndex(consensus.ConsensusType_TBFT, consensus.ConsensusType_RAFT): func(
			cc *sdk.ChainClient, extConfig, specificConf []*common.KeyValuePair) (*common.Payload, error) {
			return cc.CreateTbftToRaftPayload(extConfig)
		},
		actionIndex(consensus.ConsensusType_RAFT, consensus.ConsensusType_TBFT): func(
			cc *sdk.ChainClient, extConfig, specificConf []*common.KeyValuePair) (*common.Payload, error) {
			return cc.CreateRaftToTbftPayload(extConfig)
		},
	}
)

//actionIndex get an int16 value
//upper 8 bits represent the consensus from type
//and lower 8 bits represent the consensus to type
func actionIndex(from, to consensus.ConsensusType) actionIdx {
	return actionIdx(int16(from<<8) | int16(to))
}

//getConsensusType convert consensus name to consensus type
func getConsensusType(name string) (consensus.ConsensusType, bool) {
	t, ok := consensus.ConsensusType_value[strings.ToUpper(name)]
	return consensus.ConsensusType(t), ok
}

//getSwitchAction get the switching action based on the consensus from and to value
func getSwitchAction(from, to consensus.ConsensusType) (switchAction, error) {
	a, ok := switchActions[actionIndex(from, to)]
	if !ok {
		return nil, fmt.Errorf("switch from [%s] to [%s] is not supported", from.String(), to.String())
	}
	return a, nil
}

func configSwitchConsensusCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "consensus",
		Short: "consensus command",
		Long:  "consensus command",
	}
	cmd.AddCommand(switchConsensusCMD())
	return cmd
}

func switchConsensusCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "switch",
		Short: "switch consensus command",
		Long:  "switch consensus command",
		RunE: func(_ *cobra.Command, _ []string) error {
			return switchConsensus()
		},
	}

	attachFlags(cmd, []string{
		flagUserSignKeyFilePath, flagUserSignCrtFilePath,
		flagSdkConfPath, flagOrgId, flagEnableCertHash, flagConsensusFrom, flagConsensusTo, flagSyncResult,
		flagAdminCrtFilePaths, flagAdminKeyFilePaths, flagAdminOrgIds, flagUserTlsCrtFilePath, flagUserTlsKeyFilePath,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagConsensusFrom)
	cmd.MarkFlagRequired(flagConsensusTo)

	return cmd
}

func switchConsensus() error {
	cc, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
		userSignCrtFilePath, userSignKeyFilePath)
	if err != nil {
		return err
	}
	defer cc.Stop()

	adminKeys, adminCrts, adminOrgs, err := util.MakeAdminInfo(cc, adminKeyFilePaths, adminCrtFilePaths, adminOrgIds)
	if err != nil {
		return err
	}

	consensusFrom, consensusTo = strings.TrimSpace(consensusFrom), strings.TrimSpace(consensusTo)
	if len(consensusFrom) == 0 || len(strings.TrimSpace(consensusTo)) == 0 {
		return fmt.Errorf("invalid argument")
	}
	from, ok := getConsensusType(consensusFrom)
	if !ok {
		return fmt.Errorf("no consensus of this type[%s], please check it", consensusFrom)
	}
	to, ok := getConsensusType(consensusTo)
	if !ok {
		return fmt.Errorf("no consensus of this type[%s], please check it", consensusTo)
	}
	action, err := getSwitchAction(from, to)
	if err != nil {
		return err
	}
	payload, err := action(cc, nil, nil)
	if err != nil {
		return err
	}

	endors, err := util.MakeEndorsement(adminKeys, adminCrts, adminOrgs, cc, payload)
	if err != nil {
		return err
	}
	resp, err := cc.SendChainConfigUpdateRequest(payload, endors, timeout, syncResult)
	if err != nil {
		return err
	}
	err = util.CheckProposalRequestResp(resp, false)
	if err != nil {
		return err
	}
	fmt.Printf("consensus switch response %+v\n", resp)
	return nil
}
