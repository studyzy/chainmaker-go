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
    //log            protocol.Logger
}

func NewSimContextIterator(wsetIter, dbIter protocol.StateIterator) *SimContextIterator {
    return &SimContextIterator{
        wsetValueCache: nil,
        dbValueCache:   nil,
        wsetIter:       wsetIter,
        dbIter:         dbIter,
        //log:            logger.GetLoggerByChain(logger.MODULE_CORE, chainConf.ChainConfig().ChainId),
    }
}

func (sci *SimContextIterator) Next() bool {
    if sci.wsetValueCache != nil || sci.dbValueCache != nil {
        return true
    }
    if sci.wsetIter.Next() {
        value, err := sci.wsetIter.Value()
        if err != nil {
            //sci.log.Error("get value from wsetIter failed, ", err)
            return false
        }
        sci.wsetValueCache = value
        return true
    }
    if sci.dbIter.Next() {
        value, err := sci.dbIter.Value()
        if err != nil {
            //sci.log.Error("get value from dbIter failed, ", err)
            return false
        }
        sci.dbValueCache = value
        return true
    }

    return false
}

func (sci *SimContextIterator) Value() (*store.KV, error) {
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
        return resultCache, nil
    }
    if sci.wsetValueCache != nil {
        if !sci.dbIter.Next() {
            resultCache = sci.wsetValueCache
            sci.wsetValueCache = nil
            return resultCache, nil
        }
        dbValue, err := sci.dbIter.Value()
        if err != nil {
            //sci.log.Error("get value from dbIter failed, ", err)
            return nil, err
        }
        if string(sci.wsetValueCache.Key) == string(dbValue.Key) {
            resultCache = sci.wsetValueCache
            sci.wsetValueCache = nil
        } else if string(sci.wsetValueCache.Key) < string(dbValue.Key) {
            sci.dbValueCache = dbValue
            resultCache = sci.wsetValueCache
            sci.wsetValueCache = nil
        } else {
            resultCache = dbValue
        }
        return resultCache, nil
    }
    if sci.dbValueCache != nil {
        if !sci.wsetIter.Next() {
            resultCache = sci.dbValueCache
            sci.dbValueCache = nil
            return resultCache, nil
        }
        wsetValue, err := sci.wsetIter.Value()
        if err != nil {
            //sci.log.Error("get value from wsetIter failed, ", err)
            return nil, err
        }
        if string(sci.dbValueCache.Key) == string(wsetValue.Key) {
            sci.dbValueCache = nil
            resultCache = wsetValue
        } else if string(sci.dbValueCache.Key) < string(wsetValue.Key) {
            sci.wsetValueCache = wsetValue
            resultCache = sci.dbValueCache
            sci.dbValueCache = nil
        } else {
            resultCache = wsetValue
        }
        return resultCache, nil
    }

    var err error
    var wsetValue *store.KV = nil
    var dbValue *store.KV = nil
    if sci.wsetIter.Next() {
        wsetValue, err = sci.wsetIter.Value()
        if err != nil {
            //sci.log.Error("get value from wsetIter failed, ", err)
            return nil, err
        }
    }
    if sci.dbIter.Next() {
        dbValue, err = sci.dbIter.Value()
        if err != nil {
            //sci.log.Error("get value from dbIter failed, ", err)
            return nil, err
        }
    }
    if wsetValue != nil && dbValue != nil {
        if string(wsetValue.Key) == string(dbValue.Key) {
            return wsetValue, nil
        }
        if string(wsetValue.Key) < string(dbValue.Key) {
            sci.dbValueCache = dbValue
            return wsetValue, nil
        }
        sci.wsetValueCache = wsetValue
        return dbValue, nil
    }
    if wsetValue != nil {
        return wsetValue, nil
    }
    return dbValue, nil
}

func (sci *SimContextIterator) Release() {
    sci.wsetIter.Release()
    sci.dbIter.Release()
}
