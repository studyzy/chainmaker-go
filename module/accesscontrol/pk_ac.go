package accesscontrol

// import (
// 	"sync"

// 	"chainmaker.org/chainmaker/pb-go/config"
// 	"chainmaker.org/chainmaker/protocol"
// )

// type pkACProvider struct {
// 	acService       *accessControlService
// 	hashType        string
// 	log             protocol.Logger
// 	localOrg        string
// 	adminMember     sync.Map
// 	consensusMember sync.Map
// }

// var _ protocol.AccessControlProvider = (*pkACProvider)(nil)

// var NilPkACProvider ACProvider = (*pkACProvider)(nil)

// func (pp *pkACProvider) NewACProvider(chainConf protocol.ChainConf, localOrgId string,
// 	store protocol.BlockchainStore, log protocol.Logger) (protocol.AccessControlProvider, error) {
// }

// func newPkACProvider(chainConf protocol.ChainConf, localOrgId string,
// 	store protocol.BlockchainStore, log protocol.Logger) (*pkACProvider, error) {
// 	pkACProvider := &pkACProvider{
// 		hashType:        chainConf.ChainConfig().GetCrypto().Hash,
// 		log:             log,
// 		localOrg:        localOrgId,
// 		adminMember:     sync.Map{},
// 		consensusMember: sync.Map{},
// 	}
// 	pkACProvider.acService = initAccessControlService(pkACProvider.hashType,
// 		chainConf, store, log)
// 	//TODO load adminMember and load consensusMember
// }

// func (pp *pkACProvider) loadAdminMember(rootConf []*config.TrustRootConfig) error {
// 	if rootConf == nil {
// 		pp.log.Debug("there is no trust root")
// 	}
// 	for _, root := range rootConf {
// 		pp.acService.addOrg(root.OrgId, root.Root)
// 		for _, adminMember := range root.Root {
// 			pp.adminMember.Store(adminMember, true)
// 		}
// 	}
// }

// func (pp *pkACProvider) loadConsensusMember() error {

// }
