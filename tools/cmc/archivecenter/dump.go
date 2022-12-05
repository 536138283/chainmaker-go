package archivecenter

import (
	"context"
	"fmt"
	"time"

	"chainmaker.org/chainmaker-go/tools/cmc/util"
	"chainmaker.org/chainmaker/pb-go/v2/archivecenter"
	chainmaker_sdk_go "chainmaker.org/chainmaker/sdk-go/v2"
	"github.com/gosuri/uiprogress"
	"github.com/spf13/cobra"
)

const (
	genesisBlockHeight = 0
)

var (
	grpcTimeoutSeconds = 5
)

func newDumpCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dump",
		Short: "dump blockchain data",
		Long:  "dump blockchain data to archive center storage",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(chainId) == 0 ||
				len(sdkConfPath) == 0 ||
				len(archiveCenterConfPath) == 0 {
				return fmt.Errorf("%s or %s or %s must not be empty",
					flagChainId, flagSdkConfPath, flagArchiveConfPath)
			}
			if archiveBeginHeight >= archiveEndHeight {
				return fmt.Errorf("%s must be greater than %s",
					flagArchiveEndHeight, flagArchiveBeginHeight)
			}
			return runDumpCMD(archiveBeginHeight, archiveEndHeight)
		},
	}
	util.AttachAndRequiredFlags(cmd, flags,
		[]string{
			flagSdkConfPath, flagArchiveConfPath, flagChainId,
			flagArchiveBeginHeight, flagArchiveEndHeight,
		})
	return cmd
}

func runDumpCMD(beginHeight, endHeight uint64) error {
	// create chain client
	cc, err := util.CreateChainClient(sdkConfPath, chainId,
		"", "", "", "", "")
	if err != nil {
		return err
	}
	defer cc.Stop()
	// create archivecenter grpc client
	archiveClient, clientErr := createArchiveCenerGrpcClient(archiveCenterConfPath)
	if clientErr != nil {
		return clientErr
	}
	defer archiveClient.Stop()
	// 1. register genesis block
	genesisHash, registerErr := registerChainToArchiveCenter(cc, archiveClient)
	if registerErr != nil {
		return registerErr
	}
	clientStream, clientStreamErr := archiveClient.client.ArchiveBlocks(context.Background(),
		archiveClient.GrpcCallOption()...)
	if clientStreamErr != nil {
		return clientStreamErr
	}
	barCount := archiveEndHeight - archiveBeginHeight + 1
	progress := uiprogress.New()
	bar := progress.AddBar(int(barCount)).AppendCompleted().PrependElapsed()
	bar.PrependFunc(func(b *uiprogress.Bar) string {
		return fmt.Sprintf("Archiving Blocks (%d/%d)", b.Current(), barCount)
	})
	progress.Start()
	defer progress.Stop()
	var archiveError error
	for tempHeight := archiveBeginHeight; tempHeight <= archiveEndHeight; tempHeight++ {
		archiveError = archiveBlockByHeight(genesisHash, tempHeight,
			cc, clientStream)
		if archiveError != nil {
			break
		}
		bar.Incr()
	}
	archiveRespErr := clientStream.CloseSend()
	if archiveRespErr != nil {
		return fmt.Errorf("stream close recv error %s", archiveRespErr.Error())
	}
	if archiveError != nil {
		return archiveError
	}
	return nil
}

func registerChainToArchiveCenter(chainClient *chainmaker_sdk_go.ChainClient,
	archiveClient *ArchiveCenterClient) (string, error) {
	genesisBlock, genesisErr := chainClient.GetFullBlockByHeight(genesisBlockHeight)
	if genesisErr != nil {
		return "", fmt.Errorf("query genesis block error %s", genesisErr.Error())
	}
	genesisHash := genesisBlock.Block.GetBlockHashStr()
	ctx, ctxCancel := context.WithTimeout(context.Background(),
		time.Duration(grpcTimeoutSeconds)*time.Second)
	defer ctxCancel()
	registerResp, registerError := archiveClient.client.Register(ctx,
		&archivecenter.ArchiveBlockRequest{
			ChainUnique: genesisHash,
			Block:       genesisBlock,
		}, archiveClient.GrpcCallOption()...)
	if registerError != nil {
		return genesisHash, fmt.Errorf("register genesis rpc error %s", genesisErr.Error())
	}
	if registerResp == nil {
		return genesisHash, fmt.Errorf("register genesis rpc no response")
	}
	if registerResp.Code == 0 &&
		registerResp.RegisterStatus == archivecenter.RegisterStatus_RegisterStatusSuccess {
		return genesisHash, nil
	}
	return genesisHash, fmt.Errorf("register got code %d , message %s, status %d",
		registerResp.Code, registerResp.Message, registerResp.RegisterStatus)
}

func archiveBlockByHeight(chainGenesis string, height uint64,
	chainClient *chainmaker_sdk_go.ChainClient,
	archiveClient archivecenter.ArchiveCenterServer_ArchiveBlocksClient) error {
	block, blockError := chainClient.GetFullBlockByHeight(height)
	if blockError != nil {
		return fmt.Errorf("query block height %d got error %s",
			height, blockError.Error())

	}
	sendErr := archiveClient.Send(&archivecenter.ArchiveBlockRequest{
		ChainUnique: chainGenesis,
		Block:       block,
	})
	if sendErr != nil {
		return fmt.Errorf("send height %d got error %s",
			height, sendErr.Error())
	}
	archiveResp, archiveRespErr := archiveClient.Recv()
	if archiveRespErr != nil {
		return fmt.Errorf("send height %d got error %s", height, archiveRespErr.Error())
	}
	if archiveResp.ArchiveStatus == archivecenter.ArchiveStatus_ArchiveStatusFailed {
		return fmt.Errorf("send height %d failed %s ", height, archiveResp.Message)
	}
	return nil
}
