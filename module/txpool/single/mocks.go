/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package single

import (
	"chainmaker.org/chainmaker-go/common/msgbus"
	"chainmaker.org/chainmaker-go/mock"
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/protocol"

	"github.com/golang/mock/gomock"
)

type mockBlockChainStore struct {
	txs   map[string]*commonPb.Transaction
	store protocol.BlockchainStore
}

func newMockBlockChainStore(ctrl *gomock.Controller) *mockBlockChainStore {
	store := mock.NewMockBlockchainStore(ctrl)
	mockStore := &mockBlockChainStore{store: store, txs: make(map[string]*commonPb.Transaction)}

	store.EXPECT().GetTx(gomock.Any()).DoAndReturn(func(txId string) (*commonPb.Transaction, error) {
		tx := mockStore.txs[txId]
		return tx, nil
	}).AnyTimes()
	store.EXPECT().TxExists(gomock.Any()).DoAndReturn(func(txId string) (bool, error) {
		_, exist := mockStore.txs[txId]
		return exist, nil
	}).AnyTimes()

	return mockStore
}

func newMockMessageBus(ctrl *gomock.Controller) msgbus.MessageBus {
	mockMsgBus := mock.NewMockMessageBus(ctrl)
	mockMsgBus.EXPECT().Register(gomock.Any(), gomock.Any()).AnyTimes()
	mockMsgBus.EXPECT().Publish(gomock.Any(), gomock.Any()).AnyTimes()
	return mockMsgBus
}

func newMockAccessControlProvider(ctrl *gomock.Controller) protocol.AccessControlProvider {
	mockAc := mock.NewMockAccessControlProvider(ctrl)
	return mockAc
}

func newMockNet(ctrl *gomock.Controller) protocol.NetService {
	mockNet := mock.NewMockNetService(ctrl)
	return mockNet
}
