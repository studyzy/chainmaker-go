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

	"chainmaker.org/chainmaker/logger/v2"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	configPb "chainmaker.org/chainmaker/pb-go/v2/config"
	consensusPb "chainmaker.org/chainmaker/pb-go/v2/consensus"
	systemPb "chainmaker.org/chainmaker/pb-go/v2/syscontract"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/utils/v2"
	"github.com/gogo/protobuf/proto"
)

const (
	ConstMinQuorumForQc   = 3     //default min vote num
	ConstTransitBlock     = 0     //default epoch switch block buff
	ConstBlockNumPerEpoch = 10000 //default epoch change height
	ConstValidatorNum     = 4     //default actual consensus node num
	ConstNodeProposeRound = 1     //default continuity propose round
	//GovernanceContractName     = "government_contract"
	MinimumTimeOutMill         = 4000
	MinimumIntervalTimeOutMill = 100

	SkipTimeoutCommit        = "SkipTimeoutCommit"
	CachedLen                = "CachedLen"
	BlockNumPerEpoch         = "BlockNumPerEpoch"
	TransitBlock             = "TransitBlock"
	ValidatorNum             = "ValidatorNum"
	NodeProposeRound         = "NodeProposeRound"
	RoundTimeoutMill         = "HotstuffRoundTimeoutMill"
	RoundTimeoutIntervalMill = "HotstuffRoundTimeoutIntervalMill"

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

	contractName := systemPb.SystemContract_GOVERNANCE.String()
	bz, err := store.ReadObject(contractName, []byte(contractName))
	if err != nil {
		log.Errorf("ReadObject.Get err!contractName=%v,err=%v", contractName, err)
		return nil, err
	}

	if len(bz) == 0 {
		log.Errorf("ReadObject.Get empty!contractName=%v", contractName)
		return nil, fmt.Errorf("bytes is empty")
	}
	var governanceContract consensusPb.GovernanceContract
	err = proto.Unmarshal(bz, &governanceContract)
	if err != nil {
		log.Errorf(UnmarshalErrFmt, err)
		return nil, err
	}

	return &governanceContract, nil
}
func updateGovContractFromConfig(chainConfig *configPb.ChainConfig,
	governanceContract *consensusPb.GovernanceContract) (update bool) {
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
	if newRoundTimeoutIntervalMill != 0 &&
		governanceContract.HotstuffRoundTimeoutIntervalMill != newRoundTimeoutIntervalMill {
		governanceContract.HotstuffRoundTimeoutIntervalMill = newRoundTimeoutIntervalMill
		isChg = true
	}
	return isChg
}

func checkChainConfig(chainConfig *configPb.ChainConfig,
	governanceContract *consensusPb.GovernanceContract) (bool, error) {
	var err error
	nodes := chainConfig.Consensus.Nodes
	tempMap := map[string]int{}

	for _, node := range nodes {
		for _, nid := range node.NodeId {
			if _, ok := tempMap[nid]; !ok {
				tempMap[nid] = 1
			}
		}
	}
	if len(tempMap) < (ConstMinQuorumForQc + 1) {
		return false, fmt.Errorf("set Nodes size is too minimum: %d < %d", len(tempMap), ConstMinQuorumForQc+1)
	}

	conConf := chainConfig.Consensus
	for _, oneConf := range conConf.ExtConfig {
		switch oneConf.Key {
		case CachedLen:
			var cachedLen int64
			cachedLen, err = strconv.ParseInt(string(oneConf.Value), 10, 64)
			if err != nil || cachedLen < 0 {
				return false, fmt.Errorf("set CachedLen err")
			}
		case BlockNumPerEpoch:
			var newBlockNumPerEpoch int64
			newBlockNumPerEpoch, err = strconv.ParseInt(string(oneConf.Value), 10, 64)
			if err != nil {
				return false, fmt.Errorf("set BlockNumPerEpoch err: %s", err)
			}
			if governanceContract.Type == consensusPb.ConsensusType_HOTSTUFF && newBlockNumPerEpoch > 0 {
				return false, fmt.Errorf("set BlockNumPerEpoch err! HOTSTUFF should set <= 0, actual: %d", newBlockNumPerEpoch)
			}
		case RoundTimeoutMill:
			v, err := strconv.ParseUint(string(oneConf.Value), 10, 64)
			if err != nil {
				return false, fmt.Errorf("set %s Parse uint error: %s", RoundTimeoutMill, err)
			}
			if v < MinimumTimeOutMill {
				return false, fmt.Errorf("set %s is too minimum, %d < %d", RoundTimeoutMill, v, MinimumTimeOutMill)
			}
		case RoundTimeoutIntervalMill:
			v, err := strconv.ParseUint(string(oneConf.Value), 10, 64)
			if err != nil {
				return false, fmt.Errorf("set %s Parse uint error: %s", RoundTimeoutIntervalMill, err)
			}
			if v < MinimumIntervalTimeOutMill {
				return false, fmt.Errorf("set %s is too minimum, %d < %d", RoundTimeoutIntervalMill, v, MinimumIntervalTimeOutMill)
			}
		}
	}
	return true, nil
}

//create government data from chainConfig when genesis
func getGovernanceContractFromConfig(chainConfig *configPb.ChainConfig) (*consensusPb.GovernanceContract, error) {
	log.Debugf("get government contract from config file")

	var (
		index         = 0
		nodeIds       []string
		tempMap       = make(map[string]int)
		nodes         = chainConfig.Consensus.Nodes
		consensusType = chainConfig.Consensus.Type
		members       []*consensusPb.GovernanceMember
	)
	// 1. Initializes the members who have the right to participate in the consensus
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
		member := &consensusPb.GovernanceMember{
			Index:  int64(index),
			NodeId: nid,
		}
		members = append(members, member)
		index++
	}
	sort.Sort(indexedGovernanceMember(members))

	// 2. create governanceContract
	governanceContract := &consensusPb.GovernanceContract{
		EpochId:           0,
		Type:              consensusType,
		CurMaxIndex:       int64(index),
		SkipTimeoutCommit: false,
		N:                 uint64(len(members)),
		MinQuorumForQc:    ConstMinQuorumForQc,
		CachedLen:         0,
		NextSwitchHeight:  0,
		TransitBlock:      ConstTransitBlock,
		BlockNumPerEpoch:  0, // 0: disable epoch switch
		ValidatorNum:      uint64(len(members)),
		NodeProposeRound:  ConstNodeProposeRound,
		Members:           members,
		Validators:        nil,
		NextValidators:    nil,
		ConfigSequence:    chainConfig.Sequence,
	}
	updateGovContractFromConfig(chainConfig, governanceContract)

	if governanceContract.N > governanceContract.ValidatorNum {
		governanceContract.N = governanceContract.ValidatorNum
	}
	governanceContract.MinQuorumForQc = (2*governanceContract.N + 1) / 3
	if governanceContract.MinQuorumForQc < ConstMinQuorumForQc {
		log.Errorf("Set minQuorumForQc err!MinQuorumForQc=%v", governanceContract.MinQuorumForQc)
		governanceContract.MinQuorumForQc = ConstMinQuorumForQc
	}

	bytesSeed, _ := proto.Marshal(governanceContract)
	validators, err := createValidators(governanceContract, bytesSeed)
	if err != nil {
		log.Errorf(CreateValidatorsErrFmt, err)
		return nil, err
	}
	governanceContract.Validators = validators
	return governanceContract, nil
}

func getChainConfigFromChainStore(store protocol.BlockchainStore) (*configPb.ChainConfig, error) {
	log.Debugf("get chainConfig from chainStore")
	contractName := systemPb.SystemContract_CHAIN_CONFIG.String()
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
func getRandomList(n, chooseM int64, seed int64) []int64 {
	var (
		result       []int64
		peers        = make([]int64, n)
		i, j, curNum int64
	)
	//if n less M,choose all
	if n <= chooseM {
		for i = 0; i < n; i++ {
			result = append(result, i)
		}
		return result
	}

	//from seed%n
	for i = seed % n; curNum < chooseM; i++ {
		if i >= n {
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

func createValidators(governanceContract *consensusPb.GovernanceContract,
	seedBytes []byte) ([]*consensusPb.GovernanceMember, error) {
	var membersNum = uint64(len(governanceContract.Members))
	if membersNum <= governanceContract.ValidatorNum {
		//if membersNum less ,choose all
		validators := governanceContract.Members
		return validators, nil
	}
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

	seed := uint64(b[7]) | uint64(b[6])<<8 | uint64(b[5])<<16 | uint64(b[4])<<24 |
		uint64(b[3])<<32 | uint64(b[2])<<40 | uint64(b[1])<<48 | uint64(b[0])<<56
	seed = seed % uint64(membersNum)

	//use Josephus problem
	randomList := getRandomList(int64(membersNum), int64(governanceContract.ValidatorNum), int64(seed))
	log.Debugf("createValidators randomList=%v", randomList)
	for _, index := range randomList {
		member := governanceContract.Members[index]
		members = append(members, member)
	}
	sort.Sort(indexedGovernanceMember(members))
	return members, nil
}

func TryCreateNextValidators(block *commonPb.Block, governanceContract *consensusPb.GovernanceContract) (bool, error) {
	if governanceContract.BlockNumPerEpoch <= 0 {
		return false, nil
	}

	height := block.GetHeader().GetBlockHeight()
	if uint64(height)%governanceContract.BlockNumPerEpoch == 0 {
		//use PreBlock hash
		bytesSeed := block.Header.PreBlockHash
		validators, err := createValidators(governanceContract, bytesSeed)
		if err != nil {
			log.Errorf(CreateValidatorsErrFmt, err)
			return false, err
		}
		governanceContract.LastMinQuorumForQc = governanceContract.MinQuorumForQc
		governanceContract.NextValidators = validators
		governanceContract.NextSwitchHeight = uint64(height) + governanceContract.TransitBlock
		log.Debugf("create NextValidators. curHeight=%v,switchHeight=%v", height, governanceContract.NextSwitchHeight)
		return true, nil
	}
	return false, nil
}

//try to check switch epoch
func TrySwitchNextValidator(block *commonPb.Block, governanceContract *consensusPb.GovernanceContract) bool {
	if governanceContract.NextValidators == nil {
		return false
	}
	height := block.GetHeader().GetBlockHeight()
	if governanceContract.NextSwitchHeight == uint64(height) {
		governanceContract.Validators = governanceContract.NextValidators
		governanceContract.NextValidators = nil
		return true
	}
	return false
}

//CheckAndCreateGovernmentArgs execute after block propose,create government txRWSet,wait to add to block header
//when block commit,government txRWSet take effect
func CheckAndCreateGovernmentArgs(block *commonPb.Block, store protocol.BlockchainStore,
	proposalCache protocol.ProposalCache, ledger protocol.LedgerCache) (*commonPb.TxRWSet, error) {
	log.Debugf("CheckAndCreateGovernmentArgs start")

	// 1. get GovernanceContract
	gcr := NewGovernanceContract(store, ledger)
	governanceContract, err := gcr.GetGovernmentContract()
	if err != nil {
		log.Errorf("getGovernanceContract err!err=%v", err)
		return nil, err
	}
	var oldBytesData []byte
	if block.Header.GetBlockHeight() > 1 {
		if oldBytesData, err = proto.Marshal(governanceContract); err != nil {
			log.Errorf("proto marshal pbcc failed err!err=%v", err)
			return nil, err
		}
	}

	var (
		configChg      = false
		isConfigChg    = false
		isValidatorChg = false
	)
	// 2. check if chain config change
	if utils.IsConfBlock(block) {
		var chainConfig *configPb.ChainConfig
		chainConfig, err = getChainConfigFromBlock(block, proposalCache)
		if err != nil {
			log.Errorf("getChainConfigFromBlock err!err=%v", err)
		}
		if chainConfig != nil {
			if configChg, err = updateGovContractByConfig(chainConfig, governanceContract); err != nil {
				log.Errorf("CheckConfigChange err!err=%v", err)
				return nil, err
			}
			isConfigChg = configChg
			governanceContract.NextSwitchHeight = uint64(block.Header.BlockHeight) + governanceContract.TransitBlock
		}
	}

	// 3. if chain config no change,check if epoch switch
	if !configChg {
		log.Debugf("no chain config change, will check epoch switch")
		var isCreate bool
		if isCreate, err = TryCreateNextValidators(block, governanceContract); err != nil {
			log.Errorf("TryCreateNextValidators err!err=%v", err)
			return nil, err
		} else if isCreate {
			isValidatorChg = TrySwitchNextValidator(block, governanceContract)
		} else {
			log.Debugf("no epoch switch ...")
		}
	}

	// 4. if chain config change or switch to next epoch, change the GovernanceContract epochId
	if isValidatorChg || isConfigChg {
		governanceContract.EpochId++
		log.Debugf("EpochId change! block Height[%d], new epochId[%d], isValidatorChg[%v], isConfigChg[%v]",
			block.Header.GetBlockHeight(), governanceContract.EpochId, isValidatorChg, isConfigChg)
	}

	//5. create TxRWSet for GovernanceContract
	txRWSet, err := getGovernanceContractTxRWSet(governanceContract, oldBytesData)
	return txRWSet, err
}

func getChainConfigFromBlock(block *commonPb.Block,
	proposalCache protocol.ProposalCache) (*configPb.ChainConfig, error) {
	if !utils.IsConfBlock(block) {
		log.Errorf("block is not conf block")
		return nil, fmt.Errorf("block is not conf block")
	}

	_, rwSetMap, _ := proposalCache.GetProposedBlock(block)
	if len(rwSetMap) == 0 {
		log.Errorf("rwSetMap is nil")
		return nil, fmt.Errorf("rwSetMap is nil")
	}

	//get from rwSetMap,contract data
	var value []byte
	getChainConfigContractName := systemPb.SystemContract_CHAIN_CONFIG.String()
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

func updateGovContractByConfig(chainConfig *configPb.ChainConfig,
	governanceContract *consensusPb.GovernanceContract) (bool, error) {
	log.Debugf("updateGovContractByConfig start")
	if governanceContract.ConfigSequence == chainConfig.Sequence {
		return false, nil
	}

	isChange := false
	oldMembersMap := make(map[string]*consensusPb.GovernanceMember, len(governanceContract.Members))
	for _, member := range governanceContract.Members {
		oldMembersMap[member.NodeId] = member
	}

	var (
		nodeIds []string
		tempMap = make(map[string]int)
		nodes   = chainConfig.Consensus.Nodes
		index   = governanceContract.CurMaxIndex
		members []*consensusPb.GovernanceMember
	)
	// 1. Initializes the members who have the right to participate in the consensus
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
		//reuse old index
		if member, ok := oldMembersMap[nid]; ok {
			members = append(members, member)
		} else {
			//use new index
			member := &consensusPb.GovernanceMember{
				Index:  index,
				NodeId: nid,
			}
			members = append(members, member)
			index++
		}
	}

	// 2. if change
	isChange = updateGovContractFromConfig(chainConfig, governanceContract)
	if index != governanceContract.CurMaxIndex || len(members) != len(governanceContract.Members) || isChange {
		sort.Sort(indexedGovernanceMember(members))
		isChange = true
		n := len(members)
		//members == validators
		//if n > int(governanceContract.ValidatorNum) {
		//	n = int(governanceContract.ValidatorNum)
		//}

		minQuorumForQc := (2*n + 1) / 3
		if minQuorumForQc < ConstMinQuorumForQc {
			log.Errorf("Set minQuorumForQc err!minQuorumForQc=%v", minQuorumForQc)
			minQuorumForQc = ConstMinQuorumForQc
		}

		governanceContract.LastMinQuorumForQc = governanceContract.MinQuorumForQc
		governanceContract.CurMaxIndex = index
		governanceContract.N = uint64(n)
		governanceContract.MinQuorumForQc = uint64(minQuorumForQc)
		governanceContract.Members = members
		governanceContract.NextValidators = nil
		governanceContract.ValidatorNum = uint64(n)

		bytesSeed, _ := proto.Marshal(governanceContract)
		validators, err := createValidators(governanceContract, bytesSeed)
		if err != nil {
			log.Errorf(CreateValidatorsErrFmt, err)
			return false, err
		}
		governanceContract.Validators = validators
	}
	governanceContract.ConfigSequence = chainConfig.Sequence
	log.Debugf("updateGovContractByConfig end.isChange=%v", isChange)
	return isChange, nil
}

func getGovernanceContractTxRWSet(governanceContract *consensusPb.GovernanceContract,
	oldBytes []byte) (*commonPb.TxRWSet, error) {
	txRWSet := &commonPb.TxRWSet{
		TxId:     systemPb.SystemContract_GOVERNANCE.String(),
		TxReads:  make([]*commonPb.TxRead, 0),
		TxWrites: make([]*commonPb.TxWrite, 0, 1),
	}

	var (
		err          error
		pbccPayload  []byte
		contractName = systemPb.SystemContract_GOVERNANCE.String()
	)
	log.Debugf("begin getGovernanceContractTxRWSet ...")
	// 1. check for changes
	if pbccPayload, err = proto.Marshal(governanceContract); err != nil {
		log.Error(err)
		return nil, fmt.Errorf("proto marshal pbcc failed, %s", err.Error())
	}
	if bytes.Equal(oldBytes, pbccPayload) {
		log.Debugf("governanceContract no change")
		return txRWSet, nil
	}

	// 2. create txRWSet for the new government contract
	var oldGovernanceContract consensusPb.GovernanceContract
	if err = proto.Unmarshal(oldBytes, &oldGovernanceContract); err != nil {
		return nil, err
	}
	log.Debugf("governanceContract change older contract:[%s]", oldGovernanceContract.String())
	txWrite := &commonPb.TxWrite{
		Key:          []byte(contractName),
		Value:        pbccPayload,
		ContractName: contractName,
	}
	txRWSet.TxWrites = append(txRWSet.TxWrites, txWrite)
	return txRWSet, nil
}

func GetProposer(level uint64, nodeProposeRound uint64,
	validators []*consensusPb.GovernanceMember) (*consensusPb.GovernanceMember, error) {
	if len(validators) == 0 {
		return nil, fmt.Errorf("validators is nil")
	}
	index := (level / nodeProposeRound) % uint64(len(validators))
	newMember := &consensusPb.GovernanceMember{
		Index:  validators[index].Index,
		NodeId: validators[index].NodeId,
	}
	log.Debugf("GetProposer newMember[%v] level[%v]", newMember, level)
	return newMember, nil
}
