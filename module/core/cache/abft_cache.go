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

type AbftTxBatch struct {
	txBatch  *commonpb.Block
	code     uint32                       //校验是否成功
	rwSetMap map[string]*commonpb.TxRWSet //该交易批次读写集Map
}

type AbftCache struct {
	txBatchCache        *commonpb.Block //节点打包的单个批次缓存
	abftTxBatchCacheMap sync.Map        //节点校验后的批次集合
}

func NewAbftCache() *AbftCache {
	return &AbftCache{
		txBatchCache:        nil,
		abftTxBatchCacheMap: sync.Map{},
	}
}

// Add the TxBatch after honey badger bft
func (hc *AbftCache) AddAbftTxBatch(b *commonpb.Block, c uint32, rwSetMap map[string]*commonpb.TxRWSet) error {
	if b == nil || b.Header == nil {
		return errors.New("set the tx batch failed,block can't be empty")
	}
	hb := &AbftTxBatch{
		txBatch:  b,
		code:     c,
		rwSetMap: rwSetMap,
	}
	hc.abftTxBatchCacheMap.Store(hex.EncodeToString(b.Header.BlockHash), hb)
	return nil
}

// Get the TxBatch by code
func (hc *AbftCache) GetVerifiedAbftTxBatchsByCode(c uint32) []*AbftTxBatch {

	txBatch := make([]*AbftTxBatch, 0)

	hc.abftTxBatchCacheMap.Range(func(_, hb interface{}) bool {
		if hb.(*AbftTxBatch).code == c {
			txBatch = append(txBatch, hb.(*AbftTxBatch))
		}
		return true
	})
	return txBatch
}

// Get block by BlockHash
func (hc *AbftCache) GetVerifiedTxBatchByHash(hash []byte) (*AbftTxBatch, error) {
	if hash == nil {
		return nil, errors.New("get verified tx batch failed, tx batch can't be empty")
	}
	VerifiedTxBatch, ok := hc.abftTxBatchCacheMap.Load(hex.EncodeToString(hash))
	if !ok {
		return nil, nil
	}
	return VerifiedTxBatch.(*AbftTxBatch), nil

}

// return if a TxBatch has verified
func (hc *AbftCache) HasVerifiedTxBatch(hash []byte) bool {
	_, ok := hc.abftTxBatchCacheMap.Load(hex.EncodeToString(hash))
	return ok
}

// return if this block is success after RBC verification
func (hc *AbftCache) IsVerifiedTxBatchSuccess(hash []byte) (bool, error) {
	VerifiedTxBatch, ok := hc.abftTxBatchCacheMap.Load(hex.EncodeToString(hash))
	if !ok {
		return false, errors.New("TxBatch not exist")
	}
	return VerifiedTxBatch.(*AbftTxBatch).code == SUCCESS, nil
}

func (htb *AbftTxBatch) GetTxBatch() *commonpb.Block {
	return htb.txBatch
}
func (htb *AbftTxBatch) GetCode() uint32 {
	return htb.code
}
func (htb *AbftTxBatch) GetTxBatchRwSet() map[string]*commonpb.TxRWSet {
	return htb.rwSetMap
}

func (htb *AbftTxBatch) SetTxBatch(txBatch *commonpb.Block) {
	htb.txBatch = txBatch
}
func (htb *AbftTxBatch) SetCode(code uint32) {
	htb.code = code
}
func (htb *AbftTxBatch) SetTxBatchRwSet(rwSet map[string]*commonpb.TxRWSet) {
	htb.rwSetMap = rwSet
}

func (hc *AbftCache) GetTxBatchCache() *commonpb.Block {
	return hc.txBatchCache
}

func (hc *AbftCache) GetAbftTxBatchCacheMap() sync.Map {
	return hc.abftTxBatchCacheMap
}
func (hc *AbftCache) SetTxBatchCache(txBatch *commonpb.Block) {
	hc.txBatchCache = txBatch
}

func (hc *AbftCache) ClearAbftCache() {
	hc.txBatchCache = nil
	hc.abftTxBatchCacheMap = sync.Map{}
}
