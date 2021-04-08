/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cache

import (
	"errors"
	"sync"

	commonpb "chainmaker.org/chainmaker-go/pb/protogo/common"
)

const (
	FAIL uint32 = iota
	SUCCESS
)

type hbbftTxBatch struct {
	txBatch  *commonpb.Block
	code     uint32                       //校验是否成功
	rwSetMap map[string]*commonpb.TxRWSet //该交易批次读写集Map
}

type HbbftCache struct {
	txBatchCache *commonpb.Block

	hbbftTxBatchCacheMap sync.Map
}

func NewhbbftCacheMap() *HbbftCache {
	hc := &HbbftCache{
		hbbftTxBatchCacheMap: sync.Map{},
	}
	return hc
}

// Set the TxBatch after honey badger bft
func (hc *HbbftCache) SethbbftTxBatch(b *commonpb.Block, c uint32, rwSetMap map[string]*commonpb.TxRWSet) error {
	if b == nil || b.Header == nil {
		return errors.New("set the tx batch failed,block can't be empty")
	}
	hb := &hbbftTxBatch{
		txBatch:  b,
		code:     c,
		rwSetMap: rwSetMap,
	}
	blockHash := b.Header.BlockHash
	hc.hbbftTxBatchCacheMap.Store(string(blockHash), hb)
	return nil
}

// Get the whole TxBatchs after hbbft
func (hc *HbbftCache) GetVerifiedhbbftTxBatchs() *HbbftCache {
	return hc
}

// Get the TxBatch by code
func (hc *HbbftCache) GetVerifiedhbbftTxBatchsByCode(c uint32) *HbbftCache {
	Newhc := NewhbbftCacheMap()

	hc.hbbftTxBatchCacheMap.Range(func(_, hb interface{}) bool {
		if hb.(hbbftTxBatch).code == c {
			Newhc.SethbbftTxBatch(hb.(hbbftTxBatch).txBatch, c, hb.(hbbftTxBatch).rwSetMap)
		}
		return true
	})
	return Newhc
}

// Get block by BlockHash
func (hc *HbbftCache) GetVerifiedTxBatchByHash(hash []byte) (*hbbftTxBatch, error) {
	if hash == nil {
		return nil, errors.New("get verified tx batch failed, tx batch can't be empty")
	}
	if VerifiedTxBatch, ok := hc.hbbftTxBatchCacheMap.Load(string(hash)); ok {
		return VerifiedTxBatch.(*hbbftTxBatch), nil
	}
	return nil, nil
}

// return if a TxBatch has cached
func (hc *HbbftCache) HasVerifiedTxBatch(hash []byte) bool {
	_, ok := hc.hbbftTxBatchCacheMap.Load(string(hash))
	return ok
}

// return if this block is success after RBC verification
func (hc *HbbftCache) IsVerifiedTxBatchSuccess(hash []byte) (bool, error) {
	VerifiedTxBatch, ok := hc.hbbftTxBatchCacheMap.Load(string(hash))
	if ok {
		return VerifiedTxBatch.(hbbftTxBatch).code == SUCCESS, nil
	}
	return false, errors.New("TxBatch not exist")
}
