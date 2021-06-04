/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dpos

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"time"

	pbdpos "chainmaker.org/chainmaker-go/pb/protogo/dpos"
	"chainmaker.org/chainmaker-go/utils"
)

// ValidatorsElection select validators from Candidates
func ValidatorsElection(infos []*pbdpos.CandidateInfo, n int, outSort bool) ([]*pbdpos.CandidateInfo, error) {
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
	rand.Seed(time.Now().Unix()) // 设置种子
	selectM0IdxMap := sliceToMap(rand.Perm(m0)[:n0])
	for k, _ := range selectM0IdxMap {
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
	rand.Seed(time.Now().Unix()) // 设置种子
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
	values := make(map[int]struct{}, 0)
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
	// 优先按照weight排序，相同的情况下按照peerID从小到大排序（字符串）
	wi, wj := utils.NewBigInteger(s[i].Weight), utils.NewBigInteger(s[j].Weight)
	if val := wi.Cmp(wj); val == 0 {
		return strings.Compare(s[i].PeerID, s[j].PeerID) < 0
	} else {
		return val > 0
	}
}
