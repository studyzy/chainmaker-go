/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package accesscontrol

import (
	"reflect"

	"chainmaker.org/chainmaker/protocol/v2"
)

func init() {
	RegisterACProvider(protocol.PermissionedWithCert, NilCertACProvider)
	RegisterACProvider(protocol.Identity, NilCertACProvider)
	RegisterACProvider(protocol.PermissionedWithKey, NilPermissionedPkACProvider)
	RegisterACProvider(protocol.Public, NilPkACProvider)
}

var acProviderRegistry = map[string]reflect.Type{}

type ACProvider interface {
	NewACProvider(chainConf protocol.ChainConf, localOrgId string,
		store protocol.BlockchainStore, log protocol.Logger) (protocol.AccessControlProvider, error)
}

func RegisterACProvider(authType string, acp ACProvider) {
	_, found := acProviderRegistry[authType]
	if found {
		panic("accesscontrol provider[" + authType + "] already registered!")
	}
	acProviderRegistry[authType] = reflect.TypeOf(acp)
}

func NewACProviderByMemberType(authType string) ACProvider {
	t, found := acProviderRegistry[authType]
	if !found {
		panic("accesscontrol provider[" + authType + "] not found!")
	}
	return reflect.New(t).Elem().Interface().(ACProvider)
}
