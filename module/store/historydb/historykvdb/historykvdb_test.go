/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package historykvdb

import (
	"chainmaker.org/chainmaker-go/store/cache"
	"chainmaker.org/chainmaker-go/store/dbprovider/leveldbprovider"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"chainmaker.org/chainmaker-go/localconf"
	acPb "chainmaker.org/chainmaker-go/pb/protogo/accesscontrol"
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	storePb "chainmaker.org/chainmaker-go/pb/protogo/store"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/protocol/test"
	"chainmaker.org/chainmaker-go/store/historydb"
	"chainmaker.org/chainmaker-go/store/serialization"
	"github.com/stretchr/testify/assert"
)

var log = &test.GoLogger{}

func generateBlockHash(chainId string, height int64) []byte {
	blockHash := sha256.Sum256([]byte(fmt.Sprintf("%s-%d", chainId, height)))
	return blockHash[:]
}

func generateTxId(chainId string, height int64, index int) string {
	txIdBytes := sha256.Sum256([]byte(fmt.Sprintf("%s-%d-%d", chainId, height, index)))
	return hex.EncodeToString(txIdBytes[:32])
}

func createConfigBlock(chainId string, height int64) *storePb.BlockWithRWSet {
	block := &commonPb.Block{
		Header: &commonPb.BlockHeader{
			ChainId:     chainId,
			BlockHeight: height,
		},
		Txs: []*commonPb.Transaction{
			{
				Header: &commonPb.TxHeader{
					ChainId: chainId,
					TxType:  commonPb.TxType_UPDATE_CHAIN_CONFIG,
					Sender: &acPb.SerializedMember{
						OrgId: "org1",
					},
				},
				Result: &commonPb.Result{
					Code: commonPb.TxStatusCode_SUCCESS,
					ContractResult: &commonPb.ContractResult{
						Result: []byte("ok"),
					},
				},
			},
		},
	}

	block.Header.BlockHash = generateBlockHash(chainId, height)
	block.Txs[0].Header.TxId = generateTxId(chainId, height, 0)
	return &storePb.BlockWithRWSet{
		Block:    block,
		TxRWSets: []*commonPb.TxRWSet{},
	}
}

func createBlockAndRWSets(chainId string, height int64, txNum int) *storePb.BlockWithRWSet {
	block := &commonPb.Block{
		Header: &commonPb.BlockHeader{
			ChainId:     chainId,
			BlockHeight: height,
		},
	}

	for i := 0; i < txNum; i++ {
		payload, _ := (&commonPb.TransactPayload{
			ContractName: "contract1",
			Method:       "Function1",
			Parameters:   nil,
		}).Marshal()
		tx := &commonPb.Transaction{
			Header: &commonPb.TxHeader{
				ChainId: chainId,
				TxId:    generateTxId(chainId, height, i),
				Sender: &acPb.SerializedMember{
					OrgId:      "org1",
					MemberInfo: []byte("User" + strconv.Itoa(i)),
				},
			},
			RequestPayload: payload,
			Result: &commonPb.Result{
				Code: commonPb.TxStatusCode_SUCCESS,
				ContractResult: &commonPb.ContractResult{
					Result: []byte("ok"),
				},
			},
		}
		block.Txs = append(block.Txs, tx)
	}

	block.Header.BlockHash = generateBlockHash(chainId, height)
	var txRWSets []*commonPb.TxRWSet
	for i := 0; i < txNum; i++ {
		key := fmt.Sprintf("key_%d", i)
		value := fmt.Sprintf("value_%d", i)
		txRWset := &commonPb.TxRWSet{
			TxId: block.Txs[i].Header.TxId,
			TxWrites: []*commonPb.TxWrite{
				{
					Key:          []byte(key),
					Value:        []byte(value),
					ContractName: "contract1",
				},
			},
		}
		txRWSets = append(txRWSets, txRWset)
	}

	return &storePb.BlockWithRWSet{
		Block:    block,
		TxRWSets: txRWSets,
	}
}

var testChainId = "testchainid_1"
var block0 = createConfigBlock(testChainId, 0)
var block1 = createBlockAndRWSets(testChainId, 1, 10)
var block2 = createBlockAndRWSets(testChainId, 2, 2)

/*var block3, _ = createBlockAndRWSets(testChainId, 3, 2)
var configBlock4 = createConfigBlock(testChainId, 4)
var block5, _ = createBlockAndRWSets(testChainId, 5, 3)*/

func createBlock(chainId string, height int64) *commonPb.Block {
	block := &commonPb.Block{
		Header: &commonPb.BlockHeader{
			ChainId:     chainId,
			BlockHeight: height,
		},
		Txs: []*commonPb.Transaction{
			{
				Header: &commonPb.TxHeader{
					ChainId: chainId,
					Sender: &acPb.SerializedMember{
						OrgId: "org1",
					},
				},
				Result: &commonPb.Result{
					Code: commonPb.TxStatusCode_SUCCESS,
					ContractResult: &commonPb.ContractResult{
						Result: []byte("ok"),
					},
				},
			},
		},
	}

	block.Header.BlockHash = generateBlockHash(chainId, height)
	block.Txs[0].Header.TxId = generateTxId(chainId, height, 0)
	return block
}

func initProvider() protocol.DBHandle {
	conf := &localconf.StorageConfig{}
	path := filepath.Join(os.TempDir(), fmt.Sprintf("%d", time.Now().Nanosecond()))
	conf.StorePath = path

	lvlConfig := &localconf.DbConfig{
		LevelDbConfig: &localconf.LevelDbConfig{
			StorePath: path,
		},
	}
	p := leveldbprovider.NewLevelDBHandle(testChainId, "test", lvlConfig, log)
	return p
}

//初始化DB并同时初始化创世区块
func initKvDb() *HistoryKvDB {
	db := NewHistoryKvDB(initProvider(), cache.NewStoreCacheMgr(testChainId, log), log)
	_, blockInfo, _ := serialization.SerializeBlock(block0)
	db.InitGenesis(blockInfo)
	return db
}

func TestHistoryKvDB_CommitBlock(t *testing.T) {
	db := initKvDb()
	block1.TxRWSets[0].TxWrites[0].Value = nil
	_, blockInfo, err := serialization.SerializeBlock(block1)
	assert.Nil(t, err)
	err = db.CommitBlock(blockInfo)
	assert.Nil(t, err)
	time.Sleep(time.Second)
}

func TestHistoryKvDB_GetHistoryForKey(t *testing.T) {
	db := initKvDb()
	block1.TxRWSets[0].TxWrites[0].Value = nil
	_, blockInfo, err := serialization.SerializeBlock(block1)
	assert.Nil(t, err)
	err = db.CommitBlock(blockInfo)
	assert.Nil(t, err)
	time.Sleep(time.Second)
	result, err := db.GetHistoryForKey("contract1", []byte("key_1"))
	assert.Nil(t, err)

	assert.Equal(t, 1, getCount(result))

}
func getCount(i historydb.HistoryIterator) int {
	count := 0
	for i.Next() {
		count++
		v, e := i.Value()
		fmt.Printf("\n=====\n%+v \n%s\n", v, e)
	}
	return count
}
func TestHistoryKvDB_GetAccountTxHistory(t *testing.T) {
	db := initKvDb()
	block1.TxRWSets[0].TxWrites[0].Value = nil
	_, blockInfo, err := serialization.SerializeBlock(block1)
	assert.Nil(t, err)
	err = db.CommitBlock(blockInfo)
	time.Sleep(time.Second)
	assert.Nil(t, err)
	result, err := db.GetAccountTxHistory([]byte("User1"))
	assert.Nil(t, err)
	assert.Equal(t, 1, getCount(result))
	for result.Next() {
		v, _ := result.Value()
		t.Logf("\n------\n%#v\n------\n", v)
	}
}
func TestHistoryKvDB_GetContractTxHistory(t *testing.T) {
	db := initKvDb()
	block1.TxRWSets[0].TxWrites[0].Value = nil
	_, blockInfo, err := serialization.SerializeBlock(block1)
	err = db.CommitBlock(blockInfo)
	assert.Nil(t, err)
	time.Sleep(time.Second)
	result, err := db.GetContractTxHistory("contract1")
	assert.Nil(t, err)
	assert.Equal(t, 10, getCount(result))
	for result.Next() {
		v, _ := result.Value()
		t.Logf("%#v", v)
	}
}
