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

// subNodes 函数负责初始化并管理一组并发任务，这些任务针对每个区块链节点订阅新区块事件。
// 参数:
// statistician (*Statistician): 一个Statistician实例，用于在整个订阅过程中收集统计数据和管理线程。
func subNodes(statistician *Statistician) {
	threads, err := threadFactory(nodeNum, nil, nil, statistician)
	if err != nil {
		fmt.Println("subNodes threadFactory err:", err)
		return
	}
	params := make([]*commonPb.TxRequest, nodeNum)
	// 获取区块高度
	blockHeight, err := getBlockHeight()
	if err != nil {
		fmt.Println("getBlockHeight err:", err)
		return
	}
	fmt.Println("blockHeight:", blockHeight)
	// 构建针对每个节点的订阅请求参数
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
	// 并发启动订阅任务
	for i := 0; i < nodeNum; i++ {
		go func(index int) {
			if err := subscribeNewBlock(context.Background(), threads[index], params[index]); err != nil {
				fmt.Println("error sendSubscribe :", err)
				return
			}
		}(i)
	}
}

// subscribeNewBlock 向指定节点订阅新区块事件。当新区块到来时，它会接收并处理区块链头信息，然后更新统计信息。
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
				blockInfo := &commonPb.BlockInfo{}
				if err = proto.Unmarshal(result.Data, blockInfo); err != nil {
					return
				}
				thread.statistician.cReqStatC <- &cReqStat{
					blockInfo.Block.Header, thread.index, blockInfo.Block.Txs,
				}
			}

		}
	}()
	return err
}

// sendTx 向特定节点发送交易请求。如果请求超时，则返回错误信息
func sendTx(client apiPb.RpcNodeClient, orgId string, loopId int, req *commonPb.TxRequest) error {
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
	// 记录交易id与成功发起交易的时间戳
	txLatency.Store(result.TxId, time.Now().UnixMilli())
	return nil
}

// getBlockHeight 查询当前区块链的高度。首先创建一个查询线程，然后构建查询高度的请求并发送，最后解析返回的区块信息以获取高度
func getBlockHeight() (uint64, error) {
	threads, err := threadFactory(1, nil, nil, nil)
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
	return blockInfo.Block.Header.BlockHeight, nil
}
