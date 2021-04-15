/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0

wasi: WebAssembly System Interface
*/
package wasi

import "sync"

var (
	currentDbs    = make(map[string]string, 0)
	currentDbSync = &sync.RWMutex{}
)

func getCurrentDb(chain string) string {
	currentDbSync.RLock()
	currentDbSync.RUnlock()
	return currentDbs[chain]
}

func setCurrentDb(chain string, dbName string) {
	currentDbSync.Lock()
	defer currentDbSync.Unlock()
	currentDbs[chain] = dbName
}
