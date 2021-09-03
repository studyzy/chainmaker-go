/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package chainconf

import (
	"chainmaker.org/chainmaker/common/v2/msgbus"
	"chainmaker.org/chainmaker/protocol/v2"
)

type options struct {
	chainId         string
	msgBus          msgbus.MessageBus
	blockchainStore protocol.BlockchainStore
}

// Option is a option for chain config.
type Option func(f *options) error

// Apply all options.
func (f *options) Apply(opts ...Option) error {
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(f); err != nil {
			return err
		}
	}
	return nil
}

// WithMsgBus bind a msg-bus.
func WithMsgBus(msgBus msgbus.MessageBus) Option {
	return func(f *options) error {
		f.msgBus = msgBus
		return nil
	}
}

// WithChainId set the chain id.
func WithChainId(chainId string) Option {
	return func(f *options) error {
		f.chainId = chainId
		return nil
	}
}

// WithBlockchainStore set the block chain store.
func WithBlockchainStore(blockchainStore protocol.BlockchainStore) Option {
	return func(f *options) error {
		f.blockchainStore = blockchainStore
		return nil
	}
}
