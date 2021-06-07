/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import "sync"

const LOCKED = "LOCKED" // LOCKED mark

// reentrantLocks, avoid the same block hash
type ReentrantLocks struct {
	ReentrantLocks map[string]interface{}
	Mu             sync.Mutex
}

func (l *ReentrantLocks) Lock(key string) bool {
	l.Mu.Lock()
	defer l.Mu.Unlock()
	if l.ReentrantLocks[key] == nil {
		l.ReentrantLocks[key] = LOCKED
		return true
	}
	return false
}

func (l *ReentrantLocks) Unlock(key string) bool {
	l.Mu.Lock()
	defer l.Mu.Unlock()
	delete(l.ReentrantLocks, key)
	return true
}
