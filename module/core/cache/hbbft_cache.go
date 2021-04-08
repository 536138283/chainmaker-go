package cache

import (
	commonpb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/protocol"
	"errors"
	"sync"
)

type hbbftTxBatch struct {
	txBatch 	*commonpb.Block
	code 		uint32 //校验是否成功
	rwSetMap 	map[string]*commonpb.TxRWSet //该交易批次读写集Map
}

var hbbftTxBatchCacheMap sync.Map //string 交易批次哈希
type HbbftCache struct
{
	hbbftTxBatchCacheMap 	sync.Map
	order 					[]string
}

func NewhbbftCacheMap() protocol.HbbftCache {
	hc := &HbbftCache{
		hbbftTxBatchCacheMap: 	sync.Map{},
		order: 					make([]string,0),
	}
	return hc
}

// Set the TxBatch after honey badger bft
func (hc *HbbftCache) SethbbftTxBatch(b *commonpb.Block, c uint32, rwSetMap map[string]*commonpb.TxRWSet) error{
	if b == nil || b.Header == nil {
		return nil
	}
	hb := &hbbftTxBatch{
		txBatch: 	b,
		code: 		c,
		rwSetMap: 	rwSetMap,
	}
	blockHash := b.Header.BlockHash
	hc.hbbftTxBatchCacheMap.Store(string(blockHash),hb)
	hc.order = append(hc.order, string(blockHash))
	return nil
}

// Get the whole TxBatchs after hbbft
func (hc *HbbftCache) GetVerifiedhbbftTxBatchs() protocol.HbbftCache {
	return hc
}

// Get the TxBatch by code
func (hc *HbbftCache) GetVerifiedhbbftTxBatchsByCode(c uint32) protocol.HbbftCache {
	Newhc := NewhbbftCacheMap()

	hc.hbbftTxBatchCacheMap.Range(func(_, hb interface{}) bool {
		if hb.(hbbftTxBatch).code == c {
			Newhc.SethbbftTxBatch(hb.(hbbftTxBatch).txBatch, c, hb.(hbbftTxBatch).rwSetMap)
		}
		return true
	})
	return Newhc
}

// Get Block's information
func (hc *HbbftCache) GetVerifiedTxBatch(b *commonpb.Block) (*commonpb.Block, uint32, map[string]*commonpb.TxRWSet) {
	if b == nil || b.Header == nil {
		return nil, 2, nil
	}
	blockHash := b.Header.BlockHash
	if VerifiedTxBatch, ok := hc.hbbftTxBatchCacheMap.Load(string(blockHash)); ok {
		return VerifiedTxBatch.(hbbftTxBatch).txBatch, VerifiedTxBatch.(hbbftTxBatch).code, VerifiedTxBatch.(hbbftTxBatch).rwSetMap
	}
	return nil, 2, nil
}

// Get block by BlockHash
func (hc *HbbftCache) GetVerifiedTxBatchByHash(hash []byte) (*commonpb.Block, uint32, map[string]*commonpb.TxRWSet){
	if hash == nil {
		return nil, 2, nil
	}
	if VerifiedTxBatch, ok := hc.hbbftTxBatchCacheMap.Load(string(hash)); ok {
		return VerifiedTxBatch.(hbbftTxBatch).txBatch, VerifiedTxBatch.(hbbftTxBatch).code, VerifiedTxBatch.(hbbftTxBatch).rwSetMap
	}
	return nil, 2, nil
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
		return VerifiedTxBatch.(hbbftTxBatch).code == 1, nil
	}
	return false, errors.New("TxBatch not exist")
}

// Get the TxBatch that is most recently set
func (hc *HbbftCache) GetLastVerifiedTxBatch() (*commonpb.Block, uint32, map[string]*commonpb.TxRWSet) {
	hash := hc.order[len(hc.order)-1]
	return hc.GetVerifiedTxBatchByHash([]byte(hash))
}