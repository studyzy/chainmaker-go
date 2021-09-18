/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package accesscontrol

import (
	"fmt"
	"strings"
	"sync"

	"chainmaker.org/chainmaker/pb-go/v2/consensus"
	"chainmaker.org/chainmaker/protocol/v2"
)

type AcFactory struct {
}

var once sync.Once
var acInstance *AcFactory

func ACFactory() *AcFactory {
	once.Do(func() { acInstance = new(AcFactory) })
	return acInstance
}

func (af *AcFactory) NewACProvider(chainConf protocol.ChainConf, localOrgId string,
	store protocol.BlockchainStore, log protocol.Logger) (protocol.AccessControlProvider, error) {

	chainConf.ChainConfig().AuthType = strings.ToLower(chainConf.ChainConfig().AuthType)

	// 兼容1.x ChainConfig authType
	if chainConf.ChainConfig().AuthType == Identity {
		chainConf.ChainConfig().AuthType = AuthTypeToStringMap[PermissionedWithCert]
	}

	authType, ok := StringToAuthTypeMap[chainConf.ChainConfig().AuthType]
	if !ok {
		return nil, fmt.Errorf("new ac provider failed, invalid auth type in chain config")
	}

	// authType 和 consensusType 是否匹配
	switch authType {
	case PermissionedWithCert:
		if chainConf.ChainConfig().Consensus.Type == consensus.ConsensusType_DPOS {
			return nil,
				fmt.Errorf("new ac provider failed, the consensus type does not match the authentication type")
		}
	case PermissionedWithKey:
		if chainConf.ChainConfig().Consensus.Type == consensus.ConsensusType_DPOS {
			return nil,
				fmt.Errorf("new ac provider failed, the consensus type does not match the authentication type")
		}
	case Public:
		if chainConf.ChainConfig().Consensus.Type == consensus.ConsensusType_TBFT ||
			chainConf.ChainConfig().Consensus.Type == consensus.ConsensusType_HOTSTUFF ||
			chainConf.ChainConfig().Consensus.Type == consensus.ConsensusType_RAFT ||
			chainConf.ChainConfig().Consensus.Type == consensus.ConsensusType_MBFT {
			return nil,
				fmt.Errorf("new ac provider failed, the consensus type does not match the authentication type")
		}
	}

	p := NewACProviderByMemberType(authType)
	return p.NewACProvider(chainConf, localOrgId, store, log)
}
