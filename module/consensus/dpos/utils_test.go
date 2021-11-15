/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dpos

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	"chainmaker.org/chainmaker/vm-native/v2/dposmgr"

	pbdpos "chainmaker.org/chainmaker/pb-go/v2/consensus/dpos"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"
	"chainmaker.org/chainmaker/protocol/v2/mock"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestCandidateInfos(t *testing.T) {
	var tests = []*pbdpos.CandidateInfo{
		{PeerId: "peer0", Weight: "100"},
		{PeerId: "peer1", Weight: "100"},
		{PeerId: "peer2", Weight: "100"},
		{PeerId: "peer3", Weight: "0"},
		{PeerId: "peer4", Weight: "300"},
		{PeerId: "peer5", Weight: "500"},
	}
	sort.Sort(CandidateInfos(tests))
	require.Equal(t, tests[0].Weight, "500")
	require.Equal(t, tests[1].Weight, "300")
	require.Equal(t, tests[2].Weight, "100")
	require.Equal(t, tests[3].Weight, "100")
	require.Equal(t, tests[4].Weight, "100")
	require.Equal(t, tests[5].Weight, "0")
	require.Equal(t, tests[0].PeerId, "peer5")
	require.Equal(t, tests[1].PeerId, "peer4")
	require.Equal(t, tests[2].PeerId, "peer0")
	require.Equal(t, tests[3].PeerId, "peer1")
	require.Equal(t, tests[4].PeerId, "peer2")
	require.Equal(t, tests[5].PeerId, "peer3")
}

func TestValidatorsElection2(t *testing.T) {
	var candidates = []*pbdpos.CandidateInfo{
		{PeerId: "3tPTFsFAYjtEkJa3MYfV63TK2uYP9e3DZq97KrZMYxhy", Weight: "25000000000000000000000"},
		{PeerId: "4WUXfiUpLkx7meaNu8TNS5rNM7YtZk6fkNWXihc54PbM", Weight: "250000000000000000000000"},
		{PeerId: "4yKy5YebxygcXuid6F2vnfMhpTL94qbJELLodbCMg1Tn", Weight: "250000000000000000000000"},
		{PeerId: "AwLW3zpAsmhMDMqp1DkCCFajh9pTTXHcpeBRZybRTF2X", Weight: "250000000000000000000000"},
		{PeerId: "3BugkfMLdgXsif1Zg9sCwi4BBxFxqdjEQNjCmYgtGAtr", Weight: "250000000000000000000000"},
	}
	seed := make([]byte, 32)
	num, err := hex.Decode(seed, []byte("0efdfa8a4db5715fd03fa0ace3c01ca09e19b15a98e78cd05c09983921880282"))
	require.NoError(t, err)
	require.EqualValues(t, num, 32)
	vals, err := ValidatorsElection(candidates, 4, seed, true)
	require.NoError(t, err)
	for _, v := range vals {
		fmt.Println(v)
	}
	for i := 0; i < 3; i++ {
		time.Sleep(time.Second * 2)
		tmp, err := ValidatorsElection(candidates, 4, seed, true)
		require.NoError(t, err)
		for i, v := range vals {
			if !strings.EqualFold(v.String(), tmp[i].String()) {
				fmt.Println("expect: ", vals)
				fmt.Println("actual: ", tmp)
				require.False(t, true)
			}
		}
	}
}

func TestValidatorsElection(t *testing.T) {
	var tests = []*pbdpos.CandidateInfo{
		{PeerId: "peer0", Weight: "100"},
		{PeerId: "peer1", Weight: "100"},
		{PeerId: "peer2", Weight: "100"},
		{PeerId: "peer3", Weight: "0"},
		{PeerId: "peer4", Weight: "300"},
		{PeerId: "peer5", Weight: "500"},
		{PeerId: "peer6", Weight: "200"},
		{PeerId: "peer7", Weight: "400"},
		{PeerId: "peer8", Weight: "550"},
		{PeerId: "peer9", Weight: "250"},
		{PeerId: "peer10", Weight: "150"},
		{PeerId: "peer11", Weight: "600"},
		{PeerId: "peer12", Weight: "601"},
		{PeerId: "peer13", Weight: "660"},
		{PeerId: "peer14", Weight: "1000"},
	}
	seed := make([]byte, 32)
	_, _ = rand.Read(seed)
	validators, err := ValidatorsElection(tests, 0, seed, false)
	require.NotNil(t, err)
	require.Nil(t, validators)
	validators, err = ValidatorsElection(tests, len(tests)+1, seed, false)
	require.NotNil(t, err)
	require.Nil(t, validators)
	validators, err = ValidatorsElection(tests, len(tests), seed, false)
	require.Equal(t, len(validators), len(tests))
	require.Nil(t, err)
	validators, err = ValidatorsElection(tests, 5, seed, false)
	require.Nil(t, err)
	require.Equal(t, len(validators), 5)
	for i := 0; i < len(validators); i++ {
		fmt.Printf("%v -> %s -> %s \n", i+1, validators[i].PeerId, validators[i].Weight)
	}
	for i := 0; i < 10; i++ {
		fmt.Println("----------------------------------")
		validators, err = ValidatorsElection(tests, i+1, seed, true)
		require.Nil(t, err)
		require.Equal(t, len(validators), i+1)
		for i := 0; i < len(validators); i++ {
			fmt.Printf("%v -> %s -> %s \n", i+1, validators[i].PeerId, validators[i].Weight)
		}
	}
	fmt.Println("----------------------------------")
	validators, err = ValidatorsElection(tests, len(tests)-1, seed, false)
	require.Nil(t, err)
	require.Equal(t, len(validators), len(tests)-1)
	var count = 0
	for i := 0; i < len(validators); i++ {
		peerId := validators[i].PeerId
		for j := 0; j < len(tests); j++ {
			if strings.EqualFold(peerId, tests[j].PeerId) {
				count++
				break
			}
		}
		fmt.Printf("%v -> %s -> %s \n", i+1, validators[i].PeerId, validators[i].Weight)
	}
	require.Equal(t, len(tests)-1, count)

	validators, err = ValidatorsElection(tests, 5, seed, true)
	require.Nil(t, err)
	require.Equal(t, len(validators), 5)
	for i := 0; i < 500; i++ {
		tmp, err := ValidatorsElection(tests, 5, seed, true)
		require.NoError(t, err)
		for i, v := range validators {
			if !strings.EqualFold(v.String(), tmp[i].String()) {
				fmt.Println("expect: ", validators)
				fmt.Println("actual: ", tmp)
				require.False(t, true)
			}
		}
		//require.EqualValues(t, validators, tmp)
	}
}

//func TestRandPerm(t *testing.T) {
//	for i := 0; i < 1000; i++ {
//		rand.Seed(time.Now().Unix() + int64(i*20)) // 设置种子
//		randSlice := rand.Perm(20)[:8]
//		hasSeen := make(map[int]struct{}, len(randSlice))
//		for _, v := range randSlice {
//			if _, ok := hasSeen[v]; ok {
//				require.False(t, ok, "should not be repetition in randSlice")
//			} else {
//				hasSeen[v] = struct{}{}
//			}
//		}
//		fmt.Println(randSlice)
//	}
//}

func TestGetLatestEpochInfo(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := mock.NewMockBlockchainStore(ctrl)
	mockStore.EXPECT().ReadObject(gomock.Any(), gomock.Any()).DoAndReturn(func(contractName string, key []byte) ([]byte, error) {
		epoch := &syscontract.Epoch{EpochId: 100, NextEpochCreateHeight: 990, ProposerVector: []string{
			"vector1", "vector2", "vector3", "vector4"}}
		//return proto.Marshal(epoch)
		return epoch.Marshal()
	}).AnyTimes()
	epoch, err := GetLatestEpochInfo(mockStore)
	require.NoError(t, err)
	require.EqualValues(t, epoch.EpochId, 100)
	require.EqualValues(t, epoch.NextEpochCreateHeight, 990)
	require.EqualValues(t, epoch.ProposerVector, []string{
		"vector1", "vector2", "vector3", "vector4",
	})
}

func TestGetNodeIDsFromValidators(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	name := syscontract.SystemContract_DPOS_STAKE.String()
	nodeIDs := make(map[string]string)
	nodeIDs[name+string(dposmgr.ToNodeIDKey("val1"))] = "nodeId1"
	nodeIDs[name+string(dposmgr.ToNodeIDKey("val2"))] = "nodeId2"
	nodeIDs[name+string(dposmgr.ToNodeIDKey("val3"))] = "nodeId3"

	mockStore := mock.NewMockBlockchainStore(ctrl)
	mockStore.EXPECT().ReadObject(gomock.Any(), gomock.Any()).DoAndReturn(func(contractName string, key []byte) ([]byte, error) {
		val, exist := nodeIDs[contractName+string(key)]
		if exist {
			return []byte(val), nil
		}
		return nil, fmt.Errorf("not find key: %s in contract: %s", key, contractName)
	}).AnyTimes()
	ids, err := GetNodeIDsFromValidators(mockStore, []string{"val1", "val2", "val3"})
	require.NoError(t, err)
	require.EqualValues(t, ids, []string{"nodeId1", "nodeId2", "nodeId3"})
}
