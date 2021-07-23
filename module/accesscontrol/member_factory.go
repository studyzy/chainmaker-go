package accesscontrol

import (
	"sync"

	pbac "chainmaker.org/chainmaker/pb-go/accesscontrol"
	"chainmaker.org/chainmaker/protocol"
)

type memberFactory struct {
}

var once sync.Once
var _instance *memberFactory

func Factory() *memberFactory {
	once.Do(func() { _instance = new(memberFactory) })
	return _instance
}

func (mf *memberFactory) NewMember(pbMember *pbac.Member, ac *accessControl) (protocol.Member, error) {
	p := NewMemberByMemberType(pbMember.MemberType)
	return p.NewMember(pbMember, ac)
}
