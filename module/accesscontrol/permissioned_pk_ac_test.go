/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package accesscontrol

import (
	"fmt"
	"testing"

	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/protocol/v2/mock"
	"github.com/golang/mock/gomock"

	"chainmaker.org/chainmaker/common/v2/crypto/asym"
	"chainmaker.org/chainmaker/common/v2/helper"
	logger2 "chainmaker.org/chainmaker/logger/v2"
	pbac "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	"github.com/stretchr/testify/require"
)

var (
	test1PermissionedPKACProvider protocol.AccessControlProvider
	test2PermissionedPKACProvider protocol.AccessControlProvider
	test1PublicPKACProvider       protocol.AccessControlProvider
	test2PublicPKACProvider       protocol.AccessControlProvider
)

func TestParsePublicKey(t *testing.T) {
	_, cleanFunc, err := createTempDirWithCleanFunc()
	require.Nil(t, err)
	defer cleanFunc()
	sk, err := asym.PrivateKeyFromPEM([]byte(TestSK1), nil)
	if err != nil {
		fmt.Println(err)
	}
	commonNodeId, err := helper.CreateLibp2pPeerIdWithPublicKey(sk.PublicKey())
	if err != nil {
		fmt.Println(err)
	}
	pk, err := asym.PublicKeyFromPEM([]byte(TestPK1))
	if err != nil {
		fmt.Println(err)
	}
	openNodeId, err := helper.CreateLibp2pPeerIdWithPublicKey(pk)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("common:", commonNodeId)
	fmt.Println("open:", openNodeId)
}

func TestPermissionedPKGetMemberStatus(t *testing.T) {
	_, cleanFunc, err := createTempDirWithCleanFunc()
	require.Nil(t, err)
	defer cleanFunc()
	ctl := gomock.NewController(t)
	store := mock.NewMockBlockchainStore(ctl)
	store.EXPECT().ReadObject(syscontract.SystemContract_PUBKEY_MANAGE.String(),
		gomock.Any()).Return(nil, nil)
	logger := logger2.GetLogger(logger2.MODULE_ACCESS)
	ppkProvider, err := newPermissionedPkACProvider(testPermissionedPKChainConfig, testOrg1, store, logger)
	require.Nil(t, err)
	require.NotNil(t, ppkProvider)

	pbMember := &pbac.Member{
		OrgId:      testOrg1,
		MemberType: pbac.MemberType_PUBLIC_KEY,
		MemberInfo: []byte(TestPK1),
	}

	memberStatus, err := ppkProvider.GetMemberStatus(pbMember)
	require.Nil(t, err)
	require.Equal(t, pbac.MemberStatus_NORMAL, memberStatus)

	pbMember = &pbac.Member{
		OrgId:      testOrg1,
		MemberType: pbac.MemberType_PUBLIC_KEY,
		MemberInfo: []byte(TestPK9),
	}

	memberStatus, err = ppkProvider.GetMemberStatus(pbMember)
	require.NotNil(t, err)
	require.Equal(t, pbac.MemberStatus_INVALID, memberStatus)
}

func TestPermissionedPKVerifyP2PPrincipal(t *testing.T) {
	testPkOrgMember := testInitPermissionedPKFunc(t)
	//P2P
	pkOrgMemberInfo := testPkOrgMember[testOrg1]
	endorsement, err := testPermissionedPKCreateEndorsementEntry(pkOrgMemberInfo, protocol.RoleConsensusNode, testPKHashType, testMsg)
	require.Nil(t, err)
	require.NotNil(t, endorsement)
	ok, err := testVerifyPrincipal(test1PermissionedPKACProvider, protocol.ResourceNameP2p, []*common.EndorsementEntry{endorsement})
	require.Nil(t, err)
	require.Equal(t, true, ok)

	//P2P invalid
	pkOrgMemberInfo = testPkOrgMember[testOrg1]
	endorsement, err = testPermissionedPKCreateEndorsementEntry(pkOrgMemberInfo, protocol.RoleAdmin, testPKHashType, testMsg)
	require.Nil(t, err)
	require.NotNil(t, endorsement)
	ok, err = testVerifyPrincipal(test1PermissionedPKACProvider, protocol.ResourceNameP2p, []*common.EndorsementEntry{endorsement})
	require.NotNil(t, err)
	require.Equal(t, false, ok)
}

func TestPermissionedPKVerifyConsensusPrincipal(t *testing.T) {
	testPkOrgMember := testInitPermissionedPKFunc(t)
	//consensus
	pkOrgMemberInfo := testPkOrgMember[testOrg1]
	endorsement, err := testPermissionedPKCreateEndorsementEntry(pkOrgMemberInfo, protocol.RoleConsensusNode, testPKHashType, testMsg)
	require.Nil(t, err)
	require.NotNil(t, endorsement)
	ok, err := testVerifyPrincipal(test1PermissionedPKACProvider, protocol.ResourceNameConsensusNode, []*common.EndorsementEntry{endorsement})
	require.Nil(t, err)
	require.Equal(t, true, ok)

	//consensus invalid
	pkOrgMemberInfo = testPkOrgMember[testOrg1]
	endorsement, err = testPermissionedPKCreateEndorsementEntry(pkOrgMemberInfo, protocol.RoleAdmin, testPKHashType, testMsg)
	require.Nil(t, err)
	require.NotNil(t, endorsement)
	ok, err = testVerifyPrincipal(test1PermissionedPKACProvider, protocol.ResourceNameConsensusNode, []*common.EndorsementEntry{endorsement})
	require.NotNil(t, err)
	require.Equal(t, false, ok)
}

func TestPermissionedPKVerifySelfPrincipal(t *testing.T) {
	testPkOrgMember := testInitPermissionedPKFunc(t)
	//self
	pkOrgMemberInfo := testPkOrgMember[testOrg1]
	endorsement, err := testPermissionedPKCreateEndorsementEntry(pkOrgMemberInfo, protocol.RoleAdmin, testPKHashType, testMsg)
	require.Nil(t, err)
	require.NotNil(t, endorsement)
	principal, err := test1PermissionedPKACProvider.CreatePrincipalForTargetOrg(protocol.ResourceNameUpdateSelfConfig,
		[]*common.EndorsementEntry{endorsement}, []byte(testMsg), testOrg1)
	require.Nil(t, err)
	require.NotNil(t, principal)
	ok, err := test1PermissionedPKACProvider.VerifyPrincipal(principal)
	require.Nil(t, err)
	require.Equal(t, true, ok)

	//self invalid
	pkOrgMemberInfo = testPkOrgMember[testOrg1]
	endorsement, err = testPermissionedPKCreateEndorsementEntry(pkOrgMemberInfo, protocol.RoleAdmin, testPKHashType, testMsg)
	require.Nil(t, err)
	require.NotNil(t, endorsement)
	principal, err = test1PermissionedPKACProvider.CreatePrincipalForTargetOrg(protocol.ResourceNameUpdateSelfConfig,
		[]*common.EndorsementEntry{endorsement}, []byte(testMsg), testOrg2)
	require.Nil(t, err)
	require.NotNil(t, principal)
	ok, err = test1PermissionedPKACProvider.VerifyPrincipal(principal)
	require.NotNil(t, err)
	require.Equal(t, false, ok)
}

func TestPermissionedPKVerifyMajorityPrincipal(t *testing.T) {
	testPkOrgMember := testInitPermissionedPKFunc(t)
	//majority
	pkOrgMemberInfo1 := testPkOrgMember[testOrg1]
	endorsement1, err := testPermissionedPKCreateEndorsementEntry(pkOrgMemberInfo1, protocol.RoleAdmin, testPKHashType, testMsg)
	require.Nil(t, err)
	require.NotNil(t, endorsement1)

	pkOrgMemberInfo2 := testPkOrgMember[testOrg2]
	endorsement2, err := testPermissionedPKCreateEndorsementEntry(pkOrgMemberInfo2, protocol.RoleAdmin, testPKHashType, testMsg)
	require.Nil(t, err)
	require.NotNil(t, endorsement2)

	pkOrgMemberInfo3 := testPkOrgMember[testOrg3]
	endorsement3, err := testPermissionedPKCreateEndorsementEntry(pkOrgMemberInfo3, protocol.RoleAdmin, testPKHashType, testMsg)
	require.Nil(t, err)
	require.NotNil(t, endorsement3)

	ok, err := testVerifyPrincipal(test2PermissionedPKACProvider, protocol.ResourceNameUpdateConfig,
		[]*common.EndorsementEntry{endorsement1, endorsement2, endorsement3})
	require.Nil(t, err)
	require.Equal(t, true, ok)

	validEndorsements, err := testPermissionedPKGetValidEndorsements(test2PermissionedPKACProvider, protocol.ResourceNameUpdateConfig,
		[]*common.EndorsementEntry{endorsement1, endorsement2, endorsement3})

	require.Nil(t, err)
	require.Equal(t, len(validEndorsements), 3)

	//majority invalid

	ok, err = testVerifyPrincipal(test2PermissionedPKACProvider, protocol.ResourceNameUpdateConfig,
		[]*common.EndorsementEntry{endorsement1, endorsement2})
	require.NotNil(t, err)
	require.Equal(t, false, ok)

	validEndorsements, err = testPermissionedPKGetValidEndorsements(test2PermissionedPKACProvider, protocol.ResourceNameUpdateConfig,
		[]*common.EndorsementEntry{endorsement1, endorsement2})

	require.Nil(t, err)
	require.Equal(t, len(validEndorsements), 2)

}

func testPermissionedPKCreateEndorsementEntry(testOrgPKMember *testPkOrgMember, roleType protocol.Role, hashType, msg string) (*common.EndorsementEntry, error) {
	var (
		sigResource    []byte
		err            error
		signerResource *pbac.Member
	)
	switch roleType {
	case protocol.RoleConsensusNode:
		sigResource, err = testOrgPKMember.consensus.Sign(hashType, []byte(msg))
		if err != nil {
			return nil, err
		}

		signerResource, err = testOrgPKMember.consensus.GetMember()
		if err != nil {
			return nil, err
		}
	case protocol.RoleAdmin:
		sigResource, err = testOrgPKMember.admin.Sign(hashType, []byte(msg))
		if err != nil {
			return nil, err
		}

		signerResource, err = testOrgPKMember.admin.GetMember()
		if err != nil {
			return nil, err
		}
	}

	return &common.EndorsementEntry{
		Signer:    signerResource,
		Signature: sigResource,
	}, nil
}

func testPermissionedPKGetValidEndorsements(provider protocol.AccessControlProvider,
	resourceName string, endorsements []*common.EndorsementEntry) ([]*common.EndorsementEntry, error) {
	principal, err := provider.CreatePrincipal(resourceName, endorsements, []byte(testMsg))
	if err != nil {
		return nil, err
	}
	return provider.(*permissionedPkACProvider).GetValidEndorsements(principal)
}
