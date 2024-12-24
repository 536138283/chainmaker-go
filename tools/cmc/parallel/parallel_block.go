package parallel

import (
	"bytes"
	"chainmaker.org/chainmaker/common/v2/crypto"
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
	"fmt"
	"github.com/gogo/protobuf/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
	"io/ioutil"
	"strconv"
	"time"
)

func parallelBlock(parallelMethod string) error {
	timeoutChan := make(chan struct{}, threadNum)
	doneChan := make(chan struct{}, threadNum)

	// Statistician updater
	statistician := &Statistician{
		reqStatC: make(chan *reqStat, threadNum),

		nodeMinSuccessElapsed: make([]int64, nodeNum),
		nodeMaxSuccessElapsed: make([]int64, nodeNum),
		nodeSumSuccessElapsed: make([]int64, nodeNum),
		nodeSuccessReqCount:   make([]int, nodeNum),
		nodeTotalReqCount:     make([]int, nodeNum),
	}
	var threads []*Thread
	for i := 0; i < threadNum; i++ {
		thread, err := threadFactory(i, parallelMethod, doneChan, timeoutChan, statistician)
		if err != nil {
			return err
		}
		threads = append(threads, thread)
	}
	for _, thread := range threads {
		if err := thread.BindRequestParams(parallelMethod); err != nil {
			return err
		}
	}
	statistician.startTime = time.Now()
	statistician.lastStartTime = time.Now()

	go statistician.Start()
	go parallelStart(threads)
	printTicker := time.NewTicker(time.Duration(printTime) * time.Second)
	go printResult(printTicker, statistician)
	listenAndExitParallel(timeoutChan, doneChan, printTicker)
	// last once statistics
	fmt.Println("Statistics for the entire test")
	statistician.endTime = time.Now()
	statistician.PrintDetails(true)
	// close client conn
	for _, t := range threads {
		t.Stop()
	}
	return nil
}

type ParamBuild interface {
	Build() (*commonPb.TxRequest, error)
}

type Query struct {
	threadId    int
	loopId      int
	sk3         crypto.PrivateKey
	orgId       string
	userCrtPath string
}

func (q Query) Build() (*commonPb.TxRequest, error) {
	// 构造Payload
	pairs := makeKvs(q.threadId, q.loopId)
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
	return buildRequestParam(q.sk3, q.orgId, q.userCrtPath, payload, nil)
}

type Invoke struct {
	threadId    int
	loopId      int
	sk3         crypto.PrivateKey
	orgId       string
	userCrtPath string
}

func (i Invoke) Build() (*commonPb.TxRequest, error) {
	// 构造Payload
	pairs := makeKvs(i.threadId, i.loopId)
	if showKey {
		j, err := json.Marshal(pairs)
		if err != nil {
			fmt.Println(err)
		}
		rate := totalRandomSentTxs * 100 / totalSentTxs
		fmt.Printf("totalSentTxs:%d\t totalRandomSentTxs:%d\t randomRate:%d \t param:%s\t \n",
			totalSentTxs, totalRandomSentTxs, rate, string(j))
	}

	// 支持evm
	//var err error
	var abiData *[]byte
	if abiPath != "" {
		abiData = abiCache.Read(abiPath)
		runTime = 5 //evm
	}

	method1, pairs1, err := makePairs(method, abiPath, pairs, commonPb.RuntimeType(runTime), abiData)

	payload, err := constructInvokePayload(chainId, contractName, method1, pairs1, gasLimit)
	if err != nil {
		return nil, err
	}
	return buildRequestParam(i.sk3, i.orgId, i.userCrtPath, payload, nil)
}

type Create struct {
	threadId    int
	loopId      int
	sk3         crypto.PrivateKey
	orgId       string
	userCrtPath string
}

func (c Create) Build() (*commonPb.TxRequest, error) {
	wasmBin, err := ioutil.ReadFile(wasmPath)
	if err != nil {
		return nil, err
	}
	var pairs []*commonPb.KeyValuePair
	payload, _ := utils.GenerateInstallContractPayload(fmt.Sprintf(templateStr, contractName, c.threadId,
		c.loopId, time.Now().Unix()), "1.0.0", commonPb.RuntimeType(runTime), wasmBin, pairs)
	// gas limit
	if gasLimit > 0 {
		var limit = &commonPb.Limit{GasLimit: gasLimit}
		payload.Limit = limit
	}
	endorsement, err := acSign(payload)
	if err != nil {
		return nil, err
	}
	// payload
	return buildRequestParam(c.sk3, c.orgId, c.userCrtPath, payload, endorsement)
}

type Upgrade struct {
	threadId    int
	loopId      int
	sk3         crypto.PrivateKey
	orgId       string
	userCrtPath string
}

func (u Upgrade) Build() (*commonPb.TxRequest, error) {
	txId := utils.GetTimestampTxId()
	wasmBin, err := ioutil.ReadFile(wasmPath)
	if err != nil {
		return nil, err
	}
	var pairs []*commonPb.KeyValuePair
	payload, _ := GenerateUpgradeContractPayload(fmt.Sprintf(templateStr, contractName, u.threadId, u.loopId, time.Now().Unix()),
		version, commonPb.RuntimeType(runTime), wasmBin, pairs)
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
	// payload
	return buildRequestParam(u.sk3, u.orgId, u.userCrtPath, payload, endorsement)
}

type SubscribeBlock struct {
	threadId    int
	sk3         crypto.PrivateKey
	userCrtPath string
}

func (s SubscribeBlock) Build() (*commonPb.TxRequest, error) {
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
	return buildRequestParam(s.sk3, "", s.userCrtPath, req.Payload, nil)
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
		file := fileCache.Read(userCrtPath)
		if useShortCrt {
			certId, err := certCache.Read(userCrtPath, *file, hashAlgo)
			if err != nil {
				return nil, fmt.Errorf("fail to compute the identity for certificate [%v]", err)
			}
			sender.OrgId = orgId
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

func sendInvokeRequest(client apiPb.RpcNodeClient, orgId string, loopId int, req *commonPb.TxRequest) error {
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
