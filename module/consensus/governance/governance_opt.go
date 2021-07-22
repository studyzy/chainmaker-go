/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package governance

import (
	"fmt"
	"sort"
	"strconv"

	"chainmaker.org/chainmaker/pb-go/syscontract"

	"chainmaker.org/chainmaker-go/logger"
	"chainmaker.org/chainmaker-go/utils"
	commonPb "chainmaker.org/chainmaker/pb-go/common"
	configPb "chainmaker.org/chainmaker/pb-go/config"
	consensusPb "chainmaker.org/chainmaker/pb-go/consensus"
	"chainmaker.org/chainmaker/protocol"
	"github.com/gogo/protobuf/proto"
)

const (
	ConstMinQuorumForQc        = 3 //default min vote num
	ConstNodeProposeRound      = 1 //default continuity propose round
	MinimumTimeOutMill         = 15000
	MinimumIntervalTimeOutMill = 100

	CachedLen                = "CachedLen"
	RoundTimeoutMill         = "HotstuffRoundTimeoutMill"
	RoundTimeoutIntervalMill = "HotstuffRoundTimeoutIntervalMill"

	UnmarshalErrFmt = "proto.Unmarshal err!err=%v"
)

type indexedGovernanceMember []*consensusPb.GovernanceMember

var log = logger.GetLogger(logger.MODULE_CONSENSUS)

//Len returns the size of indexedValidators
func (iv indexedGovernanceMember) Len() int { return len(iv) }

//Swap swaps the ith object with jth object in indexedPeers
func (iv indexedGovernanceMember) Swap(i, j int) { iv[i], iv[j] = iv[j], iv[i] }

//Less checks the ith object's index < the jth object's index
func (iv indexedGovernanceMember) Less(i, j int) bool { return iv[i].Index < iv[j].Index }

type IntSlice64 []int64

func (s IntSlice64) Len() int { return len(s) }

func (s IntSlice64) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

func (s IntSlice64) Less(i, j int) bool { return s[i] < s[j] }

// ok
func getGovernanceContractFromChainStore(store protocol.BlockchainStore) (*consensusPb.GovernanceContract, error) {
	contractName := syscontract.SystemContract_GOVERNANCE.String()
	bz, err := store.ReadObject(contractName, []byte(contractName))
	if err != nil {
		return nil, fmt.Errorf("get contractName=%s from db failed, reason: %s", contractName, err)
	}
	if len(bz) == 0 {
		return nil, fmt.Errorf("get contractName=%s bytes is empty", contractName)
	}
	var governanceContract *consensusPb.GovernanceContract
	if err = proto.Unmarshal(bz, governanceContract); err != nil {
		return nil, fmt.Errorf("unmarshal contractName=%s failed, reason: %s", contractName, err)
	}
	return governanceContract, nil
}

// ok
func updateGovContractFromConfig(chainConfig *configPb.ChainConfig, governanceContract *consensusPb.GovernanceContract) (update bool) {
	isChg := false
	conConf := chainConfig.Consensus

	newCachedLen := uint64(0)
	newRoundTimeoutMill := uint64(0)
	newRoundTimeoutIntervalMill := uint64(0)

	for _, oneConf := range conConf.ExtConfig {
		switch oneConf.Key {
		case RoundTimeoutMill:
			if v, err := strconv.ParseUint(string(oneConf.Value), 10, 64); err == nil {
				if v < MinimumTimeOutMill {
					log.Warnf("%s is too minimum, %d < %d", RoundTimeoutMill, v, MinimumTimeOutMill)
					continue
				}
				newRoundTimeoutMill = v
			}
		case RoundTimeoutIntervalMill:
			if v, err := strconv.ParseUint(string(oneConf.Value), 10, 64); err == nil {
				if v < MinimumIntervalTimeOutMill {
					log.Warnf("%s is too minimun, %d < %d", RoundTimeoutIntervalMill, v, MinimumIntervalTimeOutMill)
					continue
				}
				newRoundTimeoutIntervalMill = v
			}
		case CachedLen:
			cachedLen, err := strconv.ParseUint(string(oneConf.Value), 10, 64)
			if err != nil {
				continue
			}
			newCachedLen = cachedLen
		}
	}
	if newCachedLen != 0 && governanceContract.CachedLen != newCachedLen {
		governanceContract.CachedLen = newCachedLen
		isChg = true
	}
	if newRoundTimeoutMill != 0 && governanceContract.HotstuffRoundTimeoutMill != newRoundTimeoutMill {
		governanceContract.HotstuffRoundTimeoutMill = newRoundTimeoutMill
		isChg = true
	}
	if newRoundTimeoutIntervalMill != 0 && governanceContract.HotstuffRoundTimeoutIntervalMill != newRoundTimeoutIntervalMill {
		governanceContract.HotstuffRoundTimeoutIntervalMill = newRoundTimeoutIntervalMill
		isChg = true
	}
	return isChg
}

// ok
//create government data from chainConfig when genesis
func getGovernanceContractFromConfig(chainConfig *configPb.ChainConfig) (*consensusPb.GovernanceContract, error) {
	log.Debugf("get government contract from config file")
	// 1. Initializes the members who have the right to participate in the consensus
	members, index := getMembersFromConfig(chainConfig)

	// 2. create GovernanceContract
	governanceContract := &consensusPb.GovernanceContract{
		N:                uint64(len(members)),
		EpochId:          0,
		Validators:       members,
		CurMaxIndex:      int64(index),
		MinQuorumForQc:   ConstMinQuorumForQc,
		ValidatorNum:     uint64(len(members)),
		NodeProposeRound: ConstNodeProposeRound,
		ConfigSequence:   chainConfig.Sequence,
	}
	updateGovContractFromConfig(chainConfig, governanceContract)
	governanceContract.MinQuorumForQc = (2*governanceContract.ValidatorNum + 1) / 3
	if governanceContract.MinQuorumForQc < ConstMinQuorumForQc {
		return nil, fmt.Errorf("quorum[%d] is too minimum: %d", governanceContract.MinQuorumForQc, ConstMinQuorumForQc)
	}
	return governanceContract, nil
}

// ok
func getMembersFromConfig(chainConfig *configPb.ChainConfig) ([]*consensusPb.GovernanceMember, int) {
	var (
		index   = 0
		nodes   = chainConfig.Consensus.Nodes
		nodeIds = make([]string, 0, len(nodes))
		tempMap = make(map[string]int, len(nodes))
		members = make([]*consensusPb.GovernanceMember, 0, len(nodes))
	)
	for _, node := range nodes {
		for _, nid := range node.NodeId {
			if _, ok := tempMap[nid]; !ok {
				tempMap[nid] = 1
				nodeIds = append(nodeIds, nid)
			}
		}
	}
	sort.Sort(sort.StringSlice(nodeIds))
	for _, nid := range nodeIds {
		members = append(members, &consensusPb.GovernanceMember{
			Index:  int64(index),
			NodeId: nid,
		})
		index++
	}
	sort.Sort(indexedGovernanceMember(members))
	return members, index
}

// ok
func getChainConfigFromChainStore(store protocol.BlockchainStore) (*configPb.ChainConfig, error) {
	contractName := syscontract.SystemContract_CHAIN_CONFIG.String()
	bz, err := store.ReadObject(contractName, []byte(contractName))
	if err != nil {
		log.Errorf("store.ReadObject err!contractName=%v,err=%v", contractName, err)
		return nil, err
	}
	var chainConfig configPb.ChainConfig
	if err = proto.Unmarshal(bz, &chainConfig); err != nil {
		log.Errorf(UnmarshalErrFmt, err)
		return nil, err
	}
	return &chainConfig, nil
}

// ok
func getChainConfigFromBlock(block *commonPb.Block, proposalCache protocol.ProposalCache) (*configPb.ChainConfig, error) {
	// 1. base check
	if !utils.IsConfBlock(block) {
		log.Errorf("block is not conf block")
		return nil, fmt.Errorf("block is not conf block")
	}
	_, rwSetMap, _ := proposalCache.GetProposedBlock(block)
	if len(rwSetMap) == 0 {
		log.Errorf("rwSetMap is nil")
		return nil, fmt.Errorf("rwSetMap is nil")
	}

	// 2. get from rwSetMap,contract data
	var value []byte
	getChainConfigContractName := syscontract.SystemContract_CHAIN_CONFIG.String()
	for _, rwSet := range rwSetMap {
		for _, txWriteItem := range rwSet.TxWrites {
			if txWriteItem.ContractName == getChainConfigContractName && getChainConfigContractName == string(txWriteItem.Key) {
				value = txWriteItem.Value
				break
			}
		}
	}
	if value == nil {
		log.Errorf("TxWrites no match")
		return nil, fmt.Errorf("TxWrites no match")
	}

	// 3. unmarshal chainConfig
	var chainConfig configPb.ChainConfig
	if err := proto.Unmarshal(value, &chainConfig); err != nil {
		log.Errorf(UnmarshalErrFmt, err)
		return nil, err
	}
	return &chainConfig, nil
}

// ok
func updateGovContractByConfig(chainConfig *configPb.ChainConfig, governanceContract *consensusPb.GovernanceContract) (bool, error) {
	log.Debugf("updateGovContractByConfig start")
	if governanceContract.ConfigSequence == chainConfig.Sequence {
		return false, nil
	}
	// 1. Initializes the members who have the right to participate in the consensus
	newMembers, index := getNewMembers(chainConfig, governanceContract)

	// 2. if change
	isChange := updateGovContractFromConfig(chainConfig, governanceContract)
	if index != governanceContract.CurMaxIndex || len(newMembers) != len(governanceContract.Members) || isChange {
		sort.Sort(indexedGovernanceMember(newMembers))
		isChange = true
		n := len(newMembers)
		minQuorumForQc := (2*n + 1) / 3
		if minQuorumForQc < ConstMinQuorumForQc {
			log.Errorf("Set minQuorumForQc err!minQuorumForQc=%v", minQuorumForQc)
			minQuorumForQc = ConstMinQuorumForQc
		}

		governanceContract.N = uint64(n)
		governanceContract.CurMaxIndex = index
		governanceContract.ValidatorNum = uint64(n)
		governanceContract.LastMinQuorumForQc = governanceContract.MinQuorumForQc
		governanceContract.MinQuorumForQc = uint64(minQuorumForQc)
		governanceContract.Validators = newMembers
	}
	governanceContract.ConfigSequence = chainConfig.Sequence
	log.Debugf("updateGovContractByConfig end.isChange=%v", isChange)
	return isChange, nil
}

func getNewMembers(chainConfig *configPb.ChainConfig, governanceContract *consensusPb.GovernanceContract) ([]*consensusPb.GovernanceMember, int64) {
	oldMembersMap := make(map[string]*consensusPb.GovernanceMember, len(governanceContract.Validators))
	for _, member := range governanceContract.Validators {
		oldMembersMap[member.NodeId] = member
	}

	var (
		index      = governanceContract.CurMaxIndex
		newNodes   = chainConfig.Consensus.Nodes
		tempMap    = make(map[string]int, len(newNodes))
		newNodeIds = make([]string, 0, len(newNodes))
		newMembers = make([]*consensusPb.GovernanceMember, 0, len(newNodes))
	)

	for _, node := range newNodes {
		for _, nid := range node.NodeId {
			if _, ok := tempMap[nid]; !ok {
				tempMap[nid] = 1
				newNodeIds = append(newNodeIds, nid)
			}
		}
	}
	sort.Sort(sort.StringSlice(newNodeIds))
	for _, nid := range newNodeIds {
		//reuse old index
		if member, ok := oldMembersMap[nid]; ok {
			newMembers = append(newMembers, member)
		} else {
			//use new index
			newMembers = append(newMembers, &consensusPb.GovernanceMember{
				Index: index, NodeId: nid,
			})
			index++
		}
	}
	return newMembers, index
}

// ok
func getGovernanceContractTxRWSet(GovernanceContract *consensusPb.GovernanceContract) (*commonPb.TxRWSet, error) {
	txRWSet := &commonPb.TxRWSet{
		TxId:     syscontract.SystemContract_GOVERNANCE.String(),
		TxReads:  make([]*commonPb.TxRead, 0, 0),
		TxWrites: make([]*commonPb.TxWrite, 0, 1),
	}

	var (
		err          error
		pbccPayload  []byte
		contractName = syscontract.SystemContract_GOVERNANCE.String()
	)
	// 1. check for changes
	if pbccPayload, err = proto.Marshal(GovernanceContract); err != nil {
		log.Error(err)
		return nil, fmt.Errorf("proto marshal pbcc failed, %s", err.Error())
	}

	// 2. create txRWSet for the new government contract
	txWrite := &commonPb.TxWrite{
		Key:          []byte(contractName),
		Value:        pbccPayload,
		ContractName: contractName,
	}
	txRWSet.TxWrites = append(txRWSet.TxWrites, txWrite)
	return txRWSet, nil
}
