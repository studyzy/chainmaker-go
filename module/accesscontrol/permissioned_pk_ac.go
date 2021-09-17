/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package accesscontrol

import (
	"encoding/pem"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"chainmaker.org/chainmaker/common/v2/crypto"
	"chainmaker.org/chainmaker/common/v2/crypto/asym"
	pbac "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/config"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"
	"chainmaker.org/chainmaker/protocol/v2"
	"github.com/gogo/protobuf/proto"
)

var _ protocol.AccessControlProvider = (*permissionedPkACProvider)(nil)

var NilPermissionedPkACProvider ACProvider = (*permissionedPkACProvider)(nil)

type permissionedPkACProvider struct {
	acService *accessControlService

	// local org id
	localOrg string

	// admin list in permissioned public key mode
	adminMember *sync.Map

	// consensus list in permissioned public key mode
	consensusMember *sync.Map
}

type adminMemberModel struct {
	publicKey crypto.PublicKey
	pkPEM     string
	orgId     string
}

type consensusMemberModel struct {
	nodeId string
	orgId  string
}

func (pp *permissionedPkACProvider) NewACProvider(chainConf protocol.ChainConf, localOrgId string,
	store protocol.BlockchainStore, log protocol.Logger) (protocol.AccessControlProvider, error) {
	pPkACProvider, err := newPermissionedPkACProvider(chainConf.ChainConfig(), localOrgId, store, log)
	if err != nil {
		return nil, err
	}
	chainConf.AddWatch(pPkACProvider)
	chainConf.AddVmWatch(pPkACProvider)
	return pPkACProvider, nil
}

func newPermissionedPkACProvider(chainConfig *config.ChainConfig, localOrgId string,
	store protocol.BlockchainStore, log protocol.Logger) (*permissionedPkACProvider, error) {
	ppacProvider := &permissionedPkACProvider{
		adminMember:     &sync.Map{},
		consensusMember: &sync.Map{},
		localOrg:        localOrgId,
	}

	ppacProvider.acService = initAccessControlService(chainConfig.GetCrypto().Hash, localOrgId, chainConfig, store, log)

	err := ppacProvider.initAdminMembers(chainConfig.TrustRoots)
	if err != nil {
		return nil, err
	}

	err = ppacProvider.initConsensusMember(chainConfig.Consensus.Nodes)
	if err != nil {
		return nil, err
	}

	return ppacProvider, nil
}

func (pp *permissionedPkACProvider) initAdminMembers(trustRootList []*config.TrustRootConfig) error {
	var (
		tempSyncMap, orgList sync.Map
		orgNum               int32
	)
	for _, trustRoot := range trustRootList {
		for _, root := range trustRoot.Root {
			pk, err := asym.PublicKeyFromPEM([]byte(root))
			if err != nil {
				return fmt.Errorf("init admin member failed: parse the public key from PEM failed")
			}
			adminMember := &adminMemberModel{
				publicKey: pk,
				pkPEM:     root,
				orgId:     trustRoot.OrgId,
			}
			tempSyncMap.Store(root, adminMember)
		}

		_, ok := orgList.Load(trustRoot.OrgId)
		if !ok {
			orgList.Store(trustRoot.OrgId, struct{}{})
			orgNum++
		}
	}
	atomic.StoreInt32(&pp.acService.orgNum, orgNum)
	pp.acService.orgList = &orgList
	pp.adminMember = &tempSyncMap
	return nil
}

func (pp *permissionedPkACProvider) initConsensusMember(consensusConf []*config.OrgConfig) error {
	var tempSyncMap sync.Map
	for _, conf := range consensusConf {
		for _, node := range conf.NodeId {

			consensusMember := &consensusMemberModel{
				nodeId: node,
				orgId:  conf.OrgId,
			}
			tempSyncMap.Store(node, consensusMember)
		}
	}
	pp.consensusMember = &tempSyncMap
	return nil
}

func (pp *permissionedPkACProvider) NewMember(member *pbac.Member) (protocol.Member, error) {
	return pp.acService.newPkMember(member, pp.adminMember, pp.consensusMember)
}

func (pp *permissionedPkACProvider) Module() string {
	return ModuleNameAccessControl
}

func (pp *permissionedPkACProvider) Watch(chainConfig *config.ChainConfig) error {
	pp.acService.hashType = chainConfig.GetCrypto().GetHash()
	err := pp.initAdminMembers(chainConfig.TrustRoots)
	if err != nil {
		return fmt.Errorf("update chainconfig error: %s", err.Error())
	}

	err = pp.initConsensusMember(chainConfig.Consensus.Nodes)
	if err != nil {
		return fmt.Errorf("update chainconfig error: %s", err.Error())
	}

	pp.acService.initResourcePolicy(chainConfig.ResourcePolicies, pp.localOrg)

	pp.acService.memberCache.Clear()

	return nil
}

func (pp *permissionedPkACProvider) ContractNames() []string {
	return []string{syscontract.SystemContract_PUBKEY_MANAGEMENT.String()}
}

func (pp *permissionedPkACProvider) Callback(contractName string, payloadBytes []byte) error {
	switch contractName {
	case syscontract.SystemContract_PUBKEY_MANAGEMENT.String():
		return pp.systemContractCallbackPublicKeyManagementCase(payloadBytes)
	default:
		pp.acService.log.Debugf("unwatched smart contract [%s]", contractName)
		return nil
	}
}

func (pp *permissionedPkACProvider) systemContractCallbackPublicKeyManagementCase(payloadBytes []byte) error {
	var payload common.Payload
	err := proto.Unmarshal(payloadBytes, &payload)
	if err != nil {
		return fmt.Errorf("resolve payload failed: %v", err)
	}
	switch payload.Method {
	case syscontract.CertManageFunction_CERTS_FREEZE.String():
		return pp.systemContractCallbackPublicKeyManagementDeleteCase(&payload)
	default:
		pp.acService.log.Debugf("unwatched method [%s]", payload.Method)
		return nil
	}
}

func (permissionedPkACProvider *permissionedPkACProvider) systemContractCallbackPublicKeyManagementDeleteCase(payload *common.Payload) error {
	for _, param := range payload.Parameters {
		if param.Key == PARAM_CERTS {
			certList := strings.Replace(string(param.Value), ",", "\n", -1)
			certBlock, rest := pem.Decode([]byte(certList))
			for certBlock != nil {
				cp.frozenList.Store(string(certBlock.Bytes), true)

				certBlock, rest = pem.Decode(rest)
			}
			return nil
		}
	}
	return nil
}
