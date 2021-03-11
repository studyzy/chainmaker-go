/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tbft

import (
	"crypto/rand"
	"crypto/sha1"
	"github.com/golang/mock/gomock"
	"reflect"
	"testing"

	"chainmaker.org/chainmaker-go/logger"
	"chainmaker.org/chainmaker-go/mock"
	commonpb "chainmaker.org/chainmaker-go/pb/protogo/common"
	configpb "chainmaker.org/chainmaker-go/pb/protogo/config"
	consensuspb "chainmaker.org/chainmaker-go/pb/protogo/consensus"
	tbftpb "chainmaker.org/chainmaker-go/pb/protogo/consensus/tbft"
	"chainmaker.org/chainmaker-go/protocol"
	"github.com/stretchr/testify/require"
)

const (
	chainId     = "test"
	org1Id      = "wx-org1"
	org2Id      = "wx-org2"
	org3Id      = "wx-org3"
	org4Id      = "wx-org4"
	org1Address = "/ip4/192.168.2.2/tcp/6666/p2p/QmQZn3pZCcuEf34FSvucqkvVJEvfzpNjQTk17HS6CYMR35"
	org2Address = "/ip4/192.168.2.3/tcp/6666/p2p/QmeRZz3AjhzydkzpiuuSAtmqt8mU8XcRH2hynQN4tLgYg6"
	org3Address = "/ip4/192.168.2.4/tcp/6666/p2p/QmTSMcqwp4X6oPP5WrNpsMpotQMSGcxVshkGLJUhCrqGbu"
	org4Address = "/ip4/192.168.2.5/tcp/6666/p2p/QmUryDgjNoxfMXHdDRFZ5Pe55R1vxTPA3ZgCteHze2ET27"
	org1NetId   = "QmQZn3pZCcuEf34FSvucqkvVJEvfzpNjQTk17HS6CYMR35"
	org2NetId   = "QmeRZz3AjhzydkzpiuuSAtmqt8mU8XcRH2hynQN4tLgYg6"
)

var cmLogger *logger.CMLogger

func init() {
	// logger, _ := zap.NewDevelopment(zap.AddCaller())
	// clog = logger.Sugar()
	cmLogger = logger.GetLogger(chainId)
}

func TestGetValidatorListFromConfig(t *testing.T) {
	type args struct {
		chainConfig *configpb.ChainConfig
	}
	tests := []struct {
		name           string
		args           args
		wantValidators []string
		wantErr        bool
	}{
		{
			"one org with one address",
			args{
				chainConfig: &configpb.ChainConfig{
					Consensus: &configpb.ConsensusConfig{
						Nodes: []*configpb.OrgConfig{
							{
								OrgId:   org1Id,
								Address: []string{org1Address},
							},
						},
					},
				},
			},
			[]string{org1NetId},
			false,
		},
		{
			"two org, each with one address",
			args{
				chainConfig: &configpb.ChainConfig{
					Consensus: &configpb.ConsensusConfig{
						Nodes: []*configpb.OrgConfig{
							{
								OrgId:   org1Id,
								Address: []string{org1Address},
							},
							{
								OrgId:   org2Id,
								Address: []string{org2Address},
							},
						},
					},
				},
			},
			[]string{org1NetId, org2NetId},
			false,
		},
		{
			"two org, each with two addresses",
			args{
				chainConfig: &configpb.ChainConfig{
					Consensus: &configpb.ConsensusConfig{
						Nodes: []*configpb.OrgConfig{
							{
								OrgId: org1Id,
								Address: []string{
									org1Address,
									org3Address,
								},
							},
							{
								OrgId: org2Id,
								Address: []string{
									org2Address,
									org4Address,
								},
							},
						},
					},
				},
			},
			[]string{
				org1NetId,
				"QmTSMcqwp4X6oPP5WrNpsMpotQMSGcxVshkGLJUhCrqGbu",
				org2NetId,
				"QmUryDgjNoxfMXHdDRFZ5Pe55R1vxTPA3ZgCteHze2ET27",
			},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotValidators, err := GetValidatorListFromConfig(tt.args.chainConfig)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetValidatorListFromConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotValidators, tt.wantValidators) {
				t.Errorf("GetValidatorListFromConfig() = %v, want %v", gotValidators, tt.wantValidators)
			}
		})
	}
}

func TestVerifyBlockSignaturesOneNodeSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	chainConf := mock.NewMockChainConf(ctrl)
	chainConf.EXPECT().GetChainConfigFromFuture(gomock.Any()).AnyTimes().Return(&configpb.ChainConfig{
		Consensus: &configpb.ConsensusConfig{
			Type: consensuspb.ConsensusType_TBFT,
			Nodes: []*configpb.OrgConfig{
				{
					OrgId:   org1Id,
					Address: []string{org1Address},
				},
			},
		},
	}, nil)

	var blockHeight int64 = 10
	blockHash := sha1.Sum(nil)
	rand.Read(blockHash[:])
	block := &commonpb.Block{
		Header: &commonpb.BlockHeader{
			BlockHeight: blockHeight,
			BlockHash:   blockHash[:],
		},
		AdditionalData: &commonpb.AdditionalData{
			ExtraData: map[string][]byte{
				protocol.TBFTAddtionalDataKey: nil,
			},
		},
	}

	chainConfig, _ := chainConf.GetChainConfigFromFuture(blockHeight)
	validators, _ := GetValidatorListFromConfig(chainConfig)
	validatorSet := newValidatorSet(cmLogger, validators, 1)
	voteSet := NewVoteSet(cmLogger, tbftpb.VoteType_VotePrecommit, blockHeight, 0, validatorSet)
	vote := NewVote(tbftpb.VoteType_VotePrecommit, org1NetId, blockHeight, 0, blockHash[:])
	added, err := voteSet.AddVote(vote)
	require.Nil(t, err)
	require.True(t, added)
	qc := mustMarshal(voteSet.ToProto())
	block.AdditionalData.ExtraData[protocol.TBFTAddtionalDataKey] = qc

	ac := mock.NewMockAccessControlProvider(ctrl)
	ac.EXPECT().CreatePrincipal(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)
	ac.EXPECT().VerifyPrincipal(gomock.Any()).AnyTimes().Return(true, nil)

	if err := VerifyBlockSignatures(chainConf, ac, block); err != nil {
		t.Errorf("VerifyBlockSignatures() error = %v, wantErr %v", err, nil)
	}
}

func TestVerifyBlockSignaturesOneNodeFail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	chainConf := mock.NewMockChainConf(ctrl)
	chainConf.EXPECT().GetChainConfigFromFuture(gomock.Any()).AnyTimes().Return(&configpb.ChainConfig{
		Consensus: &configpb.ConsensusConfig{
			Type: consensuspb.ConsensusType_TBFT,
			Nodes: []*configpb.OrgConfig{
				{
					OrgId:   org1Id,
					Address: []string{org1Address},
				},
			},
		},
	}, nil)

	var blockHeight int64 = 10
	blockHash := sha1.Sum(nil)
	rand.Read(blockHash[:])
	block := &commonpb.Block{
		Header: &commonpb.BlockHeader{
			BlockHeight: blockHeight,
			BlockHash:   blockHash[:],
		},
		AdditionalData: &commonpb.AdditionalData{
			ExtraData: map[string][]byte{
				protocol.TBFTAddtionalDataKey: nil,
			},
		},
	}

	// chainConfig, _ := chainConf.GetChainConfigFromFuture(blockHeight)
	// validators, _ := GetValidatorListFromConfig(chainConfig)
	// validatorSet := newValidatorSet(validators)
	// voteSet := NewVoteSet(tbftpb.VoteType_VotePrecommit, blockHeight, 0, validatorSet)
	// vote := NewVote(tbftpb.VoteType_VotePrecommit, org1Id, blockHeight, 0, blockHash[:])
	// voteSet.AddVote(vote)
	// qc := mustMarshal(voteSet.ToProto())
	// block.AdditionalData.ExtraData[protocol.TBFTAddtionalDataKey] = qc

	ac := mock.NewMockAccessControlProvider(ctrl)
	ac.EXPECT().CreatePrincipal(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)
	ac.EXPECT().VerifyPrincipal(gomock.Any()).AnyTimes().Return(true, nil)

	if err := VerifyBlockSignatures(chainConf, ac, block); err == nil {
		t.Errorf("VerifyBlockSignatures() error = %v, but expecte error", err)
	}
}

func TestVerifyBlockSignaturesFourNodeSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	chainConf := mock.NewMockChainConf(ctrl)
	chainConf.EXPECT().GetChainConfigFromFuture(gomock.Any()).AnyTimes().Return(&configpb.ChainConfig{
		Consensus: &configpb.ConsensusConfig{
			Type: consensuspb.ConsensusType_TBFT,
			Nodes: []*configpb.OrgConfig{
				{
					OrgId:   org1Id,
					Address: []string{org1Address},
				},
				{
					OrgId:   org2Id,
					Address: []string{org2Address},
				},
				{
					OrgId:   org3Id,
					Address: []string{org3Address},
				},
				{
					OrgId:   org4Id,
					Address: []string{org4Address},
				},
			},
		},
	}, nil)

	var blockHeight int64 = 10
	blockHash := sha1.Sum(nil)
	rand.Read(blockHash[:])
	block := &commonpb.Block{
		Header: &commonpb.BlockHeader{
			BlockHeight: blockHeight,
			BlockHash:   blockHash[:],
		},
		AdditionalData: &commonpb.AdditionalData{
			ExtraData: map[string][]byte{
				protocol.TBFTAddtionalDataKey: nil,
			},
		},
	}

	chainConfig, _ := chainConf.GetChainConfigFromFuture(blockHeight)
	validators, _ := GetValidatorListFromConfig(chainConfig)
	validatorSet := newValidatorSet(cmLogger, validators, 1)
	voteSet := NewVoteSet(cmLogger, tbftpb.VoteType_VotePrecommit, blockHeight, 0, validatorSet)

	nodes := []string{
		org1NetId,
		org2NetId,
		"QmTSMcqwp4X6oPP5WrNpsMpotQMSGcxVshkGLJUhCrqGbu",
		"QmUryDgjNoxfMXHdDRFZ5Pe55R1vxTPA3ZgCteHze2ET27",
	}
	for _, id := range nodes {
		vote := NewVote(tbftpb.VoteType_VotePrecommit, id, blockHeight, 0, blockHash[:])
		voteSet.AddVote(vote)
	}
	qc := mustMarshal(voteSet.ToProto())
	block.AdditionalData.ExtraData[protocol.TBFTAddtionalDataKey] = qc

	ac := mock.NewMockAccessControlProvider(ctrl)
	ac.EXPECT().CreatePrincipal(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)
	ac.EXPECT().VerifyPrincipal(gomock.Any()).AnyTimes().Return(true, nil)
	if err := VerifyBlockSignatures(chainConf, ac, block); err != nil {
		t.Errorf("VerifyBlockSignatures() error = %v, wantErr %v", err, nil)
	}
}

func TestVerifyBlockSignaturesFourNodeFail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	chainConf := mock.NewMockChainConf(ctrl)
	chainConf.EXPECT().GetChainConfigFromFuture(gomock.Any()).AnyTimes().Return(&configpb.ChainConfig{
		Consensus: &configpb.ConsensusConfig{
			Type: consensuspb.ConsensusType_TBFT,
			Nodes: []*configpb.OrgConfig{
				{
					OrgId:   org1Id,
					Address: []string{org1Address},
				},
				{
					OrgId:   org2Id,
					Address: []string{org2Address},
				},
				{
					OrgId:   org3Id,
					Address: []string{org3Address},
				},
				{
					OrgId:   org4Id,
					Address: []string{org4Address},
				},
			},
		},
	}, nil)

	var blockHeight int64 = 10
	blockHash := sha1.Sum(nil)
	rand.Read(blockHash[:])
	block := &commonpb.Block{
		Header: &commonpb.BlockHeader{
			BlockHeight: blockHeight,
			BlockHash:   blockHash[:],
		},
		AdditionalData: &commonpb.AdditionalData{
			ExtraData: map[string][]byte{
				protocol.TBFTAddtionalDataKey: nil,
			},
		},
	}

	chainConfig, _ := chainConf.GetChainConfigFromFuture(blockHeight)
	validators, _ := GetValidatorListFromConfig(chainConfig)
	validatorSet := newValidatorSet(cmLogger, validators, 1)
	voteSet := NewVoteSet(cmLogger, tbftpb.VoteType_VotePrecommit, blockHeight, 0, validatorSet)

	nodes := []string{
		org1NetId,
		// org2Id,
		// "QmTSMcqwp4X6oPP5WrNpsMpotQMSGcxVshkGLJUhCrqGbu",
		// "QmUryDgjNoxfMXHdDRFZ5Pe55R1vxTPA3ZgCteHze2ET27",
	}
	for _, id := range nodes {
		vote := NewVote(tbftpb.VoteType_VotePrecommit, id, blockHeight, 0, blockHash[:])
		voteSet.AddVote(vote)
	}
	qc := mustMarshal(voteSet.ToProto())
	block.AdditionalData.ExtraData[protocol.TBFTAddtionalDataKey] = qc

	ac := mock.NewMockAccessControlProvider(ctrl)
	ac.EXPECT().CreatePrincipal(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)
	ac.EXPECT().VerifyPrincipal(gomock.Any()).AnyTimes().Return(true, nil)
	if err := VerifyBlockSignatures(chainConf, ac, block); err == nil {
		t.Errorf("VerifyBlockSignatures() error = %v, but expecte error", err)
	}
}
