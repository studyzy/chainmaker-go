package accesscontrol

import (
	"fmt"
	"strings"
	"sync"

	"chainmaker.org/chainmaker-go/localconf"
	"chainmaker.org/chainmaker/common/v2/concurrentlru"
	"chainmaker.org/chainmaker/common/v2/crypto"
	"chainmaker.org/chainmaker/common/v2/crypto/asym"
	pbac "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	"chainmaker.org/chainmaker/pb-go/v2/config"
	"chainmaker.org/chainmaker/protocol/v2"
)

var _ protocol.AccessControlProvider = (*pkACProvider)(nil)

var NilPkACProvider ACProvider = (*pkACProvider)(nil)

const AdminPublicKey = "public"

type pkACProvider struct {

	//chainconfig authType
	authType AuthType

	hashType string

	log protocol.Logger

	localOrg string

	adminMember *sync.Map

	consensusMember *sync.Map

	memberCache *concurrentlru.Cache

	dataStore protocol.BlockchainStore
}

type publicAdminMemberModel struct {
	publicKey crypto.PublicKey
	pkPEM     string
}

func (p *pkACProvider) NewACProvider(chainConf protocol.ChainConf, localOrgId string,
	store protocol.BlockchainStore, log protocol.Logger) (protocol.AccessControlProvider, error) {
	pkAcProvider, err := newPkACProvider(chainConf.ChainConfig(), localOrgId, store, log)
	if err != nil {
		return nil, err
	}
	chainConf.AddWatch(pkAcProvider)
	return pkAcProvider, nil
}

func newPkACProvider(chainConfig *config.ChainConfig, localOrgId string,
	store protocol.BlockchainStore, log protocol.Logger) (*pkACProvider, error) {
	pkAcProvider := &pkACProvider{
		authType:        StringToAuthTypeMap[chainConfig.AuthType],
		adminMember:     &sync.Map{},
		consensusMember: &sync.Map{},
		localOrg:        localOrgId,
		memberCache:     concurrentlru.New(localconf.ChainMakerConfig.NodeConfig.CertCacheSize),
		log:             log,
	}

	return pkAcProvider, nil
}

func (p *pkACProvider) initAdminMembers(trustRootList []*config.TrustRootConfig) error {
	var (
		tempSyncMap sync.Map
	)

	if len(trustRootList) == 0 {
		return fmt.Errorf("init admin member failed: trsut root can't be empty")
	}

	for _, trustRoot := range trustRootList {
		if strings.ToLower(trustRoot.OrgId) == AdminPublicKey {
			for _, root := range trustRoot.Root {
				pk, err := asym.PublicKeyFromPEM([]byte(root))
				if err != nil {
					return fmt.Errorf("init admin member failed: parse the public key from PEM failed")
				}
				adminMember := &publicAdminMemberModel{
					publicKey: pk,
					pkPEM:     root,
				}
				tempSyncMap.Store(root, adminMember)
			}
		}
	}
	p.adminMember = &tempSyncMap
	return nil
}

func (p *pkACProvider) lookUpMemberInCache(memberInfo string) (*memberCached, bool) {
	ret, ok := p.memberCache.Get(memberInfo)
	if ok {
		return ret.(*memberCached), true
	}
	return nil, false
}

func (p *pkACProvider) addMemberToCache(memberInfo string, member *memberCached) {
	p.memberCache.Add(memberInfo, member)
}

func (p *pkACProvider) Module() string {
	return ModuleNameAccessControl
}

func (p *pkACProvider) Watch(chainConfig *config.ChainConfig) error {

	return nil
}

func (p *pkACProvider) NewMember(member *pbac.Member) (protocol.Member, error) {
	memberCached, ok := p.lookUpMemberInCache(string(member.MemberInfo))
	if ok {
		p.log.Debugf("member found in local cache")
		return memberCached.member, nil
	}
	return publicNewPkMemberFromAcs(member, p.adminMember, p.consensusMember, p.hashType)
}
