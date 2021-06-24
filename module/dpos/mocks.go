package dpos

import (
	"chainmaker.org/chainmaker/protocol/mock"
	configpb "chainmaker.org/chainmaker/pb-go/config"
	"chainmaker.org/chainmaker/pb-go/consensus"
	"chainmaker.org/chainmaker/protocol"
	"github.com/golang/mock/gomock"
)

func newMockBlockChainStore(ctrl *gomock.Controller) protocol.BlockchainStore {
	mockStore := mock.NewMockBlockchainStore(ctrl)
	mockStore.EXPECT().ReadObject(gomock.Any(), gomock.Any()).DoAndReturn(
		func(contractName string, key []byte) ([]byte, error) {
			return nil, nil
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
