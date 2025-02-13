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

// StressBuilder 构建不同场景下压测所需请求参数的接口，并返回请求对象
// 非压测请求实现Builder接口以满足参数构建需要
// 目前支持4种实现方式：Invoke、Query、Create和Upgrade每个类型都定义了自己的Build方法，用于生成不同类型的交易请求：
// - Invoke：构建调用智能合约的交易请求。
// - Query：构建查询智能合约的交易请求。
// - Create：构建部署（安装）智能合约的交易请求。
// - Upgrade：构建升级智能合约的交易请求。
type StressBuilder interface {
	Build(requestId int64, index int) (*commonPb.TxRequest, error)
}

type Query struct{}

// Build 实现StressBuilder接口，用于构建查询智能合约的交易请求
func (i Query) Build(requestId int64, index int) (*commonPb.TxRequest, error) {
	// 构建交易Payload，包含一组键值对（kvs）
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
	// 使用构造的键值对创建查询Payload
	payload, err := constructQueryPayload(chainId, contractName, method, pairs, gasLimit)
	if err != nil {
		return nil, err
	}
	// 构建完整的交易请求
	return buildRequestParam(privateKeys[index], orgIDs[index], signCrtPaths[index], payload, nil)
}

type Invoke struct{}

// Build 实现StressBuilder接口，用于构建调用智能合约的交易请求
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
	// 如果有ABI路径，则读取ABI数据，并根据ABI调整方法名和参数对
	var abiData *[]byte
	if abiPath != "" {
		abiData = abiCache.Read(abiPath)
		runTime = 5 //evm
	}
	method1, pairs1, err := makePairs(method, abiPath, pairs, commonPb.RuntimeType(runTime), abiData)
	if err != nil {
		return nil, err
	}
	// 构造调用智能合约的Payload
	payload, err := constructInvokePayload(chainId, contractName, method1, pairs1, gasLimit)
	if err != nil {
		return nil, err
	}
	// 构建完整的交易请求
	return buildRequestParam(privateKeys[index], orgIDs[index], signCrtPaths[index], payload, nil)
}

type Create struct{}

// Build 实现StressBuilder接口，用于构建部署（安装）智能合约的交易请求
func (c Create) Build(requestId int64, index int) (*commonPb.TxRequest, error) {
	// 读取WASM字节码文件
	wasmBin, err := os.ReadFile(wasmPath)
	if err != nil {
		return nil, err
	}
	var pairs []*commonPb.KeyValuePair
	// 使用模板字符串、版本信息、运行时类型等生成Payload
	payload, _ := utils.GenerateInstallContractPayload(fmt.Sprintf(templateStr, contractName, index,
		requestId, time.Now().Unix()), "1.0.0", commonPb.RuntimeType(runTime), wasmBin, pairs)
	// 如果设置了gas限制，则添加到Payload
	if gasLimit > 0 {
		var limit = &commonPb.Limit{GasLimit: gasLimit}
		payload.Limit = limit
	}
	endorsement, err := acSign(payload)
	if err != nil {
		return nil, err
	}
	// 签名并构建交易请求
	return buildRequestParam(privateKeys[index], orgIDs[index], signCrtPaths[index], payload, endorsement)
}

type Upgrade struct{}

// Build 实现StressBuilder接口，用于构建升级智能合约的交易请求
func (u Upgrade) Build(requestId int64, index int) (*commonPb.TxRequest, error) {
	// 生成唯一的交易ID，读取WASM字节码
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

// Builder 接口定义了构建非交易请求的方法规范，用于根据索引创建具体的交易请求。
// 非压测的请求需要实现Builder接口构建请求参数，参数实现逻辑，在实现方法自定义实现
// 任何实现此接口的类型都需要提供一个Build方法，该方法接收一个int类型的索引作为参数，
// 并返回一个指向commonPb.TxRequest结构体的指针以及一个潜在的错误。
type Builder interface {
	Build(index int) (*commonPb.TxRequest, error)
}

// SubscribeBlock 结构体用于订阅特定高度起始的区块信息。
type SubscribeBlock struct {
	blockHeight uint64 // 订阅开始的区块高度
}

// Build 实现了Builder接口中的Build方法，为指定索引创建一个订阅区块的交易请求。
// 它构造TxRequest，设置Payload中的各种参数以执行订阅操作，并处理相关的二进制编码和错误处理。
func (s SubscribeBlock) Build(index int) (*commonPb.TxRequest, error) {
	req := &commonPb.TxRequest{}
	startBlockHeightByte := make([]byte, 8)
	binary.LittleEndian.PutUint64(startBlockHeightByte, s.blockHeight)
	endBlockHeightBuf := new(bytes.Buffer)
	err := binary.Write(endBlockHeightBuf, binary.LittleEndian, int64(-1))
	if err != nil {
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
				Value: []byte(strconv.FormatBool(false)),
			},
		},
	}
	return buildRequestParam(privateKeys[index], orgIDs[index], signCrtPaths[index], req.Payload, nil)
}

// QueryBlockHeight 结构体用于查询区块链高度。
type QueryBlockHeight struct{}

// Build 实现了Builder接口中的Build方法，为指定索引创建一个查询区块高度的交易请求。
// 它构造TxRequest，设置Payload以查询特定高度的区块信息，并进行相应的参数组装。
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

// makeKvs 生成一组键值对（KeyValuePairs）基于全局配置的规则，用于构造交易请求的Payload。
// 每个键值对的生成会考虑是否需要唯一性、随机性、递增或递减等特性。
// 请求ID（requestId）会被嵌入到满足条件的键值对的值中，以确保唯一性或生成随机数据。
// 全局变量globalPairs定义了一系列键值对的基础配置，包括是否具有唯一性（Unique）、
// 随机生成的比率（RandomRate）、以及整数值的递增或递减（Increase/Decrease）。
// 函数内部通过原子操作来安全地更新全局统计变量，如总发送交易数（totalSentTxs）和
// 随机发送的交易数（totalRandomSentTxs）。
func makeKvs(requestId int64) []*commonPb.KeyValuePair {
	var outKvs []*commonPb.KeyValuePair
	// 原子增加总发送交易计数器
	atomic.AddInt64(&totalSentTxs, 1)
	// 遍历全局键值对配置列表
	for _, p := range globalPairs {
		var val []byte
		// 根据配置特性生成键值对的值
		switch {
		case p.Unique:
			// 如果要求唯一性，格式化字符串并加入requestId和当前时间戳
			val = []byte(fmt.Sprintf(templateStr, p.Value, requestId, time.Now().UnixNano()))
		case 0 < p.RandomRate && p.RandomRate < 100:
			// 按照随机率判断是否生成随机数据，如果满足则同样加入requestId和时间戳
			if isRandom(p.RandomRate) {
				val = []byte(fmt.Sprintf(templateStr, p.Value, requestId, time.Now().UnixNano()))
				// 原子增加随机发送交易计数器
				atomic.AddInt64(&totalRandomSentTxs, 1)
			} else {
				val = []byte(p.Value)
			}
		case p.Decrease:
			// 如果配置了递减，锁定并递减整数值，然后使用新值
			p.mu.Lock()
			val = []byte(fmt.Sprintf("%d", p.IntValue))
			p.IntValue--
			p.mu.Unlock()
			atomic.AddInt64(&totalRandomSentTxs, 1)
		case p.Increase:
			// 如果配置了递增，同上，但递增整数值
			p.mu.Lock()
			val = []byte(fmt.Sprintf("%d", p.IntValue))
			p.IntValue++
			p.mu.Unlock()
			atomic.AddInt64(&totalRandomSentTxs, 1)
		default:
			val = []byte(p.Value)
		}
		// 将键值对添加到输出列表
		outKvs = append(outKvs, &commonPb.KeyValuePair{
			Key:   p.Key,
			Value: val,
		})
	}
	return outKvs
}

// buildRequestParam 是一个辅助函数，用于构建TxRequest的通用部分，如签名、发送者信息等。
// 它接收私钥、组织ID、证书路径、Payload以及背书者列表作为参数，并返回一个完整的TxRequest实例。
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

func getChainClient() ([]*sdk.ChainClient, error) {
	var chainClients []*sdk.ChainClient
	for i := range hosts {
		clientConf := clientConf{
			host:    hosts[i],
			tls:     hostnames[i],
			caPath:  caPaths[i],
			orgId:   orgIDs[i],
			chainId: chainId,
			userKey: userKeyPaths[i],
			userCrt: userCrtPaths[i],
			signKey: signKeyPaths[i],
			signCrt: signCrtPaths[i],
		}
		chainClient, err := newChainClient(clientConf)
		if err != nil {
			return nil, err
		}
		chainClients = append(chainClients, chainClient)
	}
	return chainClients, nil
}

type clientConf struct {
	host    string
	tls     string
	caPath  string
	orgId   string
	chainId string
	userKey string
	userCrt string
	signKey string
	signCrt string
}

func newChainClient(cf clientConf) (*sdk.ChainClient, error) {
	nodeConf := sdk.NewNodeConfig(
		// 节点地址，格式：127.0.0.1:12301
		sdk.WithNodeAddr(cf.host),
		// 节点连接数
		sdk.WithNodeConnCnt(threadNum),
		// 节点是否启用TLS认证
		sdk.WithNodeUseTLS(true),
		// 根证书路径，支持多个
		sdk.WithNodeCAPaths([]string{cf.caPath}),
		// TLS Hostname
		sdk.WithNodeTLSHostName(cf.tls),
	)
	return sdk.NewChainClient(
		// 设置归属组织
		sdk.WithChainClientOrgId(cf.orgId),
		sdk.WithChainClientChainId(chainId),
		// 设置客户端用户私钥路径
		sdk.WithUserKeyFilePath(cf.userKey),
		// 设置客户端用户证书
		sdk.WithUserCrtFilePath(cf.userCrt),
		// 设置客户端签名证书
		sdk.WithUserSignCrtFilePath(cf.signCrt),
		// 设置客户端签名私钥
		sdk.WithUserSignKeyFilePath(cf.signKey),
		// 添加节点1
		sdk.AddChainClientNodeConfig(nodeConf),
	)
}
