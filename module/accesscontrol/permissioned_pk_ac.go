/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package accesscontrol

import (
	"chainmaker.org/chainmaker-go/localconf"
	"chainmaker.org/chainmaker/common/v2/concurrentlru"
	bcx509 "chainmaker.org/chainmaker/common/v2/crypto/x509"
	"chainmaker.org/chainmaker/pb-go/v2/config"
	"chainmaker.org/chainmaker/protocol/v2"
	"crypto/x509"
	"sync"
)

var _ protocol.AccessControlProvider = (*permissionedPkACProvider)(nil)

var NilPermissionedPkACProvider ACProvider = (*permissionedPkACProvider)(nil)

type permissionedPkACProvider struct {
	//chainconfig authType
	authType AuthType

	acService *accessControlService

	hashType string

	log protocol.Logger

	localOrg string

	adminMember *sync.Map

	consensusMember *sync.Map
}
