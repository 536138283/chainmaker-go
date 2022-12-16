package snapshot

import (
	"fmt"

	"errors"

	"chainmaker.org/chainmaker-go/tools/cmc/util"
	"chainmaker.org/chainmaker/pb-go/v3/common"
	sdk "chainmaker.org/chainmaker/sdk-go/v3"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	// Common flags
	flagSnapshotHeight = "snapshot-height"
	flagSdkConfPath    = "sdk-conf-path"
	flagChainId        = "chain-id"
	// Secret Key for calc Hmac
	flagSecretKey = "secret-key"

	// Send Snapshot Request timeout
	snapshotRequestTimeout = 20 // 20s
)

var (
	snapshotHeight uint64
	// sdk config file path
	sdkConfPath string
	chainId     string
	secretKey   string

	flags *pflag.FlagSet
)

func init() {
	flags = &pflag.FlagSet{}

	flags.Uint64Var(&snapshotHeight, flagSnapshotHeight, 0, "This number is the block height of the "+
		"corresponding block when the snapshot was made")
	flags.StringVar(&chainId, flagChainId, "", "Chain ID")
	flags.StringVar(&sdkConfPath, flagSdkConfPath, "", "specify sdk config path")
	flags.StringVar(&secretKey, flagSecretKey, "", "Secret Key for calc Hmac")
}

// NewSnapshotCMD , create a new snapshot command
func NewSnapshotCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "make_snapshot",
		Short: "make snapshot",
		Long:  "make a snapshot by a given height",
		RunE: func(cmd *cobra.Command, args []string) error {
			// try target is block height
			return runMakeSnapshotByHeightCMD(snapshotHeight)
		},
	}

	util.AttachAndRequiredFlags(cmd, flags, []string{
		flagSdkConfPath, flagChainId, flagSnapshotHeight, flagSecretKey,
	})

	return cmd
}

// runMakeSnapshotByHeightCMD `make_snapshot` command implementation
func runMakeSnapshotByHeightCMD(targetBlkHeight uint64) error {
	//// 1.Chain Client
	cc, err := util.CreateChainClient(sdkConfPath, chainId, "", "", "", "", "")
	if err != nil {
		return err
	}
	defer cc.Stop()

	return snapshotBlockOnChain(cc, targetBlkHeight)

}

// snapshotBlockOnChain Build & Sign & Send a make snapshot Request
func snapshotBlockOnChain(cc *sdk.ChainClient, height uint64) error {
	var (
		err                error
		payload            *common.Payload
		signedPayloadBytes *common.Payload
		resp               *common.TxResponse
	)

	payload, err = cc.CreateSnapshotBlockPayload(height)
	if err != nil {
		return err
	}

	signedPayloadBytes, err = cc.SignSnapshotPayload(payload)
	if err != nil {
		return err
	}

	resp, err = cc.SendSnapshotRequest(signedPayloadBytes, snapshotRequestTimeout)
	if err != nil {
		return err
	}

	return util.CheckProposalRequestResp(resp, false)
}

// NewGetSnapshotStatusCMD , create new cmd of get snapshot status
func NewGetSnapshotStatusCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get_snapshot_status",
		Short: "get snapshot status",
		Long:  "get a snapshot status",
		RunE: func(cmd *cobra.Command, args []string) error {
			// try target is block height
			return runGetnapshotStatusCMD()
		},
	}

	util.AttachAndRequiredFlags(cmd, flags, []string{
		flagSdkConfPath, flagChainId, flagSecretKey,
	})

	return cmd
}

// runGetnapshotStatusCMD `get_snapshot_status` command implementation
func runGetnapshotStatusCMD() error {
	//// 1.Chain Client
	cc, err := util.CreateChainClient(sdkConfPath, chainId, "", "", "", "", "")
	if err != nil {
		return err
	}
	defer cc.Stop()

	return getSnapshotStatus(cc)

}

// getSnapshotStatus Build & Sign & Send a get snapshot status Request
func getSnapshotStatus(cc *sdk.ChainClient) error {
	var (
		err                error
		payload            *common.Payload
		signedPayloadBytes *common.Payload
		resp               *common.TxResponse
	)

	payload, err = cc.CreateGetSnapshotStatusPayload()
	if err != nil {
		return err
	}

	signedPayloadBytes, err = cc.SignSnapshotPayload(payload)
	if err != nil {
		return err
	}

	resp, err = cc.SendSnapshotRequest(signedPayloadBytes, snapshotRequestTimeout)
	if err != nil {
		return err
	}

	if resp.Code == common.TxStatusCode_MAKE_SNAPSHOT_STATUS_FINISH {
		fmt.Printf("make snapshot finish \n")
		return nil
	}
	if resp.Code == common.TxStatusCode_MAKE_SNAPSHOT_STATUS_UNFINISHED {
		fmt.Printf("make snapshot unfinished \n")
		return nil
	}

	fmt.Printf("error, resp code :[%d],resp message :[%s]  \n", resp.Code, resp.Message)
	return errors.New(resp.Message)
}
