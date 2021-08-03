package accesscontrol

import (
	"sync"

	pbac "chainmaker.org/chainmaker/pb-go/accesscontrol"
	"chainmaker.org/chainmaker/protocol"
)

type memberFactory struct {
}

var once sync.Once
var mem_instance *memberFactory

func MemberFactory() *memberFactory {
	once.Do(func() { mem_instance = new(memberFactory) })
	return mem_instance
}

func (mf *memberFactory) NewMember(pbMember *pbac.Member, acs *accessControlService) (protocol.Member, error) {
	p := NewMemberByMemberType(pbMember.MemberType)
	return p.NewMember(pbMember, acs)
}
