/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

// Package parallel 并发处理，主要用于压测的场景
package parallel

import (
	"chainmaker.org/chainmaker/logger/v2"
	sdk "chainmaker.org/chainmaker/sdk-go/v2"
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"io/ioutil"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

var log = logger.GetLogger(logger.MODULE_CLI)

var (
	threadNum      int
	loopNum        int
	timeout        int
	printTime      int
	sleepTime      int
	climbTime      int
	requestTimeout int64
	runTime        int32
	checkResult    bool
	recordLog      bool
	showKey        bool
	useTLS         bool
	useShortCrt    bool

	hostsString           string
	hostnamesString       string
	statisticalType       string
	checkInterval         int
	signCrtPathsString    string
	signKeyPathsString    string
	userCrtPathsString    string
	userKeyPathsString    string
	userEncKeyPathsString string
	userEncCrtPathsString string
	orgIDsString          string
	hashAlgo              string
	caPathsString         string
	pairsFile             string
	pairsString           string
	globalPairs           []*KeyValuePair
	abiPath               string
	method                string
	orgIds                string // 组织列表(多个用逗号隔开)
	adminSignKeys         string // 管理员私钥列表(多个用逗号隔开)
	adminSignCrts         string // 管理员证书列表(多个用逗号隔开)
	chainId               string
	contractName          string
	version               string
	wasmPath              string

	caPaths       []string
	hosts         []string
	hostnames     []string
	signCrtPaths  []string
	signKeyPaths  []string
	encKeyPaths   []string
	encCrtPaths   []string
	encCrtBytes   [][]byte
	encKeyBytes   [][]byte
	userCrtPaths  []string
	userKeyPaths  []string
	adminKeyPaths []string
	adminCrtPaths []string
	orgIDs        []string

	nodeNum int

	fileCache = NewFileCacheReader()
	certCache = NewCertFileCacheReader()

	abiCache     = NewFileCacheReader()
	outputResult bool

	authTypeUint32 uint32
	authType       sdk.AuthType
	gasLimit       uint64

	startBlock int64 // 订阅的起始区块高度
	endBlock   int64 // 订阅的结束区块高度
)

type ValueParam struct {
	Initial      int64 `json:"initial"`
	Increase     bool  `json:"increase"`
	EndValue     int64 `json:"endValue"`
	TempIntValue int64 `json:"-"`
	LoopType     int8  `json:"loopType"`
}

type KeyValuePair struct {
	Key        string `json:"key,omitempty"`
	Value      string `json:"value,omitempty"`
	Unique     bool   `json:"unique,omitempty"`
	RandomRate int64  `json:"randomRate,omitempty"`
	Increase   bool   `json:"increase"`
	Decrease   bool   `json:"decrease"`
	// mu protect IntValue in Increase/Decrease scene.
	mu          sync.Mutex
	IntValue    int64         `json:"-"`
	ValueFormat string        `json:"valueFormat,omitempty"`
	ValueParams []*ValueParam `json:"valueParams,omitempty"`
	EndCount    int           `json:"-"` // 需要终止的参数数量，如果全部需要终止则停止压测
	ArriveCount int           `json:"-"` // 已经达成目标值的数量,与ArriveArr互相配合，主要记录ArriveArr中为true的值
	ArriveArr   []bool        `json:"-"` // 判断每个valueParam是否达成条件值如果达成则设置为true
	Values      []int64       `json:"-"`
	IntPows     []int64       `json:"-"`
}

// ParallelCMD parallel sub command
// @return *cobra.Command
func ParallelCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "parallel",
		Short: "Parallel",
		Long:  "Parallel",
		PersistentPreRun: func(_ *cobra.Command, _ []string) {
			authType = sdk.AuthType(authTypeUint32)
			caPaths = strings.Split(caPathsString, ",")
			hosts = strings.Split(hostsString, ",")
			hostnames = strings.Split(hostnamesString, ",")
			signCrtPaths = strings.Split(signCrtPathsString, ",")
			signKeyPaths = strings.Split(signKeyPathsString, ",")
			userCrtPaths = strings.Split(userCrtPathsString, ",")
			userKeyPaths = strings.Split(userKeyPathsString, ",")
			adminKeyPaths = strings.Split(adminSignKeys, ",")
			adminCrtPaths = strings.Split(adminSignCrts, ",")
			if userEncCrtPathsString != "" && userEncKeyPathsString != "" {
				encCrtPaths = strings.Split(userEncCrtPathsString, ",")
				encKeyPaths = strings.Split(userEncKeyPathsString, ",")
			}
			orgIDs = strings.Split(orgIDsString, ",")

			if authType == sdk.Public {
				if len(hosts) != len(signKeyPaths) {
					panic(fmt.Sprintf("hosts[%d], sign-keys[%d] length invalid", len(hosts), len(signKeyPaths)))
				}
			} else if authType == sdk.PermissionedWithKey {
				if len(hosts) != len(signKeyPaths) || len(hosts) != len(orgIDs) {
					panic(fmt.Sprintf("hosts[%d], sign-keys[%d], orgIDs[%d] length invalid",
						len(hosts), len(signKeyPaths), len(orgIDs)))
				}
			} else {
				if len(hosts) != len(signCrtPaths) || len(hosts) != len(signKeyPaths) || len(hosts) != len(caPaths) || len(hosts) != len(orgIDs) {
					panic(fmt.Sprintf("hosts[%d], sign-crts[%d], sign-keys[%d], ca-path[%d], orgIDs[%d] length invalid",
						len(hosts), len(signCrtPaths), len(signKeyPaths), len(caPaths), len(orgIDs)))
				}
			}

			if useTLS && (len(hosts) != len(userCrtPaths) || len(hosts) != len(userKeyPaths)) {
				panic(fmt.Sprintf("use tls, but hosts[%d], user-crts[%d], user-keys[%d] length invalid",
					len(hosts), len(userCrtPaths), len(userKeyPaths)))
			}
			if len(encCrtPaths) != len(encKeyPaths) && len(encKeyPaths) != len(hosts) && len(encCrtPaths) > 0 {
				panic(fmt.Sprintf("use env but encCrtPaths[%d], encKeyPaths[%d], hosts[%d]", len(encCrtPaths),
					len(encKeyPaths), len(hosts)))
			}

			if len(encCrtPaths) > 0 && len(encKeyPaths) > 0 && len(hosts) > 0 {
				for i := range encCrtPaths {
					keyBytes, err := ioutil.ReadFile(encKeyPaths[i])
					if err != nil {
						panic(err)
					}
					encKeyBytes = append(encKeyBytes, keyBytes)
					crtBytes, err := ioutil.ReadFile(encCrtPaths[i])
					if err != nil {
						panic(err)
					}
					encCrtBytes = append(encCrtBytes, crtBytes)
				}

			}
			nodeNum = len(hosts)
			if len(pairsFile) != 0 {
				bytes, err := ioutil.ReadFile(pairsFile)
				if err != nil {
					panic(err)
				}
				pairsString = string(bytes)
			}
			var err error
			globalPairs, err = getPairInfos()
			if err != nil {
				panic(err)
			}
		},
	}

	flags := cmd.PersistentFlags()
	flags.IntVarP(&threadNum, "threadNum", "N", 10, "specify thread number")
	flags.IntVarP(&loopNum, "loopNum", "l", 1000, "specify loop number")
	flags.IntVarP(&timeout, "timeout", "T", 2, "specify timeout(unit: s)")
	flags.IntVarP(&printTime, "printTime", "r", 1, "specify print time(unit: s)")
	flags.IntVarP(&sleepTime, "sleepTime", "S", 100, "specify sleep time(unit: ms)")
	flags.IntVarP(&climbTime, "climbTime", "L", 10, "specify climb time(unit: s)")
	flags.StringVarP(&hostsString, "hosts", "H", "localhost:17988,localhost:27988", "specify hosts")
	flags.StringVarP(&signCrtPathsString, "sign-crts", "K", "../../config/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.crt,../../config/crypto-config/wx-org2.chainmaker.org/user/client1/client1.sign.crt", "specify user crt path")
	flags.StringVarP(&signKeyPathsString, "sign-keys", "u", "../../config/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.key,../../config/crypto-config/wx-org2.chainmaker.org/user/client1/client1.sign.key", "specify user key path")
	flags.StringVar(&userCrtPathsString, "user-crts", "../../config/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.crt,../../config/crypto-config/wx-org2.chainmaker.org/user/client1/client1.tls.crt", "specify tls crt path")
	flags.StringVar(&userKeyPathsString, "user-keys", "../../config/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.key,../../config/crypto-config/wx-org2.chainmaker.org/user/client1/client1.tls.key", "specify tls key path")
	flags.StringVar(&userEncKeyPathsString, "user-enc-keys", "", "enc key path")
	flags.StringVar(&userEncCrtPathsString, "user-enc-crts", "", "enc certificate path")
	flags.StringVarP(&orgIDsString, "org-IDs", "I", "wx-org1,wx-org2", "specify user key path")
	flags.BoolVarP(&checkResult, "check-result", "Y", false, "specify whether check result")
	flags.BoolVarP(&recordLog, "record-log", "g", false, "specify whether record log")
	flags.BoolVarP(&outputResult, "output-result", "", false, "output rpc result, eg: txid")
	flags.BoolVarP(&showKey, "showKey", "", false, "bool")
	flags.StringVar(&hashAlgo, "hash-algorithm", "SHA256", "hash algorithm set in chain configuration")
	flags.StringVarP(&caPathsString, "ca-path", "P", "../../config/crypto-config/wx-org1.chainmaker.org/ca,../../config/crypto-config/wx-org2.chainmaker.org/ca", "specify ca path")
	flags.BoolVarP(&useTLS, "use-tls", "t", false, "specify whether use tls")
	flags.StringVar(&orgIds, "org-ids", "wx-org1,wx-org2,wx-org3,wx-org4", "orgIds of admin")
	flags.StringVar(&adminSignKeys, "admin-sign-keys", "../../config/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.sign.key,../../config/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.sign.key,../../config/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.sign.key,../../config/crypto-config/wx-org4.chainmaker.org/user/admin1/admin1.sign.key", "adminSignKeys of admin")
	flags.StringVar(&adminSignCrts, "admin-sign-crts", "../../config/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.sign.crt,../../config/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.sign.crt,../../config/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.sign.crt,../../config/crypto-config/wx-org4.chainmaker.org/user/admin1/admin1.sign.crt", "adminSignCrts of admin")
	flags.StringVarP(&chainId, "chain-id", "C", "chain1", "specify chain id")
	flags.StringVarP(&contractName, "contract-name", "n", "contract1", "specify contract name")
	flags.BoolVar(&useShortCrt, "use-short-crt", false, "use compressed certificate in transactions")
	flags.Int64Var(&requestTimeout, "requestTimeout", 5, "specify request timeout(unit: s)")
	flags.Uint32Var(&authTypeUint32, "auth-type", 1, "chainmaker auth type. PermissionedWithCert:1,PermissionedWithKey:2,Public:3")
	flags.Uint64Var(&gasLimit, "gas-limit", 0, "gas limit in uint64 type")
	flags.StringVarP(&hostnamesString, "tls-host-names", "", "", "specify hostname, the sequence is the same as --hosts")
	flags.StringVarP(&statisticalType, "statistical-type", "", "default", "normal statistical type or block based statistical type, input normal or block default:normal ")
	flags.IntVarP(&checkInterval, "check-interval", "", 1, "After all threads are done,check the interval time of the last block generation. ")
	cmd.AddCommand(invokeCMD())
	cmd.AddCommand(queryCMD())
	cmd.AddCommand(createContractCMD())
	cmd.AddCommand(upgradeContractCMD())
	return cmd
}

const (
	LoopTypeEnd     = int8(1)
	LoopTypeRestart = int8(2)
)

func getPairInfos() ([]*KeyValuePair, error) {
	var ps []*KeyValuePair
	err := json.Unmarshal([]byte(pairsString), &ps)
	if err != nil {
		log.Errorf("unmarshal pair content failed, origin content: %s, err: %s", pairsString, err)
		return nil, err
	}
	for _, p := range ps {
		if p.Decrease || p.Increase {
			p.IntValue, err = strconv.ParseInt(p.Value, 10, 64)
			if err != nil {
				return nil, err
			}
		}
		if p.ValueFormat != "" {
			re := regexp.MustCompile(`%0(\d+)d`)
			matches := re.FindAllStringSubmatch(p.ValueFormat, -1)
			if len(matches) != len(p.ValueParams) {
				return nil, fmt.Errorf("not enough (or more) values to fill the value format template")
			}
			p.Values = make([]int64, len(p.ValueParams))
			p.IntPows = make([]int64, len(p.ValueParams))
			for i, match := range matches {
				if p.ValueParams[i].LoopType == LoopTypeEnd {
					p.EndCount++
				}
				p.ArriveArr = make([]bool, len(p.ValueParams))
				p.Values[i] = p.ValueParams[i].Initial
				p.ValueParams[i].TempIntValue = p.Values[i]
				width, _ := strconv.Atoi(match[1])
				p.IntPows[i] = intPow(10, int64(width))
			}
		}
	}

	return ps, nil
}

func intPow(base, exp int64) int64 {
	result := int64(1)
	for i := int64(0); i < exp; i++ {
		result *= base
	}
	return result
}
