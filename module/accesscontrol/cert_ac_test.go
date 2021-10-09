/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package accesscontrol

import (
	"testing"

	logger2 "chainmaker.org/chainmaker/logger/v2"
	pbac "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/protocol/v2"
	"github.com/stretchr/testify/require"
)

var (
	test1CertACProvider protocol.AccessControlProvider
	test2CertACProvider protocol.AccessControlProvider
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

func testInitFunc(t *testing.T) map[string]*orgMember {
	_, cleanFunc, err := createTempDirWithCleanFunc()
	require.Nil(t, err)
	defer cleanFunc()

	var orgMemberMap = make(map[string]*orgMember, len(orgMemberInfoMap))
	for orgId, info := range orgMemberInfoMap {
		orgMemberMap[orgId] = initOrgMember(t, info)
	}
	test1CertACProvider = orgMemberMap[testOrg1].acProvider
	test2CertACProvider = orgMemberMap[testOrg2].acProvider
	return orgMemberMap
}

func TestVerifyReadPrincipal(t *testing.T) {
	orgMemberMap := testInitFunc(t)
	//read
	orgMemberInfo := orgMemberMap[testOrg2]
	endorsement, err := testCreateEndorsementEntry(orgMemberInfo, protocol.RoleClient, testHashType, testMsg)
	require.Nil(t, err)
	require.NotNil(t, endorsement)
	ok, err := testVerifyPrincipal(test1CertACProvider, common.TxType_QUERY_CONTRACT.String(), []*common.EndorsementEntry{endorsement})
	require.Nil(t, err)
	require.Equal(t, true, ok)

	//read invalid
	orgMemberInfo = orgMemberMap[testOrg5]
	endorsement, err = testCreateEndorsementEntry(orgMemberInfo, protocol.RoleClient, testHashType, testMsg)
	require.Nil(t, err)
	require.NotNil(t, endorsement)
	ok, err = testVerifyPrincipal(test1CertACProvider, common.TxType_QUERY_CONTRACT.String(), []*common.EndorsementEntry{endorsement})
	require.NotNil(t, err)
	require.Equal(t, false, ok)
}

func TestVerifyP2PPrincipal(t *testing.T) {
	orgMemberMap := testInitFunc(t)
	//P2P
	orgMemberInfo := orgMemberMap[testOrg1]
	endorsement, err := testCreateEndorsementEntry(orgMemberInfo, protocol.RoleConsensusNode, testHashType, testMsg)
	require.Nil(t, err)
	require.NotNil(t, endorsement)
	ok, err := testVerifyPrincipal(test1CertACProvider, protocol.ResourceNameP2p, []*common.EndorsementEntry{endorsement})
	require.Nil(t, err)
	require.Equal(t, true, ok)

	//P2P invalid
	orgMemberInfo = orgMemberMap[testOrg1]
	endorsement, err = testCreateEndorsementEntry(orgMemberInfo, protocol.RoleClient, testHashType, testMsg)
	require.Nil(t, err)
	require.NotNil(t, endorsement)
	ok, err = testVerifyPrincipal(test1CertACProvider, protocol.ResourceNameP2p, []*common.EndorsementEntry{endorsement})
	require.NotNil(t, err)
	require.Equal(t, false, ok)

	orgMemberInfo = orgMemberMap[testOrg5]
	endorsement, err = testCreateEndorsementEntry(orgMemberInfo, protocol.RoleConsensusNode, testHashType, testMsg)
	require.Nil(t, err)
	require.NotNil(t, endorsement)
	ok, err = testVerifyPrincipal(test1CertACProvider, protocol.ResourceNameP2p, []*common.EndorsementEntry{endorsement})
	require.NotNil(t, err)
	require.Equal(t, false, ok)
}

func TestVerifyConsensusPrincipal(t *testing.T) {
	orgMemberMap := testInitFunc(t)
	//consensus
	orgMemberInfo := orgMemberMap[testOrg1]
	endorsement, err := testCreateEndorsementEntry(orgMemberInfo, protocol.RoleConsensusNode, testHashType, testMsg)
	require.Nil(t, err)
	require.NotNil(t, endorsement)
	ok, err := testVerifyPrincipal(test1CertACProvider, protocol.ResourceNameConsensusNode, []*common.EndorsementEntry{endorsement})
	require.Nil(t, err)
	require.Equal(t, true, ok)

	//consensus invalid
	orgMemberInfo = orgMemberMap[testOrg1]
	endorsement, err = testCreateEndorsementEntry(orgMemberInfo, protocol.RoleAdmin, testHashType, testMsg)
	require.Nil(t, err)
	require.NotNil(t, endorsement)
	ok, err = testVerifyPrincipal(test1CertACProvider, protocol.ResourceNameConsensusNode, []*common.EndorsementEntry{endorsement})
	require.NotNil(t, err)
	require.Equal(t, false, ok)

	orgMemberInfo = orgMemberMap[testOrg5]
	endorsement, err = testCreateEndorsementEntry(orgMemberInfo, protocol.RoleConsensusNode, testHashType, testMsg)
	require.Nil(t, err)
	require.NotNil(t, endorsement)
	ok, err = testVerifyPrincipal(test1CertACProvider, protocol.ResourceNameConsensusNode, []*common.EndorsementEntry{endorsement})
	require.NotNil(t, err)
	require.Equal(t, false, ok)
}

func TestVerifySelfPrincipal(t *testing.T) {
	orgMemberMap := testInitFunc(t)
	//self
	orgMemberInfo := orgMemberMap[testOrg1]
	endorsement, err := testCreateEndorsementEntry(orgMemberInfo, protocol.RoleAdmin, testHashType, testMsg)
	require.Nil(t, err)
	require.NotNil(t, endorsement)
	principal, err := test1CertACProvider.CreatePrincipalForTargetOrg(protocol.ResourceNameUpdateSelfConfig,
		[]*common.EndorsementEntry{endorsement}, []byte(testMsg), testOrg1)
	require.Nil(t, err)
	ok, err := test1CertACProvider.VerifyPrincipal(principal)
	require.Nil(t, err)
	require.Equal(t, true, ok)

	//self invalid
	orgMemberInfo = orgMemberMap[testOrg1]
	endorsement, err = testCreateEndorsementEntry(orgMemberInfo, protocol.RoleAdmin, testHashType, testMsg)
	require.Nil(t, err)
	require.NotNil(t, endorsement)
	ok, err = testVerifyPrincipal(test1CertACProvider, protocol.ResourceNameUpdateSelfConfig, []*common.EndorsementEntry{endorsement})
	require.NotNil(t, err)
	require.Equal(t, false, ok)

}

func TestVerifyMajorityPrincipal(t *testing.T) {
	orgMemberMap := testInitFunc(t)
	//majority
	orgMemberInfo1 := orgMemberMap[testOrg1]
	endorsement1, err := testCreateEndorsementEntry(orgMemberInfo1, protocol.RoleAdmin, testHashType, testMsg)
	require.Nil(t, err)
	require.NotNil(t, endorsement1)

	orgMemberInfo2 := orgMemberMap[testOrg2]
	endorsement2, err := testCreateEndorsementEntry(orgMemberInfo2, protocol.RoleAdmin, testHashType, testMsg)
	require.Nil(t, err)
	require.NotNil(t, endorsement2)

	orgMemberInfo3 := orgMemberMap[testOrg3]
	endorsement3, err := testCreateEndorsementEntry(orgMemberInfo3, protocol.RoleAdmin, testHashType, testMsg)
	require.Nil(t, err)
	require.NotNil(t, endorsement3)

	ok, err := testVerifyPrincipal(test1CertACProvider, protocol.ResourceNameUpdateConfig,
		[]*common.EndorsementEntry{endorsement1, endorsement2, endorsement3})
	require.Nil(t, err)
	require.Equal(t, true, ok)

	validEndorsements, err := testGetValidEndorsements(test1CertACProvider, protocol.ResourceNameUpdateConfig,
		[]*common.EndorsementEntry{endorsement1, endorsement2, endorsement3})

	require.Nil(t, err)
	require.Equal(t, len(validEndorsements), 3)

	//majority invalid

	ok, err = testVerifyPrincipal(test1CertACProvider, protocol.ResourceNameUpdateConfig,
		[]*common.EndorsementEntry{endorsement1, endorsement2})
	require.NotNil(t, err)
	require.Equal(t, false, ok)

	validEndorsements, err = testGetValidEndorsements(test1CertACProvider, protocol.ResourceNameUpdateConfig,
		[]*common.EndorsementEntry{endorsement1, endorsement2})

	require.Nil(t, err)
	require.Equal(t, len(validEndorsements), 2)

}

func TestVerifyAllPrincipal(t *testing.T) {
	orgMemberMap := testInitFunc(t)
	//all
	orgMemberInfo1 := orgMemberMap[testOrg1]
	endorsement1, err := testCreateEndorsementEntry(orgMemberInfo1, protocol.RoleAdmin, testHashType, testMsg)
	require.Nil(t, err)
	require.NotNil(t, endorsement1)

	orgMemberInfo2 := orgMemberMap[testOrg2]
	endorsement2, err := testCreateEndorsementEntry(orgMemberInfo2, protocol.RoleAdmin, testHashType, testMsg)
	require.Nil(t, err)
	require.NotNil(t, endorsement2)

	orgMemberInfo3 := orgMemberMap[testOrg3]
	endorsement3, err := testCreateEndorsementEntry(orgMemberInfo3, protocol.RoleAdmin, testHashType, testMsg)
	require.Nil(t, err)
	require.NotNil(t, endorsement3)

	orgMemberInfo4 := orgMemberMap[testOrg4]
	endorsement4, err := testCreateEndorsementEntry(orgMemberInfo4, protocol.RoleAdmin, testHashType, testMsg)
	require.Nil(t, err)
	require.NotNil(t, endorsement4)

	ok, err := testVerifyPrincipal(test1CertACProvider, protocol.ResourceNameAllTest,
		[]*common.EndorsementEntry{endorsement1, endorsement2, endorsement3, endorsement4})
	require.Nil(t, err)
	require.Equal(t, true, ok)

	validEndorsements, err := testGetValidEndorsements(test1CertACProvider, protocol.ResourceNameUpdateConfig,
		[]*common.EndorsementEntry{endorsement1, endorsement2, endorsement3, endorsement4})

	require.Nil(t, err)
	require.Equal(t, len(validEndorsements), 4)

	//all invalid

	ok, err = testVerifyPrincipal(test1CertACProvider, protocol.ResourceNameUpdateConfig,
		[]*common.EndorsementEntry{endorsement1, endorsement2})
	require.NotNil(t, err)
	require.Equal(t, false, ok)

	validEndorsements, err = testGetValidEndorsements(test1CertACProvider, protocol.ResourceNameUpdateConfig,
		[]*common.EndorsementEntry{endorsement1, endorsement2})

	require.Nil(t, err)
	require.Equal(t, len(validEndorsements), 2)

}

func TestVerifyTrustMemberPrincipal(t *testing.T) {
	orgMemberMap := testInitFunc(t)
	//read
	orgMemberInfo := orgMemberMap[testOrg5]
	sigResource, err := orgMemberInfo.trustMember1.Sign(testHashType, []byte(testMsg))
	require.Nil(t, err)
	require.NotNil(t, sigResource)
	signerResource, err := orgMemberInfo.trustMember1.GetMember()
	require.Nil(t, err)
	require.NotNil(t, signerResource)

	endorsement := &common.EndorsementEntry{
		Signer:    signerResource,
		Signature: sigResource,
	}

	ok, err := testVerifyPrincipal(test1CertACProvider, common.TxType_QUERY_CONTRACT.String(), []*common.EndorsementEntry{endorsement})
	require.Nil(t, err)
	require.Equal(t, true, ok)

	//read invalid
	orgMemberInfo = orgMemberMap[testOrg5]
	endorsement, err = testCreateEndorsementEntry(orgMemberInfo, protocol.RoleClient, testHashType, testMsg)
	require.Nil(t, err)
	require.NotNil(t, endorsement)
	ok, err = testVerifyPrincipal(test1CertACProvider, common.TxType_QUERY_CONTRACT.String(), []*common.EndorsementEntry{endorsement})
	require.NotNil(t, err)
	require.Equal(t, false, ok)
}

func testCreateEndorsementEntry(orgMember *orgMember, roleType protocol.Role, hashType, msg string) (*common.EndorsementEntry, error) {
	var (
		sigResource    []byte
		err            error
		signerResource *pbac.Member
	)
	switch roleType {
	case protocol.RoleConsensusNode:
		sigResource, err = orgMember.consensus.Sign(hashType, []byte(msg))
		if err != nil {
			return nil, err
		}

		signerResource, err = orgMember.consensus.GetMember()
		if err != nil {
			return nil, err
		}
	case protocol.RoleAdmin:
		sigResource, err = orgMember.admin.Sign(hashType, []byte(msg))
		if err != nil {
			return nil, err
		}

		signerResource, err = orgMember.admin.GetMember()
		if err != nil {
			return nil, err
		}
	default:
		sigResource, err = orgMember.client.Sign(hashType, []byte(msg))
		if err != nil {
			return nil, err
		}

		signerResource, err = orgMember.client.GetMember()
		if err != nil {
			return nil, err
		}
	}

	return &common.EndorsementEntry{
		Signer:    signerResource,
		Signature: sigResource,
	}, nil
}

func testVerifyPrincipal(provider protocol.AccessControlProvider,
	resourceName string, endorsements []*common.EndorsementEntry) (bool, error) {
	principal, err := provider.CreatePrincipal(resourceName, endorsements, []byte(testMsg))
	if err != nil {
		return false, err
	}
	return provider.VerifyPrincipal(principal)
}

func testGetValidEndorsements(provider protocol.AccessControlProvider,
	resourceName string, endorsements []*common.EndorsementEntry) ([]*common.EndorsementEntry, error) {
	principal, err := provider.CreatePrincipal(resourceName, endorsements, []byte(testMsg))
	if err != nil {
		return nil, err
	}
	return provider.(*certACProvider).GetValidEndorsements(principal)
}

func TestVerifyRelatedMaterial(t *testing.T) {
	logger := logger2.GetLogger(logger2.MODULE_ACCESS)
	certProvider, err := newCertACProvider(testChainConfig, testOrg1, nil, logger)
	require.Nil(t, err)
	require.NotNil(t, certProvider)
	isRevoked, err := certProvider.VerifyRelatedMaterial(pbac.VerifyType_CRL, []byte(""))
	require.NotNil(t, err)
	require.Equal(t, false, isRevoked)
	certProvider.VerifyRelatedMaterial(pbac.VerifyType_CRL, []byte(testCRL))
	require.NotNil(t, err)
	require.Equal(t, false, isRevoked)
}
