/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package cache

import (
	"fmt"
	"testing"

	commonpb "chainmaker.org/chainmaker/pb-go/v3/common"
)

func TestAbft(t *testing.T) {
	hc := NewAbftCache()
	rwSetMap := make(map[string]*commonpb.TxRWSet)
	b0 := CreateNewTestBlock(0)
	hash0 := b0.Header.BlockHash
	ok := hc.AddVerifiedTxBatch(b0, false, rwSetMap)
	fmt.Println(ok)
	b1 := CreateNewTestBlock(1)
	hash1 := []byte{'1', '2', '3', '4', '5', '6', '7', '8', '9', '0'}
	b1.Header.BlockHash = hash1
	ok = hc.AddVerifiedTxBatch(b1, true, rwSetMap)
	fmt.Println(ok)
	b2 := CreateNewTestBlock(2)
	hash2 := []byte{'2', '3', '4', '5', '6', '7', '8', '9', '0', '1'}
	b2.Header.BlockHash = hash2
	ok = hc.AddVerifiedTxBatch(b2, true, rwSetMap)
	fmt.Println(ok)
	b3 := CreateNewTestBlock(3)
	hash3 := []byte{'3', '4', '5', '6', '7', '8', '9', '0', '1', '2'}
	b3.Header.BlockHash = hash3
	ok = hc.AddVerifiedTxBatch(b3, false, rwSetMap)
	fmt.Println(ok)

	fmt.Println("Get TxBatch by FAIL code")
	tb0 := hc.GetVerifiedTxBatchsByResult(false)
	for i := 0; i < len(tb0); i++ {
		fmt.Println(tb0[i].txBatch.Header.BlockHash)
	}
	fmt.Println("Get TxBatch by SUCCESS code")
	tb1 := hc.GetVerifiedTxBatchsByResult(false)
	for i := 0; i < len(tb1); i++ {
		fmt.Println(tb1[i].txBatch.Header.BlockHash)
	}

	fmt.Println("Get TxBatch by Hash")
	htb0, ok := hc.GetVerifiedTxBatchByHash(hash0)
	if ok == nil {
		b := htb0.GetTxBatch()
		fmt.Println(b.Header.BlockHeight)
		fmt.Println(htb0.GetVerifyResult())
		fmt.Println(len(htb0.GetTxBatchRwSet()))
	} else {
		fmt.Println(ok)
	}
	htb1, ok := hc.GetVerifiedTxBatchByHash(hash1)
	if ok == nil {
		htb1.SetVerifyResult(false)
		b := htb1.GetTxBatch()
		b.Header.BlockHash = []byte{'4', '5', '6', '7', '8', '9', '0', '1', '2', '3'}
		htb1.SetTxBatch(b)
		rw := htb1.GetTxBatchRwSet()
		htb1.SetTxBatchRwSet(rw)
	} else {
		fmt.Println(ok)
	}

	fmt.Println(hc.HasVerifiedTxBatch(hash2))
	fmt.Println(hc.IsVerifiedTxBatchSuccess(hash3))

	hc.SetProposedTxBatch(CreateNewTestBlock(4), nil)
	tbc := hc.GetProposedTxBatch()
	fmt.Println(tbc.txBatch.Header.BlockHash)
	hcMap := hc.GetVerifiedTxBatchMap()
	b, ok0 := hcMap.Load(string(hash2))
	if ok0 {
		fmt.Println(b.(commonpb.Block).Header.BlockHeight)
	}
	hc.ClearAbftCache()
}
