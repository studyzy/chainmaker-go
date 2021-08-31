/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package subscriber

import (
	"chainmaker.org/chainmaker-go/subscriber/model"
	"chainmaker.org/chainmaker/common/v2/msgbus"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	feed "github.com/ethereum/go-ethereum/event"
)

// EventSubscriber - new EventSubscriber struct
type EventSubscriber struct {
	blockFeed         feed.Feed
	contractEventFeed feed.Feed
}

// OnMessage - deal msgbus.BlockInfo message
func (s *EventSubscriber) OnMessage(msg *msgbus.Message) {
	if blockInfo, ok := msg.Payload.(*commonPb.BlockInfo); ok {
		go s.blockFeed.Send(model.NewBlockEvent{BlockInfo: blockInfo})
	}
	if conEventInfoList, ok := msg.Payload.(*commonPb.ContractEventInfoList); ok {
		go s.contractEventFeed.Send(model.NewContractEvent{ContractEventInfoList: conEventInfoList})
	}
}

// OnQuit - deal msgbus OnQuit message
func (s *EventSubscriber) OnQuit() {
	// do nothing
}

// NewSubscriber - new and register msgbus.BlockInfo object
func NewSubscriber(msgBus msgbus.MessageBus) *EventSubscriber {
	subscriber := &EventSubscriber{}
	msgBus.Register(msgbus.BlockInfo, subscriber)

	msgBus.Register(msgbus.ContractEventInfo, subscriber)
	return subscriber
}

// SubscribeBlockEvent - subscribe block event
func (s *EventSubscriber) SubscribeBlockEvent(ch chan<- model.NewBlockEvent) feed.Subscription {
	return s.blockFeed.Subscribe(ch)
}

// SubscribeContractEvent - subscribe contract event
func (s *EventSubscriber) SubscribeContractEvent(ch chan<- model.NewContractEvent) feed.Subscription {
	return s.contractEventFeed.Subscribe(ch)
}
