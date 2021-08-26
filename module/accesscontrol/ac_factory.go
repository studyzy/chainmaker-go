/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package accesscontrol

import "chainmaker.org/chainmaker/protocol"

type AcFactory struct {
}

var acInstance *AcFactory

func ACFactory() *AcFactory {
	once.Do(func() { acInstance = new(AcFactory) })
	return acInstance
}

func (af *AcFactory) NewACProvider(memberType string, chainConf protocol.ChainConf, localOrgId string,
	store protocol.BlockchainStore, log protocol.Logger) (protocol.AccessControlProvider, error) {
	p := NewACProviderByMemberType(memberType)
	return p.NewACProvider(chainConf, localOrgId, store, log)
}
