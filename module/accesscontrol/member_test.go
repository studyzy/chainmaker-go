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

	"chainmaker.org/chainmaker-go/localconf"
	logger2 "chainmaker.org/chainmaker-go/logger"
	pbac "chainmaker.org/chainmaker/pb-go/accesscontrol"
	"chainmaker.org/chainmaker/protocol"
	"github.com/stretchr/testify/require"
)

func TestMemberGetOrgId(t *testing.T) {
	localconf.ChainMakerConfig.NodeConfig.SignerCacheSize = 10
	localconf.ChainMakerConfig.NodeConfig.CertCacheSize = 10

	_, cleanFunc, err := createTempDirWithCleanFunc()
	require.Nil(t, err)
	defer cleanFunc()
	logger := logger2.GetLogger(logger2.MODULE_ACCESS)

	acInst, err := newAccessControlWithChainConfigPb(chainConf, org1Name, nil, logger)
	require.Nil(t, err)
	require.NotNil(t, acInst)

	pbMember := &pbac.Member{
		OrgId:      org1Name,
		MemberType: pbac.MemberType_CERT,
		MemberInfo: []byte(orgList[org1Name].consensusNode.certificate),
	}
	member, err := acInst.NewMember(pbMember)
	require.Nil(t, err)
	require.NotNil(t, member)
	memberOrgId := member.GetOrgId()
	require.Equal(t, org1Name, memberOrgId)
}

func TestMemberGetMemberId(t *testing.T) {

	localconf.ChainMakerConfig.NodeConfig.SignerCacheSize = 10
	localconf.ChainMakerConfig.NodeConfig.CertCacheSize = 10

	_, cleanFunc, err := createTempDirWithCleanFunc()
	require.Nil(t, err)
	defer cleanFunc()
	logger := logger2.GetLogger(logger2.MODULE_ACCESS)

	acInst, err := newAccessControlWithChainConfigPb(chainConf, org1Name, nil, logger)
	require.Nil(t, err)
	require.NotNil(t, acInst)

	pbMember := &pbac.Member{
		OrgId:      org1Name,
		MemberType: pbac.MemberType_CERT,
		MemberInfo: []byte(orgList[org1Name].consensusNode.certificate),
	}
	member, err := acInst.NewMember(pbMember)
	require.Nil(t, err)
	require.NotNil(t, member)
	memberId := member.GetMemberId()
	require.Equal(t, "consensus1.sign.wx-org1.chainmaker.org", memberId)
}

func TestMemberGetRole(t *testing.T) {
	localconf.ChainMakerConfig.NodeConfig.SignerCacheSize = 10
	localconf.ChainMakerConfig.NodeConfig.CertCacheSize = 10

	_, cleanFunc, err := createTempDirWithCleanFunc()
	require.Nil(t, err)
	defer cleanFunc()
	logger := logger2.GetLogger(logger2.MODULE_ACCESS)

	acInst, err := newAccessControlWithChainConfigPb(chainConf, org1Name, nil, logger)
	require.Nil(t, err)
	require.NotNil(t, acInst)

	pbMember := &pbac.Member{
		OrgId:      org1Name,
		MemberType: pbac.MemberType_CERT,
		MemberInfo: []byte(orgList[org1Name].consensusNode.certificate),
	}
	member, err := acInst.NewMember(pbMember)
	require.Nil(t, err)
	require.NotNil(t, member)
	role := member.GetRole()
	require.Equal(t, protocol.Role("CONSENSUS"), role)
}

func TestMemberGetMember(t *testing.T) {
	localconf.ChainMakerConfig.NodeConfig.SignerCacheSize = 10
	localconf.ChainMakerConfig.NodeConfig.CertCacheSize = 10

	_, cleanFunc, err := createTempDirWithCleanFunc()
	require.Nil(t, err)
	defer cleanFunc()
	logger := logger2.GetLogger(logger2.MODULE_ACCESS)

	acInst, err := newAccessControlWithChainConfigPb(chainConf, org1Name, nil, logger)
	require.Nil(t, err)
	require.NotNil(t, acInst)

	pbMember := &pbac.Member{
		OrgId:      org1Name,
		MemberType: pbac.MemberType_CERT,
		MemberInfo: []byte(orgList[org1Name].consensusNode.certificate),
	}
	member, err := acInst.NewMember(pbMember)
	require.Nil(t, err)
	require.NotNil(t, member)
	getPbMember, err := member.GetMember()
	require.Nil(t, err)
	require.Equal(t, pbMember, getPbMember)
	//fmt.Println(getPbMember)
	//fmt.Println(pbMember)
}

func TestMemberSignAndVerify(t *testing.T) {
	localconf.ChainMakerConfig.NodeConfig.SignerCacheSize = 10
	localconf.ChainMakerConfig.NodeConfig.CertCacheSize = 10

	td, cleanFunc, err := createTempDirWithCleanFunc()
	require.Nil(t, err)
	defer cleanFunc()
	localPrivKeyFile := filepath.Join(td, tempOrg1KeyFileName)
	localCertFile := filepath.Join(td, tempOrg1CertFileName)
	err = ioutil.WriteFile(localPrivKeyFile, []byte(orgList[org1Name].consensusNode.sk), os.ModePerm)
	require.Nil(t, err)
	err = ioutil.WriteFile(localCertFile, []byte(orgList[org1Name].consensusNode.certificate), os.ModePerm)
	require.Nil(t, err)
	signingMember, err := InitCertSigningMember(chainConf.Crypto.Hash, org1Name, localPrivKeyFile, "", localCertFile)
	require.Nil(t, err)
	require.NotNil(t, signingMember)
	signRead, err := signingMember.Sign(chainConf.Crypto.Hash, []byte(msg))
	require.Nil(t, err)
	err = signingMember.Verify(chainConf.Crypto.Hash, []byte(msg), signRead)
	require.Nil(t, err)
}
