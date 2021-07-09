/*
 Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
   SPDX-License-Identifier: Apache-2.0
*/

package scheduler

import (
    commonpb "chainmaker.org/chainmaker-go/pb/protogo/common"
    "chainmaker.org/chainmaker-go/protocol"
    "github.com/stretchr/testify/require"
    "testing"
)

func TestSimContextIteratorNextValue(t *testing.T) {
    transactPayload := &commonpb.TransactPayload{
        ContractName: "test",
        Method:       "test",
        Parameters:   nil,
    }
    payload, err := transactPayload.Marshal()
    require.Nil(t, err)
    simContext := &txSimContextImpl{
        txReadKeyMap:     make(map[string]*commonpb.TxRead, 8),
        txWriteKeyMap:    make(map[string]*commonpb.TxWrite, 8),
        sqlRowCache:      make(map[int32]protocol.SqlRows, 0),
        kvRowCache:       make(map[int32]protocol.StateIterator, 0),
        txWriteKeySql:    make([]*commonpb.TxWrite, 0),
        txWriteKeyDdlSql: make([]*commonpb.TxWrite, 0),
        tx: &commonpb.Transaction{
            Header: &commonpb.TxHeader{
                ChainId:        "chain1",
                TxId:           "12345678",
                TxType:         commonpb.TxType_INVOKE_USER_CONTRACT,
                Timestamp:      0,
                ExpirationTime: 0,
            },
            RequestPayload:   payload,
            RequestSignature: nil,
            Result:           nil,
        },
        gasUsed:      0,
        currentDepth: 0,
        hisResult:    make([]*callContractResult, 0),
    }
    simContextEmptyIterator := NewSimContextIterator(simContext, makeEmptyWSetIterator(), makeEmptyWSetIterator())
    require.False(t, simContextEmptyIterator.Next())
    val, err := simContextEmptyIterator.Value()
    require.Nil(t, err)
    require.Nil(t, val)

    i := 0
    _, vals := makeStringKeyMap()
    simContextMockDbIterator := NewSimContextIterator(simContext, makeEmptyWSetIterator(), makeMockDbIterator())
    for {
        if !simContextMockDbIterator.Next() {
            break
        }
        val, err := simContextMockDbIterator.Value()
        require.Nil(t, err)
        require.Equal(t, vals[i], val)
        i++
    }

    i = 0
    simContextMockWSetIterator := NewSimContextIterator(simContext, makeMockWSetIterator(), makeEmptyWSetIterator())
    for {
        if !simContextMockWSetIterator.Next() {
            break
        }
        val, err := simContextMockWSetIterator.Value()
        require.Nil(t, err)
        require.Equal(t, vals[i], val)
        i++
    }

    i = 0
    simContextIterator := NewSimContextIterator(simContext, makeMockWSetIterator(), makeMockDbIterator())
    for {
        if !simContextIterator.Next() {
            break
        }
        val, err := simContextIterator.Value()
        require.Nil(t, err)
        require.Equal(t, vals[i], val)
        i++
    }
}

func makeEmptyWSetIterator() protocol.StateIterator {
    return NewWsetIterator(make(map[string]interface{}))
}

func makeMockWSetIterator() protocol.StateIterator {
    stringKeyMap, _ := makeStringKeyMap()
    return NewWsetIterator(stringKeyMap)
}

func makeMockDbIterator() protocol.StateIterator {
    stringKeyMap, _ := makeStringKeyMap()
    return NewWsetIterator(stringKeyMap)
}
