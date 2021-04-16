/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package governance

import (
	"bytes"
	"chainmaker.org/chainmaker-go/common/helper"
	"chainmaker.org/chainmaker-go/logger"
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	configPb "chainmaker.org/chainmaker-go/pb/protogo/config"
	consensusPb "chainmaker.org/chainmaker-go/pb/protogo/consensus"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/utils"
	"fmt"
	"github.com/gogo/protobuf/proto"
	"sort"
	"strconv"
	"strings"
)

const (
	ConstMinQuorumForQc    = 3  //default min vote num
	ConstTransitBlock      = 0  //default epoch switch block buff
	ConstBlockNumPerEpoch  = 10 //default epoch change height
	ConstValidatorNum      = 4  //default actual consensus node num
	ConstNodeProposeRound  = 1  //default continuity propose round
	GovernmentContractName = "government_contract"

	SkipTimeoutCommit = "SkipTimeoutCommit"
	CachedLen         = "CachedLen"
	BlockNumPerEpoch  = "BlockNumPerEpoch"
	TransitBlock      = "TransitBlock"
	ValidatorNum      = "ValidatorNum"
	NodeProposeRound  = "NodeProposeRound"

	UnmarshalErrFmt        = "proto.Unmarshal err!err=%v"
	CreateValidatorsErrFmt = "createValidators err!err=%v"
)

type indexedGovernmentMember []*consensusPb.GovernmentMember

var log = logger.GetLogger(logger.MODULE_CONSENSUS)

//Len returns the size of indexedValidators
func (iv indexedGovernmentMember) Len() int { return len(iv) }

//Swap swaps the ith object with jth object in indexedPeers
func (iv indexedGovernmentMember) Swap(i, j int) { iv[i], iv[j] = iv[j], iv[i] }

//Less checks the ith object's index < the jth object's index
func (iv indexedGovernmentMember) Less(i, j int) bool { return iv[i].Index < iv[j].Index }

type IntSlice64 []int64

func (s IntSlice64) Len() int { return len(s) }

func (s IntSlice64) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

func (s IntSlice64) Less(i, j int) bool { return s[i] < s[j] }

func getGovernmentContractFromChainStore(store protocol.BlockchainStore) (*consensusPb.GovernmentContract, error) {

	contractName := GovernmentContractName
	bytes, err := store.ReadObject(contractName, []byte(contractName))
	if err != nil {
		log.Errorf("ReadObject.Get err!contractName=%v,err=%v", contractName, err)
		return nil, err
	}

	if len(bytes) == 0 {
		log.Errorf("ReadObject.Get empty!contractName=%v", contractName)
		return nil, fmt.Errorf("bytes is empty")
	}
	var governmentContract consensusPb.GovernmentContract
	err = proto.Unmarshal(bytes, &governmentContract)
	if err != nil {
		log.Errorf(UnmarshalErrFmt, err)
		return nil, err
	}

	return &governmentContract, nil
}
func setFromExtConfig(chainConfig *configPb.ChainConfig, governmentContract *consensusPb.GovernmentContract) bool {
	isChg := false
	conconf := chainConfig.Consensus

	newSkipTimeoutCommit := false
	newCachedLen := uint64(0)
	newBlockNumPerEpoch := uint64(ConstBlockNumPerEpoch)
	newTransitBlock := uint64(ConstTransitBlock)
	newValidatorNum := uint64(ConstValidatorNum)
	newNodeProposeRound := uint64(ConstNodeProposeRound)

	for _, oneconf := range conconf.ExtConfig {
		switch oneconf.Key {
		case SkipTimeoutCommit:
			if strings.ToUpper(oneconf.Value) == "TRUE" {
				newSkipTimeoutCommit = true
				continue
			}
			if strings.ToUpper(oneconf.Value) == "FALSE" {
				newSkipTimeoutCommit = false
				continue
			}

		case CachedLen:
			cachedLen, err := strconv.ParseUint(oneconf.Value, 10, 64)
			if err != nil {
				continue
			}
			newCachedLen = cachedLen
			continue
		case BlockNumPerEpoch:
			blockNumPerEpoch, err := strconv.ParseUint(oneconf.Value, 10, 64)
			if err != nil {
				continue
			}
			newBlockNumPerEpoch = blockNumPerEpoch
			continue
		case TransitBlock:
			transitBlock, err := strconv.ParseUint(oneconf.Value, 10, 64)
			if err != nil {
				continue
			}
			if governmentContract.Type == consensusPb.ConsensusType_HOTSTUFF && newTransitBlock != 0 {
				log.Warnf("set TransitBlock err!HOTSTUFF should set 0")
				continue
			}
			newTransitBlock = transitBlock
			continue
		case ValidatorNum:
			validatorNum, err := strconv.ParseUint(oneconf.Value, 10, 64)
			if err != nil {
				continue
			}
			//if less than default,no effect
			if validatorNum < ConstValidatorNum {
				log.Warnf("set validatorNum err!validatorNum[%v],min ConstValidatorNum[%v]", validatorNum, ConstValidatorNum)
				continue
			}
			newValidatorNum = validatorNum
			continue
		case NodeProposeRound:
			nodeProposeRound, err := strconv.ParseUint(oneconf.Value, 10, 64)
			if err != nil {
				continue
			}
			if nodeProposeRound < 1 {
				log.Warnf("set nodeProposeRound err!NodeProposeRound=%v", nodeProposeRound)
				continue
			}
			newNodeProposeRound = nodeProposeRound
			continue
		}
	}
	if governmentContract.SkipTimeoutCommit != newSkipTimeoutCommit {
		governmentContract.SkipTimeoutCommit = newSkipTimeoutCommit
		isChg = true
	}
	if governmentContract.ValidatorNum != newValidatorNum {
		governmentContract.ValidatorNum = newValidatorNum
		isChg = true
	}
	if governmentContract.NodeProposeRound != newNodeProposeRound {
		governmentContract.NodeProposeRound = newNodeProposeRound
		isChg = true
	}
	if governmentContract.CachedLen != newCachedLen {
		governmentContract.CachedLen = newCachedLen
		isChg = true
	}
	if newBlockNumPerEpoch != 0 && newBlockNumPerEpoch < newTransitBlock {
		log.Errorf("set ConstBlockNumPerEpoch or ConstTransitBlock err!newBlockNumPerEpoch=%v,newTransitBlock=%v", newBlockNumPerEpoch, newTransitBlock)
	} else {
		if governmentContract.BlockNumPerEpoch != newBlockNumPerEpoch {
			governmentContract.BlockNumPerEpoch = newBlockNumPerEpoch
			isChg = true
		}
		if governmentContract.TransitBlock != newTransitBlock {
			governmentContract.TransitBlock = newTransitBlock
			isChg = true
		}
	}
	return isChg
}
func checkChainConfig(chainConfig *configPb.ChainConfig, governmentContract *consensusPb.GovernmentContract) (bool, error) {
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

	conconf := chainConfig.Consensus

	newBlockNumPerEpoch := int64(ConstBlockNumPerEpoch)
	newTransitBlock := int64(ConstTransitBlock)

	for _, oneconf := range conconf.ExtConfig {
		switch oneconf.Key {
		case SkipTimeoutCommit:
			if strings.ToUpper(oneconf.Value) != "TRUE" && strings.ToUpper(oneconf.Value) != "FALSE" {
				return false, fmt.Errorf("set SkipTimeoutCommit err")
			}
			continue
		case CachedLen:
			cachedLen, err := strconv.ParseInt(oneconf.Value, 10, 64)
			if err != nil || cachedLen < 0 {
				return false, fmt.Errorf("set CachedLen err")
			}
			continue
		case BlockNumPerEpoch:
			newBlockNumPerEpoch, err = strconv.ParseInt(oneconf.Value, 10, 64)
			if err != nil || newBlockNumPerEpoch < 0 {
				return false, fmt.Errorf("set BlockNumPerEpoch err")
			}
			continue
		case TransitBlock:
			newTransitBlock, err = strconv.ParseInt(oneconf.Value, 10, 64)
			if err != nil || newTransitBlock < 0 {
				return false, fmt.Errorf("set TransitBlock err")
			}
			if governmentContract.Type == consensusPb.ConsensusType_HOTSTUFF && newTransitBlock != 0 {
				return false, fmt.Errorf("TransitBlock err,hotstuff should set 0")
			}
			continue
		case ValidatorNum:
			validatorNum, err := strconv.ParseInt(oneconf.Value, 10, 64)
			if err != nil || validatorNum < 0 {
				return false, fmt.Errorf("set ValidatorNum err")
			}
			//if less than default,no effect
			if validatorNum < ConstValidatorNum {
				return false, fmt.Errorf("set ValidatorNum err")
			}
			continue
		case NodeProposeRound:
			nodeProposeRound, err := strconv.ParseInt(oneconf.Value, 10, 64)
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

//create governance data from chainConfig when genesis
func getGovernmentContractFromConfig(chainConfig *configPb.ChainConfig) (*consensusPb.GovernmentContract, error) {

	log.Debugf("getGovernmentContractFromConfig start")
	var members []*consensusPb.GovernmentMember

	index := 0
	nodes := chainConfig.Consensus.Nodes
	consensusType := chainConfig.Consensus.Type

	tempMap := map[string]int{}
	var addrs []string
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
		member := &consensusPb.GovernmentMember{
			Index:  int64(index),
			NodeID: uid,
		}
		members = append(members, member)
		index++
	}

	sort.Sort(indexedGovernmentMember(members))

	governmentContract := &consensusPb.GovernmentContract{
		EpochId:           0,
		Type:              consensusType,
		CurMaxIndex:       int64(index),
		SkipTimeoutCommit: false,
		//IsConfigChg:       false,
		//IsValidatorChg:    false,
		N:                uint64(len(members)),
		MinQuorumForQc:   ConstMinQuorumForQc,
		CachedLen:        0,
		NextSwitchHeight: 0,
		TransitBlock:     ConstTransitBlock,
		BlockNumPerEpoch: ConstBlockNumPerEpoch,
		ValidatorNum:     ConstValidatorNum,
		NodeProposeRound: ConstNodeProposeRound,
		Members:          members,
		Validators:       nil,
		NextValidators:   nil,
		ConfigSequence:   chainConfig.Sequence,
	}
	setFromExtConfig(chainConfig, governmentContract)

	if governmentContract.N > governmentContract.ValidatorNum {
		governmentContract.N = governmentContract.ValidatorNum
	}

	governmentContract.MinQuorumForQc = ((2*governmentContract.N + 1) / 3)
	if governmentContract.MinQuorumForQc < ConstMinQuorumForQc {
		log.Errorf("Set minQuorumForQc err!MinQuorumForQc=%v", governmentContract.MinQuorumForQc)
		governmentContract.MinQuorumForQc = ConstMinQuorumForQc
	}

	bytesSeed, _ := proto.Marshal(governmentContract)

	validators, err := createValidators(governmentContract, bytesSeed)
	if err != nil {
		log.Errorf(CreateValidatorsErrFmt, err)
		return nil, err
	}

	governmentContract.Validators = validators
	return governmentContract, nil
}
func getChainConfigFromBlock(block *commonPb.Block, proposalCache protocol.ProposalCache) (*configPb.ChainConfig, error) {
	if !utils.IsConfBlock(block) {
		log.Errorf("block is not conf block")
		return nil, fmt.Errorf("block is not conf block")
	}

	_, rwSetMap,_ := proposalCache.GetProposedBlock(block)
	if rwSetMap == nil || len(rwSetMap) == 0 {
		log.Errorf("rwSetMap is nil")
		return nil, fmt.Errorf("rwSetMap is nil")
	}

	//get from rwSetMap,contract data
	var value []byte
	getChainConfigContractName := commonPb.ContractName_SYSTEM_CONTRACT_CHAIN_CONFIG.String()
	for _, rwSet := range rwSetMap {
		txWrites := rwSet.TxWrites
		for _, txWriteItem := range txWrites {
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
	err := proto.Unmarshal(value, &chainConfig)
	if err != nil {
		log.Errorf(UnmarshalErrFmt, err)
		return nil, err
	}
	return &chainConfig, nil
}

func getChainConfigFromChainStore(store protocol.BlockchainStore) (*configPb.ChainConfig, error) {
	contractName := commonPb.ContractName_SYSTEM_CONTRACT_CHAIN_CONFIG.String()
	bytes, err := store.ReadObject(contractName, []byte(contractName))
	if err != nil {
		log.Errorf("store.ReadObject err!contractName=%v,err=%v", contractName, err)
		return nil, err
	}
	var chainConfig configPb.ChainConfig
	err = proto.Unmarshal(bytes, &chainConfig)
	if err != nil {
		log.Errorf(UnmarshalErrFmt, err)
		return nil, err
	}
	return &chainConfig, nil

}

func getGovernmentContractTxRWSet(governmentContract *consensusPb.GovernmentContract, oldBytes []byte) (*commonPb.TxRWSet, error) {

	txRWSet := &commonPb.TxRWSet{
		TxId:     "",
		TxReads:  make([]*commonPb.TxRead, 0, 0),
		TxWrites: make([]*commonPb.TxWrite, 0, 1),
	}

	contractName := GovernmentContractName
	pbccPayload, err := proto.Marshal(governmentContract)
	if err != nil {
		log.Error(err)
		return nil, fmt.Errorf("proto marshal pbcc failed, %s", err.Error())
	}
	//no change return empty TxRWSet
	if bytes.Equal(oldBytes, pbccPayload) {
		log.Debugf("governmentContract no change")
		return txRWSet, nil
	}

	var oldGovernmentContract consensusPb.GovernmentContract
	err = proto.Unmarshal(oldBytes, &oldGovernmentContract)

	log.Debugf("governmentContract change!older:[%v]", oldGovernmentContract.String())
	txWrite := &commonPb.TxWrite{
		Key:          []byte(contractName),
		Value:        pbccPayload,
		ContractName: contractName,
	}

	txRWSet.TxWrites = append(txRWSet.TxWrites, txWrite)
	return txRWSet, nil
}

//Josephus problem
//Time Complexity:chooseM*seed
func getRandomList(N, chooseM int64, seed int64) []int64 {
	var result []int64
	var peers []int64 = make([]int64, N)
	var i, j, curNum int64

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

func createValidators(governmentContract *consensusPb.GovernmentContract, seedBytes []byte) ([]*consensusPb.GovernmentMember, error) {
	var membersNum uint64 = uint64(len(governmentContract.Members))
	if membersNum <= governmentContract.ValidatorNum {
		//if membersNum less ,choose all
		validators := governmentContract.Members
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
		var members []*consensusPb.GovernmentMember

		seed := uint64(b[7]) | uint64(b[6])<<8 | uint64(b[5])<<16 | uint64(b[4])<<24 | uint64(b[3])<<32 | uint64(b[2])<<40 | uint64(b[1])<<48 | uint64(b[0])<<56
		seed = seed % uint64(membersNum)

		//use Josephus problem
		randomList := getRandomList(int64(membersNum), int64(governmentContract.ValidatorNum), int64(seed))
		log.Debugf("createValidators randomList=%v", randomList)
		for _, index := range randomList {
			member := governmentContract.Members[index]
			members = append(members, member)
		}
		sort.Sort(indexedGovernmentMember(members))
		return members, nil
	}
}

func checkConfigChange(chainConfig *configPb.ChainConfig, governmentContract *consensusPb.GovernmentContract) (bool, error) {
	log.Debugf("checkConfigChange start")
	isChange := false

	if governmentContract.ConfigSequence != chainConfig.Sequence {
		oldMembersMap := make(map[string]*consensusPb.GovernmentMember)
		for _, member := range governmentContract.Members {

			oldMembersMap[member.NodeID] = member
		}

		var members []*consensusPb.GovernmentMember
		index := governmentContract.CurMaxIndex
		nodes := chainConfig.Consensus.Nodes

		tempMap := map[string]int{}
		var addrs []string
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
				member := &consensusPb.GovernmentMember{
					Index:  int64(index),
					NodeID: uid,
				}
				members = append(members, member)
				index++
			}
		}

		isChange = setFromExtConfig(chainConfig, governmentContract)
		//if change
		if index != governmentContract.CurMaxIndex || len(members) != len(governmentContract.Members) || isChange {
			sort.Sort(indexedGovernmentMember(members))
			isChange = true

			n := len(members)
			if n > int(governmentContract.ValidatorNum) {
				n = int(governmentContract.ValidatorNum)
			}

			minQuorumForQc := (2*n + 1) / 3
			if minQuorumForQc < ConstMinQuorumForQc {
				log.Errorf("Set minQuorumForQc err!minQuorumForQc=%v", minQuorumForQc)
				minQuorumForQc = ConstMinQuorumForQc
			}

			governmentContract.CurMaxIndex = index
			governmentContract.N = uint64(n)
			governmentContract.MinQuorumForQc = uint64(minQuorumForQc)
			governmentContract.Members = members
			governmentContract.NextValidators = nil

			bytesSeed, _ := proto.Marshal(governmentContract)

			validators, err := createValidators(governmentContract, bytesSeed)
			if err != nil {
				log.Errorf(CreateValidatorsErrFmt, err)
				return false, err
			}
			governmentContract.Validators = validators
		}
		governmentContract.ConfigSequence = chainConfig.Sequence
	}
	log.Debugf("checkConfigChange end.isChange=%v", isChange)
	return isChange, nil
}

func TryCreateNextValidators(block *commonPb.Block, governmentContract *consensusPb.GovernmentContract) (bool, error) {
	if governmentContract.BlockNumPerEpoch <= 0 {
		return false, nil
	}

	height := block.GetHeader().GetBlockHeight()

	if uint64(height)%governmentContract.BlockNumPerEpoch == 0 {
		//use PreBlock hash
		BytesSeed := block.Header.PreBlockHash
		validators, err := createValidators(governmentContract, BytesSeed)
		if err != nil {
			log.Errorf(CreateValidatorsErrFmt, err)
			return false, err
		}
		governmentContract.NextValidators = validators
		governmentContract.NextSwitchHeight = uint64(height) + governmentContract.TransitBlock
		log.Debugf("create NextValidators. curHeight=%v,switchHeight=%v", height, governmentContract.NextSwitchHeight)
		return true, nil
	}
	return false, nil
}

//try to check switch epoch
func TrySwitchNextValidator(block *commonPb.Block, governmentContract *consensusPb.GovernmentContract) (bool, error) {
	if governmentContract.NextValidators == nil {
		return false, nil
	}
	height := block.GetHeader().GetBlockHeight()
	if governmentContract.NextSwitchHeight == uint64(height) {
		governmentContract.Validators = governmentContract.NextValidators
		governmentContract.NextValidators = nil
		//governmentContract.NextSwitchHeight = 0
		return true, nil
	}
	return false, nil
}

//execute after block propose,create governance txRWSet,wait to add to block header
//when block commit,governance txRWSet take effect
func CheckAndCreateGovernmentArgs(block *commonPb.Block, store protocol.BlockchainStore, proposalCache protocol.ProposalCache) (*commonPb.TxRWSet, error) {
	log.Debugf("CheckAndCreateGovernmentArgs start")
	gcr := &GovernmentContractImp{
		log:                log,
		store:              store,
		governmentContract: nil,
	}

	governmentContract, err := gcr.GetGovernmentContract()
	if err != nil {
		log.Errorf("getGovernmentContract err!err=%v", err)
		return nil, err
	}

	//genesis block no governance data
	var oldBytesData []byte
	if block.Header.GetBlockHeight() > 1 {
		oldBytesData, err = proto.Marshal(governmentContract)
	}

	if err != nil {
		log.Errorf("proto marshal pbcc failed err!err=%v", err)
		return nil, err
	}
	IsConfigChg := false
	IsValidatorChg := false

	var configChg = false
	//check if chain config change
	if utils.IsConfBlock(block) {
		chainConfig, err := getChainConfigFromBlock(block, proposalCache)
		if err != nil {
			log.Errorf("getChainConfigFromBlock err!err=%v", err)
		} else if chainConfig != nil {
			configChg, err = checkConfigChange(chainConfig, governmentContract)
			if err != nil {
				log.Errorf("CheckConfigChange err!err=%v", err)
				return nil, err
			}
			IsConfigChg = configChg
		}

	}

	//if chain config no change,check if epoch switch
	if !configChg {
		log.Debugf("IsValidatorChg start")
		_, err := TryCreateNextValidators(block, governmentContract)
		if err != nil {
			log.Errorf("TryCreateNextValidators err!err=%v", err)
			return nil, err
		}

		validatorChg, err := TrySwitchNextValidator(block, governmentContract)
		if err != nil {
			log.Errorf("TrySwitchNextValidator err!err=%v", err)
			return nil, err
		}
		IsValidatorChg = validatorChg
	}

	//if governmentContract change,change the governmentContract epochId
	if IsValidatorChg || IsConfigChg {
		log.Debugf("EpochId change!height[%v] old epochid[%v] now epochid[%v]", block.Header.GetBlockHeight(), governmentContract.EpochId, (governmentContract.EpochId + 1))
		governmentContract.EpochId++
	}

	//create TxRWSet
	log.Debugf("governmentContract [%v] height[%v]", governmentContract.String(), block.GetHeader().GetBlockHeight())
	return getGovernmentContractTxRWSet(governmentContract, oldBytesData)
}

func GetProposer(level uint64, NodeProposeRound uint64, validators []*consensusPb.GovernmentMember) (*consensusPb.GovernmentMember, error) {
	if validators == nil || len(validators) == 0 {
		return nil, fmt.Errorf("validators is nil")
	}
	index := (level / NodeProposeRound) % uint64(len(validators))

	newMember := &consensusPb.GovernmentMember{
		Index:  validators[index].Index,
		NodeID: validators[index].NodeID,
	}
	log.Debugf("GetProposer newMember[%v] level[%v]", newMember, level)

	return newMember, nil
}
