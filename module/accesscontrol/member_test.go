/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package accesscontrol

import (
	"chainmaker.org/chainmaker-go/localconf"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestMemberGetOrgId(t *testing.T) {
	localconf.ChainMakerConfig.NodeConfig.SignerCacheSize = 10
	localconf.ChainMakerConfig.NodeConfig.CertCacheSize = 10

	var td = filepath.Join(os.TempDir(), tempDirName)
	os.Mkdir(td, 0666)
	localPrivKeyFile := filepath.Join(td, tempOrg1KeyFileName)
	localCertFile := filepath.Join(td, tempOrg1CertFileName)
	err := ioutil.WriteFile(localPrivKeyFile, []byte(orgList[org1Name].consensusNode.sk), 0666)
	require.Nil(t, err)
	err = ioutil.WriteFile(localCertFile, []byte(orgList[org1Name].consensusNode.certificate), 0666)
	require.Nil(t, err)
	acInst, err := newAccessControlWithChainConfigPb(localPrivKeyFile, "", localCertFile, chainConf, org1Name, nil)
	require.Nil(t, err)
	require.NotNil(t, acInst)
	member, err := acInst.NewMemberFromCertPem(org2Name, orgList[org2Name].consensusNode.certificate)
	require.Nil(t, err)
	require.NotNil(t, member)
	memberOrgId := member.GetOrgId()
	require.NotEqual(t, "", memberOrgId)
}

func TestMemberGetMemberId(t *testing.T) {
	localconf.ChainMakerConfig.NodeConfig.SignerCacheSize = 10
	localconf.ChainMakerConfig.NodeConfig.CertCacheSize = 10

	var td = filepath.Join(os.TempDir(), tempDirName)
	os.Mkdir(td, 0666)
	localPrivKeyFile := filepath.Join(td, tempOrg1KeyFileName)
	localCertFile := filepath.Join(td, tempOrg1CertFileName)
	err := ioutil.WriteFile(localPrivKeyFile, []byte(orgList[org1Name].consensusNode.sk), 0666)
	require.Nil(t, err)
	err = ioutil.WriteFile(localCertFile, []byte(orgList[org1Name].consensusNode.certificate), 0666)
	require.Nil(t, err)
	acInst, err := newAccessControlWithChainConfigPb(localPrivKeyFile, "", localCertFile, chainConf, org1Name, nil)
	require.Nil(t, err)
	require.NotNil(t, acInst)
	member, err := acInst.NewMemberFromCertPem(org2Name, orgList[org2Name].consensusNode.certificate)
	require.Nil(t, err)
	require.NotNil(t, member)
	memberId := member.GetMemberId()
	require.NotEqual(t, "", memberId)
}

func TestMemberGetRole(t *testing.T) {
	localconf.ChainMakerConfig.NodeConfig.SignerCacheSize = 10
	localconf.ChainMakerConfig.NodeConfig.CertCacheSize = 10

	var td = filepath.Join(os.TempDir(), tempDirName)
	os.Mkdir(td, 0666)
	localPrivKeyFile := filepath.Join(td, tempOrg1KeyFileName)
	localCertFile := filepath.Join(td, tempOrg1CertFileName)
	err := ioutil.WriteFile(localPrivKeyFile, []byte(orgList[org1Name].consensusNode.sk), 0666)
	require.Nil(t, err)
	err = ioutil.WriteFile(localCertFile, []byte(orgList[org1Name].consensusNode.certificate), 0666)
	require.Nil(t, err)
	acInst, err := newAccessControlWithChainConfigPb(localPrivKeyFile, "", localCertFile, chainConf, org1Name, nil)
	require.Nil(t, err)
	require.NotNil(t, acInst)
	member, err := acInst.NewMemberFromCertPem(org2Name, orgList[org2Name].consensusNode.certificate)
	require.Nil(t, err)
	require.NotNil(t, member)
	role := member.GetRole()
	require.NotNil(t, role)
}

func TestMemberGetSKI(t *testing.T) {
	localconf.ChainMakerConfig.NodeConfig.SignerCacheSize = 10
	localconf.ChainMakerConfig.NodeConfig.CertCacheSize = 10

	var td = filepath.Join(os.TempDir(), tempDirName)
	os.Mkdir(td, 0666)
	localPrivKeyFile := filepath.Join(td, tempOrg1KeyFileName)
	localCertFile := filepath.Join(td, tempOrg1CertFileName)
	err := ioutil.WriteFile(localPrivKeyFile, []byte(orgList[org1Name].consensusNode.sk), 0666)
	require.Nil(t, err)
	err = ioutil.WriteFile(localCertFile, []byte(orgList[org1Name].consensusNode.certificate), 0666)
	require.Nil(t, err)
	acInst, err := newAccessControlWithChainConfigPb(localPrivKeyFile, "", localCertFile, chainConf, org1Name, nil)
	require.Nil(t, err)
	require.NotNil(t, acInst)
	member, err := acInst.NewMemberFromCertPem(org2Name, orgList[org2Name].consensusNode.certificate)
	require.Nil(t, err)
	require.NotNil(t, member)
	ski := member.GetSKI()
	require.NotNil(t, ski)
}

func TestMemberGetCertificate(t *testing.T) {
	localconf.ChainMakerConfig.NodeConfig.SignerCacheSize = 10
	localconf.ChainMakerConfig.NodeConfig.CertCacheSize = 10

	var td = filepath.Join(os.TempDir(), tempDirName)
	os.Mkdir(td, 0666)
	localPrivKeyFile := filepath.Join(td, tempOrg1KeyFileName)
	localCertFile := filepath.Join(td, tempOrg1CertFileName)
	err := ioutil.WriteFile(localPrivKeyFile, []byte(orgList[org1Name].consensusNode.sk), 0666)
	require.Nil(t, err)
	err = ioutil.WriteFile(localCertFile, []byte(orgList[org1Name].consensusNode.certificate), 0666)
	require.Nil(t, err)
	acInst, err := newAccessControlWithChainConfigPb(localPrivKeyFile, "", localCertFile, chainConf, org1Name, nil)
	require.Nil(t, err)
	require.NotNil(t, acInst)
	member, err := acInst.NewMemberFromCertPem(org2Name, orgList[org2Name].consensusNode.certificate)
	require.Nil(t, err)
	require.NotNil(t, member)
	cert, err := member.GetCertificate()
	require.Nil(t, err)
	require.NotNil(t, cert)
}

func TestMemberSerialize(t *testing.T) {
	localconf.ChainMakerConfig.NodeConfig.SignerCacheSize = 10
	localconf.ChainMakerConfig.NodeConfig.CertCacheSize = 10

	var td = filepath.Join(os.TempDir(), tempDirName)
	os.Mkdir(td, 0666)
	localPrivKeyFile := filepath.Join(td, tempOrg1KeyFileName)
	localCertFile := filepath.Join(td, tempOrg1CertFileName)
	err := ioutil.WriteFile(localPrivKeyFile, []byte(orgList[org1Name].consensusNode.sk), 0666)
	require.Nil(t, err)
	err = ioutil.WriteFile(localCertFile, []byte(orgList[org1Name].consensusNode.certificate), 0666)
	require.Nil(t, err)
	acInst, err := newAccessControlWithChainConfigPb(localPrivKeyFile, "", localCertFile, chainConf, org1Name, nil)
	require.Nil(t, err)
	require.NotNil(t, acInst)
	member, err := acInst.NewMemberFromCertPem(org2Name, orgList[org2Name].consensusNode.certificate)
	require.Nil(t, err)
	require.NotNil(t, member)
	serializedMember, err := member.Serialize(true)
	require.Nil(t, err)
	require.NotNil(t, serializedMember)
}

func TestMemberGetSerializedMember(t *testing.T) {
	localconf.ChainMakerConfig.NodeConfig.SignerCacheSize = 10
	localconf.ChainMakerConfig.NodeConfig.CertCacheSize = 10

	var td = filepath.Join(os.TempDir(), tempDirName)
	os.Mkdir(td, 0666)
	localPrivKeyFile := filepath.Join(td, tempOrg1KeyFileName)
	localCertFile := filepath.Join(td, tempOrg1CertFileName)
	err := ioutil.WriteFile(localPrivKeyFile, []byte(orgList[org1Name].consensusNode.sk), 0666)
	require.Nil(t, err)
	err = ioutil.WriteFile(localCertFile, []byte(orgList[org1Name].consensusNode.certificate), 0666)
	require.Nil(t, err)
	acInst, err := newAccessControlWithChainConfigPb(localPrivKeyFile, "", localCertFile, chainConf, org1Name, nil)
	require.Nil(t, err)
	require.NotNil(t, acInst)
	member, err := acInst.NewMemberFromCertPem(org2Name, orgList[org2Name].consensusNode.certificate)
	require.Nil(t, err)
	require.NotNil(t, member)
	serializedMember, err := member.GetSerializedMember(true)
	require.Nil(t, err)
	require.NotNil(t, serializedMember)
}

func TestMemberSignAndVerify(t *testing.T) {
	localconf.ChainMakerConfig.NodeConfig.SignerCacheSize = 10
	localconf.ChainMakerConfig.NodeConfig.CertCacheSize = 10

	var td = filepath.Join(os.TempDir(), tempDirName)
	os.Mkdir(td, 0666)
	localPrivKeyFile := filepath.Join(td, tempOrg1KeyFileName)
	localCertFile := filepath.Join(td, tempOrg1CertFileName)
	err := ioutil.WriteFile(localPrivKeyFile, []byte(orgList[org1Name].consensusNode.sk), 0666)
	require.Nil(t, err)
	err = ioutil.WriteFile(localCertFile, []byte(orgList[org1Name].consensusNode.certificate), 0666)
	require.Nil(t, err)
	acInst, err := newAccessControlWithChainConfigPb(localPrivKeyFile, "", localCertFile, chainConf, org1Name, nil)
	require.Nil(t, err)
	require.NotNil(t, acInst)
	localSigningMember := acInst.GetLocalSigningMember()
	signRead, err := localSigningMember.Sign(acInst.GetHashAlg(), []byte(msg))
	require.Nil(t, err)
	err = localSigningMember.Verify(acInst.GetHashAlg(), []byte(msg), signRead)
	require.Nil(t, err)
}
