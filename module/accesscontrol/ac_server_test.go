/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package accesscontrol

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	logger2 "chainmaker.org/chainmaker/logger/v2"
	pbac "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/config"
	"chainmaker.org/chainmaker/protocol/v2"
	"github.com/stretchr/testify/require"
)

func TestInitAccessControlService(t *testing.T) {
	logger := logger2.GetLogger(logger2.MODULE_ACCESS)
	acServices := initAccessControlService(testHashType, protocol.Identity, nil, logger)
	acServices.initResourcePolicy(testChainConfig.ResourcePolicies, testOrg1)
	require.NotNil(t, acServices)
}

func TestValidateResourcePolicy(t *testing.T) {
	_, cleanFunc, err := createTempDirWithCleanFunc()
	require.Nil(t, err)
	defer cleanFunc()

	logger := logger2.GetLogger(logger2.MODULE_ACCESS)
	acServices := initAccessControlService(testHashType, protocol.Identity, nil, logger)
	acServices.initResourcePolicy(testChainConfig.ResourcePolicies, testOrg1)
	require.NotNil(t, acServices)

	resourcePolicy := &config.ResourcePolicy{
		ResourceName: "INIT_CONTRACT",
		Policy:       &pbac.Policy{Rule: "ANY"},
	}
	ok := acServices.validateResourcePolicy(resourcePolicy)
	require.Equal(t, true, ok)

	resourcePolicy = &config.ResourcePolicy{
		ResourceName: "P2P",
		Policy:       &pbac.Policy{Rule: "ANY"},
	}
	ok = acServices.validateResourcePolicy(resourcePolicy)
	require.Equal(t, false, ok)
}

func TestCertMemberInfo(t *testing.T) {
	td, cleanFunc, err := createTempDirWithCleanFunc()
	require.Nil(t, err)
	defer cleanFunc()

	logger := logger2.GetLogger(logger2.MODULE_ACCESS)
	acServices := initAccessControlService(testHashType, protocol.Identity, nil, logger)
	acServices.initResourcePolicy(testChainConfig.ResourcePolicies, testOrg1)
	require.NotNil(t, acServices)

	pbMember := &pbac.Member{
		OrgId:      testOrg1,
		MemberType: pbac.MemberType_CERT,
		MemberInfo: []byte(testConsensusSignOrg1.cert),
	}
	member, err := acServices.newCertMember(pbMember)
	require.Nil(t, err)
	require.Equal(t, testOrg1, member.GetOrgId())
	require.Equal(t, testConsensusRole, member.GetRole())
	require.Equal(t, testConsensusCN, member.GetMemberId())

	localPrivKeyFile := filepath.Join(td, tempOrg1KeyFileName)
	localCertFile := filepath.Join(td, tempOrg1CertFileName)
	err = ioutil.WriteFile(localPrivKeyFile, []byte(testConsensusSignOrg2.sk), os.ModePerm)
	require.Nil(t, err)
	err = ioutil.WriteFile(localCertFile, []byte(testConsensusSignOrg2.cert), os.ModePerm)
	require.Nil(t, err)
	signingMember, err := InitCertSigningMember(testChainConfig, testOrg2, localPrivKeyFile, "", localCertFile)
	require.Nil(t, err)
	require.NotNil(t, signingMember)
	signRead, err := signingMember.Sign(testChainConfig.Crypto.Hash, []byte(testMsg))
	require.Nil(t, err)
	err = signingMember.Verify(testChainConfig.Crypto.Hash, []byte(testMsg), signRead)
	require.Nil(t, err)

	cachedMember := &memberCached{
		member:    member,
		certChain: nil,
	}
	mem, err := member.GetMember()
	require.Nil(t, err)
	require.NotNil(t, mem)
	acServices.addMemberToCache(string(mem.MemberInfo), cachedMember)
	memCache, ok := acServices.lookUpMemberInCache(string(mem.MemberInfo))
	require.Equal(t, true, ok)
	require.Equal(t, cachedMember, memCache)
}

func TestVerifyPrincipalPolicy(t *testing.T) {
	_, cleanFunc, err := createTempDirWithCleanFunc()
	require.Nil(t, err)
	defer cleanFunc()
	hashType := testHashType
	logger := logger2.GetLogger(logger2.MODULE_ACCESS)
	acServices := initAccessControlService(testHashType, protocol.Identity, nil, logger)
	acServices.initResourcePolicy(testChainConfig.ResourcePolicies, testOrg1)
	require.NotNil(t, acServices)

	var orgMemberMap = make(map[string]*orgMember, len(orgMemberInfoMap))
	for orgId, info := range orgMemberInfoMap {
		orgMemberMap[orgId] = initOrgMember(t, info)
	}

	org1Member := orgMemberMap[testOrg1]

	org1AdminSig, err := org1Member.admin.Sign(hashType, []byte(testMsg))
	require.Nil(t, err)
	org1AdminPb, err := org1Member.admin.GetMember()
	require.Nil(t, err)
	endorsement := &common.EndorsementEntry{
		Signer:    org1AdminPb,
		Signature: org1AdminSig,
	}
	policy, err := acServices.lookUpPolicy(common.TxType_QUERY_CONTRACT.String())
	require.Nil(t, err)
	require.Equal(t, policyRead.GetPbPolicy(), policy)

	principal, err := acServices.createPrincipal(common.TxType_QUERY_CONTRACT.String(),
		[]*common.EndorsementEntry{endorsement}, []byte(testMsg))
	require.Nil(t, err)

	ok, err := acServices.verifyPrincipalPolicy(principal, principal, policyRead)
	require.NotNil(t, err)
	require.Equal(t, false, ok)
}
