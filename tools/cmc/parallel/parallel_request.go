package parallel

import (
	apiPb "chainmaker.org/chainmaker/pb-go/v2/api"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"context"
	"fmt"
	"github.com/gogo/protobuf/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
	"time"
)

func subNodes(statistician *Statistician) {
	threads, err := threadFactory(nodeNum, invokerMethod, nil, nil, statistician)
	if err != nil {
		fmt.Println("subNodes threadFactory err:", err)
		return
	}
	params := make([]*commonPb.TxRequest, nodeNum)

	blockHeight, err := getBlockHeight()
	if err != nil {
		fmt.Println("getBlockHeight err:", err)
		return
	}
	fmt.Println("blockHeight:", blockHeight)
	for i := 0; i < nodeNum; i++ {
		s := SubscribeBlock{
			blockHeight: blockHeight + 1,
		}
		params[i], err = s.Build(i)
		if err != nil {
			fmt.Println("error building subscribe params:", err)
			return
		}
	}
	for i := 0; i < nodeNum; i++ {
		go func(index int) {
			//defer wg.Done()
			if err := subscribeNewBlock(context.Background(), threads[index], params[index]); err != nil {
				fmt.Println("error sendSubscribe :", err)
				return
			}
		}(i)
	}
}

// 订阅区块
func subscribeNewBlock(ctx context.Context, thread *Thread, req *commonPb.TxRequest) error {
	resp, err := thread.client.Subscribe(ctx, req)
	if err != nil {
		return err
	}
	fmt.Println("subscribe start")
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				var result *commonPb.SubscribeResult
				result, err = resp.Recv()
				if err == io.EOF || err != nil {
					fmt.Printf("subscribe node[%d] receive err: %s\n", thread.index, err.Error())
					return
				}
				blockHeader := &commonPb.BlockHeader{}
				if err = proto.Unmarshal(result.Data, blockHeader); err != nil {
					return
				}
				thread.statistician.cReqStatC <- &cReqStat{
					blockHeader, thread.index, time.Now().Unix(),
				}
			}

		}
	}()
	return err
}

func sendRequest(client apiPb.RpcNodeClient, orgId string, loopId int, req *commonPb.TxRequest) error {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Duration(requestTimeout)*time.Second))
	defer cancel()
	result, err := client.SendRequest(ctx, req)
	if err != nil {
		if statusErr, ok := status.FromError(err); ok && statusErr.Code() == codes.DeadlineExceeded {
			return fmt.Errorf("client.call err: deadline\n")
		}
		return fmt.Errorf("client.call err: %v\n", err)
	}
	if outputResult {
		msg := fmt.Sprintf(resultStr, orgId, loopId, result.ContractResult, result.TxId, result)
		fmt.Println(msg)
	}
	return nil
}

func getBlockHeight() (uint64, error) {
	threads, err := threadFactory(1, "", nil, nil, nil)
	if err != nil {
		fmt.Printf("getBlockHeight err: %v\n", err)
		return 0, err
	}
	builder := QueryBlockHeight{}
	param, err := builder.Build(0)
	if err != nil {
		fmt.Printf("fail to build query block height: %v\n", err)
		return 0, err
	}
	resp, err := threads[0].client.SendRequest(context.Background(), param)
	if err != nil {
		if statusErr, ok := status.FromError(err); ok && statusErr.Code() == codes.DeadlineExceeded {
			return 0, fmt.Errorf("client.call err: deadline\n")
		}
		return 0, err
	}
	blockInfo := &commonPb.BlockInfo{}
	if err = proto.Unmarshal(resp.ContractResult.Result, blockInfo); err != nil {
		return 0, fmt.Errorf("fail to unmarshal block height: %v\n", err)
	}
	fmt.Println("height: ", blockInfo.Block.Header.BlockHeight)
	return blockInfo.Block.Header.BlockHeight, nil
}
