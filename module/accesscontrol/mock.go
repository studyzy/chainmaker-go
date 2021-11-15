/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package accesscontrol

import (
	"sync"

	"chainmaker.org/chainmaker/common/v2/concurrentlru"
	bcx509 "chainmaker.org/chainmaker/common/v2/crypto/x509"
	"chainmaker.org/chainmaker/logger/v2"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/protocol/v2"
)

var mockAcLogger = logger.GetLogger(logger.MODULE_ACCESS)

func MockAccessControl() protocol.AccessControlProvider {
	certAc := &certACProvider{
		acService: &accessControlService{
			orgList:               &sync.Map{},
			orgNum:                0,
			resourceNamePolicyMap: &sync.Map{},
			hashType:              "",
			dataStore:             nil,
			memberCache:           concurrentlru.New(0),
			log:                   mockAcLogger,
		},
		certCache:  concurrentlru.New(0),
		crl:        sync.Map{},
		frozenList: sync.Map{},
		opts: bcx509.VerifyOptions{
			Intermediates: bcx509.NewCertPool(),
			Roots:         bcx509.NewCertPool(),
		},
		localOrg: nil,
	}
	return certAc
}

func MockAccessControlWithHash(hashAlg string) protocol.AccessControlProvider {
	certAc := &certACProvider{
		acService: &accessControlService{
			orgList:               &sync.Map{},
			orgNum:                0,
			resourceNamePolicyMap: &sync.Map{},
			hashType:              hashAlg,
			dataStore:             nil,
			memberCache:           concurrentlru.New(0),
			log:                   mockAcLogger,
		},
		certCache:  concurrentlru.New(0),
		crl:        sync.Map{},
		frozenList: sync.Map{},
		opts: bcx509.VerifyOptions{
			Intermediates: bcx509.NewCertPool(),
			Roots:         bcx509.NewCertPool(),
		},
		localOrg: nil,
	}
	return certAc
}

func MockSignWithMultipleNodes(msg []byte, signers []protocol.SigningMember, hashType string) (
	[]*commonPb.EndorsementEntry, error) {
	var ret []*commonPb.EndorsementEntry
	for _, signer := range signers {
		sig, err := signer.Sign(hashType, msg)
		if err != nil {
			return nil, err
		}
		signerSerial, err := signer.GetMember()
		if err != nil {
			return nil, err
		}
		ret = append(ret, &commonPb.EndorsementEntry{
			Signer:    signerSerial,
			Signature: sig,
		})
	}
	return ret, nil
}

func NewAccessControlWithChainConfig(chainConfig protocol.ChainConf, localOrgId string,
	store protocol.BlockchainStore, log protocol.Logger) (
	protocol.AccessControlProvider, error) {
	conf := chainConfig.ChainConfig()
	acp, err := newCertACProvider(conf, localOrgId, store, log)
	if err != nil {
		return nil, err
	}
	chainConfig.AddWatch(acp)
	chainConfig.AddVmWatch(acp)
	//InitCertSigningMember(testChainConfig, localOrgId, localPrivKeyFile, localPrivKeyPwd, localCertFile)
	return acp, err
}
