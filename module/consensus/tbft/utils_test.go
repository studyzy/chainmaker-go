/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tbft

import (
	"crypto/rand"
	"crypto/sha256"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"

	"chainmaker.org/chainmaker/logger/v2"
	commonpb "chainmaker.org/chainmaker/pb-go/v2/common"
	configpb "chainmaker.org/chainmaker/pb-go/v2/config"
	consensuspb "chainmaker.org/chainmaker/pb-go/v2/consensus"
	tbftpb "chainmaker.org/chainmaker/pb-go/v2/consensus/tbft"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/protocol/v2/mock"
	"github.com/stretchr/testify/require"
)

const (
	chainId    = "test"
	org1Id     = "wx-org1"
	org2Id     = "wx-org2"
	org3Id     = "wx-org3"
	org4Id     = "wx-org4"
	org1NodeId = "QmQZn3pZCcuEf34FSvucqkvVJEvfzpNjQTk17HS6CYMR35"
	org2NodeId = "QmeRZz3AjhzydkzpiuuSAtmqt8mU8XcRH2hynQN4tLgYg6"
	org3NodeId = "QmTSMcqwp4X6oPP5WrNpsMpotQMSGcxVshkGLJUhCrqGbu"
	org4NodeId = "QmUryDgjNoxfMXHdDRFZ5Pe55R1vxTPA3ZgCteHze2ET27"
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
			"one org with one node id",
			args{
				chainConfig: &configpb.ChainConfig{
					Consensus: &configpb.ConsensusConfig{
						Nodes: []*configpb.OrgConfig{
							{
								OrgId:  org1Id,
								NodeId: []string{org1NodeId},
							},
						},
					},
				},
			},
			[]string{org1NodeId},
			false,
		},
		{
			"two org, each with one node id",
			args{
				chainConfig: &configpb.ChainConfig{
					Consensus: &configpb.ConsensusConfig{
						Nodes: []*configpb.OrgConfig{
							{
								OrgId:  org1Id,
								NodeId: []string{org1NodeId},
							},
							{
								OrgId:  org2Id,
								NodeId: []string{org2NodeId},
							},
						},
					},
				},
			},
			[]string{org1NodeId, org2NodeId},
			false,
		},
		{
			"two org, each with two node ids",
			args{
				chainConfig: &configpb.ChainConfig{
					Consensus: &configpb.ConsensusConfig{
						Nodes: []*configpb.OrgConfig{
							{
								OrgId: org1Id,
								NodeId: []string{
									org1NodeId,
									org3NodeId,
								},
							},
							{
								OrgId: org2Id,
								NodeId: []string{
									org2NodeId,
									org4NodeId,
								},
							},
						},
					},
				},
			},
			[]string{
				org1NodeId,
				"QmTSMcqwp4X6oPP5WrNpsMpotQMSGcxVshkGLJUhCrqGbu",
				org2NodeId,
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
					OrgId:  org1Id,
					NodeId: []string{org1NodeId},
				},
			},
		},
	}, nil)

	var blockHeight uint64 = 10
	blockHash := sha256.Sum256(nil)
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
	voteSet := NewVoteSet(cmLogger, tbftpb.VoteType_VOTE_PRECOMMIT, blockHeight, 0, validatorSet)
	vote := NewVote(tbftpb.VoteType_VOTE_PRECOMMIT, org1NodeId, blockHeight, 0, blockHash[:])
	added, err := voteSet.AddVote(vote)
	require.Nil(t, err)
	require.True(t, added)
	qc := mustMarshal(voteSet.ToProto())
	block.AdditionalData.ExtraData[protocol.TBFTAddtionalDataKey] = qc

	ac := mock.NewMockAccessControlProvider(ctrl)
	ac.EXPECT().CreatePrincipal(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)
	ac.EXPECT().VerifyPrincipal(gomock.Any()).AnyTimes().Return(true, nil)

	if err := VerifyBlockSignatures(chainConf, ac, block, nil); err != nil {
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
					OrgId:  org1Id,
					NodeId: []string{org1NodeId},
				},
			},
		},
	}, nil)

	var blockHeight uint64 = 10
	blockHash := sha256.Sum256(nil)
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
	// voteSet := NewVoteSet(tbftpb.VoteType_VOTE_PRECOMMIT, blockHeight, 0, validatorSet)
	// vote := NewVote(tbftpb.VoteType_VOTE_PRECOMMIT, org1Id, blockHeight, 0, blockHash[:])
	// voteSet.AddVote(vote)
	// qc := mustMarshal(voteSet.ToProto())
	// block.AdditionalData.ExtraData[protocol.TBFTAddtionalDataKey] = qc

	ac := mock.NewMockAccessControlProvider(ctrl)
	ac.EXPECT().CreatePrincipal(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)
	ac.EXPECT().VerifyPrincipal(gomock.Any()).AnyTimes().Return(true, nil)

	if err := VerifyBlockSignatures(chainConf, ac, block, nil); err == nil {
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
					OrgId:  org1Id,
					NodeId: []string{org1NodeId},
				},
				{
					OrgId:  org2Id,
					NodeId: []string{org2NodeId},
				},
				{
					OrgId:  org3Id,
					NodeId: []string{org3NodeId},
				},
				{
					OrgId:  org4Id,
					NodeId: []string{org4NodeId},
				},
			},
		},
	}, nil)

	var blockHeight uint64 = 10
	blockHash := sha256.Sum256(nil)
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
	voteSet := NewVoteSet(cmLogger, tbftpb.VoteType_VOTE_PRECOMMIT, blockHeight, 0, validatorSet)

	nodes := []string{
		org1NodeId,
		org2NodeId,
		"QmTSMcqwp4X6oPP5WrNpsMpotQMSGcxVshkGLJUhCrqGbu",
		"QmUryDgjNoxfMXHdDRFZ5Pe55R1vxTPA3ZgCteHze2ET27",
	}
	for _, id := range nodes {
		vote := NewVote(tbftpb.VoteType_VOTE_PRECOMMIT, id, blockHeight, 0, blockHash[:])
		voteSet.AddVote(vote)
	}
	qc := mustMarshal(voteSet.ToProto())
	block.AdditionalData.ExtraData[protocol.TBFTAddtionalDataKey] = qc

	ac := mock.NewMockAccessControlProvider(ctrl)
	ac.EXPECT().CreatePrincipal(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)
	ac.EXPECT().VerifyPrincipal(gomock.Any()).AnyTimes().Return(true, nil)
	if err := VerifyBlockSignatures(chainConf, ac, block, nil); err != nil {
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
					OrgId:  org1Id,
					NodeId: []string{org1NodeId},
				},
				{
					OrgId:  org2Id,
					NodeId: []string{org2NodeId},
				},
				{
					OrgId:  org3Id,
					NodeId: []string{org3NodeId},
				},
				{
					OrgId:  org4Id,
					NodeId: []string{org4NodeId},
				},
			},
		},
	}, nil)

	var blockHeight uint64 = 10
	blockHash := sha256.Sum256(nil)
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
	voteSet := NewVoteSet(cmLogger, tbftpb.VoteType_VOTE_PRECOMMIT, blockHeight, 0, validatorSet)

	nodes := []string{
		org1NodeId,
		// org2Id,
		// "QmTSMcqwp4X6oPP5WrNpsMpotQMSGcxVshkGLJUhCrqGbu",
		// "QmUryDgjNoxfMXHdDRFZ5Pe55R1vxTPA3ZgCteHze2ET27",
	}
	for _, id := range nodes {
		vote := NewVote(tbftpb.VoteType_VOTE_PRECOMMIT, id, blockHeight, 0, blockHash[:])
		voteSet.AddVote(vote)
	}
	qc := mustMarshal(voteSet.ToProto())
	block.AdditionalData.ExtraData[protocol.TBFTAddtionalDataKey] = qc

	ac := mock.NewMockAccessControlProvider(ctrl)
	ac.EXPECT().CreatePrincipal(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)
	ac.EXPECT().VerifyPrincipal(gomock.Any()).AnyTimes().Return(true, nil)
	if err := VerifyBlockSignatures(chainConf, ac, block, nil); err == nil {
		t.Errorf("VerifyBlockSignatures() error = %v, but expecte error", err)
	}
}
