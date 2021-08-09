/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package accesscontrol

import "chainmaker.org/chainmaker/protocol"

type acFactory struct {
}

var ac_instance *acFactory

func ACFactory() *acFactory {
	once.Do(func() { ac_instance = new(acFactory) })
	return ac_instance
}

func (af *acFactory) NewACProvider(memberType string, chainConf protocol.ChainConf, localOrgId string,
	store protocol.BlockchainStore, log protocol.Logger) (protocol.AccessControlProvider, error) {
	p := NewACProviderByMemberType(memberType)
	return p.NewACProvider(chainConf, localOrgId, store, log)
}
