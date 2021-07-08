/*
Copyright (C) BABEC. All rights reserved.
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
	simContext     protocol.TxSimContext
	//log            protocol.Logger
}

func NewSimContextIterator(simContext protocol.TxSimContext, wsetIter, dbIter protocol.StateIterator) *SimContextIterator {
	return &SimContextIterator{
		wsetValueCache: nil,
		dbValueCache:   nil,
		finalValue:     nil,
		wsetIter:       wsetIter,
		dbIter:         dbIter,
		simContext:     simContext,
	}
}

// Next move the iter to next and return is there value in next iter
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

// Value return the value of current iter
func (sci *SimContextIterator) Value() (*store.KV, error) {
	if sci.finalValue == nil {
		return nil, nil
	}
	contractName, err := sci.simContext.GetTx().GetContractName()
	if err != nil {
		return nil, err
	}
	sci.simContext.PutIntoReadSet(contractName, sci.finalValue.Key, sci.finalValue.Value)

	return sci.finalValue, nil
}

// Release release the iterator
func (sci *SimContextIterator) Release() {
	sci.wsetIter.Release()
	sci.dbIter.Release()
}
