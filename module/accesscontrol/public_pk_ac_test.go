/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package accesscontrol

import (
	"testing"

	pbac "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"
	"chainmaker.org/chainmaker/protocol/v2"
	"github.com/stretchr/testify/require"
)

func TestPublicPKVerifyConsensusPrincipal(t *testing.T) {
	testPkMember := testInitPublicPKFunc(t)
	//consensus
	pkMemberInfo := testPkMember[testOrg1]
	endorsement, err := testPublicPKCreateEndorsementEntry(pkMemberInfo, protocol.RoleConsensusNode, testPKHashType, testMsg)
	require.Nil(t, err)
	require.NotNil(t, endorsement)
	ok, err := testVerifyPrincipal(test1PublicPKACProvider, protocol.ResourceNameConsensusNode, []*common.EndorsementEntry{endorsement})
	require.Nil(t, err)
	require.Equal(t, true, ok)

	//consensus invalid
	pkMemberInfo = testPkMember[testOrg1]
	endorsement, err = testPublicPKCreateEndorsementEntry(pkMemberInfo, protocol.RoleAdmin, testPKHashType, testMsg)
	require.Nil(t, err)
	require.NotNil(t, endorsement)
	ok, err = testVerifyPrincipal(test1PublicPKACProvider, protocol.ResourceNameConsensusNode, []*common.EndorsementEntry{endorsement})
	require.NotNil(t, err)
	require.Equal(t, false, ok)
}

func TestPublicPKVerifyManagePrincipal(t *testing.T) {
	testPkMember := testInitPublicPKFunc(t)
	//Manage
	pkMemberInfo := testPkMember[testOrg1]
	endorsement, err := testPublicPKCreateEndorsementEntry(pkMemberInfo, protocol.RoleAdmin, testPKHashType, testMsg)
	require.Nil(t, err)
	require.NotNil(t, endorsement)
	ok, err := testVerifyPrincipal(test1PublicPKACProvider, syscontract.SystemContract_CONTRACT_MANAGE.String()+"-"+
		syscontract.ContractManageFunction_UPGRADE_CONTRACT.String(), []*common.EndorsementEntry{endorsement})
	require.Nil(t, err)
	require.Equal(t, true, ok)

	//Manage invalid
	pkMemberInfo = testPkMember[testOrg1]
	endorsement, err = testPublicPKCreateEndorsementEntry(pkMemberInfo, protocol.RoleConsensusNode, testPKHashType, testMsg)
	require.Nil(t, err)
	require.NotNil(t, endorsement)
	ok, err = testVerifyPrincipal(test1PublicPKACProvider, syscontract.SystemContract_CONTRACT_MANAGE.String()+"-"+
		syscontract.ContractManageFunction_UPGRADE_CONTRACT.String(), []*common.EndorsementEntry{endorsement})
	require.NotNil(t, err)
	require.Equal(t, false, ok)
}

func TestPublicPKVerifyAdminMajorityPrincipal(t *testing.T) {
	testPkMember := testInitPublicPKFunc(t)
	//majority
	pkMemberInfo1 := testPkMember[testOrg1]
	endorsement1, err := testPublicPKCreateEndorsementEntry(pkMemberInfo1, protocol.RoleAdmin, testPKHashType, testMsg)
	require.Nil(t, err)
	require.NotNil(t, endorsement1)

	pkMemberInfo2 := testPkMember[testOrg2]
	endorsement2, err := testPublicPKCreateEndorsementEntry(pkMemberInfo2, protocol.RoleAdmin, testPKHashType, testMsg)
	require.Nil(t, err)
	require.NotNil(t, endorsement2)

	pkMemberInfo3 := testPkMember[testOrg3]
	endorsement3, err := testPublicPKCreateEndorsementEntry(pkMemberInfo3, protocol.RoleAdmin, testPKHashType, testMsg)
	require.Nil(t, err)
	require.NotNil(t, endorsement3)

	ok, err := testVerifyPrincipal(test2PublicPKACProvider, syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_TRUST_ROOT_UPDATE.String(), []*common.EndorsementEntry{endorsement1, endorsement2, endorsement3})
	require.Nil(t, err)
	require.Equal(t, true, ok)

	validEndorsements, err := testPublicPKGetValidEndorsements(test2PublicPKACProvider,
		syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
			syscontract.ChainConfigFunction_TRUST_ROOT_UPDATE.String(),
		[]*common.EndorsementEntry{endorsement1, endorsement2, endorsement3})

	require.Nil(t, err)
	require.Equal(t, len(validEndorsements), 3)

	//majority invalid

	ok, err = testVerifyPrincipal(test2PublicPKACProvider,
		syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
			syscontract.ChainConfigFunction_TRUST_ROOT_UPDATE.String(),
		[]*common.EndorsementEntry{endorsement1, endorsement2})
	require.NotNil(t, err)
	require.Equal(t, false, ok)

	validEndorsements, err = testPublicPKGetValidEndorsements(test2PublicPKACProvider,
		syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
			syscontract.ChainConfigFunction_TRUST_ROOT_UPDATE.String(),
		[]*common.EndorsementEntry{endorsement1, endorsement2})

	require.Nil(t, err)
	require.Equal(t, len(validEndorsements), 2)

}

func TestPublicPKVerifyForbiddenPrincipal(t *testing.T) {
	testPkMember := testInitPublicPKFunc(t)
	//Forbidden
	pkMemberInfo := testPkMember[testOrg1]
	endorsement, err := testPublicPKCreateEndorsementEntry(pkMemberInfo, protocol.RoleAdmin, testPKHashType, testMsg)
	require.Nil(t, err)
	require.NotNil(t, endorsement)
	ok, err := testVerifyPrincipal(test1PublicPKACProvider, syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_NODE_ID_ADD.String(), []*common.EndorsementEntry{endorsement})
	require.NotNil(t, err)
	require.Equal(t, false, ok)
}

func testPublicPKCreateEndorsementEntry(testPKMember *testPkMember, roleType protocol.Role, hashType, msg string) (*common.EndorsementEntry, error) {
	var (
		sigResource    []byte
		err            error
		signerResource *pbac.Member
	)
	switch roleType {
	case protocol.RoleConsensusNode:
		sigResource, err = testPKMember.consensus.Sign(hashType, []byte(msg))
		if err != nil {
			return nil, err
		}

		signerResource, err = testPKMember.consensus.GetMember()
		if err != nil {
			return nil, err
		}
	case protocol.RoleAdmin:
		sigResource, err = testPKMember.admin.Sign(hashType, []byte(msg))
		if err != nil {
			return nil, err
		}

		signerResource, err = testPKMember.admin.GetMember()
		if err != nil {
			return nil, err
		}
	}

	return &common.EndorsementEntry{
		Signer:    signerResource,
		Signature: sigResource,
	}, nil
}

func testPublicPKGetValidEndorsements(provider protocol.AccessControlProvider,
	resourceName string, endorsements []*common.EndorsementEntry) ([]*common.EndorsementEntry, error) {
	principal, err := provider.CreatePrincipal(resourceName, endorsements, []byte(testMsg))
	if err != nil {
		return nil, err
	}
	return provider.(*pkACProvider).GetValidEndorsements(principal)
}
