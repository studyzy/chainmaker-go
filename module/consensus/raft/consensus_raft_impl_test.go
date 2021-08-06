/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package raft

import (
	"reflect"
	"testing"

	"chainmaker.org/chainmaker-go/mock"
	configpb "chainmaker.org/chainmaker-go/pb/protogo/config"
	consensuspb "chainmaker.org/chainmaker-go/pb/protogo/consensus"
	"github.com/golang/mock/gomock"
	"github.com/jfcg/sorty"
	"go.uber.org/zap"
)

func Test_computeUpdatedNodes(t *testing.T) {
	type args struct {
		oldSet []uint64
		newSet []uint64
	}
	tests := []struct {
		name        string
		args        args
		wantRemoved []uint64
		wantAdded   []uint64
	}{
		{
			"no change",
			args{
				oldSet: []uint64{1, 2, 3},
				newSet: []uint64{1, 2, 3},
			},
			[]uint64{},
			[]uint64{},
		},
		{
			"add 2 nodes",
			args{
				oldSet: []uint64{1, 2, 3},
				newSet: []uint64{1, 2, 3, 4, 5},
			},
			[]uint64{},
			[]uint64{4, 5},
		},
		{
			"remove 2 nodes",
			args{
				oldSet: []uint64{1, 2, 3},
				newSet: []uint64{1},
			},
			[]uint64{2, 3},
			[]uint64{},
		},
		{
			"add 2 nodes and remove 2 nodes",
			args{
				oldSet: []uint64{1, 2, 3},
				newSet: []uint64{1, 4, 5},
			},
			[]uint64{2, 3},
			[]uint64{4, 5},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRemoved, gotAdded := computeUpdatedNodes(tt.args.oldSet, tt.args.newSet)
			if !reflect.DeepEqual(gotRemoved, tt.wantRemoved) {
				t.Errorf("computeUpdatedNodes() gotRemoved = %v, want %v", gotRemoved, tt.wantRemoved)
			}
			if !reflect.DeepEqual(gotAdded, tt.wantAdded) {
				t.Errorf("computeUpdatedNodes() gotAdded = %v, want %v", gotAdded, tt.wantAdded)
			}
		})
	}
}

func sortU8(input ...uint64) []uint64 {
	output := make([]uint64, len(input))
	copy(output, input)
	sorty.SortU8(output)
	return output
}

func TestConsensusRaftImpl_getPeersFromChainConf(t *testing.T) {
	logger := NewLogger(zap.L().Sugar())
	config := &configpb.ChainConfig{
		Consensus: &configpb.ConsensusConfig{
			Type:  consensuspb.ConsensusType_RAFT,
			Nodes: nil,
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	chainConf := mock.NewMockChainConf(ctrl)
	chainConf.EXPECT().ChainConfig().AnyTimes().Return(config)

	const (
		nodeId1 = "QmcQHCuAXaFkbcsPUj7e37hXXfZ9DdN7bozseo5oX4qiC4"
		nodeId2 = "QmeyNRs2DwWjcHTpcVHoUSaDAAif4VQZ2wQDQAUNDP33gH"
		nodeId3 = "QmXf6mnQDBR9aHauRmViKzSuZgpumkn7x6rNxw1oqqRr45"
	)

	type fields struct {
		Nodes []*configpb.OrgConfig
	}
	tests := []struct {
		name   string
		fields fields
		want   []uint64
	}{
		{
			"1 org 1 node",
			fields{Nodes: []*configpb.OrgConfig{
				{
					NodeId: []string{nodeId1},
				},
			}},
			[]uint64{computeRaftIdFromNodeId(nodeId1)},
		},
		{
			"3 org 1 node",
			fields{Nodes: []*configpb.OrgConfig{
				{
					NodeId: []string{nodeId1},
				},
				{
					NodeId: []string{nodeId2},
				},
				{
					NodeId: []string{nodeId3},
				},
			}},
			sortU8(computeRaftIdFromNodeId(nodeId1), computeRaftIdFromNodeId(nodeId2), computeRaftIdFromNodeId(nodeId3)),
		},
		{
			"1 org 2 node",
			fields{Nodes: []*configpb.OrgConfig{
				{
					NodeId: []string{nodeId1, nodeId2, nodeId3},
				},
			}},
			sortU8(computeRaftIdFromNodeId(nodeId1), computeRaftIdFromNodeId(nodeId2), computeRaftIdFromNodeId(nodeId3)),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.Consensus.Nodes = tt.fields.Nodes
			consensus := &ConsensusRaftImpl{
				logger:    logger,
				chainConf: chainConf,
			}
			if got := consensus.getPeersFromChainConf(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ConsensusRaftImpl.getPeersFromChainConf() = %v, want %v", got, tt.want)
			}
		})
	}
}
