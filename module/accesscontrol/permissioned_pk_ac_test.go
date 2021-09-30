package accesscontrol

import (
	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/protocol/v2/mock"
	"fmt"
	"github.com/golang/mock/gomock"
	"testing"

	"chainmaker.org/chainmaker/common/v2/crypto/asym"
	"chainmaker.org/chainmaker/common/v2/helper"
	logger2 "chainmaker.org/chainmaker/logger/v2"
	pbac "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	"github.com/stretchr/testify/require"
)

var (
	test1PermissionedPKACProvider protocol.AccessControlProvider
	test2PermissionedPKACProvider protocol.AccessControlProvider
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
	store.EXPECT().ReadObject(syscontract.SystemContract_PUBKEY_MANAGEMENT.String(),
		gomock.Any()).Return(nil,nil)
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

func TestPermissionedPKVerifyConsensusPrincipal(t *testing.T) {
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

func TestPermissionedPKVerifySelfPrincipal(t *testing.T) {
	orgMemberMap := testInitFunc(t)
	//self
	orgMemberInfo := orgMemberMap[testOrg1]
	endorsement, err := testCreateEndorsementEntry(orgMemberInfo, protocol.RoleAdmin, testHashType, testMsg)
	require.Nil(t, err)
	require.NotNil(t, endorsement)
	principal, err := test1CertACProvider.CreatePrincipalForTargetOrg(protocol.ResourceNameUpdateSelfConfig,
		[]*common.EndorsementEntry{endorsement}, []byte(testMsg), testOrg1)
	require.NotNil(t, err)
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

func TestPermissionedPKVerifyMajorityPrincipal(t *testing.T) {
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

func testPermissionedPKCreateEndorsementEntry(testOrgPKMember *test, roleType protocol.Role, hashType, msg string) (*common.EndorsementEntry, error) {
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


func testPermissionedPKGetValidEndorsements(provider protocol.AccessControlProvider,
	resourceName string, endorsements []*common.EndorsementEntry) ([]*common.EndorsementEntry, error) {
	principal, err := provider.CreatePrincipal(resourceName, endorsements, []byte(testMsg))
	if err != nil {
		return nil, err
	}
	return test1CertACProvider.(*certACProvider).GetValidEndorsements(principal)
}