package accesscontrol

import (
	"reflect"

	pbac "chainmaker.org/chainmaker/pb-go/accesscontrol"
	"chainmaker.org/chainmaker/protocol"
)

func init() {
	RegisterMemberProvider(pbac.MemberType_CERT, NilCertMemberProvider)
	RegisterMemberProvider(pbac.MemberType_CERT_HASH, NilCertMemberProvider)
	RegisterMemberProvider(pbac.MemberType_PUBLIC_KEY, NilPkMemberProvider)
}

var memberRegistry = map[pbac.MemberType]reflect.Type{}

type MemberProvider interface {
	NewMember(member *pbac.Member, acs *accessControlService) (protocol.Member, error)
}

func RegisterMemberProvider(memberType pbac.MemberType, mp MemberProvider) {
	_, found := memberRegistry[memberType]
	if found {
		panic("accesscontrol member provider[" + memberType.String() + "] already registered!")
	}
	memberRegistry[memberType] = reflect.TypeOf(mp)
}

func NewMemberByMemberType(memberType pbac.MemberType) MemberProvider {
	t, found := memberRegistry[memberType]
	if !found {
		panic("accesscontrol member provider[" + memberType.String() + "] not found!")
	}

	return reflect.New(t).Elem().Interface().(MemberProvider)
}
