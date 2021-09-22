/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sync

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"chainmaker.org/chainmaker/logger/v2"

	"github.com/Workiva/go-datastructures/queue"
)

type MockHandler struct {
	receiveItems []queue.Item
}

func NewMockHandler() *MockHandler {
	return &MockHandler{receiveItems: make([]queue.Item, 0, 10)}
}

func (mock *MockHandler) handler(item queue.Item) (queue.Item, error) {
	mock.receiveItems = append(mock.receiveItems, item)
	return nil, nil
}
func (mock *MockHandler) getState() string {
	return ""
}

func TestAddTask(t *testing.T) {
	mock := NewMockHandler()
	routine := NewRoutine("mock", mock.handler, mock.getState, logger.GetLogger("mock"))
	require.NoError(t, routine.begin())

	require.NoError(t, routine.addTask(NodeStatusMsg{from: "node1"}))
	require.NoError(t, routine.addTask(NodeStatusMsg{from: "node2"}))
	require.NoError(t, routine.addTask(NodeStatusMsg{from: "node3"}))

	for i := range mock.receiveItems {
		item := mock.receiveItems[i].(NodeStatusMsg)
		require.EqualValues(t, fmt.Sprintf("node%d", i+1), item.from)
	}
	routine.end()
}
