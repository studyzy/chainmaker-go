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

// chain authentication mode
type AuthType uint32

const (
	// permissioned with certificate
	PermissionedWithCert AuthType = iota + 1

	// permissioned with public key
	PermissionedWithKey

	// public key
	Public
)

var AuthTypeToStringMap = map[AuthType]string{
	PermissionedWithCert: "permissionedWithCert",
	PermissionedWithKey:  "permissionedWithKey",
	Public:               "public",
}

var StringToAuthTypeMap = map[string]AuthType{
	"permissionedWithCert": PermissionedWithCert,
	"permissionedWithKey":  PermissionedWithKey,
	"public":               Public,
}

func init() {
	RegisterACProvider(PermissionedWithCert, NilCertACProvider)
	RegisterACProvider(PermissionedWithKey, NilPermissionedPkACProvider)
	RegisterACProvider(Public, NilPkACProvider)
}

var acProviderRegistry = map[AuthType]reflect.Type{}

type ACProvider interface {
	NewACProvider(chainConf protocol.ChainConf, localOrgId string,
		store protocol.BlockchainStore, log protocol.Logger) (protocol.AccessControlProvider, error)
}

func RegisterACProvider(authType AuthType, acp ACProvider) {
	_, found := acProviderRegistry[authType]
	if found {
		panic("accesscontrol provider[" + AuthTypeToStringMap[authType] + "] already registered!")
	}
	acProviderRegistry[authType] = reflect.TypeOf(acp)
}

func NewACProviderByMemberType(authType AuthType) ACProvider {
	t, found := acProviderRegistry[authType]
	if !found {
		panic("accesscontrol provider[" + AuthTypeToStringMap[authType] + "] not found!")
	}
	return reflect.New(t).Elem().Interface().(ACProvider)
}
