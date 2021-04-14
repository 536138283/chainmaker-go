/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cache

import (
	"encoding/hex"
	"errors"
	"sync"

	commonpb "chainmaker.org/chainmaker-go/pb/protogo/common"
)

const (
	FAIL uint32 = iota
	SUCCESS
)

type HbbftTxBatch struct {
	txBatch  *commonpb.Block
	code     uint32                       //校验是否成功
	rwSetMap map[string]*commonpb.TxRWSet //该交易批次读写集Map
}

type HbbftCache struct {
	txBatchCache *commonpb.Block //节点打包的单个批次缓存
	//txBatchCacheMap      map[string]*commonpb.Block //节点收到RBC后需要校验的批次集合（防止校验遗漏）
	hbbftTxBatchCacheMap sync.Map //节点校验后的批次集合
}

func NewHbbftCache() *HbbftCache {
	return &HbbftCache{
		txBatchCache: nil,
		//txBatchCacheMap:      make(map[string]*commonpb.Block),
		hbbftTxBatchCacheMap: sync.Map{},
	}
}

// Add the TxBatch after honey badger bft
func (hc *HbbftCache) AddHbbftTxBatch(b *commonpb.Block, c uint32, rwSetMap map[string]*commonpb.TxRWSet) error {
	if b == nil || b.Header == nil {
		return errors.New("set the tx batch failed,block can't be empty")
	}
	hb := &HbbftTxBatch{
		txBatch:  b,
		code:     c,
		rwSetMap: rwSetMap,
	}
	hc.hbbftTxBatchCacheMap.Store(hex.EncodeToString(b.Header.BlockHash), hb)
	return nil
}

// Get the TxBatch by code
func (hc *HbbftCache) GetVerifiedHbbftTxBatchsByCode(c uint32) []*HbbftTxBatch {

	txBatch := make([]*HbbftTxBatch, 0)

	hc.hbbftTxBatchCacheMap.Range(func(_, hb interface{}) bool {
		if hb.(*HbbftTxBatch).code == c {
			txBatch = append(txBatch, hb.(*HbbftTxBatch))
		}
		return true
	})
	return txBatch
}

// Get block by BlockHash
func (hc *HbbftCache) GetVerifiedTxBatchByHash(hash []byte) (*HbbftTxBatch, error) {
	if hash == nil {
		return nil, errors.New("get verified tx batch failed, tx batch can't be empty")
	}
	if VerifiedTxBatch, ok := hc.hbbftTxBatchCacheMap.Load(hex.EncodeToString(hash)); ok {
		return VerifiedTxBatch.(*HbbftTxBatch), nil
	}
	return nil, nil
}

// return if a TxBatch has verified
func (hc *HbbftCache) HasVerifiedTxBatch(hash []byte) bool {
	_, ok := hc.hbbftTxBatchCacheMap.Load(hex.EncodeToString(hash))
	return ok
}

// return if this block is success after RBC verification
func (hc *HbbftCache) IsVerifiedTxBatchSuccess(hash []byte) (bool, error) {
	VerifiedTxBatch, ok := hc.hbbftTxBatchCacheMap.Load(hex.EncodeToString(hash))
	if ok {
		return VerifiedTxBatch.(*HbbftTxBatch).code == SUCCESS, nil
	}
	return false, errors.New("TxBatch not exist")
}

//func (hc *HbbftCache) AddTxBatch(txBatch *commonpb.Block) {
//	hc.txBatchCacheMap[hex.EncodeToString(txBatch.Header.BlockHash)] = txBatch
//}

func (htb *HbbftTxBatch) GetTxBatch() *commonpb.Block {
	return htb.txBatch
}
func (htb *HbbftTxBatch) GetCode() uint32 {
	return htb.code
}
func (htb *HbbftTxBatch) GetTxBatchRwSet() map[string]*commonpb.TxRWSet {
	return htb.rwSetMap
}

func (htb *HbbftTxBatch) SetTxBatch(txBatch *commonpb.Block) {
	htb.txBatch = txBatch
}
func (htb *HbbftTxBatch) SetCode(code uint32) {
	htb.code = code
}
func (htb *HbbftTxBatch) SetTxBatchRwSet(rwSet map[string]*commonpb.TxRWSet) {
	htb.rwSetMap = rwSet
}

func (hc *HbbftCache) GetTxBatchCache() *commonpb.Block {
	return hc.txBatchCache
}

//func (hc *HbbftCache) GetTxBatchCacheMap() map[string]*commonpb.Block {
//	return hc.txBatchCacheMap
//}
func (hc *HbbftCache) GetHbbftTxBatchCacheMap() sync.Map {
	return hc.hbbftTxBatchCacheMap
}
func (hc *HbbftCache) SetTxBatchCache(txBatch *commonpb.Block) {
	hc.txBatchCache = txBatch
}

func (hc *HbbftCache) ClearHbbftCache() {
	//hc.txBatchCacheMap = make(map[string]*commonpb.Block, 0)
	hc.txBatchCache = nil
	hc.hbbftTxBatchCacheMap = sync.Map{}
}

//func (hc *HbbftCache) GetTxBatchCacheByHash(hash []byte) *commonpb.Block {
//	txBatch, ok := hc.txBatchCacheMap[hex.EncodeToString(hash)]
//	if ok {
//		return txBatch
//	} else {
//		return nil
//	}
//}
