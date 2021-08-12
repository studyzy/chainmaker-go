/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package accesscontrol

import (
	"reflect"

	pbac "chainmaker.org/chainmaker/pb-go/accesscontrol"
	"chainmaker.org/chainmaker/protocol"
)

func init() {
	RegisterACProvider(pbac.MemberType_CERT.String(), NilCertACProvider)
	RegisterACProvider(pbac.MemberType_CERT_HASH.String(), NilCertACProvider)
}

var acProviderRegistry = map[string]reflect.Type{}

type ACProvider interface {
	NewACProvider(chainConf protocol.ChainConf, localOrgId string,
		store protocol.BlockchainStore, log protocol.Logger) (protocol.AccessControlProvider, error)
}

func RegisterACProvider(memberType string, acp ACProvider) {
	_, found := acProviderRegistry[memberType]
	if found {
		panic("accesscontrol provider[" + memberType + "] already registered!")
	}
	acProviderRegistry[memberType] = reflect.TypeOf(acp)
}

func NewACProviderByMemberType(memberType string) ACProvider {
	t, found := acProviderRegistry[memberType]
	if !found {
		panic("accesscontrol provider[" + memberType + "] not found!")
	}
	return reflect.New(t).Elem().Interface().(ACProvider)
}
