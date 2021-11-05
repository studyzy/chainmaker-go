/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package accesscontrol

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"encoding/hex"

	"chainmaker.org/chainmaker/common/v2/crypto"
	"chainmaker.org/chainmaker/common/v2/crypto/asym"
	"chainmaker.org/chainmaker/localconf/v2"
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
	pkBytes   []byte
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
	chainConfig.AuthType = strings.ToLower(chainConfig.AuthType)
	ppacProvider.acService = initAccessControlService(chainConfig.GetCrypto().Hash,
		chainConfig.AuthType, store, log)

	err := ppacProvider.initAdminMembers(chainConfig.TrustRoots)
	if err != nil {
		return nil, err
	}

	err = ppacProvider.initConsensusMember(chainConfig.Consensus.Nodes)
	if err != nil {
		return nil, err
	}

	ppacProvider.acService.initResourcePolicy(chainConfig.ResourcePolicies, localOrgId)

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

			pkBytes, err := pk.Bytes()
			if err != nil {
				return fmt.Errorf("init admin member failed: %s", err.Error())
			}

			adminMember := &adminMemberModel{
				publicKey: pk,
				pkBytes:   pkBytes,
				orgId:     trustRoot.OrgId,
			}
			adminKey := hex.EncodeToString(pkBytes)
			tempSyncMap.Store(adminKey, adminMember)
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
	return []string{syscontract.SystemContract_PUBKEY_MANAGE.String()}
}

func (pp *permissionedPkACProvider) Callback(contractName string, payloadBytes []byte) error {
	switch contractName {
	case syscontract.SystemContract_PUBKEY_MANAGE.String():
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
	case syscontract.PubkeyManageFunction_PUBKEY_DELETE.String():
		return pp.systemContractCallbackPublicKeyManagementDeleteCase(&payload)
	default:
		pp.acService.log.Debugf("unwatched method [%s]", payload.Method)
		return nil
	}
}

func (pp *permissionedPkACProvider) systemContractCallbackPublicKeyManagementDeleteCase(payload *common.Payload) error {
	for _, param := range payload.Parameters {
		if param.Key == PUBLIC_KEYS {
			pk, err := asym.PublicKeyFromPEM(param.Value)
			if err != nil {
				return fmt.Errorf("delete member cache failed, [%v]", err.Error())
			}
			pkStr, err := pk.String()
			if err != nil {
				return fmt.Errorf("delete member cache failed, [%v]", err.Error())
			}
			pp.acService.memberCache.Remove(pkStr)
			pp.acService.log.Debugf("The public key was removed from the cache,[%v]", pkStr)
		}
	}
	return nil
}

// all-in-one validation for signing members: certificate chain/whitelist, signature, policies
func (pp *permissionedPkACProvider) refinePrincipal(principal protocol.Principal) (protocol.Principal, error) {
	endorsements := principal.GetEndorsement()
	msg := principal.GetMessage()
	refinedEndorsement := pp.refineEndorsements(endorsements, msg)
	if len(refinedEndorsement) <= 0 {
		return nil, fmt.Errorf("refine endorsements failed, all endorsers have failed verification")
	}

	refinedPrincipal, err := pp.CreatePrincipal(principal.GetResourceName(), refinedEndorsement, msg)
	if err != nil {
		return nil, fmt.Errorf("create principal failed: [%s]", err.Error())
	}

	return refinedPrincipal, nil
}

func (pp *permissionedPkACProvider) refineEndorsements(endorsements []*common.EndorsementEntry,
	msg []byte) []*common.EndorsementEntry {

	refinedSigners := map[string]bool{}
	var refinedEndorsement []*common.EndorsementEntry

	for _, endorsementEntry := range endorsements {
		endorsement := &common.EndorsementEntry{
			Signer: &pbac.Member{
				OrgId:      endorsementEntry.Signer.OrgId,
				MemberInfo: endorsementEntry.Signer.MemberInfo,
				MemberType: endorsementEntry.Signer.MemberType,
			},
			Signature: endorsementEntry.Signature,
		}

		memInfo := string(endorsement.Signer.MemberInfo)

		remoteMember, err := pp.NewMember(endorsement.Signer)
		if err != nil {
			pp.acService.log.Infof("new member failed: [%s]", err.Error())
			continue
		}

		if err := remoteMember.Verify(pp.GetHashAlg(), msg, endorsement.Signature); err != nil {
			pp.acService.log.Infof("signer member verify signature failed: [%s]", err.Error())
			pp.acService.log.Debugf("information for invalid signature:\norganization: %s\npubkey: %s\nmessage: %s\n"+
				"signature: %s", endorsement.Signer.OrgId, memInfo, hex.Dump(msg), hex.Dump(endorsement.Signature))
			continue
		}

		if _, ok := refinedSigners[memInfo]; !ok {
			refinedSigners[memInfo] = true
			refinedEndorsement = append(refinedEndorsement, endorsement)
		}
	}
	return refinedEndorsement
}

func (pp *permissionedPkACProvider) NewMember(member *pbac.Member) (protocol.Member, error) {
	return pp.acService.newPkMember(member, pp.adminMember, pp.consensusMember)
}

// GetHashAlg return hash algorithm the access control provider uses
func (pp *permissionedPkACProvider) GetHashAlg() string {
	return pp.acService.hashType
}

// ValidateResourcePolicy checks whether the given resource principal is valid
func (pp *permissionedPkACProvider) ValidateResourcePolicy(resourcePolicy *config.ResourcePolicy) bool {
	return pp.acService.validateResourcePolicy(resourcePolicy)
}

// CreatePrincipalForTargetOrg creates a principal for "SELF" type principal,
// which needs to convert SELF to a sepecific organization id in one authentication
func (pp *permissionedPkACProvider) CreatePrincipalForTargetOrg(resourceName string,
	endorsements []*common.EndorsementEntry, message []byte,
	targetOrgId string) (protocol.Principal, error) {
	return pp.acService.createPrincipalForTargetOrg(resourceName, endorsements, message, targetOrgId)
}

// CreatePrincipal creates a principal for one time authentication
func (pp *permissionedPkACProvider) CreatePrincipal(resourceName string, endorsements []*common.EndorsementEntry,
	message []byte) (
	protocol.Principal, error) {
	return pp.acService.createPrincipal(resourceName, endorsements, message)
}

func (pp *permissionedPkACProvider) LookUpPolicy(resourceName string) (*pbac.Policy, error) {
	return pp.acService.lookUpPolicy(resourceName)
}

func (pp *permissionedPkACProvider) LookUpExceptionalPolicy(resourceName string) (*pbac.Policy, error) {
	return pp.acService.lookUpExceptionalPolicy(resourceName)
}

func (pp *permissionedPkACProvider) GetMemberStatus(member *pbac.Member) (pbac.MemberStatus, error) {
	if _, err := pp.newNodeMember(member); err != nil {
		pp.acService.log.Infof("get member status: %s", err.Error())
		return pbac.MemberStatus_INVALID, err
	}
	return pbac.MemberStatus_NORMAL, nil
}

func (pp *permissionedPkACProvider) VerifyRelatedMaterial(verifyType pbac.VerifyType, data []byte) (bool, error) {
	return true, nil
}

// VerifyPrincipal verifies if the principal for the resource is met
func (pp *permissionedPkACProvider) VerifyPrincipal(principal protocol.Principal) (bool, error) {

	if atomic.LoadInt32(&pp.acService.orgNum) <= 0 {
		return false, fmt.Errorf("authentication failed: empty organization list or trusted node list on this chain")
	}

	refinedPrincipal, err := pp.refinePrincipal(principal)
	if err != nil {
		return false, fmt.Errorf("authentication failed, [%s]", err.Error())
	}

	if localconf.ChainMakerConfig.DebugConfig.IsSkipAccessControl {
		return true, nil
	}

	p, err := pp.acService.lookUpPolicyByResourceName(principal.GetResourceName())
	if err != nil {
		return false, fmt.Errorf("authentication failed, [%s]", err.Error())
	}

	return pp.acService.verifyPrincipalPolicy(principal, refinedPrincipal, p)
}

//GetValidEndorsements filters all endorsement entries and returns all valid ones
func (pp *permissionedPkACProvider) GetValidEndorsements(principal protocol.Principal) (
	[]*common.EndorsementEntry, error) {
	if atomic.LoadInt32(&pp.acService.orgNum) <= 0 {
		return nil, fmt.Errorf("authentication fail: empty organization list or trusted node list on this chain")
	}
	refinedPolicy, err := pp.refinePrincipal(principal)
	if err != nil {
		return nil, fmt.Errorf("authentication fail, not a member on this chain: [%v]", err)
	}
	endorsements := refinedPolicy.GetEndorsement()

	p, err := pp.acService.lookUpPolicyByResourceName(principal.GetResourceName())
	if err != nil {
		return nil, fmt.Errorf("authentication fail: [%v]", err)
	}
	orgListRaw := p.GetOrgList()
	roleListRaw := p.GetRoleList()
	orgList := map[string]bool{}
	roleList := map[protocol.Role]bool{}
	for _, orgRaw := range orgListRaw {
		orgList[orgRaw] = true
	}
	for _, roleRaw := range roleListRaw {
		roleList[roleRaw] = true
	}
	return pp.acService.getValidEndorsements(orgList, roleList, endorsements), nil
}

func (pp *permissionedPkACProvider) newNodeMember(member *pbac.Member) (protocol.Member, error) {
	return pp.acService.newNodePkMember(member, pp.consensusMember)
}
