/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package accesscontrol

import (
	"chainmaker.org/chainmaker-go/localconf"
	"chainmaker.org/chainmaker/common/v2/concurrentlru"
	bcx509 "chainmaker.org/chainmaker/common/v2/crypto/x509"
	"chainmaker.org/chainmaker/pb-go/v2/config"
	"chainmaker.org/chainmaker/protocol/v2"
	"crypto/x509"
	"sync"
)

var _ protocol.AccessControlProvider = (*pkACProvider)(nil)

var NilPkACProvider ACProvider = (*pkACProvider{})(nil)

type pkACProvider struct {
	//chainconfig authType
	authType string

	acService *accessControlService

	hashType string

	log protocol.Logger

	localOrg string

	adminMember *sync.Map

	consensusMember *sync.Map
}

func (pp *pkACProvider) NewPkProvider(chainConf protocol.ChainConf, localOrgId string,
	store protocol.BlockchainStore, log protocol.Logger) (protocol.AccessControlProvider, error) {
	pkACProvider, err := newPkACProvider(chainConf.ChainConfig(), localOrgId, store, log)
	if err != nil {
		return nil, err
	}
	chainConf.AddWatch(pkACProvider)
	chainConf.AddVmWatch(pkACProvider)
	return pkACProvider, nil
}

func newPkACProvider(chainConfig *config.ChainConfig, localOrgId string,
	store protocol.BlockchainStore, log protocol.Logger) (*certACProvider, error) {
	pkACProvider := &pkACProvider{
		hashType:   chainConfig.GetCrypto().Hash,
		localOrg: nil,
		log:      log,
		authType:
	}
	certACProvider.acService = initAccessControlService(certACProvider.hashType, localOrgId, chainConfig, store, log)

	err := certACProvider.initTrustRoots(chainConfig.TrustRoots, localOrgId)
	if err != nil {
		return nil, err
	}

	certACProvider.opts.KeyUsages = make([]x509.ExtKeyUsage, 1)
	certACProvider.opts.KeyUsages[0] = x509.ExtKeyUsageAny

	if err := certACProvider.loadCRL(); err != nil {
		return nil, err
	}

	if err := certACProvider.loadCertFrozenList(); err != nil {
		return nil, err
	}
	return certACProvider, nil
}