/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package consensus

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"chainmaker.org/chainmaker-go/consensus/tbft"
	"chainmaker.org/chainmaker/common/v2/msgbus"
	"chainmaker.org/chainmaker/localconf/v2"
	configpb "chainmaker.org/chainmaker/pb-go/v2/config"
	consensuspb "chainmaker.org/chainmaker/pb-go/v2/consensus"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/protocol/v2/mock"

	"github.com/golang/mock/gomock"
)

const (
	chainID     = "test"
	id          = "QmQZn3pZCcuEf34FSvucqkvVJEvfzpNjQTk17HS6CYMR35"
	org1Id      = "wx-org1"
	org1Address = "/ip4/192.168.2.2/tcp/6666/p2p/QmQZn3pZCcuEf34FSvucqkvVJEvfzpNjQTk17HS6CYMR35"
)

var (
	nodeList = []string{org1Address}
)

func TestNewConsensusEngine(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	prePath := localconf.ChainMakerConfig.GetStorePath()
	defer func() {
		localconf.ChainMakerConfig.StorageConfig["store_path"] = prePath
	}()
	localconf.ChainMakerConfig.StorageConfig["store_path"] = filepath.Join(os.TempDir(), fmt.Sprintf("%d", time.Now().Nanosecond()))

	signer := mock.NewMockSigningMember(ctrl)
	ledgerCache := mock.NewMockLedgerCache(ctrl)
	ledgerCache.EXPECT().CurrentHeight().AnyTimes().Return(uint64(1), nil)
	chainConf := mock.NewMockChainConf(ctrl)
	chainConf.EXPECT().ChainConfig().AnyTimes().Return(&configpb.ChainConfig{
		Consensus: &configpb.ConsensusConfig{
			Type: consensuspb.ConsensusType_TBFT,
			Nodes: []*configpb.OrgConfig{
				{
					OrgId:  org1Id,
					NodeId: []string{id},
				},
			},
		},
	})
	dbHandle := mock.NewMockDBHandle(ctrl)
	dbHandle.EXPECT().Get(gomock.Any()).AnyTimes().Return(nil, nil)

	type args struct {
		consensusType  consensuspb.ConsensusType
		chainID        string
		id             string
		nodeList       []string
		signer         protocol.SigningMember
		ac             protocol.AccessControlProvider
		dbHandle       protocol.DBHandle
		ledgerCache    protocol.LedgerCache
		proposalCache  protocol.ProposalCache
		blockVerifier  protocol.BlockVerifier
		blockCommitter protocol.BlockCommitter
		netService     protocol.NetService
		msgBus         msgbus.MessageBus
		chainConf      protocol.ChainConf
		store          protocol.BlockchainStore
	}
	tests := []struct {
		name    string
		args    args
		want    protocol.ConsensusEngine
		wantErr bool
	}{
		{"new TBFT consensus engine",
			args{
				consensusType: consensuspb.ConsensusType_TBFT,
				chainID:       chainID,
				id:            id,
				nodeList:      nodeList,
				signer:        signer,
				ledgerCache:   ledgerCache,
				dbHandle:      dbHandle,
				chainConf:     chainConf,
			},
			&tbft.ConsensusTBFTImpl{},
			false,
		},
		// {"new POW consensus engine", args{pb.ConsensusType_POW}, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var factory Factory
			got, err := factory.NewConsensusEngine(
				tt.args.consensusType,
				tt.args.chainID,
				tt.args.id,
				tt.args.nodeList,
				tt.args.signer,
				tt.args.ac,
				tt.args.dbHandle,
				tt.args.ledgerCache,
				tt.args.proposalCache,
				tt.args.blockVerifier,
				tt.args.blockCommitter,
				tt.args.netService,
				tt.args.msgBus,
				tt.args.chainConf,
				tt.args.store,
				nil,
			)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewConsensusEngine() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if reflect.TypeOf(got) != reflect.TypeOf(tt.want) {
				t.Errorf("NewConsensusEngine() = %v, want %v", got, tt.want)
			}
		})
	}
}
