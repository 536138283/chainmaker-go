/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cache

import (
	"encoding/hex"
	"errors"
	"sync"

	commonpb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/utils/v2"
)

// VerifiedTxBatchCache Abft tx batch structure
type VerifiedTxBatchCache struct {
	txBatch      *commonpb.Block
	verifyResult bool
	rwSetMap     map[string]*commonpb.TxRWSet
	rwMu         sync.RWMutex
}

// ProposedTxBatchCache After propose txbatch cache
type ProposedTxBatchCache struct {
	fingerPrint utils.BlockFingerPrint
	txBatch     *commonpb.Block
	rwSetMap    map[string]*commonpb.TxRWSet
}

// AbftCache abft complete structure
type AbftCache struct {
	proposedTxBatchCache    *ProposedTxBatchCache //After propose txbatch cache
	verifiedTxBatchCacheMap sync.Map              //After propose txbatch cache map
	rwMu                    sync.RWMutex
}

// NewAbftCache return AbftCache
func NewAbftCache() *AbftCache {
	return &AbftCache{
		proposedTxBatchCache:    nil,
		verifiedTxBatchCacheMap: sync.Map{},
	}
}

// AddVerifiedTxBatch Add the TxBatch after honey badger bft
func (hc *AbftCache) AddVerifiedTxBatch(b *commonpb.Block, c bool, rwSetMap map[string]*commonpb.TxRWSet) error {
	if b == nil || b.Header == nil {
		return errors.New("set the tx batch failed,block can't be empty")
	}
	hb := &VerifiedTxBatchCache{
		txBatch:      b,
		verifyResult: c,
		rwSetMap:     rwSetMap,
	}
	hc.rwMu.Lock()
	defer hc.rwMu.Unlock()
	hc.verifiedTxBatchCacheMap.Store(hex.EncodeToString(b.Header.BlockHash), hb)
	return nil
}

// GetVerifiedTxBatchsByResult Get the TxBatch by result
func (hc *AbftCache) GetVerifiedTxBatchsByResult(c bool) []*VerifiedTxBatchCache {
	txBatch := make([]*VerifiedTxBatchCache, 0)
	hc.rwMu.RLock()
	defer hc.rwMu.RUnlock()
	hc.verifiedTxBatchCacheMap.Range(func(_, hb interface{}) bool {
		if hb.(*VerifiedTxBatchCache).verifyResult == c {
			txBatch = append(txBatch, hb.(*VerifiedTxBatchCache))
		}
		return true
	})
	return txBatch
}

// GetVerifiedTxBatchByHash Get block by BlockHash
func (hc *AbftCache) GetVerifiedTxBatchByHash(hash []byte) (*VerifiedTxBatchCache, error) {
	if hash == nil {
		return nil, errors.New("get verified tx batch failed, tx batch can't be empty")
	}
	hc.rwMu.RLock()
	defer hc.rwMu.RUnlock()
	verifiedTxBatch, ok := hc.verifiedTxBatchCacheMap.Load(hex.EncodeToString(hash))
	if !ok {
		return nil, errors.New("get verified tx batch failed, tx batch is not exits")
	}
	return verifiedTxBatch.(*VerifiedTxBatchCache), nil

}

// HasVerifiedTxBatch return if a TxBatch has verified
func (hc *AbftCache) HasVerifiedTxBatch(hash []byte) bool {
	hc.rwMu.RLock()
	defer hc.rwMu.RUnlock()
	_, ok := hc.verifiedTxBatchCacheMap.Load(hex.EncodeToString(hash))
	return ok
}

// IsVerifiedTxBatchSuccess return if this block is success after RBC verification
func (hc *AbftCache) IsVerifiedTxBatchSuccess(hash []byte) (bool, error) {
	hc.rwMu.RLock()
	defer hc.rwMu.RUnlock()
	verifiedTxBatch, ok := hc.verifiedTxBatchCacheMap.Load(hex.EncodeToString(hash))
	if !ok {
		return false, errors.New("tx batch not exist")
	}
	return verifiedTxBatch.(*VerifiedTxBatchCache).verifyResult, nil
}

// GetTxBatch return Block
func (vtbc *VerifiedTxBatchCache) GetTxBatch() *commonpb.Block {
	vtbc.rwMu.RLock()
	defer vtbc.rwMu.RUnlock()
	return vtbc.txBatch
}

// GetVerifyResult return bool
func (vtbc *VerifiedTxBatchCache) GetVerifyResult() bool {
	vtbc.rwMu.RLock()
	defer vtbc.rwMu.RUnlock()
	return vtbc.verifyResult
}

// GetTxBatchRwSet return map string TxRWSet
func (vtbc *VerifiedTxBatchCache) GetTxBatchRwSet() map[string]*commonpb.TxRWSet {
	vtbc.rwMu.RLock()
	defer vtbc.rwMu.RUnlock()
	return vtbc.rwSetMap
}

// SetTxBatch params txBatch
func (vtbc *VerifiedTxBatchCache) SetTxBatch(txBatch *commonpb.Block) {
	vtbc.rwMu.Lock()
	defer vtbc.rwMu.Unlock()
	vtbc.txBatch = txBatch
}

// SetVerifyResult params result bool
func (vtbc *VerifiedTxBatchCache) SetVerifyResult(result bool) {
	vtbc.rwMu.Lock()
	defer vtbc.rwMu.Unlock()
	vtbc.verifyResult = result
}

// SetTxBatchRwSet params rwSet map
func (vtbc *VerifiedTxBatchCache) SetTxBatchRwSet(rwSet map[string]*commonpb.TxRWSet) {
	vtbc.rwMu.Lock()
	defer vtbc.rwMu.Unlock()
	vtbc.rwSetMap = rwSet
}

// GetTxBatch return block
func (ptbc *ProposedTxBatchCache) GetTxBatch() *commonpb.Block {
	return ptbc.txBatch
}

// GetFingerPrint return BlockFingerPrint
func (ptbc *ProposedTxBatchCache) GetFingerPrint() utils.BlockFingerPrint {
	return ptbc.fingerPrint
}

// GetRwSetMap return map string TxRWSet
func (ptbc *ProposedTxBatchCache) GetRwSetMap() map[string]*commonpb.TxRWSet {
	return ptbc.rwSetMap
}

// GetProposedTxBatch return ProposedTxBatchCache
func (hc *AbftCache) GetProposedTxBatch() *ProposedTxBatchCache {
	hc.rwMu.RLock()
	defer hc.rwMu.RUnlock()
	return hc.proposedTxBatchCache
}

// GetVerifiedTxBatchMap return sync map
func (hc *AbftCache) GetVerifiedTxBatchMap() *sync.Map {
	hc.rwMu.RLock()
	defer hc.rwMu.RUnlock()
	return &hc.verifiedTxBatchCacheMap
}

// SetProposedTxBatch params txBatch, rwSetMap
func (hc *AbftCache) SetProposedTxBatch(txBatch *commonpb.Block, rwSetMap map[string]*commonpb.TxRWSet) {
	hc.rwMu.Lock()
	defer hc.rwMu.Unlock()
	hc.proposedTxBatchCache = &ProposedTxBatchCache{
		fingerPrint: utils.CalcBlockFingerPrint(txBatch),
		txBatch:     txBatch,
		rwSetMap:    rwSetMap,
	}
}

// ClearAbftCache clear abft cache
func (hc *AbftCache) ClearAbftCache() {
	hc.rwMu.Lock()
	defer hc.rwMu.Unlock()
	hc.proposedTxBatchCache = nil
	hc.verifiedTxBatchCacheMap = sync.Map{}
}
