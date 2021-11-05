/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dpos

import (
	"bytes"
	"encoding/binary"
	"fmt"

	configPb "chainmaker.org/chainmaker/pb-go/v2/config"
	"chainmaker.org/chainmaker/pb-go/v2/consensus"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/protocol/v2/mock"
	native "chainmaker.org/chainmaker/vm-native/v2/dposmgr"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/mock/gomock"
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
				epoch := &syscontract.Epoch{NextEpochCreateHeight: 100}
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
	mockConf.EXPECT().ChainConfig().Return(&configPb.ChainConfig{
		ChainId: "test_chain",
		Consensus: &configPb.ConsensusConfig{
			Type: consensus.ConsensusType_DPOS,
		},
	}).AnyTimes()
	return mockConf
}
