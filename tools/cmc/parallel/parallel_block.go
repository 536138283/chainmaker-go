package parallel

import (
	"bytes"
	"chainmaker.org/chainmaker/common/v2/crypto"
	"chainmaker.org/chainmaker/common/v2/crypto/asym"
	acPb "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	apiPb "chainmaker.org/chainmaker/pb-go/v2/api"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"
	sdk "chainmaker.org/chainmaker/sdk-go/v2"
	sdkutils "chainmaker.org/chainmaker/sdk-go/v2/utils"
	"chainmaker.org/chainmaker/utils/v2"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gogo/protobuf/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"sync/atomic"
	"time"
)

type RequestParam struct {
	Param     *commonPb.TxRequest
	RequestId int64
}

// 请求参数队列，存放请求参数，队列的数量为节点的数量
// 队列的index为队列的下标
var paramQueues []chan RequestParam

var privateKeys []crypto.PrivateKey

// 每一个请求参数的唯一id
var requestId int64

// 生产信号，当该chan接收到数据时，开始生产请求参数
var produceSignal chan int

// 生产因子，用来控制生产消息的数量
var productFactor int

func initParallel() {
	initProductFactor(100)
	produceSignal = make(chan int, 1)
	paramQueues = make([]chan RequestParam, nodeNum)
	for i := 0; i < nodeNum; i++ {
		paramQueues[i] = make(chan RequestParam, productFactor)
		// 解析签名私钥
		file, err := os.ReadFile(signKeyPaths[i])
		if err != nil {
			fmt.Printf("read node[%s] sign key err: %s\n", hosts[i], err.Error())
			return
		}
		signKey, err := asym.PrivateKeyFromPEM(file, nil)
		if err != nil {
			fmt.Printf("analysis node[%s] sign key err: %s\n", hosts[i], err.Error())
			return
		}
		privateKeys = append(privateKeys, signKey)
	}
}

// 对半法加载成产因子
func initProductFactor(factor int) {
	if threadNum*loopNum/factor > threadNum {
		productFactor = threadNum * loopNum / nodeNum / factor
		return
	} else {
		productFactor = productFactor / 2
	}
}

// 构建请求参数
// param：
// @method：方法接收器，接受不同的cmd命令构建不同的方法调用所需要的请求参数
// 该方法用于批量生成请求参数，生成个数由生产因子控制，当index为-1的时候像每个chan成产数据，否则像指定chan生产数据
func producer(method string) {
	fmt.Printf("producer method: %s\n", method)
	var builder Builder
	switch method {
	case invokerMethod:
		builder = Invoke{}
	case queryMethod:
		builder = Query{}
	case createContractStr:
		builder = Create{}
	case upgradeContractStr:
		builder = Upgrade{}
	}
	for {
		if requestId >= int64(threadNum*loopNum) {
			close(produceSignal)
			return
		}
		select {
		case index := <-produceSignal:
			if index == -1 {
				for i := 0; i < productFactor; i++ {
					for nodeIndex := 0; nodeIndex < nodeNum; nodeIndex++ {
						atomic.AddInt64(&requestId, 1)
						param, err := builder.Build(requestId, nodeIndex)
						if err != nil {
							fmt.Printf("producer err: %s\n", err.Error())
						}
						paramQueues[nodeIndex] <- RequestParam{
							Param:     param,
							RequestId: requestId,
						}
					}

				}
			} else {
				go func() {
					for i := 0; i < productFactor; i++ {
						atomic.AddInt64(&requestId, 1)
						param, err := builder.Build(requestId, index)
						if err != nil {
							fmt.Printf("producer err: %s\n", err.Error())
						}
						paramQueues[index] <- RequestParam{
							Param:     param,
							RequestId: requestId,
						}
					}
				}()
			}

		}
	}

}

type Builder interface {
	Build(requestId int64, index int) (*commonPb.TxRequest, error)
}

type Query struct {
}

func (i Query) Build(requestId int64, index int) (*commonPb.TxRequest, error) {
	// 构造Payload
	pairs := makeKvs(111)
	if showKey {
		j, err := json.Marshal(pairs)
		if err != nil {
			fmt.Println(err)
		}
		rate := totalRandomSentTxs * 100 / totalSentTxs
		fmt.Printf("totalSentTxs:%d\t totalRandomSentTxs:%d\t randomRate:%d \t param:%s\t \n",
			totalSentTxs, totalRandomSentTxs, rate, string(j))
	}
	payload, err := constructQueryPayload(chainId, contractName, method, pairs, gasLimit)
	if err != nil {
		return nil, err
	}
	return buildRequestParam(privateKeys[index], orgIDs[index], signCrtPaths[index], payload, nil)
}

type Invoke struct {
}

func (i Invoke) Build(requestId int64, index int) (*commonPb.TxRequest, error) {
	pairs := makeKvs(requestId)
	if showKey {
		j, err := json.Marshal(pairs)
		if err != nil {
			fmt.Println(err)
		}
		rate := totalRandomSentTxs * 100 / totalSentTxs
		fmt.Printf("totalSentTxs:%d\t totalRandomSentTxs:%d\t randomRate:%d \t param:%s\t \n",
			totalSentTxs, totalRandomSentTxs, rate, string(j))
	}
	var abiData *[]byte
	if abiPath != "" {
		abiData = abiCache.Read(abiPath)
		runTime = 5 //evm
	}
	method1, pairs1, err := makePairs(method, abiPath, pairs, commonPb.RuntimeType(runTime), abiData)
	if err != nil {
		return nil, err
	}
	payload, err := constructInvokePayload(chainId, contractName, method1, pairs1, gasLimit)
	if err != nil {
		return nil, err
	}
	fmt.Println(index, "  requestId:", requestId)
	return buildRequestParam(privateKeys[index], orgIDs[index], signCrtPaths[index], payload, nil)
}

type Create struct {
}

func (c Create) Build(requestId int64, index int) (*commonPb.TxRequest, error) {
	wasmBin, err := os.ReadFile(wasmPath)
	if err != nil {
		return nil, err
	}
	var pairs []*commonPb.KeyValuePair
	payload, _ := utils.GenerateInstallContractPayload(fmt.Sprintf(templateStr, contractName, index,
		requestId, time.Now().Unix()), "1.0.0", commonPb.RuntimeType(runTime), wasmBin, pairs)
	// gas limit
	if gasLimit > 0 {
		var limit = &commonPb.Limit{GasLimit: gasLimit}
		payload.Limit = limit
	}
	endorsement, err := acSign(payload)
	if err != nil {
		return nil, err
	}
	return buildRequestParam(privateKeys[index], orgIDs[index], signCrtPaths[index], payload, endorsement)
}

type Upgrade struct {
}

func (u Upgrade) Build(requestId int64, index int) (*commonPb.TxRequest, error) {
	txId := utils.GetTimestampTxId()
	wasmBin, err := ioutil.ReadFile(wasmPath)
	if err != nil {
		return nil, err
	}
	var pairs []*commonPb.KeyValuePair
	payload, err := utils.GenerateInstallContractPayload(
		fmt.Sprintf(templateStr, contractName, index, requestId, time.Now().Unix()),
		version, commonPb.RuntimeType(runTime), wasmBin, pairs)
	if err != nil {
		return nil, err
	}
	payload.Method = syscontract.ContractManageFunction_UPGRADE_CONTRACT.String()
	// gas limit
	if gasLimit > 0 {
		var limit = &commonPb.Limit{GasLimit: gasLimit}
		payload.Limit = limit
	}
	payload.TxId = txId
	payload.ChainId = chainId
	payload.Timestamp = time.Now().Unix()
	endorsement, err := acSign(payload)
	if err != nil {
		return nil, err
	}
	return buildRequestParam(privateKeys[index], orgIDs[index], signCrtPaths[index], payload, endorsement)
}

func makeKvs(requestId int64) []*commonPb.KeyValuePair {
	var outKvs []*commonPb.KeyValuePair
	atomic.AddInt64(&totalSentTxs, 1)
	for _, p := range globalPairs {
		var val []byte
		switch {
		case p.Unique:
			val = []byte(fmt.Sprintf(templateStr, p.Value, requestId, time.Now().UnixNano()))
		case 0 < p.RandomRate && p.RandomRate < 100:
			if isRandom(p.RandomRate) {
				val = []byte(fmt.Sprintf(templateStr, p.Value, requestId, time.Now().UnixNano()))
				atomic.AddInt64(&totalRandomSentTxs, 1)
			} else {
				val = []byte(p.Value)
			}
		case p.Decrease:
			p.mu.Lock()
			val = []byte(fmt.Sprintf("%d", p.IntValue))
			p.IntValue--
			p.mu.Unlock()
			atomic.AddInt64(&totalRandomSentTxs, 1)
		case p.Increase:
			p.mu.Lock()
			val = []byte(fmt.Sprintf("%d", p.IntValue))
			p.IntValue++
			p.mu.Unlock()
			atomic.AddInt64(&totalRandomSentTxs, 1)
		default:
			val = []byte(p.Value)
		}

		outKvs = append(outKvs, &commonPb.KeyValuePair{
			Key:   p.Key,
			Value: val,
		})
	}
	return outKvs
}

type SubscribeBlock struct {
}

func (s SubscribeBlock) Build(index int) (*commonPb.TxRequest, error) {
	req := &commonPb.TxRequest{}
	startBlockHeightByte := make([]byte, 8)
	binary.LittleEndian.PutUint64(make([]byte, 8), 0)
	//endBlockHeightByte := make([]byte, 8)
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, int64(-1))
	if err != nil {
		fmt.Println("Error:", err)
		return nil, err
	}
	req.Payload = &commonPb.Payload{
		ChainId:      chainId,
		TxId:         utils.GetRandTxId(),
		TxType:       commonPb.TxType_SUBSCRIBE,
		Timestamp:    time.Now().Unix(),
		ContractName: syscontract.SystemContract_SUBSCRIBE_MANAGE.String(),
		Method:       syscontract.SubscribeFunction_SUBSCRIBE_BLOCK.String(),
		Parameters: []*commonPb.KeyValuePair{
			{
				Key:   syscontract.SubscribeBlock_START_BLOCK.String(),
				Value: startBlockHeightByte,
			},
			{
				Key:   syscontract.SubscribeBlock_END_BLOCK.String(),
				Value: buf.Bytes(),
			},
			{
				Key:   syscontract.SubscribeBlock_WITH_RWSET.String(),
				Value: []byte(strconv.FormatBool(false)),
			},
			{
				Key:   syscontract.SubscribeBlock_ONLY_HEADER.String(),
				Value: []byte(strconv.FormatBool(true)),
			},
		},
	}
	return buildRequestParam(privateKeys[index], orgIDs[index], signCrtPaths[index], req.Payload, nil)
}

func buildRequestParam(sk3 crypto.PrivateKey, orgId, userCrtPath string,
	payload *commonPb.Payload, endorsers []*commonPb.EndorsementEntry) (*commonPb.TxRequest, error) {
	req := &commonPb.TxRequest{}
	sender := &acPb.Member{}
	switch {
	case authType == sdk.Public || authType == sdk.PermissionedWithKey:
		pubKey := sk3.PublicKey()
		memberInfo, err := pubKey.String()
		if err != nil {
			return nil, err
		}
		if authType == sdk.PermissionedWithKey {
			sender.OrgId = orgId
		}
		sender.MemberInfo = []byte(memberInfo)
		sender.MemberType = acPb.MemberType_PUBLIC_KEY
	default:
		sender.OrgId = orgId
		file := fileCache.Read(userCrtPath)
		if useShortCrt {
			certId, err := certCache.Read(userCrtPath, *file, hashAlgo)
			if err != nil {
				return nil, fmt.Errorf("fail to compute the identity for certificate [%v]", err)
			}

			sender.MemberInfo = *certId
			sender.MemberType = acPb.MemberType_CERT_HASH
		} else {
			sender.MemberInfo = *file
		}
	}
	hashType, err := getHashType(hashAlgo)
	if err != nil {
		return nil, err
	}
	req.Payload = payload
	// 拼接后，计算Hash，对hash计算签名
	rawTxBytes, err := utils.CalcUnsignedTxRequestBytes(req)
	if err != nil {
		return nil, err
	}
	signBytes, err := sdkutils.SignPayloadBytesWithHashType(sk3, hashType, rawTxBytes)
	if err != nil {
		return nil, err
	}
	if len(endorsers) > 0 {
		req.Endorsers = endorsers
	}
	req.Sender = &commonPb.EndorsementEntry{Signer: sender, Signature: signBytes}
	return req, nil
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

// 订阅区块 subscribeBlock
// 订阅交易 subscribeTx
// getCurrentBlockHeight
func sendSubscribe(ctx context.Context, client apiPb.RpcNodeClient, req *commonPb.TxRequest) (<-chan interface{}, error) {
	resp, err := client.Subscribe(ctx, req)
	if err != nil {
		return nil, err
	}
	fmt.Println("subscribe start")
	c := make(chan interface{})

	go func() {
		defer close(c)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				var result *commonPb.SubscribeResult
				result, err = resp.Recv()
				if err == io.EOF {
					return
				}

				if err != nil {
					return
				}

				var ret interface{}
				blockInfo := &commonPb.BlockInfo{}
				if err = proto.Unmarshal(result.Data, blockInfo); err == nil {
					ret = blockInfo
					break
				}
				blockHeader := &commonPb.BlockHeader{}
				if err = proto.Unmarshal(result.Data, blockHeader); err == nil {
					ret = blockHeader
					break
				}
				fmt.Println("block：", blockInfo)
				close(c)
				c <- ret
			}
		}
	}()
	return c, err
}

func subNodes(statistician *Statistician) {
	threads, err := threadFactory(nodeNum, invokerMethod, nil, nil, statistician)
	if err != nil {
		fmt.Println("subNodes threadFactory err:", err)
		return
	}
	params := make([]*commonPb.TxRequest, nodeNum)
	for i := 0; i < nodeNum; i++ {
		s := SubscribeBlock{}
		params[i], err = s.Build(i)
		if err != nil {
			fmt.Println("error building subscribe params:", err)
			return
		}
	}
	for i := 0; i < nodeNum; i++ {
		go func(index int) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			c, err := sendSubscribe(context.Background(), threads[index].client, params[index])
			if err != nil {
				fmt.Println("error sendSubscribe :", err)
				return
			}
			for {
				select {
				case block, ok := <-c:
					if !ok {
						fmt.Println(errors.New("chan is close"))
						return
					}

					if block == nil {
						fmt.Println(errors.New("received block is nil"))
						return
					}

					blockHeader, ok := block.(*commonPb.BlockHeader)
					if !ok {
						fmt.Println(errors.New("received data is not *common.BlockHeader"))
						return
					}
					statistician.cReqStatC <- &cReqStat{
						blockHeader,
					}
					fmt.Printf("recv blockHeader [%d] => %+v\n", blockHeader.BlockHeight, blockHeader)
				case <-ctx.Done():
					return
				}
			}
		}(i)
	}
}
