package accesscontrol

import (
	"fmt"
	"testing"
	"time"

	logger2 "chainmaker.org/chainmaker-go/logger"
	pbac "chainmaker.org/chainmaker/pb-go/accesscontrol"
	"chainmaker.org/chainmaker/pb-go/common"
	"chainmaker.org/chainmaker/pb-go/syscontract"
	"chainmaker.org/chainmaker/protocol"
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

	// read
	sigRead, err := orgMemberMap[testOrg1].client.Sign(hashType, []byte(testMsg))
	require.Nil(t, err)
	signerRead, err := orgMemberMap[testOrg1].client.GetMember()
	require.Nil(t, err)
	endorsementReadZephyrus := &common.EndorsementEntry{
		Signer:    signerRead,
		Signature: sigRead,
	}
	principalRead, err := orgMemberMap[testOrg1].acProvider.CreatePrincipal(common.TxType_QUERY_CONTRACT.String(), []*common.EndorsementEntry{endorsementReadZephyrus}, []byte(testMsg))
	require.Nil(t, err)

	ok, err := orgMemberMap[testOrg2].acProvider.VerifyPrincipal(principalRead)
	require.Nil(t, err)
	require.Equal(t, true, ok)
	validEndorsements, err := orgMemberMap[testOrg2].acProvider.(*certACProvider).GetValidEndorsements(principalRead)
	require.Nil(t, err)
	require.Equal(t, len(validEndorsements), 1)
	require.Equal(t, endorsementReadZephyrus.String(), validEndorsements[0].String())

	// read invalid
	sigRead, err = orgMemberMap[testOrg5].client.Sign(hashType, []byte(testMsg))
	require.Nil(t, err)
	signerRead, err = orgMemberMap[testOrg5].client.GetMember()
	require.Nil(t, err)
	endorsementRead := &common.EndorsementEntry{
		Signer:    signerRead,
		Signature: sigRead,
	}
	principalRead, err = orgMemberMap[testOrg2].acProvider.CreatePrincipal(common.TxType_QUERY_CONTRACT.String(), []*common.EndorsementEntry{endorsementRead}, []byte(testMsg))
	require.Nil(t, err)
	ok, err = orgMemberMap[testOrg2].acProvider.VerifyPrincipal(principalRead)
	require.NotNil(t, err)
	require.Equal(t, false, ok)

	// write
	sigWrite, err := orgMemberMap[testOrg4].admin.Sign(hashType, []byte(testMsg))
	require.Nil(t, err)
	signerWrite, err := orgMemberMap[testOrg4].admin.GetMember()
	require.Nil(t, err)
	endorsementWrite := &common.EndorsementEntry{
		Signer:    signerWrite,
		Signature: sigWrite,
	}
	principalWrite, err := orgMemberMap[testOrg1].acProvider.CreatePrincipal(protocol.ResourceNameWriteData, []*common.EndorsementEntry{endorsementWrite}, []byte(testMsg))
	require.Nil(t, err)
	ok, err = orgMemberMap[testOrg1].acProvider.VerifyPrincipal(principalWrite)
	require.Nil(t, err)
	require.Equal(t, true, ok)

	// invalid
	sigWrite, err = orgMemberMap[testOrg5].admin.Sign(hashType, []byte(testMsg))
	require.Nil(t, err)
	signerWrite, err = orgMemberMap[testOrg5].admin.GetMember()
	require.Nil(t, err)
	endorsementWrite = &common.EndorsementEntry{
		Signer:    signerWrite,
		Signature: sigWrite,
	}
	principalWrite, err = orgMemberMap[testOrg1].acProvider.CreatePrincipal(protocol.ResourceNameWriteData, []*common.EndorsementEntry{endorsementWrite}, []byte(testMsg))
	require.Nil(t, err)
	ok, err = orgMemberMap[testOrg1].acProvider.VerifyPrincipal(principalWrite)
	require.NotNil(t, err)
	require.Equal(t, false, ok)

	// P2P
	sigP2P, err := orgMemberMap[testOrg1].consensus.Sign(hashType, []byte(testMsg))
	require.Nil(t, err)
	signerP2P, err := orgMemberMap[testOrg1].consensus.GetMember()
	require.Nil(t, err)
	endorsementP2P := &common.EndorsementEntry{
		Signer:    signerP2P,
		Signature: sigP2P,
	}
	principalP2P, err := orgMemberMap[testOrg2].acProvider.CreatePrincipal(protocol.ResourceNameP2p, []*common.EndorsementEntry{endorsementP2P}, []byte(testMsg))
	require.Nil(t, err)
	ok, err = orgMemberMap[testOrg2].acProvider.VerifyPrincipal(principalP2P)
	require.Nil(t, err)
	require.Equal(t, true, ok)

	// invalid
	sigP2P, err = orgMemberMap[testOrg1].admin.Sign(hashType, []byte(testMsg))
	require.Nil(t, err)
	signerP2P, err = orgMemberMap[testOrg1].admin.GetMember()
	require.Nil(t, err)
	endorsementP2P = &common.EndorsementEntry{
		Signer:    signerRead,
		Signature: sigRead,
	}
	principalP2P, err = orgMemberMap[testOrg2].acProvider.CreatePrincipal(protocol.ResourceNameP2p, []*common.EndorsementEntry{endorsementP2P}, []byte(testMsg))
	require.Nil(t, err)
	ok, err = orgMemberMap[testOrg2].acProvider.VerifyPrincipal(principalP2P)
	require.NotNil(t, err)
	require.Equal(t, false, ok)

	// consensus
	sigConsensus, err := orgMemberMap[testOrg1].consensus.Sign(hashType, []byte(testMsg))
	require.Nil(t, err)
	signerConsensus, err := orgMemberMap[testOrg1].consensus.GetMember()
	require.Nil(t, err)
	endorsementConsensus := &common.EndorsementEntry{
		Signer:    signerConsensus,
		Signature: sigConsensus,
	}
	principalConsensus, err := orgMemberMap[testOrg2].acProvider.CreatePrincipal(protocol.ResourceNameConsensusNode, []*common.EndorsementEntry{endorsementConsensus}, []byte(testMsg))
	require.Nil(t, err)
	ok, err = orgMemberMap[testOrg2].acProvider.VerifyPrincipal(principalConsensus)
	require.Nil(t, err)
	require.Equal(t, true, ok)

	// invalid
	sigConsensus, err = orgMemberMap[testOrg1].client.Sign(hashType, []byte(testMsg))
	require.Nil(t, err)
	signerConsensus, err = orgMemberMap[testOrg1].client.GetMember()
	require.Nil(t, err)
	endorsementConsensus = &common.EndorsementEntry{
		Signer:    signerConsensus,
		Signature: sigConsensus,
	}
	principalConsensus, err = orgMemberMap[testOrg2].acProvider.CreatePrincipal(protocol.ResourceNameConsensusNode, []*common.EndorsementEntry{endorsementConsensus}, []byte(testMsg))
	require.Nil(t, err)
	ok, err = orgMemberMap[testOrg2].acProvider.VerifyPrincipal(principalConsensus)
	require.NotNil(t, err)
	require.Equal(t, false, ok)

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
	ok, err = orgMemberMap[testOrg2].acProvider.VerifyPrincipal(principalSelf)
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

	validEndorsements, err = orgMemberMap[testOrg2].acProvider.(*certACProvider).GetValidEndorsements(principalMajority)
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
	principalRead, err = orgMemberMap[testOrg2].acProvider.CreatePrincipalForTargetOrg(protocol.ResourceNameUpdateSelfConfig, []*common.EndorsementEntry{endorsementEurus}, []byte(testMsg), testOrg4)
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
