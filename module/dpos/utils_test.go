/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dpos

import (
	pbdpos "chainmaker.org/chainmaker-go/pb/protogo/dpos"
	"fmt"
	"github.com/stretchr/testify/require"
	"sort"
	"strings"
	"testing"
)

func TestCandidateInfos(t *testing.T) {
	var tests = []*pbdpos.CandidateInfo {
		{"peer0", "100"},
		{"peer1", "100"},
		{"peer2", "100"},
		{"peer3", "0"},
		{"peer4", "300"},
		{"peer5", "500"},
	}
	sort.Sort(CandidateInfos(tests))
	require.Equal(t, tests[0].Weight, "500")
	require.Equal(t, tests[1].Weight, "300")
	require.Equal(t, tests[2].Weight, "100")
	require.Equal(t, tests[3].Weight, "100")
	require.Equal(t, tests[4].Weight, "100")
	require.Equal(t, tests[5].Weight, "0")
	require.Equal(t, tests[0].PeerID, "peer5")
	require.Equal(t, tests[1].PeerID, "peer4")
	require.Equal(t, tests[2].PeerID, "peer0")
	require.Equal(t, tests[3].PeerID, "peer1")
	require.Equal(t, tests[4].PeerID, "peer2")
	require.Equal(t, tests[5].PeerID, "peer3")
}

func TestValidatorsElection(t *testing.T) {
	var tests = []*pbdpos.CandidateInfo{
		{"peer0", "100"},
		{"peer1", "100"},
		{"peer2", "100"},
		{"peer3", "0"},
		{"peer4", "300"},
		{"peer5", "500"},
		{"peer6", "200"},
		{"peer7", "400"},
		{"peer8", "550"},
		{"peer9", "250"},
		{"peer10", "150"},
		{"peer11", "600"},
		{"peer12", "601"},
		{"peer13", "660"},
		{"peer14", "1000"},
	}
	validators, err := ValidatorsElection(tests, 0, false)
	require.NotNil(t, err)
	require.Nil(t, validators)
	validators, err = ValidatorsElection(tests, len(tests) + 1, false)
	require.NotNil(t, err)
	require.Nil(t, validators)
	validators, err = ValidatorsElection(tests, len(tests), false)
	require.Equal(t, len(validators), len(tests))
	require.Nil(t, err)
	validators, err = ValidatorsElection(tests, 5, false)
	require.Nil(t, err)
	require.Equal(t, len(validators), 5)
	for i := 0; i < len(validators); i++ {
		fmt.Printf("%v -> %s -> %s \n", i + 1, validators[i].PeerID, validators[i].Weight)
	}
	for i := 0; i < 10; i++ {
		fmt.Println("----------------------------------")
		validators, err = ValidatorsElection(tests, i + 1, true)
		require.Nil(t, err)
		require.Equal(t, len(validators), i + 1)
		for i := 0; i < len(validators); i++ {
			fmt.Printf("%v -> %s -> %s \n", i + 1, validators[i].PeerID, validators[i].Weight)
		}
	}
	fmt.Println("----------------------------------")
	validators, err = ValidatorsElection(tests, len(tests) - 1, false)
	require.Nil(t, err)
	require.Equal(t, len(validators), len(tests) - 1)
	var count = 0
	for i := 0; i < len(validators); i++ {
		peerID := validators[i].PeerID
		for j := 0; j < len(tests); j++ {
			if strings.EqualFold(peerID, tests[j].PeerID) {
				count++
				break
			}
		}
		fmt.Printf("%v -> %s -> %s \n", i + 1, validators[i].PeerID, validators[i].Weight)
	}
	require.Equal(t, len(tests) - 1, count)
}