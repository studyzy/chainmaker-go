/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package accesscontrol

import (
	"fmt"
	"testing"
	"time"

	logger2 "chainmaker.org/chainmaker-go/logger"
	"chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	pbac "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"
	"chainmaker.org/chainmaker/protocol/v2"
	"github.com/stretchr/testify/require"
)

func TestGetMemberStatus(t *testing.T) {
	logger := logger2.GetLogger(logger2.MODULE_ACCESS)
	certProvider, err := newCertACProvider(testChainConfig, testOrg1, nil, logger)
	require.Nil(t, err)
	require.NotNil(t, certProvider)

	pbMember := &pbac.Member{
		OrgId:      testOrg1,
		MemberType: pbac.MemberType_CERT,
		MemberInfo: []byte(testConsensusSignOrg1.cert),
	}

	memberStatus, err := certProvider.GetMemberStatus(pbMember)
	require.Nil(t, err)
	require.Equal(t, pbac.MemberStatus_NORMAL, memberStatus)
}

func TestVerifyPrincipal(t *testing.T) {
	hashType := testHashType
	_, cleanFunc, err := createTempDirWithCleanFunc()
	require.Nil(t, err)
	defer cleanFunc()

	var orgMemberMap = make(map[string]*orgMember, len(orgMemberInfoMap))
	for orgId, info := range orgMemberInfoMap {
		orgMemberMap[orgId] = initOrgMember(t, info)
	}

	tests := []struct {
		// give
		orgMemberMap    map[string]*orgMember
		currentOrgId    string
		acProviderOrgId string
		hashType        string
		msg             string
		resourceName    string
		signType        string
		// want
		wantErr          error
		wantVerifyResult bool
	}{
		{ // read
			orgMemberMap:     orgMemberMap,
			currentOrgId:     testOrg1,
			acProviderOrgId:  testOrg2,
			hashType:         testHashType,
			msg:              testMsg,
			resourceName:     common.TxType_QUERY_CONTRACT.String(),
			signType:         common.TxType_QUERY_CONTRACT.String(),
			wantErr:          nil,
			wantVerifyResult: true,
		},
		{ // read invalid
			orgMemberMap:     orgMemberMap,
			currentOrgId:     testOrg5,
			acProviderOrgId:  testOrg2,
			hashType:         testHashType,
			msg:              testMsg,
			resourceName:     common.TxType_QUERY_CONTRACT.String(),
			signType:         common.TxType_QUERY_CONTRACT.String(),
			wantErr:          nil,
			wantVerifyResult: false,
		},
		{ // write
			orgMemberMap:     orgMemberMap,
			currentOrgId:     testOrg4,
			acProviderOrgId:  testOrg2,
			hashType:         testHashType,
			msg:              testMsg,
			resourceName:     protocol.ResourceNameWriteData,
			signType:         protocol.ResourceNameWriteData,
			wantErr:          nil,
			wantVerifyResult: true,
		},
		{ // write invalid
			orgMemberMap:     orgMemberMap,
			currentOrgId:     testOrg5,
			acProviderOrgId:  testOrg2,
			hashType:         testHashType,
			msg:              testMsg,
			resourceName:     protocol.ResourceNameWriteData,
			signType:         protocol.ResourceNameWriteData,
			wantErr:          nil,
			wantVerifyResult: false,
		},
		{ // P2P
			orgMemberMap:     orgMemberMap,
			currentOrgId:     testOrg1,
			acProviderOrgId:  testOrg2,
			hashType:         testHashType,
			msg:              testMsg,
			resourceName:     protocol.ResourceNameP2p,
			signType:         string(protocol.RoleConsensusNode),
			wantErr:          nil,
			wantVerifyResult: true,
		},
		{ // P2P invalid ?
			orgMemberMap:     orgMemberMap,
			currentOrgId:     testOrg1,
			acProviderOrgId:  testOrg2,
			hashType:         testHashType,
			msg:              testMsg,
			resourceName:     protocol.ResourceNameP2p,
			signType:         string(protocol.RoleAdmin),
			wantErr:          nil,
			wantVerifyResult: false,
		},
		{ // consensus
			orgMemberMap:     orgMemberMap,
			currentOrgId:     testOrg1,
			acProviderOrgId:  testOrg2,
			hashType:         testHashType,
			msg:              testMsg,
			resourceName:     protocol.ResourceNameConsensusNode,
			signType:         string(protocol.RoleConsensusNode),
			wantErr:          nil,
			wantVerifyResult: true,
		},
		{ // consensus invalid
			orgMemberMap:     orgMemberMap,
			currentOrgId:     testOrg1,
			acProviderOrgId:  testOrg2,
			hashType:         testHashType,
			msg:              testMsg,
			resourceName:     protocol.ResourceNameConsensusNode,
			signType:         string(protocol.RoleClient),
			wantErr:          nil,
			wantVerifyResult: false,
		},
		//{ // self
		//	orgMemberMap:     orgMemberMap,
		//	currentOrgId:     testOrg4,
		//	acProviderOrgId:  testOrg2,
		//	hashType:         testHashType,
		//	msg:              testMsg,
		//	resourceName:     protocol.ResourceNameUpdateSelfConfig,
		//	signType:         string(protocol.RoleAdmin),
		//	wantErr:          nil,
		//	wantVerifyResult: true,
		//},
		//{ // self invalid
		//	orgMemberMap:     orgMemberMap,
		//	currentOrgId:     testOrg4,
		//	acProviderOrgId:  testOrg3,
		//	hashType:         testHashType,
		//	msg:              testMsg,
		//	resourceName:     protocol.ResourceNameUpdateSelfConfig,
		//	signType:         string(protocol.RoleAdmin),
		//	wantErr:          nil,
		//	wantVerifyResult: false,
		//},
	}

	for _, test := range tests {
		endorsementResourceZephyrus, err := createEndorsementEntry(test.orgMemberMap, test.signType, test.currentOrgId, test.hashType, test.msg)
		require.Nil(t, err)
		currentPrincipal, err := createCurrentPrincipal(test.orgMemberMap, test.resourceName, test.acProviderOrgId, endorsementResourceZephyrus)
		require.Nil(t, err)
		verifyResult, validEndorsements, err := testVerifyCurrentPrincipal(test.orgMemberMap, test.acProviderOrgId, currentPrincipal)
		if test.wantVerifyResult {
			require.Nil(t, err)
			require.Equal(t, test.wantVerifyResult, verifyResult)
			require.Equal(t, endorsementResourceZephyrus.String(), validEndorsements[0].String())
		} else {
			require.NotNil(t, err)
			require.Equal(t, test.wantVerifyResult, verifyResult)
		}

	}

	// read
	sigRead, err := orgMemberMap[testOrg1].client.Sign(hashType, []byte(testMsg))
	require.Nil(t, err)
	signerRead, err := orgMemberMap[testOrg1].client.GetMember()
	require.Nil(t, err)
	endorsementReadZephyrus := &common.EndorsementEntry{
		Signer:    signerRead,
		Signature: sigRead,
	}

	// self
	sigSelf, err := orgMemberMap[testOrg4].admin.Sign(hashType, []byte(testMsg))
	require.Nil(t, err)
	signerSelf, err := orgMemberMap[testOrg4].admin.GetMember()
	require.Nil(t, err)
	endorsementSelf := &common.EndorsementEntry{
		Signer:    signerSelf,
		Signature: sigSelf,
	}
	principalSelf, err := orgMemberMap[testOrg2].acProvider.CreatePrincipalForTargetOrg(protocol.ResourceNameUpdateSelfConfig, []*common.EndorsementEntry{endorsementSelf}, []byte(testMsg), testOrg4)
	require.Nil(t, err)
	ok, err := orgMemberMap[testOrg2].acProvider.VerifyPrincipal(principalSelf)
	require.Nil(t, err)
	require.Equal(t, true, ok)

	// invalid
	sigSelf, err = orgMemberMap[testOrg4].admin.Sign(hashType, []byte(testMsg))
	require.Nil(t, err)
	signerSelf, err = orgMemberMap[testOrg4].admin.GetMember()
	require.Nil(t, err)
	endorsementSelf = &common.EndorsementEntry{
		Signer:    signerSelf,
		Signature: sigSelf,
	}
	principalSelf, err = orgMemberMap[testOrg2].acProvider.CreatePrincipalForTargetOrg(protocol.ResourceNameUpdateSelfConfig, []*common.EndorsementEntry{endorsementSelf}, []byte(testMsg), testOrg3)
	require.Nil(t, err)
	ok, err = orgMemberMap[testOrg2].acProvider.VerifyPrincipal(principalSelf)
	require.NotNil(t, err)
	require.Equal(t, false, ok)

	// majority
	sigEurus, err := orgMemberMap[testOrg4].admin.Sign(hashType, []byte(testMsg))
	require.Nil(t, err)
	signerEurus, err := orgMemberMap[testOrg4].admin.GetMember()
	require.Nil(t, err)
	endorsementEurus := &common.EndorsementEntry{
		Signer:    signerEurus,
		Signature: sigEurus,
	}
	sigAuster, err := orgMemberMap[testOrg3].admin.Sign(hashType, []byte(testMsg))
	require.Nil(t, err)
	signerAuster, err := orgMemberMap[testOrg3].admin.GetMember()
	require.Nil(t, err)
	endorsementAuster := &common.EndorsementEntry{
		Signer:    signerAuster,
		Signature: sigAuster,
	}
	sigZephyrus, err := orgMemberMap[testOrg1].admin.Sign(hashType, []byte(testMsg))
	require.Nil(t, err)
	signerZephyrus, err := orgMemberMap[testOrg1].admin.GetMember()
	require.Nil(t, err)
	endorsementZephyrus := &common.EndorsementEntry{
		Signer:    signerZephyrus,
		Signature: sigZephyrus,
	}
	sigBoreas, err := orgMemberMap[testOrg2].admin.Sign(hashType, []byte(testMsg))
	require.Nil(t, err)
	signerBoreas, err := orgMemberMap[testOrg2].admin.GetMember()
	require.Nil(t, err)
	endorsementBoreas := &common.EndorsementEntry{
		Signer:    signerBoreas,
		Signature: sigBoreas,
	}
	principalMajority, err := orgMemberMap[testOrg2].acProvider.CreatePrincipal(protocol.ResourceNameUpdateConfig, []*common.EndorsementEntry{endorsementAuster, endorsementBoreas, endorsementZephyrus, endorsementEurus}, []byte(testMsg))
	require.Nil(t, err)
	ok, err = orgMemberMap[testOrg2].acProvider.VerifyPrincipal(principalMajority)
	require.Nil(t, err)
	require.Equal(t, true, ok)

	validEndorsements, err := orgMemberMap[testOrg2].acProvider.(*certACProvider).GetValidEndorsements(principalMajority)
	require.Nil(t, err)
	require.Equal(t, len(validEndorsements), 4)

	principalMajority, err = orgMemberMap[testOrg2].acProvider.CreatePrincipal(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+syscontract.ChainConfigFunction_CONSENSUS_EXT_ADD.String(), []*common.EndorsementEntry{endorsementAuster, endorsementBoreas, endorsementZephyrus}, []byte(testMsg))
	require.Nil(t, err)
	ok, err = orgMemberMap[testOrg2].acProvider.VerifyPrincipal(principalMajority)
	require.Nil(t, err)
	require.Equal(t, true, ok)
	validEndorsements, err = orgMemberMap[testOrg2].acProvider.(*certACProvider).GetValidEndorsements(principalMajority)
	require.Nil(t, err)
	require.Equal(t, len(validEndorsements), 3)
	require.Equal(t, endorsementAuster.String(), validEndorsements[0].String())
	require.Equal(t, endorsementBoreas.String(), validEndorsements[1].String())
	require.Equal(t, endorsementZephyrus.String(), validEndorsements[2].String())

	// abnormal
	sigThuellai, err := orgMemberMap[testOrg5].admin.Sign(hashType, []byte(testMsg))
	require.Nil(t, err)
	signerThuellai, err := orgMemberMap[testOrg5].admin.GetMember()
	require.Nil(t, err)
	endorsementThuellai := &common.EndorsementEntry{
		Signer:    signerThuellai,
		Signature: sigThuellai,
	}
	principalMajority, err = orgMemberMap[testOrg2].acProvider.CreatePrincipal(protocol.ResourceNameUpdateConfig, []*common.EndorsementEntry{endorsementAuster, endorsementBoreas, endorsementThuellai, endorsementZephyrus, endorsementEurus}, []byte(testMsg))
	require.Nil(t, err)
	ok, err = orgMemberMap[testOrg2].acProvider.VerifyPrincipal(principalMajority)
	require.Nil(t, err)
	require.Equal(t, true, ok)
	validEndorsements, err = orgMemberMap[testOrg2].acProvider.(*certACProvider).GetValidEndorsements(principalMajority)
	require.Nil(t, err)
	require.Equal(t, len(validEndorsements), 4)

	// invalid
	principalMajority, err = orgMemberMap[testOrg2].acProvider.CreatePrincipal(protocol.ResourceNameUpdateConfig, []*common.EndorsementEntry{endorsementAuster, endorsementBoreas, endorsementThuellai, endorsementAuster}, []byte(testMsg))
	require.Nil(t, err)
	ok, err = orgMemberMap[testOrg2].acProvider.VerifyPrincipal(principalMajority)
	require.NotNil(t, err)
	require.Equal(t, false, ok)
	validEndorsements, err = orgMemberMap[testOrg2].acProvider.(*certACProvider).GetValidEndorsements(principalMajority)
	require.Nil(t, err)
	require.Equal(t, len(validEndorsements), 2)
	// all
	principalAll, err := orgMemberMap[testOrg2].acProvider.CreatePrincipal(protocol.ResourceNameAllTest, []*common.EndorsementEntry{endorsementAuster, endorsementBoreas, endorsementZephyrus, endorsementEurus}, []byte(testMsg))
	require.Nil(t, err)
	ok, err = orgMemberMap[testOrg2].acProvider.VerifyPrincipal(principalAll)
	require.Nil(t, err)
	require.Equal(t, true, ok)
	// abnormal
	principalAll, err = orgMemberMap[testOrg2].acProvider.CreatePrincipal(protocol.ResourceNameAllTest, []*common.EndorsementEntry{endorsementAuster, endorsementBoreas, endorsementZephyrus, endorsementEurus, endorsementThuellai}, []byte(testMsg))
	require.Nil(t, err)
	ok, err = orgMemberMap[testOrg2].acProvider.VerifyPrincipal(principalAll)
	require.Nil(t, err)
	require.Equal(t, true, ok)
	// invalid
	principalAll, err = orgMemberMap[testOrg2].acProvider.CreatePrincipal(protocol.ResourceNameAllTest, []*common.EndorsementEntry{endorsementBoreas, endorsementZephyrus, endorsementEurus}, []byte(testMsg))
	require.Nil(t, err)
	ok, err = orgMemberMap[testOrg2].acProvider.VerifyPrincipal(principalAll)
	require.NotNil(t, err)
	require.Equal(t, false, ok)

	// threshold
	policyLimit, err := orgMemberMap[testOrg2].acProvider.CreatePrincipal("test_2", []*common.EndorsementEntry{endorsementAuster, endorsementZephyrus}, []byte(testMsg))
	require.Nil(t, err)
	ok, err = orgMemberMap[testOrg2].acProvider.VerifyPrincipal(policyLimit)
	require.Nil(t, err)
	require.Equal(t, true, ok)
	policyLimit, err = orgMemberMap[testOrg2].acProvider.CreatePrincipal("test_2_admin", []*common.EndorsementEntry{endorsementAuster, endorsementZephyrus}, []byte(testMsg))
	require.Nil(t, err)
	ok, err = orgMemberMap[testOrg2].acProvider.VerifyPrincipal(policyLimit)
	require.Nil(t, err)
	require.Equal(t, true, ok)
	// invalid
	policyLimit, err = orgMemberMap[testOrg2].acProvider.CreatePrincipal("test_2", []*common.EndorsementEntry{endorsementAuster, endorsementThuellai}, []byte(testMsg))
	require.Nil(t, err)
	ok, err = orgMemberMap[testOrg2].acProvider.VerifyPrincipal(policyLimit)
	require.NotNil(t, err)
	require.Equal(t, false, ok)
	policyLimit, err = orgMemberMap[testOrg2].acProvider.CreatePrincipal("test_2_admin", []*common.EndorsementEntry{endorsementAuster, endorsementReadZephyrus}, []byte(testMsg))
	require.Nil(t, err)
	ok, err = orgMemberMap[testOrg2].acProvider.VerifyPrincipal(policyLimit)
	require.NotNil(t, err)
	require.Equal(t, false, ok)
	// portion
	policyPortion, err := orgMemberMap[testOrg2].acProvider.CreatePrincipal("test_3/4", []*common.EndorsementEntry{endorsementAuster, endorsementReadZephyrus, endorsementEurus}, []byte(testMsg))
	require.Nil(t, err)
	ok, err = orgMemberMap[testOrg2].acProvider.VerifyPrincipal(policyPortion)
	require.Nil(t, err)
	require.Equal(t, true, ok)
	policyPortion, err = orgMemberMap[testOrg2].acProvider.CreatePrincipal("test_3/4_admin", []*common.EndorsementEntry{endorsementAuster, endorsementZephyrus, endorsementEurus}, []byte(testMsg))
	require.Nil(t, err)
	ok, err = orgMemberMap[testOrg2].acProvider.VerifyPrincipal(policyPortion)
	require.Nil(t, err)
	require.Equal(t, true, ok)
	// invalid
	policyPortion, err = orgMemberMap[testOrg2].acProvider.CreatePrincipal("test_3/4", []*common.EndorsementEntry{endorsementAuster, endorsementAuster, endorsementBoreas, endorsementThuellai}, []byte(testMsg))
	require.Nil(t, err)
	ok, err = orgMemberMap[testOrg2].acProvider.VerifyPrincipal(policyPortion)
	require.NotNil(t, err)
	require.Equal(t, false, ok)
	policyPortion, err = orgMemberMap[testOrg2].acProvider.CreatePrincipal("test_3/4_admin", []*common.EndorsementEntry{endorsementAuster, endorsementReadZephyrus, endorsementEurus}, []byte(testMsg))
	require.Nil(t, err)
	ok, err = orgMemberMap[testOrg2].acProvider.VerifyPrincipal(policyPortion)
	require.NotNil(t, err)
	require.Equal(t, false, ok)
	// bench
	var timeStart, timeEnd int64
	count := int64(100)
	// any
	sigAny, err := orgMemberMap[testOrg1].client.Sign(hashType, []byte(testMsg))
	require.Nil(t, err)
	signerAny, err := orgMemberMap[testOrg1].client.GetMember()
	require.Nil(t, err)
	endorsementAny := &common.EndorsementEntry{
		Signer:    signerAny,
		Signature: sigAny,
	}
	principalAny, err := orgMemberMap[testOrg2].acProvider.CreatePrincipal(common.TxType_QUERY_CONTRACT.String(), []*common.EndorsementEntry{endorsementAny}, []byte(testMsg))
	require.Nil(t, err)
	timeStart = time.Now().UnixNano()
	for i := 0; i < int(count); i++ {
		ok, err = orgMemberMap[testOrg2].acProvider.VerifyPrincipal(principalAny)
	}
	timeEnd = time.Now().UnixNano()
	require.Nil(t, err)
	require.Equal(t, true, ok)
	fmt.Printf("Verify ANY average time (over %d runs in nanoseconds): %d\n", count, (timeEnd-timeStart)/count)
	// self
	principalRead, err := orgMemberMap[testOrg2].acProvider.CreatePrincipalForTargetOrg(protocol.ResourceNameUpdateSelfConfig, []*common.EndorsementEntry{endorsementEurus}, []byte(testMsg), testOrg4)
	require.Nil(t, err)
	timeStart = time.Now().UnixNano()
	for i := 0; i < int(count); i++ {
		ok, err = orgMemberMap[testOrg2].acProvider.VerifyPrincipal(principalRead)
	}
	timeEnd = time.Now().UnixNano()
	require.Nil(t, err)
	require.Equal(t, true, ok)
	fmt.Printf("Verify SELF average time (over %d runs in nanoseconds): %d\n", count, (timeEnd-timeStart)/count)
	// consensus
	sigZephyrusConsensus, err := orgMemberMap[testOrg1].consensus.Sign(hashType, []byte(testMsg))
	require.Nil(t, err)
	signerZephyrusConsensus, err := orgMemberMap[testOrg1].consensus.GetMember()
	require.Nil(t, err)
	endorsementZephyrusConsensus := &common.EndorsementEntry{
		Signer:    signerZephyrusConsensus,
		Signature: sigZephyrusConsensus,
	}
	policyConsensus, err := orgMemberMap[testOrg2].acProvider.CreatePrincipal(protocol.ResourceNameConsensusNode, []*common.EndorsementEntry{endorsementZephyrusConsensus}, []byte(testMsg))
	require.Nil(t, err)
	timeStart = time.Now().UnixNano()
	for i := 0; i < int(count); i++ {
		ok, err = orgMemberMap[testOrg2].acProvider.VerifyPrincipal(policyConsensus)
	}
	timeEnd = time.Now().UnixNano()
	require.Nil(t, err)
	require.Equal(t, true, ok)
	fmt.Printf("Verify CONSENSUS average time (over %d runs in nanoseconds): %d\n", count, (timeEnd-timeStart)/count)
}

func createCurrentPrincipal(orgMemberMap map[string]*orgMember, resourceName, acProviderOrgId string, endorsementResourceZephyrus ...*common.EndorsementEntry) (protocol.Principal, error) {
	var endorsementResourceZephyrusList []*common.EndorsementEntry
	endorsementResourceZephyrusList = append(endorsementResourceZephyrusList, endorsementResourceZephyrus...)
	resourcePrincipal, err := orgMemberMap[acProviderOrgId].acProvider.CreatePrincipal(resourceName, endorsementResourceZephyrusList, []byte(testMsg))
	return resourcePrincipal, err
}

func createEndorsementEntry(orgMemberMap map[string]*orgMember, signType, currentOrgId, hashType, msg string) (*common.EndorsementEntry, error) {
	var (
		sigResource    []byte
		err            error
		signerResource *accesscontrol.Member
	)
	switch signType {
	case string(protocol.RoleConsensusNode):
		sigResource, err = orgMemberMap[currentOrgId].consensus.Sign(hashType, []byte(msg))
		if err != nil {
			return nil, err
		}

		signerResource, err = orgMemberMap[currentOrgId].consensus.GetMember()
		if err != nil {
			return nil, err
		}
	case string(protocol.RoleAdmin):
		sigResource, err = orgMemberMap[currentOrgId].admin.Sign(hashType, []byte(msg))
		if err != nil {
			return nil, err
		}

		signerResource, err = orgMemberMap[currentOrgId].admin.GetMember()
		if err != nil {
			return nil, err
		}
	default:
		sigResource, err = orgMemberMap[currentOrgId].client.Sign(hashType, []byte(msg))
		if err != nil {
			return nil, err
		}

		signerResource, err = orgMemberMap[currentOrgId].client.GetMember()
		if err != nil {
			return nil, err
		}
	}

	return &common.EndorsementEntry{
		Signer:    signerResource,
		Signature: sigResource,
	}, nil
}

func testVerifyCurrentPrincipal(orgMemberMap map[string]*orgMember, acProviderOrgId string, resourcePrincipal protocol.Principal) (bool, []*common.EndorsementEntry, error) {

	ok, err := orgMemberMap[acProviderOrgId].acProvider.VerifyPrincipal(resourcePrincipal)
	if err != nil {
		return false, nil, err
	}

	validEndorsements, err := orgMemberMap[acProviderOrgId].acProvider.(*certACProvider).GetValidEndorsements(resourcePrincipal)
	if err != nil {
		return ok, nil, err
	}
	return ok, validEndorsements, nil
}
