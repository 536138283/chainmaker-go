package parallel

import (
	"bytes"
	"chainmaker.org/chainmaker/common/v2/crypto"
	acPb "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"
	sdk "chainmaker.org/chainmaker/sdk-go/v2"
	sdkutils "chainmaker.org/chainmaker/sdk-go/v2/utils"
	"chainmaker.org/chainmaker/utils/v2"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"strconv"
	"sync/atomic"
	"time"
)

type StressBuilder interface {
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

type Builder interface {
	Build(index int) (*commonPb.TxRequest, error)
}

type SubscribeBlock struct {
	blockHeight uint64
}

func (s SubscribeBlock) Build(index int) (*commonPb.TxRequest, error) {
	req := &commonPb.TxRequest{}
	startBlockHeightByte := make([]byte, 8)
	binary.LittleEndian.PutUint64(startBlockHeightByte, s.blockHeight)
	endBlockHeightBuf := new(bytes.Buffer)
	err := binary.Write(endBlockHeightBuf, binary.LittleEndian, int64(-1))
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
				Value: endBlockHeightBuf.Bytes(),
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

type QueryBlockHeight struct{}

func (s QueryBlockHeight) Build(index int) (*commonPb.TxRequest, error) {
	var kvs []*commonPb.KeyValuePair
	kvs = append(kvs, &commonPb.KeyValuePair{
		Key:   sdkutils.KeyBlockContractBlockHeight,
		Value: []byte(strconv.FormatUint(math.MaxUint, 10)),
	})
	kvs = append(kvs, &commonPb.KeyValuePair{
		Key:   sdkutils.KeyBlockContractWithRWSet,
		Value: []byte(strconv.FormatBool(false)),
	})
	kvs = append(kvs, &commonPb.KeyValuePair{
		Key:   sdkutils.KeyBlockContractTruncateValueLen,
		Value: []byte(strconv.FormatInt(1000, 10)),
	})
	payload := sdkutils.NewPayload(
		sdkutils.WithChainId(chainId),
		sdkutils.WithTxType(commonPb.TxType_QUERY_CONTRACT),
		sdkutils.WithTxId(""),
		sdkutils.WithTimestamp(time.Now().Unix()),
		sdkutils.WithContractName(syscontract.SystemContract_CHAIN_QUERY.String()),
		sdkutils.WithMethod(syscontract.ChainQueryFunction_GET_BLOCK_BY_HEIGHT.String()),
		sdkutils.WithParameters(kvs),
		sdkutils.WithSequence(0),
		sdkutils.WithLimit(nil),
	)
	return buildRequestParam(privateKeys[index], orgIDs[index], signCrtPaths[index], payload, nil)
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
