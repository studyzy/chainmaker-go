/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package governance

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"chainmaker.org/chainmaker-go/common/helper"
	"chainmaker.org/chainmaker-go/logger"
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	configPb "chainmaker.org/chainmaker-go/pb/protogo/config"
	consensusPb "chainmaker.org/chainmaker-go/pb/protogo/consensus"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/utils"
	"github.com/gogo/protobuf/proto"
)

const (
	ConstMinQuorumForQc    = 3     //default min vote num
	ConstTransitBlock      = 0     //default epoch switch block buff
	ConstBlockNumPerEpoch  = 10000 //default epoch change height
	ConstValidatorNum      = 4     //default actual consensus node num
	ConstNodeProposeRound  = 1     //default continuity propose round
	GovernanceContractName = "government_contract"

	SkipTimeoutCommit = "SkipTimeoutCommit"
	CachedLen         = "CachedLen"
	BlockNumPerEpoch  = "BlockNumPerEpoch"
	TransitBlock      = "TransitBlock"
	ValidatorNum      = "ValidatorNum"
	NodeProposeRound  = "NodeProposeRound"

	UnmarshalErrFmt        = "proto.Unmarshal err!err=%v"
	CreateValidatorsErrFmt = "createValidators err!err=%v"
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

func getGovernanceContractFromChainStore(store protocol.BlockchainStore) (*consensusPb.GovernanceContract, error) {
	log.Debugf("get government contract from chainStore")
	contractName := GovernanceContractName
	bz, err := store.ReadObject(contractName, []byte(contractName))
	if err != nil {
		log.Errorf("ReadObject.Get err!contractName=%v,err=%v", contractName, err)
		return nil, err
	}

	if len(bz) == 0 {
		log.Errorf("ReadObject.Get empty!contractName=%v", contractName)
		return nil, fmt.Errorf("bytes is empty")
	}
	var GovernanceContract consensusPb.GovernanceContract
	err = proto.Unmarshal(bz, &GovernanceContract)
	if err != nil {
		log.Errorf(UnmarshalErrFmt, err)
		return nil, err
	}

	return &GovernanceContract, nil
}
func updateGovContractFromConfig(chainConfig *configPb.ChainConfig, GovernanceContract *consensusPb.GovernanceContract) (update bool) {
	isChg := false
	conConf := chainConfig.Consensus

	newCachedLen := uint64(0)
	newSkipTimeoutCommit := false
	newTransitBlock := uint64(ConstTransitBlock)
	newValidatorNum := uint64(ConstValidatorNum)
	newBlockNumPerEpoch := uint64(ConstBlockNumPerEpoch)
	newNodeProposeRound := uint64(ConstNodeProposeRound)

	for _, oneConf := range conConf.ExtConfig {
		switch oneConf.Key {
		case SkipTimeoutCommit:
			if strings.ToUpper(oneConf.Value) == "TRUE" {
				newSkipTimeoutCommit = true
				continue
			}
			if strings.ToUpper(oneConf.Value) == "FALSE" {
				newSkipTimeoutCommit = false
				continue
			}
		case CachedLen:
			cachedLen, err := strconv.ParseUint(oneConf.Value, 10, 64)
			if err != nil {
				continue
			}
			newCachedLen = cachedLen
		case BlockNumPerEpoch:
			blockNumPerEpoch, err := strconv.ParseUint(oneConf.Value, 10, 64)
			if err != nil {
				continue
			}
			newBlockNumPerEpoch = blockNumPerEpoch
		case TransitBlock:
			transitBlock, err := strconv.ParseUint(oneConf.Value, 10, 64)
			if err != nil {
				continue
			}
			if GovernanceContract.Type == consensusPb.ConsensusType_HOTSTUFF && newTransitBlock != 0 {
				log.Warnf("set TransitBlock err!HOTSTUFF should set 0")
				continue
			}
			newTransitBlock = transitBlock
		case ValidatorNum:
			validatorNum, err := strconv.ParseUint(oneConf.Value, 10, 64)
			if err != nil {
				continue
			}
			//if less than default,no effect
			if validatorNum < ConstValidatorNum {
				log.Warnf("set validatorNum err!validatorNum[%v],min ConstValidatorNum[%v]", validatorNum, ConstValidatorNum)
				continue
			}
			newValidatorNum = validatorNum
		case NodeProposeRound:
			nodeProposeRound, err := strconv.ParseUint(oneConf.Value, 10, 64)
			if err != nil {
				continue
			}
			if nodeProposeRound < 1 {
				log.Warnf("set nodeProposeRound err!NodeProposeRound=%v", nodeProposeRound)
				continue
			}
			newNodeProposeRound = nodeProposeRound
		}
	}
	if GovernanceContract.SkipTimeoutCommit != newSkipTimeoutCommit {
		GovernanceContract.SkipTimeoutCommit = newSkipTimeoutCommit
		isChg = true
	}
	if GovernanceContract.ValidatorNum != newValidatorNum {
		GovernanceContract.ValidatorNum = newValidatorNum
		isChg = true
	}
	if GovernanceContract.NodeProposeRound != newNodeProposeRound {
		GovernanceContract.NodeProposeRound = newNodeProposeRound
		isChg = true
	}
	if GovernanceContract.CachedLen != newCachedLen {
		GovernanceContract.CachedLen = newCachedLen
		isChg = true
	}
	if newBlockNumPerEpoch != 0 && newBlockNumPerEpoch < newTransitBlock {
		log.Errorf("set ConstBlockNumPerEpoch or ConstTransitBlock err!newBlockNumPerEpoch=%v,newTransitBlock=%v", newBlockNumPerEpoch, newTransitBlock)
	} else {
		if GovernanceContract.BlockNumPerEpoch != newBlockNumPerEpoch {
			GovernanceContract.BlockNumPerEpoch = newBlockNumPerEpoch
			isChg = true
		}
		if GovernanceContract.TransitBlock != newTransitBlock {
			GovernanceContract.TransitBlock = newTransitBlock
			isChg = true
		}
	}
	return isChg
}
func checkChainConfig(chainConfig *configPb.ChainConfig, GovernanceContract *consensusPb.GovernanceContract) (bool, error) {
	var err error
	nodes := chainConfig.Consensus.Nodes
	tempMap := map[string]int{}

	for _, node := range nodes {
		for _, addr := range node.Address {
			if _, ok := tempMap[addr]; !ok {
				tempMap[addr] = 1
			}
		}
	}
	n := len(tempMap)
	if n < (ConstMinQuorumForQc + 1) {
		return false, fmt.Errorf("set Nodes size err")
	}

	conConf := chainConfig.Consensus
	newTransitBlock := int64(ConstTransitBlock)
	newBlockNumPerEpoch := int64(ConstBlockNumPerEpoch)

	for _, oneConf := range conConf.ExtConfig {
		switch oneConf.Key {
		case SkipTimeoutCommit:
			if strings.ToUpper(oneConf.Value) != "TRUE" && strings.ToUpper(oneConf.Value) != "FALSE" {
				return false, fmt.Errorf("set SkipTimeoutCommit err")
			}
		case CachedLen:
			cachedLen, err := strconv.ParseInt(oneConf.Value, 10, 64)
			if err != nil || cachedLen < 0 {
				return false, fmt.Errorf("set CachedLen err")
			}
		case BlockNumPerEpoch:
			newBlockNumPerEpoch, err = strconv.ParseInt(oneConf.Value, 10, 64)
			if err != nil || newBlockNumPerEpoch < 0 {
				return false, fmt.Errorf("set BlockNumPerEpoch err")
			}
		case TransitBlock:
			newTransitBlock, err = strconv.ParseInt(oneConf.Value, 10, 64)
			if err != nil || newTransitBlock < 0 {
				return false, fmt.Errorf("set TransitBlock err")
			}
			if GovernanceContract.Type == consensusPb.ConsensusType_HOTSTUFF && newTransitBlock != 0 {
				return false, fmt.Errorf("TransitBlock err,hotstuff should set 0")
			}
		case ValidatorNum:
			validatorNum, err := strconv.ParseInt(oneConf.Value, 10, 64)
			if err != nil || validatorNum < 0 {
				return false, fmt.Errorf("set ValidatorNum err")
			}
			//if less than default,no effect
			if validatorNum < ConstValidatorNum {
				return false, fmt.Errorf("set ValidatorNum err")
			}
		case NodeProposeRound:
			nodeProposeRound, err := strconv.ParseInt(oneConf.Value, 10, 64)
			if err != nil || nodeProposeRound < 1 {
				return false, fmt.Errorf("set nodeProposeRound err")
			}
		}
	}
	if newBlockNumPerEpoch != 0 && newBlockNumPerEpoch < newTransitBlock {
		return false, fmt.Errorf("newBlockNumPerEpoch less than transitBlock err")
	}
	return true, nil
}

//create government data from chainConfig when genesis
func getGovernanceContractFromConfig(chainConfig *configPb.ChainConfig) (*consensusPb.GovernanceContract, error) {
	log.Debugf("get government contract from config file")

	var (
		index         = 0
		addrs         []string
		tempMap       = make(map[string]int)
		nodes         = chainConfig.Consensus.Nodes
		consensusType = chainConfig.Consensus.Type
		members       []*consensusPb.GovernanceMember
	)
	// 1. Initializes the members who have the right to participate in the consensus
	for _, node := range nodes {
		for _, addr := range node.Address {
			if _, ok := tempMap[addr]; !ok {
				tempMap[addr] = 1
				addrs = append(addrs, addr)
			}
		}
	}
	sort.Sort(sort.StringSlice(addrs))

	for _, addr := range addrs {
		uid, err := helper.GetNodeUidFromAddr(addr)
		if err != nil {
			continue
		}
		member := &consensusPb.GovernanceMember{
			Index:  int64(index),
			NodeID: uid,
		}
		members = append(members, member)
		index++
	}
	sort.Sort(indexedGovernanceMember(members))

	// 2. create GovernanceContract
	GovernanceContract := &consensusPb.GovernanceContract{
		EpochId:           0,
		Type:              consensusType,
		CurMaxIndex:       int64(index),
		SkipTimeoutCommit: false,
		N:                 uint64(len(members)),
		MinQuorumForQc:    ConstMinQuorumForQc,
		CachedLen:         0,
		NextSwitchHeight:  0,
		TransitBlock:      ConstTransitBlock,
		BlockNumPerEpoch:  ConstBlockNumPerEpoch,
		ValidatorNum:      ConstValidatorNum,
		NodeProposeRound:  ConstNodeProposeRound,
		Members:           members,
		Validators:        nil,
		NextValidators:    nil,
		ConfigSequence:    chainConfig.Sequence,
	}
	updateGovContractFromConfig(chainConfig, GovernanceContract)

	if GovernanceContract.N > GovernanceContract.ValidatorNum {
		GovernanceContract.N = GovernanceContract.ValidatorNum
	}
	GovernanceContract.MinQuorumForQc = (2*GovernanceContract.N + 1) / 3
	if GovernanceContract.MinQuorumForQc < ConstMinQuorumForQc {
		log.Errorf("Set minQuorumForQc err!MinQuorumForQc=%v", GovernanceContract.MinQuorumForQc)
		GovernanceContract.MinQuorumForQc = ConstMinQuorumForQc
	}

	bytesSeed, _ := proto.Marshal(GovernanceContract)
	validators, err := createValidators(GovernanceContract, bytesSeed)
	if err != nil {
		log.Errorf(CreateValidatorsErrFmt, err)
		return nil, err
	}
	GovernanceContract.Validators = validators
	return GovernanceContract, nil
}

func getChainConfigFromChainStore(store protocol.BlockchainStore) (*configPb.ChainConfig, error) {
	log.Debugf("get chainConfig from chainStore")
	contractName := commonPb.ContractName_SYSTEM_CONTRACT_CHAIN_CONFIG.String()
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

//Josephus problem
//Time Complexity:chooseM*seed
func getRandomList(N, chooseM int64, seed int64) []int64 {
	var (
		result       []int64
		peers        = make([]int64, N)
		i, j, curNum int64
	)
	//if N less M,choose all
	if N <= chooseM {
		for i = 0; i < N; i++ {
			result = append(result, i)
		}
		return result
	}

	//from seed%N
	for i = seed % N; curNum < chooseM; i++ {
		if i >= N {
			i = 0
		}
		if peers[i] == 0 {
			j++
			if j > seed {
				result = append(result, i)
				peers[i] = 1
				j = 0
				curNum++
			}
		}
	}
	sort.Sort(IntSlice64(result))
	return result
}

func createValidators(GovernanceContract *consensusPb.GovernanceContract, seedBytes []byte) ([]*consensusPb.GovernanceMember, error) {
	var membersNum = uint64(len(GovernanceContract.Members))
	if membersNum <= GovernanceContract.ValidatorNum {
		//if membersNum less ,choose all
		validators := GovernanceContract.Members
		return validators, nil
	} else {
		for len(seedBytes) < 40 {
			seedBytes = append(seedBytes, 0)
		}

		var b [8]byte
		for i := 0; i < 8; i++ {
			for j := 0; j < 4; j++ {
				b[i] = seedBytes[i] ^ seedBytes[i+8*j]
			}
		}
		var members []*consensusPb.GovernanceMember

		seed := uint64(b[7]) | uint64(b[6])<<8 | uint64(b[5])<<16 | uint64(b[4])<<24 | uint64(b[3])<<32 | uint64(b[2])<<40 | uint64(b[1])<<48 | uint64(b[0])<<56
		seed = seed % uint64(membersNum)

		//use Josephus problem
		randomList := getRandomList(int64(membersNum), int64(GovernanceContract.ValidatorNum), int64(seed))
		log.Debugf("createValidators randomList=%v", randomList)
		for _, index := range randomList {
			member := GovernanceContract.Members[index]
			members = append(members, member)
		}
		sort.Sort(indexedGovernanceMember(members))
		return members, nil
	}
}

func TryCreateNextValidators(block *commonPb.Block, GovernanceContract *consensusPb.GovernanceContract) (bool, error) {
	if GovernanceContract.BlockNumPerEpoch <= 0 {
		return false, nil
	}

	height := block.GetHeader().GetBlockHeight()
	if uint64(height)%GovernanceContract.BlockNumPerEpoch == 0 {
		//use PreBlock hash
		BytesSeed := block.Header.PreBlockHash
		validators, err := createValidators(GovernanceContract, BytesSeed)
		if err != nil {
			log.Errorf(CreateValidatorsErrFmt, err)
			return false, err
		}
		GovernanceContract.NextValidators = validators
		GovernanceContract.NextSwitchHeight = uint64(height) + GovernanceContract.TransitBlock
		log.Debugf("create NextValidators. curHeight=%v,switchHeight=%v", height, GovernanceContract.NextSwitchHeight)
		return true, nil
	}
	return false, nil
}

//try to check switch epoch
func TrySwitchNextValidator(block *commonPb.Block, GovernanceContract *consensusPb.GovernanceContract) bool {
	if GovernanceContract.NextValidators == nil {
		return false
	}
	height := block.GetHeader().GetBlockHeight()
	if GovernanceContract.NextSwitchHeight == uint64(height) {
		GovernanceContract.Validators = GovernanceContract.NextValidators
		GovernanceContract.NextValidators = nil
		return true
	}
	return false
}

//CheckAndCreateGovernmentArgs execute after block propose,create government txRWSet,wait to add to block header
//when block commit,government txRWSet take effect
func CheckAndCreateGovernmentArgs(block *commonPb.Block,
	store protocol.BlockchainStore, proposalCache protocol.ProposalCache, ledger protocol.LedgerCache) (*commonPb.TxRWSet, error) {
	log.Debugf("CheckAndCreateGovernmentArgs start")

	// 1. get GovernanceContract
	gcr := NewGovernanceContract(store, ledger).(*GovernanceContractImp)
	GovernanceContract, err := gcr.GetGovernmentContract()
	if err != nil {
		log.Errorf("getGovernanceContract err!err=%v", err)
		return nil, err
	}
	var oldBytesData []byte
	if block.Header.GetBlockHeight() > 1 {
		if oldBytesData, err = proto.Marshal(GovernanceContract); err != nil {
			log.Errorf("proto marshal pbcc failed err!err=%v", err)
			return nil, err
		}
	}
	var (
		configChg      = false
		IsConfigChg    = false
		IsValidatorChg = false
	)
	// 2. check if chain config change
	if utils.IsConfBlock(block) {
		chainConfig, err := getChainConfigFromBlock(block, proposalCache)
		if err != nil {
			log.Errorf("getChainConfigFromBlock err!err=%v", err)
		}
		if chainConfig != nil {
			if configChg, err = updateGovContractByConfig(chainConfig, GovernanceContract); err != nil {
				log.Errorf("CheckConfigChange err!err=%v", err)
				return nil, err
			}
			IsConfigChg = configChg
		}
	}

	// 3. if chain config no change,check if epoch switch
	if !configChg {
		log.Debugf("IsValidatorChg start")
		if _, err := TryCreateNextValidators(block, GovernanceContract); err != nil {
			log.Errorf("TryCreateNextValidators err!err=%v", err)
			return nil, err
		}
		IsValidatorChg = TrySwitchNextValidator(block, GovernanceContract)
	}

	// 4. if chain config change or switch to next epoch, change the GovernanceContract epochId
	if IsValidatorChg || IsConfigChg {
		log.Debugf("EpochId change! height[%v] old epochId[%v] now epochId[%v]",
			block.Header.GetBlockHeight(), GovernanceContract.EpochId, (GovernanceContract.EpochId + 1))
		GovernanceContract.EpochId++
	}

	//5. create TxRWSet for GovernanceContract
	log.Debugf("GovernanceContract [%v] height[%v]", GovernanceContract.String(), block.GetHeader().GetBlockHeight())
	txRWSet, err := getGovernanceContractTxRWSet(GovernanceContract, oldBytesData)
	return txRWSet, err
}

func getChainConfigFromBlock(block *commonPb.Block, proposalCache protocol.ProposalCache) (*configPb.ChainConfig, error) {
	if !utils.IsConfBlock(block) {
		log.Errorf("block is not conf block")
		return nil, fmt.Errorf("block is not conf block")
	}

	_, rwSetMap := proposalCache.GetProposedBlock(block)
	if len(rwSetMap) == 0 {
		log.Errorf("rwSetMap is nil")
		return nil, fmt.Errorf("rwSetMap is nil")
	}

	//get from rwSetMap,contract data
	var value []byte
	getChainConfigContractName := commonPb.ContractName_SYSTEM_CONTRACT_CHAIN_CONFIG.String()
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

	var chainConfig configPb.ChainConfig
	if err := proto.Unmarshal(value, &chainConfig); err != nil {
		log.Errorf(UnmarshalErrFmt, err)
		return nil, err
	}
	return &chainConfig, nil
}

func updateGovContractByConfig(chainConfig *configPb.ChainConfig, GovernanceContract *consensusPb.GovernanceContract) (bool, error) {
	log.Debugf("updateGovContractByConfig start")
	if GovernanceContract.ConfigSequence == chainConfig.Sequence {
		return false, nil
	}

	isChange := false
	oldMembersMap := make(map[string]*consensusPb.GovernanceMember, len(GovernanceContract.Members))
	for _, member := range GovernanceContract.Members {
		oldMembersMap[member.NodeID] = member
	}

	var (
		addrs   []string
		tempMap = make(map[string]int)
		nodes   = chainConfig.Consensus.Nodes
		index   = GovernanceContract.CurMaxIndex
		members []*consensusPb.GovernanceMember
	)
	// 1. Initializes the members who have the right to participate in the consensus
	for _, node := range nodes {
		for _, addr := range node.Address {
			if _, ok := tempMap[addr]; !ok {
				tempMap[addr] = 1
				addrs = append(addrs, addr)
			}
		}
	}
	sort.Sort(sort.StringSlice(addrs))
	for _, addr := range addrs {
		uid, err := helper.GetNodeUidFromAddr(addr)
		if err != nil {
			continue
		}
		//reuse old index
		if member, ok := oldMembersMap[uid]; ok {
			members = append(members, member)
		} else {
			//use new index
			member := &consensusPb.GovernanceMember{
				Index:  index,
				NodeID: uid,
			}
			members = append(members, member)
			index++
		}
	}

	// 2. if change
	isChange = updateGovContractFromConfig(chainConfig, GovernanceContract)
	if index != GovernanceContract.CurMaxIndex || len(members) != len(GovernanceContract.Members) || isChange {
		sort.Sort(indexedGovernanceMember(members))
		isChange = true
		n := len(members)
		if n > int(GovernanceContract.ValidatorNum) {
			n = int(GovernanceContract.ValidatorNum)
		}

		minQuorumForQc := (2*n + 1) / 3
		if minQuorumForQc < ConstMinQuorumForQc {
			log.Errorf("Set minQuorumForQc err!minQuorumForQc=%v", minQuorumForQc)
			minQuorumForQc = ConstMinQuorumForQc
		}

		GovernanceContract.CurMaxIndex = index
		GovernanceContract.N = uint64(n)
		GovernanceContract.MinQuorumForQc = uint64(minQuorumForQc)
		GovernanceContract.Members = members
		GovernanceContract.NextValidators = nil

		bytesSeed, _ := proto.Marshal(GovernanceContract)
		validators, err := createValidators(GovernanceContract, bytesSeed)
		if err != nil {
			log.Errorf(CreateValidatorsErrFmt, err)
			return false, err
		}
		GovernanceContract.Validators = validators
	}
	GovernanceContract.ConfigSequence = chainConfig.Sequence
	log.Debugf("updateGovContractByConfig end.isChange=%v", isChange)
	return isChange, nil
}

func getGovernanceContractTxRWSet(GovernanceContract *consensusPb.GovernanceContract, oldBytes []byte) (*commonPb.TxRWSet, error) {
	txRWSet := &commonPb.TxRWSet{
		TxId:     "",
		TxReads:  make([]*commonPb.TxRead, 0, 0),
		TxWrites: make([]*commonPb.TxWrite, 0, 1),
	}

	var (
		err          error
		pbccPayload  []byte
		contractName = GovernanceContractName
	)
	// 1. check for changes
	if pbccPayload, err = proto.Marshal(GovernanceContract); err != nil {
		log.Error(err)
		return nil, fmt.Errorf("proto marshal pbcc failed, %s", err.Error())
	}
	if bytes.Equal(oldBytes, pbccPayload) {
		log.Debugf("GovernanceContract no change")
		return txRWSet, nil
	}

	// 2. create txRWSet for the new government contract
	var oldGovernanceContract consensusPb.GovernanceContract
	if err = proto.Unmarshal(oldBytes, &oldGovernanceContract); err != nil {
		return nil, err
	}
	log.Debugf("GovernanceContract change!older:[%v]", oldGovernanceContract.String())
	txWrite := &commonPb.TxWrite{
		Key:          []byte(contractName),
		Value:        pbccPayload,
		ContractName: contractName,
	}
	txRWSet.TxWrites = append(txRWSet.TxWrites, txWrite)
	return txRWSet, nil
}

func GetProposer(level uint64, NodeProposeRound uint64, validators []*consensusPb.GovernanceMember) (*consensusPb.GovernanceMember, error) {
	if validators == nil || len(validators) == 0 {
		return nil, fmt.Errorf("validators is nil")
	}
	index := (level / NodeProposeRound) % uint64(len(validators))
	newMember := &consensusPb.GovernanceMember{
		Index:  validators[index].Index,
		NodeID: validators[index].NodeID,
	}
	log.Debugf("GetProposer newMember[%v] level[%v]", newMember, level)
	return newMember, nil
}
