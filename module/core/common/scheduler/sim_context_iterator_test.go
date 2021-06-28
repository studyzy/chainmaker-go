/*
 Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
   SPDX-License-Identifier: Apache-2.0
*/

package scheduler

import (
    "chainmaker.org/chainmaker-go/protocol"
    "github.com/stretchr/testify/require"
    "testing"
)

func TestSimContextIteratorNextValue(t *testing.T) {
    simContextEmptyIterator := NewSimContextIterator(makeEmptyWSetIterator(), makeEmptyWSetIterator())
    require.False(t, simContextEmptyIterator.Next())
    val, err := simContextEmptyIterator.Value()
    require.Nil(t, err)
    require.Nil(t, val)

    i := 0
    _, vals := makeStringKeyMap()
    simContextMockDbIterator := NewSimContextIterator(makeEmptyWSetIterator(), makeMockDbIterator())
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
    simContextMockWSetIterator := NewSimContextIterator(makeMockWSetIterator(), makeEmptyWSetIterator())
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
    simContextIterator := NewSimContextIterator(makeMockWSetIterator(), makeMockDbIterator())
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