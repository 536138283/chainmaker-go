/*
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 */

package snapshot

import (
	"chainmaker.org/chainmaker/pb-go/v2/common"
	"sync"
)

const (
	Prime32             = uint32(16777619)
	ShardNum            = 64
	KeysPerShardDefault = 1024
	LowLevel            = 10  // 低水位，在低水位一下的采用串行写入方式
	HighLevel           = 100 // 高水位，高水位以上采用分组并行方式，处于Low~High中间的采用分组串行写入
)

type WrapSv struct {
	k  string
	sv *sv
}

type ShardSet struct {
	shardNum int
	shards   []*Shard
}

func newShardSet() *ShardSet {
	shards := make([]*Shard, ShardNum)
	for i := 0; i < ShardNum; i++ {
		shards[i] = newShard()
	}
	return &ShardSet{
		shardNum: ShardNum,
		shards:   shards,
	}
}

func (s *ShardSet) getShardNum(k string) int {
	return shardNum(k, s.shardNum)
}

func (s *ShardSet) getByLock(k string) (*sv, bool) {
	shard := s.shards[shardNum(k, s.shardNum)]
	return shard.getByLock(k)
}

func (s *ShardSet) putByLock(k string, sv *sv) {
	shard := s.shards[shardNum(k, s.shardNum)]
	shard.putByLock(k, sv)
}

func (s *ShardSet) put(k string, sv *sv) {
	shard := s.shards[shardNum(k, s.shardNum)]
	shard.putByLock(k, sv)
}

func (s *ShardSet) putNoLock(k string, sv *sv) {
	shard := s.shards[shardNum(k, s.shardNum)]
	shard.put(k, sv)
}

func (s *ShardSet) putReads(applySeq int, txReads []*common.TxRead) {
	n := len(txReads)
	if n <= LowLevel {
		for _, txRead := range txReads {
			finalKey := constructKey(txRead.ContractName, txRead.Key)
			sv := &sv{
				seq:   applySeq,
				value: txRead.Value,
			}
			s.putByLock(finalKey, sv)
		}
		return
	}
	shards := make([][]*ksv, s.shardNum)
	for _, txRead := range txReads {
		k := constructKey(txRead.ContractName, txRead.Key)
		v := &sv{
			seq:   applySeq,
			value: txRead.Value,
		}
		sn := shardNum(k, s.shardNum)
		if shards[sn] != nil {
			shards[sn] = append(shards[sn], &ksv{
				k: k,
				v: v,
			})
		} else {
			// 设置容量
			shard := make([]*ksv, 0, 16)
			shard[0] = &ksv{
				k: k,
				v: v,
			}
			shards[sn] = shard
		}
	}
	// 两种写入模型，如果数量较少则串行同步分组写入，否则并行同步分组写入
	if n <= HighLevel {
		s.putsBySerial(shards)
		return
	}
	s.putsByParallel(shards)
}

func (s *ShardSet) putWrites(applySeq int, txWrites []*common.TxWrite) {
	n := len(txWrites)
	if n <= LowLevel {
		for _, txWrite := range txWrites {
			finalKey := constructKey(txWrite.ContractName, txWrite.Key)
			sv := &sv{
				seq:   applySeq,
				value: txWrite.Value,
			}
			s.putByLock(finalKey, sv)
		}
		return
	}
	shards := make([][]*ksv, s.shardNum)
	for _, txWrite := range txWrites {
		k := constructKey(txWrite.ContractName, txWrite.Key)
		v := &sv{
			seq:   applySeq,
			value: txWrite.Value,
		}
		sn := shardNum(k, s.shardNum)
		if shards[sn] != nil {
			shards[sn] = append(shards[sn], &ksv{
				k: k,
				v: v,
			})
		} else {
			// 设置容量
			shard := make([]*ksv, 0, 16)
			shard[0] = &ksv{
				k: k,
				v: v,
			}
			shards[sn] = shard
		}
	}
	// 两种写入模型，如果数量较少则串行同步分组写入，否则并行同步分组写入
	if n <= HighLevel {
		s.putsBySerial(shards)
		return
	}
	s.putsByParallel(shards)
}

func (s *ShardSet) puts(ks []string, svs []*sv) {
	n := len(ks)
	if n <= LowLevel {
		// 串行写入即可
		for i := 0; i < n; i++ {
			s.putByLock(ks[i], svs[i])
		}
		return
	}
	// 首先进行分组
	shards := s.group(ks, svs)
	// 两种写入模型，如果数量较少则串行同步分组写入，否则并行同步分组写入
	if n <= HighLevel {
		s.putsBySerial(shards)
		return
	}
	s.putsByParallel(shards)
}

func (s *ShardSet) group(ks []string, svs []*sv) [][]*ksv {
	// 首先进行分组
	shards := make([][]*ksv, s.shardNum)
	for i := 0; i < len(ks); i++ {
		k := ks[i]
		v := svs[i]
		sn := shardNum(ks[i], s.shardNum)
		if shards[sn] != nil {
			shards[sn] = append(shards[sn], &ksv{
				k: k,
				v: v,
			})
		} else {
			// 设置容量
			shard := make([]*ksv, 0, 16)
			shard[0] = &ksv{
				k: k,
				v: v,
			}
			shards[sn] = shard
		}
	}
	return shards
}

func (s *ShardSet) putsBySerial(shards [][]*ksv) {
	// 分组完成后将分组的内容加入到具体分片中
	for i, ksvs := range shards {
		if ksvs != nil {
			s.shards[i].puts(ksvs)
		}
	}
}

func (s *ShardSet) putsByParallel(shards [][]*ksv) {
	var wg sync.WaitGroup
	wg.Add(len(shards))
	// 分组完成后将分组的内容加入到具体分片中
	for i, ksvs := range shards {
		if ksvs != nil {
			go func(i int, ksvs []*ksv) {
				s.shards[i].puts(ksvs)
				wg.Done()
			}(i, ksvs)
			continue
		}
		wg.Done()
	}
	wg.Wait()
}

type Shard struct {
	sync.RWMutex
	m map[string]*sv
}

func newShard() *Shard {
	return &Shard{
		m: make(map[string]*sv, KeysPerShardDefault),
	}
}

func (s *Shard) getByLock(k string) (*sv, bool) {
	s.RLock()
	defer s.RUnlock()
	return s.get(k)
}

func (s *Shard) putByLock(k string, kv *sv) {
	s.Lock()
	defer s.Unlock()
	s.put(k, kv)
}

func (s *Shard) puts(ksvs []*ksv) {
	s.Lock()
	defer s.Unlock()
	for i := 0; i < len(ksvs); i++ {
		s.put(ksvs[i].k, ksvs[i].v)
	}
}

func (s *Shard) putsByLock(news *Shard) {
	s.Lock()
	defer s.Unlock()
	for k, v := range news.m {
		s.put(k, v)
	}
}

func (s *Shard) get(k string) (*sv, bool) {
	if kv, exist := s.m[k]; exist {
		return kv, exist
	}
	return nil, false
}

func (s *Shard) put(k string, sv *sv) {
	s.m[k] = sv
}

type ksv struct {
	k string
	v *sv
}

func shardNum(key string, shardedNum int) int {
	return int(fnv32(key) % uint32(shardedNum))
}

func fnv32(key string) uint32 {
	hash := uint32(2166136261)
	keyLength := len(key)
	for i := 0; i < keyLength; i++ {
		hash *= Prime32
		hash ^= uint32(key[i])
	}
	return hash
}
