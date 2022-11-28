/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package scheduler

import (
	"math/big"
	"sync"

	"github.com/holiman/uint256"
)

const (
	// AdjustWindowSize adjust window size
	AdjustWindowSize = 64
	// MinAdjustTimes min adjust times
	MinAdjustTimes = 2
	// MinPoolCapacity min pool capacity
	MinPoolCapacity = 2
	// BaseConflictRate base conflict rate
	BaseConflictRate = 0.05
	// TopConflictRate top conflict rate
	TopConflictRate = 0.2
	// DescendCoefficient descend coefficient
	DescendCoefficient = 0.25
	// AscendCoefficient ascend coefficient
	AscendCoefficient = 3
)

// TxExecType tx exec type
type TxExecType int

const (
	// ConflictTx conflict tx value 0
	ConflictTx TxExecType = iota
	// NormalTx normail tx value 1
	NormalTx
)

// ConflictsBitWindow holds a bitWindow to adjust goroutine pool size for runtime.
type ConflictsBitWindow struct {
	bitWindow         *uint256.Int
	bitWindowCapacity int
	maxPoolCapacity   int
	conflictsNum      int
	execCount         int
	mu                sync.Mutex
}

// NewConflictsBitWindow returns an empty queue.
func NewConflictsBitWindow(txBatchSize int) *ConflictsBitWindow {
	return &ConflictsBitWindow{
		bitWindow:         uint256.NewInt(0),
		bitWindowCapacity: AdjustWindowSize,
		maxPoolCapacity:   txBatchSize,
	}
}

// Enqueue puts the given value v at the tail of the queue.
func (q *ConflictsBitWindow) Enqueue(v TxExecType, currPoolCapacity int) int {
	q.mu.Lock()
	defer q.mu.Unlock()
	v256 := uint256.NewInt(0)
	if v == ConflictTx {
		v256, _ = uint256.FromBig(big.NewInt(1))
		q.conflictsNum++
	}

	flag, _ := uint256.FromBig(big.NewInt(1))
	if flag.And(flag.Lsh(flag, uint(q.bitWindowCapacity-1)), q.bitWindow).Cmp(uint256.NewInt(0)) > 0 {
		q.conflictsNum--
	}

	q.bitWindow.Or(q.bitWindow.Lsh(q.bitWindow, 1), v256)
	q.execCount++
	if q.execCount%q.bitWindowCapacity == 0 {
		return q.getNewPoolCapacity(currPoolCapacity)
	}
	return -1
}

// getNewPoolCapacity update and return the pool capacity in get stage.
func (q *ConflictsBitWindow) getNewPoolCapacity(currPoolCapacity int) int {
	conflictsRate := q.getConflictsRate()
	targetCapacity := -1
	if conflictsRate < BaseConflictRate {
		targetCapacity = int(float64(currPoolCapacity) * AscendCoefficient)
	} else if conflictsRate > TopConflictRate {
		targetCapacity = int(float64(currPoolCapacity) * DescendCoefficient)
	}
	if targetCapacity > q.maxPoolCapacity {
		return q.maxPoolCapacity
	}
	if targetCapacity < MinPoolCapacity {
		return MinPoolCapacity
	}
	return targetCapacity
}

// getConflictsRate return the conflicts rate in slide window.
func (q *ConflictsBitWindow) getConflictsRate() float64 {
	return float64(q.conflictsNum) / float64(q.bitWindowCapacity)
}

// setMaxPoolCapacity set max pool capacity
func (q *ConflictsBitWindow) setMaxPoolCapacity(maxPoolCapacity int) {
	q.mu.Lock()
	defer q.mu.Unlock()
	// if max pool capcacity less than min pool capacity, set the max pool capacity equal min pool capacity
	if maxPoolCapacity < MinPoolCapacity {
		maxPoolCapacity = MinPoolCapacity
	}
	q.maxPoolCapacity = maxPoolCapacity
}
