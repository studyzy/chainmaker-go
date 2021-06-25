/*
 Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
   SPDX-License-Identifier: Apache-2.0
*/

package scheduler

import (
    "chainmaker.org/chainmaker-go/common/sortedmap"
    "chainmaker.org/chainmaker-go/pb/protogo/store"
    "fmt"
)

type WsetIterator struct {
    stringKeySortedMap *sortedmap.StringKeySortedMap
    //log protocol.Logger
}

func NewWsetIterator(wsets map[string]interface{}) *WsetIterator {
    return &WsetIterator{
        stringKeySortedMap: sortedmap.NewStringKeySortedMapWithInterfaceData(wsets),
        //log: logger.GetLoggerByChain(logger.MODULE_CORE, chainConf.ChainConfig().ChainId),
    }
}

func (wi *WsetIterator) Next() bool {
    return wi.stringKeySortedMap.Length() > 0
}

func (wi *WsetIterator) Value() (*store.KV, error) {
    var kv *store.KV
    var keyStr string
    var ok bool
    wi.stringKeySortedMap.Range(func(key string, val interface{}) (isContinue bool) {
        keyStr = key
        kv, ok = val.(*store.KV)
        return false
    })
    if !ok {
        return nil, fmt.Errorf("get value from wsetIterator failed, value type error")
    }
    wi.stringKeySortedMap.Remove(keyStr)
    return kv, nil
}

func (wi *WsetIterator) Release() {
    // do nothing
}
