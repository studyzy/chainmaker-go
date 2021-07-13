/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dpos

import (
	"bytes"
	"encoding/binary"
	"fmt"

	native "chainmaker.org/chainmaker-go/vm/native/dposmgr"
	commonpb "chainmaker.org/chainmaker/pb-go/common"
	configpb "chainmaker.org/chainmaker/pb-go/config"
	"chainmaker.org/chainmaker/pb-go/consensus"
	"chainmaker.org/chainmaker/protocol"
	"chainmaker.org/chainmaker/protocol/mock"

	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/proto"
)

var (
	testAddr        = "addr1-balance"
	testAddrBalance = 9999
)

func newMockBlockChainStore(ctrl *gomock.Controller) protocol.BlockchainStore {
	mockStore := mock.NewMockBlockchainStore(ctrl)
	mockStore.EXPECT().ReadObject(gomock.Any(), gomock.Any()).DoAndReturn(
		func(contractName string, key []byte) ([]byte, error) {
			if bytes.Equal(key, []byte(native.BalanceKey(testAddr))) {
				return []byte(fmt.Sprintf("%d", testAddrBalance)), nil
			}
			if bytes.Equal(key, []byte(native.KeyMinSelfDelegation)) {
				return []byte("200000"), nil
			}
			if bytes.Equal(key, []byte(native.BalanceKey(native.StakeContractAddr()))) {
				return []byte("10000"), nil
			}
			if bytes.Equal(key, []byte(native.KeyCurrentEpoch)) {
				epoch := &commonpb.Epoch{NextEpochCreateHeight: 100}
				bz, err := proto.Marshal(epoch)
				return bz, err
			}
			if bytes.Equal(key, []byte(native.KeyEpochBlockNumber)) {
				bz := make([]byte, 8)
				binary.BigEndian.PutUint64(bz, 4)
				return bz, nil
			}
			return nil, nil
		}).AnyTimes()

	iter := mock.NewMockStateIterator(ctrl)
	iter.EXPECT().Release().AnyTimes()
	iter.EXPECT().Value().AnyTimes()
	iter.EXPECT().Next().AnyTimes()
	mockStore.EXPECT().SelectObject(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(contractName string, startKey []byte, limit []byte) (protocol.StateIterator, error) {
			return iter, nil
		}).AnyTimes()
	return mockStore
}

func newMockChainConf(ctrl *gomock.Controller) protocol.ChainConf {
	mockConf := mock.NewMockChainConf(ctrl)
	mockConf.EXPECT().ChainConfig().Return(&configpb.ChainConfig{
		ChainId: "test_chain",
		Consensus: &configpb.ConsensusConfig{
			Type: consensus.ConsensusType_DPOS,
		},
	}).AnyTimes()
	return mockConf
}
