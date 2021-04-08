package cache

import (
	commonpb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"fmt"
	"testing"
)

func TestHbbft(t *testing.T) {
	hbbftCache := NewhbbftCacheMap()
	rwSetMap := make(map[string]*commonpb.TxRWSet)
	b0 := CreateNewTestBlock(0)
	hbbftCache.SethbbftTxBatch(b0, 0, rwSetMap)
	hash0 := b0.Header.BlockHash
	hc := hbbftCache.GetVerifiedhbbftTxBatchs()
	b1 := CreateNewTestBlock(1)
	b1.Header.BlockHash = []byte{'1','2','3','4','5','6','7','8','9','0'}
	hash1 := b1.Header.BlockHash
	hc.SethbbftTxBatch(b1,1,rwSetMap)
	b, c, m := hc.GetVerifiedTxBatchByHash(hash1)
	hbbftCache.SethbbftTxBatch(b,c,m)
	hc0 := hbbftCache.GetVerifiedhbbftTxBatchsByCode(1)
	hc0.hbbftTxBatchCacheMap.Range(func(k, _ interface{}) bool {
		fmt.Println(k)
		return true
	})
	fmt.Println(hbbftCache.HasVerifiedTxBatch(hash1))
	fmt.Println(hbbftCache.IsVerifiedTxBatchSuccess(hash0))
}