package parallel

import (
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	sdk "chainmaker.org/chainmaker/sdk-go/v2"
	"context"
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"math"
	"time"
)

// subNodes 函数负责初始化并管理一组并发任务，这些任务针对每个区块链节点订阅新区块事件。
// 参数:
// statistician (*Statistician): 一个Statistician实例，用于在整个订阅过程中收集统计数据和管理线程。
func subNodes(statistician *Statistician) {
	// 并发启动订阅任务
	for i := 0; i < nodeNum; i++ {
		go func(index int) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			blockChan, err := defaultSdkClients[index].SubscribeBlock(context.Background(), math.MaxInt64,
				-1, false, false)
			if err != nil {
				fmt.Println("error sendSubscribe :", err)
				return
			}
			// 接收区块并发送到统计对象
			for {
				select {
				case block := <-blockChan:
					blockInfo, ok := block.(*commonPb.BlockInfo)
					if !ok {
						return
					}
					statistician.cReqStatC <- &cReqStat{
						blockInfo.Block.Header, index, blockInfo.Block.Txs,
					}
				case <-ctx.Done():
					return
				}
			}

		}(i)
	}
}

// sendTx 向特定节点发送交易请求。如果请求超时，则返回错误信息
func sendTx(client *sdk.ChainClient, orgId string, loopId int, req *commonPb.TxRequest) error {
	// 防止在收到响应之前上链的数据不一致情况，这里提前记录交易id
	txLatency.Store(req.Payload.TxId, time.Now().UnixNano()/1e6)
	result, err := client.SendTxRequest(req, requestTimeout, true)
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

// getBlockHeight 查询当前区块链的高度。首先创建一个查询线程，然后构建查询高度的请求并发送，最后解析返回的区块信息以获取高度
func getBlockHeight() (uint64, error) {
	blockInfo, err := defaultSdkClients[0].GetBlockByHeight(math.MaxUint64, false)
	if err != nil {
		return 0, err
	}
	return blockInfo.Block.Header.BlockHeight, nil
}
