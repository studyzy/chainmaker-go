/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package accesscontrol

import (
	"encoding/hex"
	"fmt"
	"sync"

	commonCert "chainmaker.org/chainmaker/common/v2/cert"
	bccrypto "chainmaker.org/chainmaker/common/v2/crypto"
	"chainmaker.org/chainmaker/common/v2/crypto/asym"
	"chainmaker.org/chainmaker/common/v2/helper"
	pbac "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"
	"chainmaker.org/chainmaker/protocol/v2"
	"github.com/gogo/protobuf/proto"
)

var _ protocol.Member = (*pkMember)(nil)

// an instance whose member type is a certificate
type pkMember struct {

	// pem public key
	id string

	// organization identity who owns this member
	orgId string

	// public key uid
	uid string

	// the public key used for authentication
	pk bccrypto.PublicKey

	// role of this member
	role protocol.Role

	// hash type from chain configuration
	hashType string
}

func (pm *pkMember) GetMemberId() string {
	return pm.id
}

func (pm *pkMember) GetOrgId() string {
	return pm.orgId
}

func (pm *pkMember) GetRole() protocol.Role {
	return pm.role
}

func (pm *pkMember) GetUid() string {
	return pm.uid
}

func (pm *pkMember) Verify(hashType string, msg []byte, sig []byte) error {

	hash, ok := bccrypto.HashAlgoMap[hashType]
	if !ok {
		return fmt.Errorf("cert member verify signature failed: unsupport hash type")
	}
	ok, err := pm.pk.VerifyWithOpts(msg, sig, &bccrypto.SignOpts{
		Hash: hash,
		UID:  bccrypto.CRYPTO_DEFAULT_UID,
	})
	if err != nil {
		return fmt.Errorf("cert member verify signature failed: [%s]", err.Error())
	}
	if !ok {
		return fmt.Errorf("cert member verify signature failed: invalid signature")
	}
	return nil
}

func (pm *pkMember) GetMember() (*pbac.Member, error) {
	memberInfo, err := pm.pk.String()
	if err != nil {
		return nil, fmt.Errorf("get pb member failed: %s", err.Error())
	}
	return &pbac.Member{
		OrgId:      pm.orgId,
		MemberInfo: []byte(memberInfo),
		MemberType: pbac.MemberType_PUBLIC_KEY,
	}, nil
}

type signingPKMember struct {
	// Extends Identity
	pkMember

	// Sign the message
	sk bccrypto.PrivateKey
}

// When using public key instead of certificate,
//hashType is used to specify the hash algorithm while the signature algorithm is decided by the public key itself.
func (spm *signingPKMember) Sign(hashType string, msg []byte) ([]byte, error) {
	hash, ok := bccrypto.HashAlgoMap[hashType]
	if !ok {
		return nil, fmt.Errorf("sign failed: unsupport hash type")
	}
	return spm.sk.SignWithOpts(msg, &bccrypto.SignOpts{
		Hash: hash,
		UID:  bccrypto.CRYPTO_DEFAULT_UID,
	})
}

func newPkMemberFromAcs(member *pbac.Member, adminList, consensusList *sync.Map,
	acs *accessControlService) (*pkMember, error) {

	if member.MemberType != pbac.MemberType_PUBLIC_KEY {
		return nil, fmt.Errorf("new public key member failed: memberType and authType do not match")
	}
	pk, err := asym.PublicKeyFromPEM(member.MemberInfo)
	if err != nil {
		return nil, fmt.Errorf("new public key member failed: parse the public key from PEM failed")
	}
	pkBytes, err := pk.Bytes()
	if err != nil {
		return nil, fmt.Errorf("new public key member failed: %s", err.Error())
	}

	adminMember, ok := adminList.Load(hex.EncodeToString(pkBytes))
	if ok {
		admin, _ := adminMember.(*adminMemberModel)
		return newPkMemberFromParam(admin.orgId, admin.pkBytes, protocol.RoleAdmin, acs.hashType)
	}

	var nodeId string
	nodeId, err = helper.CreateLibp2pPeerIdWithPublicKey(pk)
	if err != nil {
		return nil, fmt.Errorf("new public key member failed: create libp2p peer id with pk failed")
	}

	consensusMember, ok := consensusList.Load(nodeId)
	if ok {
		consensus, _ := consensusMember.(*consensusMemberModel)
		return newPkMemberFromParam(consensus.orgId, pkBytes,
			protocol.RoleConsensusNode, acs.hashType)
	}

	publicKeyIdex := pubkeyHash(pkBytes)
	publicKeyInfoBytes, err := acs.dataStore.ReadObject(syscontract.SystemContract_PUBKEY_MANAGE.String(),
		[]byte(publicKeyIdex))
	if err != nil {
		return nil, fmt.Errorf("new public key member failed: %s", err.Error())
	}

	if publicKeyInfoBytes == nil {
		return nil, fmt.Errorf("new public key member failed: the public key doesn't belong to a member on chain")
	}

	var publickInfo pbac.PKInfo
	err = proto.Unmarshal(publicKeyInfoBytes, &publickInfo)
	if err != nil {
		return nil, fmt.Errorf("new public key member failed: %s", err.Error())
	}

	return newPkMemberFromParam(publickInfo.OrgId, pkBytes,
		protocol.Role(publickInfo.Role), acs.hashType)
}

func newPkNodeMemberFromAcs(member *pbac.Member, consensusList *sync.Map,
	acs *accessControlService) (*pkMember, error) {

	if member.MemberType != pbac.MemberType_PUBLIC_KEY {
		return nil, fmt.Errorf("new public key node member failed: memberType and authType do not match")
	}
	pk, err := asym.PublicKeyFromPEM(member.MemberInfo)
	if err != nil {
		return nil, fmt.Errorf("new public key node member failed: parse the public key from PEM failed")
	}
	pkBytes, err := pk.Bytes()
	if err != nil {
		return nil, fmt.Errorf("new public key node member failed: %s", err.Error())
	}

	var nodeId string
	nodeId, err = helper.CreateLibp2pPeerIdWithPublicKey(pk)
	if err != nil {
		return nil, fmt.Errorf("new public key node member failed: create libp2p peer id with pk failed")
	}

	consensusMember, ok := consensusList.Load(nodeId)
	if ok {
		consensus, _ := consensusMember.(*consensusMemberModel)
		return newPkMemberFromParam(consensus.orgId, pkBytes,
			protocol.RoleConsensusNode, acs.hashType)
	}

	publicKeyIdex := pubkeyHash(pkBytes)
	publicKeyInfoBytes, err := acs.dataStore.ReadObject(syscontract.SystemContract_PUBKEY_MANAGE.String(),
		[]byte(publicKeyIdex))
	if err != nil {
		return nil, fmt.Errorf("new public key node member failed: %s", err.Error())
	}

	if publicKeyInfoBytes == nil {
		return nil, fmt.Errorf("new public key node member failed: the public key doesn't belong to a member on chain")
	}

	var publickInfo pbac.PKInfo
	err = proto.Unmarshal(publicKeyInfoBytes, &publickInfo)
	if err != nil {
		return nil, fmt.Errorf("new public key node member failed: %s", err.Error())
	}

	if publickInfo.Role != string(protocol.RoleCommonNode) {

		return nil, fmt.Errorf("new public key node member failed: the public key is not a node member")
	}

	return newPkMemberFromParam(publickInfo.OrgId, pkBytes,
		protocol.Role(publickInfo.Role), acs.hashType)
}

func publicNewPkMemberFromAcs(member *pbac.Member, adminList,
	consensusList *sync.Map, hashType string) (*pkMember, error) {
	if member.MemberType != pbac.MemberType_PUBLIC_KEY {
		return nil, fmt.Errorf("new public key member failed: memberType and authType do not match")
	}

	pk, err := asym.PublicKeyFromPEM(member.MemberInfo)
	if err != nil {
		return nil, fmt.Errorf("new public key member failed: parse the public key from PEM failed")
	}
	pkBytes, err := pk.Bytes()
	if err != nil {
		return nil, fmt.Errorf("new public key member failed: %s", err.Error())
	}

	adminMember, ok := adminList.Load(hex.EncodeToString(pkBytes))
	if ok {
		admin, _ := adminMember.(*publicAdminMemberModel)
		return newPkMemberFromParam("", admin.pkBytes, protocol.RoleAdmin, hashType)
	}

	var nodeId string
	nodeId, err = helper.CreateLibp2pPeerIdWithPublicKey(pk)
	if err != nil {
		return nil, fmt.Errorf("new public key member failed: create libp2p peer id with pk failed")
	}

	_, ok = consensusList.Load(nodeId)
	if ok {
		return newPkMemberFromParam("", pkBytes,
			protocol.RoleConsensusNode, hashType)
	}
	return newPkMemberFromParam("", pkBytes, protocol.Role(""), hashType)
}

func newPkMemberFromParam(orgId string, pkBytes []byte, role protocol.Role,
	hashType string) (*pkMember, error) {

	hash, ok := bccrypto.HashAlgoMap[hashType]
	if !ok {
		return nil, fmt.Errorf("sign failed: unsupport hash type")
	}

	var pkMem pkMember
	pkMem.orgId = orgId
	pkMem.hashType = hashType

	pk, err := asym.PublicKeyFromDER(pkBytes)
	if err != nil {
		return nil, fmt.Errorf("setup pk member failed, err: %s", err.Error())
	}

	pkMem.pk = pk
	pkMem.role = role
	ski, err := commonCert.ComputeSKI(hash, pk.ToStandardKey())

	if err != nil {
		return nil, fmt.Errorf("setup pk member failed, err: %s", err.Error())
	}

	pkMem.uid = hex.EncodeToString(ski)
	pkPem, err := pk.String()
	if err != nil {
		return nil, fmt.Errorf("setup pk member failed, err: %s", err.Error())
	}
	pkMem.id = pkPem

	return &pkMem, nil
}
