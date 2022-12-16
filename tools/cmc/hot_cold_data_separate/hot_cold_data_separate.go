package hot_cold_data_separate

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
	flagStartHeight = "start-height"
	flagEndHeight   = "end-height"
	flagSdkConfPath = "sdk-conf-path"
	flagChainId     = "chain-id"
	flagJobID       = "job-id"
	// Secret Key for calc Hmac
	flagSecretKey = "secret-key"

	// Send  Request timeout
	requestTimeout = 20 // 20s
)

var (
	startHeight uint64
	endHeight   uint64
	jobID       string
	// sdk config file path
	sdkConfPath string
	chainId     string
	secretKey   string

	flags *pflag.FlagSet
)

func init() {
	flags = &pflag.FlagSet{}

	flags.Uint64Var(&startHeight, flagStartHeight, 0, "This number is the specified starting block height"+
		" when doing hot and cold data")
	flags.Uint64Var(&endHeight, flagEndHeight, 0, "This number is the specified ending block height"+
		" when doing hot and cold data")
	flags.StringVar(&jobID, flagJobID, "", "job id for hot cold data separate job")
	flags.StringVar(&chainId, flagChainId, "", "Chain ID")
	flags.StringVar(&sdkConfPath, flagSdkConfPath, "", "specify sdk config path")
	flags.StringVar(&secretKey, flagSecretKey, "", "Secret Key for calc Hmac")
}

// NewHotColdDataSeparateCMD , new command for hot-cold-data-separate
func NewHotColdDataSeparateCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hot_cold_separate",
		Short: "do hot cold separate",
		Long:  "do a job of hot cold data separate by given start height and end height",
		RunE: func(cmd *cobra.Command, args []string) error {
			// try target is block height
			return runHotColdDataSeparate()
		},
	}

	util.AttachAndRequiredFlags(cmd, flags, []string{
		flagSdkConfPath, flagChainId, flagStartHeight, flagEndHeight, flagSecretKey,
	})

	return cmd
}

// runHotColdDataSeparate `hot_cold_separate` command implementation
func runHotColdDataSeparate() error {
	//// 1.Chain Client
	cc, err := util.CreateChainClient(sdkConfPath, chainId, "", "", "", "", "")
	if err != nil {
		return err
	}
	defer cc.Stop()

	return hotColdSeparateOnChain(cc, startHeight, endHeight)

}

// hotColdSeparateOnChain Build & Sign & Send a make hot cold separate Request
func hotColdSeparateOnChain(cc *sdk.ChainClient, startHeight, endHeight uint64) error {
	var (
		err                error
		payload            *common.Payload
		signedPayloadBytes *common.Payload
		resp               *common.TxResponse
	)

	payload, err = cc.CreateHotColdDataSeparateBlockPayload(startHeight, endHeight)
	if err != nil {
		return err
	}

	signedPayloadBytes, err = cc.SignHotColdDataSeparatePayload(payload)
	if err != nil {
		return err
	}

	resp, err = cc.SendHotColdDataSeparateRequest(signedPayloadBytes, requestTimeout)
	if err != nil {
		return err
	}

	err = util.CheckProposalRequestResp(resp, false)
	if err != nil {
		return err
	}
	fmt.Printf("job of hot cold data separate start, jobID:[%s] \n", resp.Message)
	return nil
}

// NewGetHotColdDataSeparateJobCMD , new command for getting hot-cold-data-separate job info
func NewGetHotColdDataSeparateJobCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get_hot_cold_separate_jobInfo",
		Short: "get hot cold separate job info ",
		Long:  "get detail information about hot cold data separate's job ",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGetHotColdSeparateJobInfoCMD()
		},
	}

	util.AttachAndRequiredFlags(cmd, flags, []string{
		flagSdkConfPath, flagChainId, flagSecretKey, flagJobID,
	})

	return cmd
}

// runGetHotColdSeparateJobInfoCMD `get_hot_cold_separate_jobInfo` command implementation
func runGetHotColdSeparateJobInfoCMD() error {
	//// 1.Chain Client
	cc, err := util.CreateChainClient(sdkConfPath, chainId, "", "", "", "", "")
	if err != nil {
		return err
	}
	defer cc.Stop()

	return getJobInfo(cc)

}

// getJobInfo Build & Sign & Send a get job info  Request
func getJobInfo(cc *sdk.ChainClient) error {
	var (
		err                error
		payload            *common.Payload
		signedPayloadBytes *common.Payload
		resp               *common.TxResponse
	)

	payload, err = cc.CreateGetHotColdDataSeparatePayload(jobID)
	if err != nil {
		return err
	}

	signedPayloadBytes, err = cc.SignHotColdDataSeparatePayload(payload)
	if err != nil {
		return err
	}

	resp, err = cc.SendHotColdDataSeparateRequest(signedPayloadBytes, requestTimeout)
	if err != nil {
		return err
	}

	if resp.Code == common.TxStatusCode_SUCCESS {
		fmt.Printf("get jobInfo:[%s] \n", resp.Message)
		return nil
	}

	fmt.Printf("error, resp code :[%d],resp message :[%s]  \n", resp.Code, resp.Message)
	return errors.New(resp.Message)
}
