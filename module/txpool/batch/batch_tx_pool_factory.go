/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package batch

import (
	"time"

	"chainmaker.org/chainmaker-go/common/msgbus"
)

type Option func(p *BatchTxPool) error

func (p *BatchTxPool) Apply(opts ...Option) error {
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(p); err != nil {
			return err
		}
	}
	return nil
}

func WithMsgBus(msgBus msgbus.MessageBus) Option {
	return func(p *BatchTxPool) error {
		p.SetMsgBus(msgBus)
		return nil
	}
}

func WithPoolSize(size int) Option {
	return func(p *BatchTxPool) error {
		if size > 0 {
			p.SetPoolSize(size)
		}
		return nil
	}
}

func WithBatchMaxSize(maxSize int) Option {
	return func(p *BatchTxPool) error {
		if maxSize > 0 {
			p.SetBatchMaxSize(maxSize)
		}
		return nil
	}
}

func WithBatchCreateTimeout(timeout time.Duration) Option {
	return func(p *BatchTxPool) error {
		if timeout > 0 {
			p.SetBatchCreateTimeout(timeout)
		}
		return nil
	}
}
