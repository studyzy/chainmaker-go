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

	var emptyAuthType = ""
	if chainConf.ChainConfig().AuthType == emptyAuthType {
		chainConf.ChainConfig().AuthType = protocol.PermissionedWithCert
	}

	// authType 和 consensusType 是否匹配
	switch chainConf.ChainConfig().AuthType {
	case protocol.PermissionedWithCert, protocol.Identity:
		if chainConf.ChainConfig().Consensus.Type == consensus.ConsensusType_DPOS {
			return nil,
				fmt.Errorf("new ac provider failed, the consensus type does not match the authentication type")
		}
	case protocol.PermissionedWithKey:
		if chainConf.ChainConfig().Consensus.Type == consensus.ConsensusType_DPOS {
			return nil,
				fmt.Errorf("new ac provider failed, the consensus type does not match the authentication type")
		}
	case protocol.Public:
		if chainConf.ChainConfig().Consensus.Type == consensus.ConsensusType_TBFT ||
			chainConf.ChainConfig().Consensus.Type == consensus.ConsensusType_HOTSTUFF ||
			chainConf.ChainConfig().Consensus.Type == consensus.ConsensusType_RAFT ||
			chainConf.ChainConfig().Consensus.Type == consensus.ConsensusType_MBFT {
			return nil,
				fmt.Errorf("new ac provid" +
					"er failed, the consensus type does not match the authentication type")
		}
	default:
		return nil,
			fmt.Errorf("new ac provider failed, the auth type doesn't exist")
	}

	p := NewACProviderByMemberType(chainConf.ChainConfig().AuthType)
	return p.NewACProvider(chainConf, localOrgId, store, log)
}
