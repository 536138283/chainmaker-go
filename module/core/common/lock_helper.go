/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import "sync"

// LOCKED mark
const LOCKED = "LOCKED"

// ReentrantLocks avoid the same block hash
type ReentrantLocks struct {
	ReentrantLocks map[string]interface{}
	Mu             sync.Mutex
}

// Lock used by ReentrantLocks
func (l *ReentrantLocks) Lock(key string) bool {
	l.Mu.Lock()
	defer l.Mu.Unlock()
	if l.ReentrantLocks[key] == nil {
		l.ReentrantLocks[key] = LOCKED
		return true
	}
	return false
}

// Unlock used by ReentrantLocks
func (l *ReentrantLocks) Unlock(key string) bool {
	l.Mu.Lock()
	defer l.Mu.Unlock()
	delete(l.ReentrantLocks, key)
	return true
}
