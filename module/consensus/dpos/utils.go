/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dpos

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"sort"
	"strings"

	"chainmaker.org/chainmaker/vm-native/v2/dposmgr"

	configPb "chainmaker.org/chainmaker/pb-go/v2/config"
	pbdpos "chainmaker.org/chainmaker/pb-go/v2/consensus/dpos"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/utils/v2"
	"github.com/gogo/protobuf/proto"
	goproto "github.com/gogo/protobuf/proto"
)

// ValidatorsElection select validators from Candidates
func ValidatorsElection(
	infos []*pbdpos.CandidateInfo, n int, seed []byte, outSort bool) ([]*pbdpos.CandidateInfo, error) {
	if n == 0 {
		return nil, fmt.Errorf("can not select zero validators")
	}
	m := len(infos)
	if n > m {
		return nil, fmt.Errorf("the number of candidate is not enough, candidate[%v] validator[%v]", m, n)
	}
	if n == m {
		validators := make([]*pbdpos.CandidateInfo, 0)
		validators = append(validators, infos...)
		return validators, nil
	}
	// when m > n
	// 首先对结果进行排序
	sort.Sort(CandidateInfos(infos))
	// n、m 都分为 两部分，
	var (
		m0, _      = distributionM(m, n)
		n0, n1     = distributionN(m, n)
		validators = make([]*pbdpos.CandidateInfo, 0)
	)
	// 首选从m0个对象中选择n0个结果
	// |m0|after(m0) ~ m1|
	// |n0|n1|
	seedInt := binary.LittleEndian.Uint64(seed)
	rand.Seed(int64(seedInt)) // 设置种子
	selectM0IdxMap := sliceToMap(rand.Perm(m0)[:n0])
	for k := range selectM0IdxMap {
		validators = append(validators, infos[k])
	}
	// 构建新的数组
	newSelectArray := make([]*pbdpos.CandidateInfo, 0)
	for i := 0; i < len(infos); i++ {
		if _, ok := selectM0IdxMap[i]; !ok {
			newSelectArray = append(newSelectArray, infos[i])
		}
	}

	// 从新的数组中选择n1个结果
	// 理论上不需要再排序，因为最开始已经排序过
	rand.Seed(int64(seedInt)) // 设置种子
	selectM1IdxArray := rand.Perm(len(newSelectArray))[:n1]
	for _, v := range selectM1IdxArray {
		validators = append(validators, newSelectArray[v])
	}
	if outSort {
		// 输出需要进行排序
		sort.Sort(CandidateInfos(validators))
	}
	return validators, nil
}

func distributionM(m, n int) (int, int) {
	return n, m - n
}

func distributionN(m, n int) (int, int) {
	n0 := n / 2
	return n0, n - n0
}

func sliceToMap(array []int) map[int]struct{} {
	values := make(map[int]struct{})
	for _, v := range array {
		values[v] = struct{}{}
	}
	return values
}

// CandidateInfos array for sort
type CandidateInfos []*pbdpos.CandidateInfo

func (s CandidateInfos) Len() int {
	return len(s)
}

func (s CandidateInfos) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s CandidateInfos) Less(i, j int) bool {
	// 优先按照weight排序，相同的情况下按照PeerId从小到大排序（字符串）
	wi, wj := utils.NewBigInteger(s[i].Weight), utils.NewBigInteger(s[j].Weight)
	val := wi.Cmp(wj)
	if val == 0 {
		return strings.Compare(s[i].PeerId, s[j].PeerId) < 0
	}
	return val > 0
}

func GetLatestEpochInfo(store protocol.BlockchainStore) (*syscontract.Epoch, error) {
	val, err := store.ReadObject(syscontract.SystemContract_DPOS_STAKE.String(), []byte(dposmgr.KeyCurrentEpoch))
	if err != nil {
		return nil, fmt.Errorf("read contract: %s key: %s, error: %s",
			syscontract.SystemContract_DPOS_STAKE.String(), dposmgr.KeyCurrentEpoch, err)
	}
	epoch := syscontract.Epoch{}
	if err = proto.Unmarshal(val, &epoch); err != nil {
		return nil, fmt.Errorf("unmarshal epoch failed, reason: %s", err)
	}
	return &epoch, nil
}

func GetNodeIDsFromValidators(store protocol.BlockchainStore, validators []string) ([]string, error) {
	if len(validators) == 0 {
		return nil, fmt.Errorf("validators is null")
	}
	nodeIDs := make([]string, 0, len(validators))
	for _, validator := range validators {
		nodeID, err := store.ReadObject(syscontract.SystemContract_DPOS_STAKE.String(), dposmgr.ToNodeIDKey(validator))
		if err != nil || len(nodeID) == 0 {
			return nil, fmt.Errorf("read nodeID of the validator[%s] failed, reason: %s", validator, err)
		}
		nodeIDs = append(nodeIDs, string(nodeID))
	}
	return nodeIDs, nil
}

func GetChainConfig(store protocol.BlockchainStore) (*configPb.ChainConfig, error) {
	var chainConfig configPb.ChainConfig
	bytes, err := store.ReadObject(
		syscontract.SystemContract_CHAIN_CONFIG.String(),
		[]byte(syscontract.SystemContract_CHAIN_CONFIG.String()),
	)
	if err != nil || len(bytes) == 0 {
		return nil, fmt.Errorf("read chainConfig failed, reason: %s", err)
	}
	err = goproto.Unmarshal(bytes, &chainConfig)
	if err != nil {
		err = fmt.Errorf(
			"unmarshal chainConfig failed, contractName %s err: %+v",
			syscontract.SystemContract_CHAIN_CONFIG.String(),
			err,
		)
		return nil, err
	}
	return &chainConfig, nil
}
