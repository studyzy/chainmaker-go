/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package verifier

import "sync"

// reentrantLocks, avoid the same block hash
type reentrantLocks struct {
	reentrantLocks map[string]interface{}
	mu             sync.Mutex
}

func (l *reentrantLocks) lock(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.reentrantLocks[key] == nil {
		l.reentrantLocks[key] = LOCKED
		return true
	}
	return false
}

func (l *reentrantLocks) unlock(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.reentrantLocks, key)
	return true
}
