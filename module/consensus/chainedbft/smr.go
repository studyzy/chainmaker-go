/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package chainedbft

import (
	"sort"
	"sync"

	"chainmaker.org/chainmaker-go/consensus/chainedbft/liveness"
	safetyrules "chainmaker.org/chainmaker-go/consensus/chainedbft/safety_rules"
	timeservice "chainmaker.org/chainmaker-go/consensus/chainedbft/time_service"
	"chainmaker.org/chainmaker-go/consensus/chainedbft/types"
	"chainmaker.org/chainmaker-go/consensus/chainedbft/utils"
	"chainmaker.org/chainmaker/logger/v2"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	chainedbftpb "chainmaker.org/chainmaker/pb-go/v2/consensus/chainedbft"
)

//chainedbftSMR manages current smr of consensus at height and level
type chainedbftSMR struct {
	sync.RWMutex
	committee    *committee                 // The collection of validators for the current epoch
	preCommittee *committee                 // The collection of validators for the prev epoch
	state        chainedbftpb.ConsStateType // The current consensus state of the local node
	paceMaker    *liveness.Pacemaker        // govern the advancement of levels and height of the local node
	chainStore   *chainStore                // access data on the chain and in the cache, commit block data on the chain
	safetyRules  *safetyrules.SafetyRules   // validate incoming qc and block and update its' state to the newest

	logger *logger.CMLogger
	server *ConsensusChainedBftImpl
}

//newChainedBftSMR returns an instance of consensus smr
func newChainedBftSMR(chainID string,
	epoch *epochManager, chainStore *chainStore, ts *timeservice.TimerService,
	server *ConsensusChainedBftImpl) *chainedbftSMR {
	smr := &chainedbftSMR{
		chainStore: chainStore,
		logger:     logger.GetLoggerByChain(logger.MODULE_CONSENSUS, chainID),
		server:     server,
	}
	smr.safetyRules = safetyrules.NewSafetyRules(smr.logger, chainStore.blockPool, chainStore.blockChainStore)
	smr.initByEpoch(epoch, ts)
	return smr
}

// initByEpoch init committee and paceMaker, reset safetyRules
func (cs *chainedbftSMR) initByEpoch(epoch *epochManager, ts *timeservice.TimerService) {
	cs.initCommittee(epoch.useValidators)
	cs.paceMaker = liveness.NewPacemaker(cs.logger, epoch.index, epoch.createHeight, epoch.epochId, ts)
	cs.forwardNewHeightIfNeed()
}

//initCommittee initializes a committee with valdiators
func (cs *chainedbftSMR) initCommittee(validators []*types.Validator) {
	cs.preCommittee = cs.committee
	peers := make([]*peer, 0, len(validators))
	for _, validator := range validators {
		peer := &peer{
			id:     validator.NodeID,
			index:  validator.Index,
			active: true,
		}
		cs.logger.Debugf("initCommittee peer id [%v], peer index [%v]", peer.id, peer.index)
		peers = append(peers, peer)
	}
	sort.Sort(indexedPeers(peers))
	cs.committee = newCommittee(peers)
}

//forwardNewHeightIfNeed resets the consensus smr by chainStore, and update state to ConsStateType_NEW_HEIGHT
func (cs *chainedbftSMR) forwardNewHeightIfNeed() {
	lastBlock := cs.chainStore.getCurrentCertifiedBlock()
	cs.logger.Debugf("forwardNewHeightIfNeed to chainStore state, smr height [%v],"+
		" chainStore height [%v]", cs.getHeight(), lastBlock.Header.BlockHeight)
	if cs.getHeight() > 0 && uint64(cs.getHeight()) != lastBlock.Header.BlockHeight {
		cs.logger.Warnf("mismatched height [%v], expected [%v]",
			lastBlock.Header.BlockHeight, cs.getHeight())
		return
	}

	cs.state = chainedbftpb.ConsStateType_NEW_HEIGHT
	level, err := utils.GetLevelFromBlock(lastBlock)
	if err != nil {
		cs.logger.Errorf("get level from block error: %s, block %v", err, lastBlock)
		return
	}
	cs.safetyRules.SetLastCommittedBlock(lastBlock, level)
}

func (cs *chainedbftSMR) updateState(newState chainedbftpb.ConsStateType) {
	cs.Lock()
	defer cs.Unlock()
	cs.state = newState
}

func (cs *chainedbftSMR) peers() []*peer {
	return cs.committee.getPeers()
}

func (cs *chainedbftSMR) getPeerByIndex(index uint64) *peer {
	return cs.committee.getPeerByIndex(index)
}

func (cs *chainedbftSMR) isValidIdx(index uint64) bool {
	return cs.committee.isValidIdx(index)
}

func (cs *chainedbftSMR) min(qcHeight uint64) int {
	epochSwitchHeight := cs.server.governanceContract.GetSwitchHeight()
	if epochSwitchHeight == qcHeight {
		return int(cs.server.governanceContract.GetLastGovMembersValidatorMinCount())
	}
	return int(cs.server.governanceContract.GetGovMembersValidatorMinCount())
}

func (cs *chainedbftSMR) getLastVote() (uint64, *chainedbftpb.ConsensusPayload) {
	return cs.safetyRules.GetLastVoteLevel(), cs.safetyRules.GetLastVoteMsg()
}

func (cs *chainedbftSMR) setLastVote(vote *chainedbftpb.ConsensusPayload, level uint64) {
	cs.safetyRules.SetLastVote(vote, level)
}

func (cs *chainedbftSMR) safeNode(proposal *chainedbftpb.ProposalData) error {
	return cs.safetyRules.SafeNode(proposal)
}

func (cs *chainedbftSMR) updateLockedQC(qc *chainedbftpb.QuorumCert) {
	cs.safetyRules.UpdateLockedQC(qc)
}

func (cs *chainedbftSMR) commitRules(qc *chainedbftpb.QuorumCert) (commit bool, commitBlock *common.Block,
	commitLevel uint64) {
	return cs.safetyRules.CommitRules(qc)
}

func (cs *chainedbftSMR) setLastCommittedBlock(block *common.Block, level uint64) {
	cs.safetyRules.SetLastCommittedBlock(block, level)
}

func (cs *chainedbftSMR) getLastCommittedLevel() uint64 {
	return cs.safetyRules.GetLastCommittedLevel()
}

func (cs *chainedbftSMR) getHeight() uint64 {
	return cs.paceMaker.GetHeight()
}
func (cs *chainedbftSMR) getEpochId() uint64 {
	return cs.paceMaker.GetEpochId()
}

func (cs *chainedbftSMR) getCurrentLevel() uint64 {
	return cs.paceMaker.GetCurrentLevel()
}

func (cs *chainedbftSMR) processLocalTimeout(level uint64) bool {
	return cs.paceMaker.ProcessLocalTimeout(level)
}

func (cs *chainedbftSMR) getHighestTCLevel() uint64 {
	return cs.paceMaker.GetHighestTCLevel()
}

// processCertificates Update the status of local Pacemaker with the latest received QC;
// height The height of the received block
// hqcLevel The highest QC level in local node;
// htcLevel The received tcLevel
// hcLevel The latest committed QC level in local node.
// Returns true if the consensus goes to the next level, otherwise false.
//func (cs *chainedbftSMR) processCertificates(height, hqcLevel, htcLevel, hcLevel uint64) bool {
func (cs *chainedbftSMR) processCertificates(qc *chainedbftpb.QuorumCert, tc *chainedbftpb.QuorumCert,
	hcLevel uint64) bool {
	return cs.paceMaker.ProcessCertificates(qc, tc, hcLevel)
	//return cs.paceMaker.ProcessCertificates(height, hqcLevel, htcLevel, hcLevel)
}

func (cs *chainedbftSMR) updateTC(tc *chainedbftpb.QuorumCert) {
	cs.paceMaker.UpdateTC(tc)
}

func (cs *chainedbftSMR) getTC() *chainedbftpb.QuorumCert {
	return cs.paceMaker.GetTC()
}
