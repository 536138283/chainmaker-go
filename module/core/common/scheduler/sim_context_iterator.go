/*
 Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
   SPDX-License-Identifier: Apache-2.0
*/

package scheduler

import (
	"chainmaker.org/chainmaker-go/pb/protogo/store"
	"chainmaker.org/chainmaker-go/protocol"
)

type SimContextIterator struct {
	wsetValueCache *store.KV
	dbValueCache   *store.KV
	wsetIter       protocol.StateIterator
	dbIter         protocol.StateIterator
	finalValue     *store.KV
	//log            protocol.Logger
}

func NewSimContextIterator(wsetIter, dbIter protocol.StateIterator) *SimContextIterator {
	return &SimContextIterator{
		wsetValueCache: nil,
		dbValueCache:   nil,
		finalValue:     nil,
		wsetIter:       wsetIter,
		dbIter:         dbIter,
		//log:            logger.GetLoggerByChain(logger.MODULE_CORE, chainConf.ChainConfig().ChainId),
	}
}

func (sci *SimContextIterator) Next() bool {
	if sci.wsetValueCache == nil {
		if sci.wsetIter.Next() {
			value, err := sci.wsetIter.Value()
			if err != nil {
				//sci.log.Error("get value from wsetIter failed, ", err)
				return false
			}
			sci.wsetValueCache = value
		}
	}
	if sci.dbValueCache == nil {
		if sci.dbIter.Next() {
			value, err := sci.dbIter.Value()
			if err != nil {
				//sci.log.Error("get value from dbIter failed, ", err)
				return false
			}
			sci.dbValueCache = value
		}
	}
	if sci.wsetValueCache == nil && sci.dbValueCache == nil {
		return false
	}

	var resultCache *store.KV
	if sci.wsetValueCache != nil && sci.dbValueCache != nil {
		if string(sci.wsetValueCache.Key) == string(sci.dbValueCache.Key) {
			sci.dbValueCache = nil
			resultCache = sci.wsetValueCache
			sci.wsetValueCache = nil
		} else if string(sci.wsetValueCache.Key) < string(sci.dbValueCache.Key) {
			resultCache = sci.wsetValueCache
			sci.wsetValueCache = nil
		} else {
			resultCache = sci.dbValueCache
			sci.dbValueCache = nil
		}
	} else if sci.wsetValueCache != nil {
		resultCache = sci.wsetValueCache
		sci.wsetValueCache = nil
	} else if sci.dbValueCache != nil {
		resultCache = sci.dbValueCache
		sci.dbValueCache = nil
	}
	sci.finalValue = resultCache
	return true
}

func (sci *SimContextIterator) Value() (*store.KV, error) {
	return sci.finalValue, nil
}

func (sci *SimContextIterator) Release() {
	sci.wsetIter.Release()
	sci.dbIter.Release()
}
