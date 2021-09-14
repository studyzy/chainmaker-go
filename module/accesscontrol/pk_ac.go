package accesscontrol

import (
	"sync"

	"chainmaker.org/chainmaker/protocol/v2"
)

var _ protocol.AccessControlProvider = (*pkACProvider)(nil)

var NilPkACProvider ACProvider = (*pkACProvider)(nil)

type pkACProvider struct {

	//chainconfig authType
	authType AuthType

	hashType string

	log protocol.Logger

	localOrg string

	adminMember *sync.Map

	consensusMember *sync.Map
}
