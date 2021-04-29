/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cache

import (
	"chainmaker.org/chainmaker-go/utils"
	"encoding/hex"
	"errors"
	"sync"

	commonpb "chainmaker.org/chainmaker-go/pb/protogo/common"
)

//Abft tx batch structure
type VerifiedTxBatchCache struct {
	txBatch      *commonpb.Block
	verifyResult bool
	rwSetMap     map[string]*commonpb.TxRWSet
}

//After propose txbatch cache
type ProposedTxBatchCache struct {
	fingerPrint utils.BlockFingerPrint
	txBatch     *commonpb.Block
	rwSetMap    map[string]*commonpb.TxRWSet
}

//abft complete structure
type AbftCache struct {
	proposedTxBatchCache    *ProposedTxBatchCache //After propose txbatch cache
	verifiedTxBatchCacheMap sync.Map              //After propose txbatch cache map
}

func NewAbftCache() *AbftCache {
	return &AbftCache{
		proposedTxBatchCache:    nil,
		verifiedTxBatchCacheMap: sync.Map{},
	}
}

// Add the TxBatch after honey badger bft
func (hc *AbftCache) AddVerifiedTxBatch(b *commonpb.Block, c bool, rwSetMap map[string]*commonpb.TxRWSet) error {
	if b == nil || b.Header == nil {
		return errors.New("set the tx batch failed,block can't be empty")
	}
	hb := &VerifiedTxBatchCache{
		txBatch:      b,
		verifyResult: c,
		rwSetMap:     rwSetMap,
	}
	hc.verifiedTxBatchCacheMap.Store(hex.EncodeToString(b.Header.BlockHash), hb)
	return nil
}

// Get the TxBatch by result
func (hc *AbftCache) GetVerifiedTxBatchsByResult(c bool) []*VerifiedTxBatchCache {

	txBatch := make([]*VerifiedTxBatchCache, 0)

	hc.verifiedTxBatchCacheMap.Range(func(_, hb interface{}) bool {
		if hb.(*VerifiedTxBatchCache).verifyResult == c {
			txBatch = append(txBatch, hb.(*VerifiedTxBatchCache))
		}
		return true
	})
	return txBatch
}

// Get block by BlockHash
func (hc *AbftCache) GetVerifiedTxBatchByHash(hash []byte) (*VerifiedTxBatchCache, error) {
	if hash == nil {
		return nil, errors.New("get verified tx batch failed, tx batch can't be empty")
	}
	VerifiedTxBatch, ok := hc.verifiedTxBatchCacheMap.Load(hex.EncodeToString(hash))
	if !ok {
		return nil, errors.New("get verified tx batch failed, tx batch is not exits")
	}
	return VerifiedTxBatch.(*VerifiedTxBatchCache), nil

}

// return if a TxBatch has verified
func (hc *AbftCache) HasVerifiedTxBatch(hash []byte) bool {
	_, ok := hc.verifiedTxBatchCacheMap.Load(hex.EncodeToString(hash))
	return ok
}

// return if this block is success after RBC verification
func (hc *AbftCache) IsVerifiedTxBatchSuccess(hash []byte) (bool, error) {
	VerifiedTxBatch, ok := hc.verifiedTxBatchCacheMap.Load(hex.EncodeToString(hash))
	if !ok {
		return false, errors.New("tx batch not exist")
	}
	return VerifiedTxBatch.(*VerifiedTxBatchCache).verifyResult, nil
}

func (vtbc *VerifiedTxBatchCache) GetTxBatch() *commonpb.Block {
	return vtbc.txBatch
}
func (vtbc *VerifiedTxBatchCache) GetVerifyResult() bool {
	return vtbc.verifyResult
}
func (vtbc *VerifiedTxBatchCache) GetTxBatchRwSet() map[string]*commonpb.TxRWSet {
	return vtbc.rwSetMap
}

func (vtbc *VerifiedTxBatchCache) SetTxBatch(txBatch *commonpb.Block) {
	vtbc.txBatch = txBatch
}
func (vtbc *VerifiedTxBatchCache) SetVerifyResult(result bool) {
	vtbc.verifyResult = result
}
func (vtbc *VerifiedTxBatchCache) SetTxBatchRwSet(rwSet map[string]*commonpb.TxRWSet) {
	vtbc.rwSetMap = rwSet
}
func (ptbc *ProposedTxBatchCache) GetTxBatch() *commonpb.Block {
	return ptbc.txBatch
}
func (ptbc *ProposedTxBatchCache) GetFingerPrint() utils.BlockFingerPrint {
	return ptbc.fingerPrint
}
func (ptbc *ProposedTxBatchCache) GetRwSetMap() map[string]*commonpb.TxRWSet {
	return ptbc.rwSetMap
}
func (hc *AbftCache) GetProposedTxBatch() *ProposedTxBatchCache {
	return hc.proposedTxBatchCache
}
func (hc *AbftCache) GetVerifiedTxBatchMap() sync.Map {
	return hc.verifiedTxBatchCacheMap
}
func (hc *AbftCache) SetProposedTxBatch(txBatch *commonpb.Block, rwSetMap map[string]*commonpb.TxRWSet) {
	hc.proposedTxBatchCache = &ProposedTxBatchCache{
		fingerPrint: utils.CalcBlockFingerPrint(txBatch),
		txBatch:  txBatch,
		rwSetMap: rwSetMap,
	}
}

func (hc *AbftCache) ClearAbftCache() {
	hc.proposedTxBatchCache = nil
	hc.verifiedTxBatchCacheMap = sync.Map{}
}
