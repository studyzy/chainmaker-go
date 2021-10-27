/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package net

import (
	"errors"
	"strings"
	"sync"

	"chainmaker.org/chainmaker/common/v2/msgbus"
	rootLog "chainmaker.org/chainmaker/logger/v2"
	"chainmaker.org/chainmaker/net-common/common/priorityblocker"
	configPb "chainmaker.org/chainmaker/pb-go/v2/config"
	netPb "chainmaker.org/chainmaker/pb-go/v2/net"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"
	"chainmaker.org/chainmaker/protocol/v2"
	"github.com/gogo/protobuf/proto"
)

var (
	ErrorChainMsgBusBeenBound = errors.New("chain msg bus has been bound")
	ErrorNetNotRunning        = errors.New("net instance is not running")
)

const (
	topicNamePrefix            = "topic"
	consensusTopicNamePrefix   = "consensus_topic"
	msgBusTopicPrefix          = "msgbus_topic"
	msgBusConsensusTopicPrefix = "msgbus_consensus_topic"
	msgBusMsgFlagPrefix        = "msgbus"

	topicSeparator = "::"

	moduleSync = "NetService"
)

// CreateFlagWithPrefixAndMsgType will join prefix with msg type string.
func CreateFlagWithPrefixAndMsgType(prefix string, msgType netPb.NetMsg_MsgType) string {
	var builder strings.Builder
	builder.WriteString(prefix)
	builder.WriteString(topicSeparator)
	builder.WriteString(msgType.String())
	return builder.String()
}

var _ protocol.NetService = (*NetService)(nil)

// NetService provide a net service for modules.
type NetService struct {
	chainId              string
	localNet             protocol.Net
	msgBus               msgbus.MessageBus
	logger               *rootLog.CMLogger
	configWatcher        *ConfigWatcher
	consensusNodeIds     map[string]struct{}
	consensusNodeIdsLock sync.RWMutex

	ac            protocol.AccessControlProvider
	revokeNodeIds sync.Map // nolint: structcheck,unused // node id of node cert revoked , map[string]struct{}
	vmWatcher     *VmWatcher
}

// NewNetService create a new net service instance.
func NewNetService(chainId string, localNet protocol.Net, ac protocol.AccessControlProvider) *NetService {
	logger := rootLog.GetLoggerByChain(rootLog.MODULE_NET, chainId)
	ns := &NetService{
		chainId:          chainId,
		localNet:         localNet,
		consensusNodeIds: make(map[string]struct{}),
		ac:               ac,
		logger:           logger,
	}
	return ns
}

// BroadcastMsg broadcast a net msg to other nodes belongs to the same chain.
func (ns *NetService) BroadcastMsg(msg []byte, msgType netPb.NetMsg_MsgType) error {
	err := ns.broadcastMsg(msg, CreateFlagWithPrefixAndMsgType(topicNamePrefix, msgType))
	if err != nil {
		return err
	}
	return nil
}

func (ns *NetService) broadcastMsg(msg []byte, topic string) error {
	return ns.localNet.BroadcastWithChainId(ns.chainId, topic, msg)
}

// Subscribe a pub-sub topic for receiving the msg that be broadcast by the other node.
func (ns *NetService) Subscribe(msgType netPb.NetMsg_MsgType, handler protocol.MsgHandler) error {
	err := ns.subscribe(handler, msgType, CreateFlagWithPrefixAndMsgType(topicNamePrefix, msgType))
	if err != nil {
		return err
	}
	return nil
}

func (ns *NetService) subscribe(handler protocol.MsgHandler, msgType netPb.NetMsg_MsgType, topic string) error {
	h := func(publisher string, msg []byte) error {
		return handler(publisher, msg, msgType)
	}
	return ns.localNet.SubscribeWithChainId(ns.chainId, topic, h)
}

// CancelSubscribe stop receiving the msg from the pub-sub topic subscribed.
func (ns *NetService) CancelSubscribe(msgType netPb.NetMsg_MsgType) error {
	return ns.localNet.CancelSubscribeWithChainId(ns.chainId, CreateFlagWithPrefixAndMsgType(topicNamePrefix, msgType))
}

func (ns *NetService) getConsensusNodeIdList() []string {
	result := make([]string, 0)
	ns.consensusNodeIdsLock.RLock()
	defer ns.consensusNodeIdsLock.RUnlock()
	if ns.consensusNodeIds != nil {
		for node := range ns.consensusNodeIds {
			result = append(result, node)
		}
	}
	return result
}

func (ns *NetService) isConsensusNodeIdListEmpty() bool {
	ns.consensusNodeIdsLock.RLock()
	defer ns.consensusNodeIdsLock.RUnlock()
	return len(ns.consensusNodeIds) == 0
}

func (ns *NetService) consensusBroadcastMsg(msg []byte, topic string) error {
	consensusNodeIdList := ns.getConsensusNodeIdList()
	if len(consensusNodeIdList) == 0 {
		return nil
	}
	var wg sync.WaitGroup
	wg.Add(len(consensusNodeIdList))
	for i := range consensusNodeIdList {
		to := consensusNodeIdList[i]
		if to == ns.localNet.GetNodeUid() {
			wg.Done()
			continue
		}
		go func() {
			defer wg.Done()
			if err := ns.localNet.SendMsg(ns.chainId, to, topic, msg); err != nil {
				ns.logger.Warnf("[NetService] send consensus broadcast msg failed, %s", err.Error())
			}
		}()
	}
	wg.Wait()
	return nil
}

// ConsensusBroadcastMsg only broadcast a net msg to other consensus nodes belongs to the same chain.
func (ns *NetService) ConsensusBroadcastMsg(msg []byte, msgType netPb.NetMsg_MsgType) error {
	pbMsg := NewNetMsg(msg, msgType, "")
	return ns.consensusBroadcastMsg(msg, CreateFlagWithPrefixAndMsgType(consensusTopicNamePrefix, pbMsg.Type))
}

// ConsensusSubscribe create a listener for receiving the msg
// which type is the given netPb.NetMsg_MsgType
// that be broadcast with ConsensusBroadcastMsg method by the other consensus node.
func (ns *NetService) ConsensusSubscribe(msgType netPb.NetMsg_MsgType, handler protocol.MsgHandler) error {
	err := ns.receiveMsg(handler, CreateFlagWithPrefixAndMsgType(consensusTopicNamePrefix, msgType), msgType)
	if err != nil {
		return err
	}
	return nil
}

// CancelConsensusSubscribe stop receiving the msg
// which type is the given netPb.NetMsg_MsgType
// that be broadcast with ConsensusBroadcastMsg method by the other consensus node.
func (ns *NetService) CancelConsensusSubscribe(msgType netPb.NetMsg_MsgType) error {
	return ns.cancelReceiveMsg(CreateFlagWithPrefixAndMsgType(consensusTopicNamePrefix, msgType))
}

// SendMsg send a net msg to the nodes which node ids are the given strings.
func (ns *NetService) SendMsg(msg []byte, msgType netPb.NetMsg_MsgType, to ...string) error {
	msgFlag := msgType.String()
	for _, n := range to {
		if n == ns.localNet.GetNodeUid() {
			continue
		}
		err := ns.localNet.SendMsg(ns.chainId, n, msgFlag, msg)
		if err != nil {
			ns.logger.Debugf("[NetService] send msg failed(to:%s, flag:%s), %s", n, msgFlag, err.Error())
			return err
		}
	}
	return nil
}

// ReceiveMsg create a listener for receiving the msg
// which type is the given netPb.NetMsg_MsgType
// that be sent with ConsensusBroadcastMsg method by the other consensus node.
func (ns *NetService) ReceiveMsg(msgType netPb.NetMsg_MsgType, handler protocol.MsgHandler) error {
	msgFlag := msgType.String()
	return ns.receiveMsg(handler, msgFlag, msgType)
}

func (ns *NetService) receiveMsg(handler protocol.MsgHandler, flag string, msgType netPb.NetMsg_MsgType) error {
	h := func(from string, data []byte) error {
		err := handler(from, data, msgType)
		if err != nil {
			return err
		}
		return nil
	}

	return ns.localNet.DirectMsgHandle(ns.chainId, flag, h)
}

func (ns *NetService) cancelReceiveMsg(flag string) error {

	return ns.localNet.CancelDirectMsgHandle(ns.chainId, flag)
}

// MsgForMsgBusHandler is a handler function that receive the msg from net than publish to msg-bus.
type MsgForMsgBusHandler func(chainId string, from string, msg []byte) error

func (ns *NetService) receiveMsgForMsgBus(handler MsgForMsgBusHandler, flag string) error {
	h := func(from string, data []byte) error {
		err := handler(ns.chainId, from, data)
		if err != nil {
			return err
		}
		return nil
	}

	return ns.localNet.DirectMsgHandle(ns.chainId, flag, h)
}

func (ns *NetService) subscribeTopicForMsgBus(handler MsgForMsgBusHandler, topic string) error {
	h := func(from string, data []byte) error {
		err := handler(ns.chainId, from, data)
		if err != nil {
			return err
		}
		return nil
	}

	return ns.localNet.SubscribeWithChainId(ns.chainId, topic, h)
}

// GetNodeUidByCertId return the id of the node connected to us which mapped to tls cert id given.
// node id and tls cert id relation will be mapped after connection created success.
func (ns *NetService) GetNodeUidByCertId(certId string) (string, error) {
	return ns.localNet.GetNodeUidByCertId(certId)
}

// GetChainNodesInfo return the base info of the nodes connected.
func (ns *NetService) GetChainNodesInfo() ([]*protocol.ChainNodeInfo, error) {
	return ns.localNet.ChainNodesInfo(ns.chainId)
}

// GetChainNodesInfoProvider return a protocol.ChainNodesInfoProvider.
func (ns *NetService) GetChainNodesInfoProvider() protocol.ChainNodesInfoProvider {
	return ns
}

// Start the net-service.
func (ns *NetService) Start() error {
	if !ns.localNet.IsRunning() {
		ns.logger.Errorf("[NetService] start the net first pls.")
		return ErrorNetNotRunning
	}
	// add access control
	ns.localNet.AddAC(ns.chainId, ns.ac)

	// init pub-sub
	if err := ns.localNet.InitPubSub(ns.chainId, 0); err != nil {
		ns.logger.Errorf("[NetService] init pubsub failed, %s", err.Error())
		return err
	}

	// re verify peers
	ns.localNet.ReVerifyPeers(ns.chainId)

	if err := ns.initBindMsgBus(); err != nil {
		return err
	}

	ns.setFlagPriority()

	ns.logger.Infof("[NetService] net service started.")
	return nil
}

// Stop the net-service.
func (ns *NetService) Stop() error {
	return nil
}

// ConfigWatcher return a implementation of protocol.Watcher. It is used for refreshing the config.
func (ns *NetService) ConfigWatcher() protocol.Watcher {
	if ns.configWatcher == nil {
		ns.configWatcher = &ConfigWatcher{ns: ns}
	}
	return ns.configWatcher
}

// ConfigWatcher is a implementation of protocol.Watcher.
type ConfigWatcher struct {
	ns *NetService
}

// Module
func (cw *ConfigWatcher) Module() string {
	return moduleSync
}

// Watch
func (cw *ConfigWatcher) Watch(chainConfig *configPb.ChainConfig) error {
	// refresh chainConfig
	cw.ns.logger.Infof("[NetService] refreshing chain config...")
	// 1.refresh consensus nodeIds
	// 1.1 get all new nodeIds
	newConsensusNodeIds := make(map[string]struct{})
	for _, node := range chainConfig.Consensus.Nodes {
		for _, nodeId := range node.NodeId {
			newConsensusNodeIds[nodeId] = struct{}{}
		}
	}
	// 1.2 refresh consensus nodeIds
	cw.ns.consensusNodeIdsLock.Lock()
	cw.ns.consensusNodeIds = newConsensusNodeIds
	cw.ns.consensusNodeIdsLock.Unlock()
	cw.ns.logger.Infof("[NetService] refresh ids of consensus nodes ok ")
	// 2.re-verify peers
	cw.ns.localNet.ReVerifyPeers(cw.ns.chainId)
	cw.ns.logger.Infof("[NetService] re-verify peers ok")
	cw.ns.logger.Infof("[NetService] refresh chain config ok")
	return nil
}

// VmWatcher return an implementation of protocol.VmWatcher.
// It is used for refreshing revoked peer which use revoked tls cert.
func (ns *NetService) VmWatcher() protocol.VmWatcher {
	if ns.vmWatcher == nil {
		ns.vmWatcher = &VmWatcher{ns: ns}
	}
	return ns.vmWatcher
}

type VmWatcher struct {
	ns *NetService
}

func (v *VmWatcher) Module() string {
	return moduleSync
}

func (v *VmWatcher) ContractNames() []string {
	return []string{syscontract.SystemContract_CERT_MANAGE.String(),
		syscontract.SystemContract_PUBKEY_MANAGE.String()}
}

func (v *VmWatcher) Callback(contractName string, _ []byte) error {
	switch contractName {
	case syscontract.SystemContract_CERT_MANAGE.String():
		v.ns.logger.Infof("[module: %s] call back, [contractName: %s]", v.Module(), contractName)
		v.ns.localNet.ReVerifyPeers(v.ns.chainId)
		return nil
	case syscontract.SystemContract_PUBKEY_MANAGE.String():
		v.ns.logger.Infof("[module: %s] call back, [contractName: %s]", v.Module(), contractName)
		v.ns.localNet.ReVerifyPeers(v.ns.chainId)
		return nil
	default:
		return nil
	}
}

// HandleMsgBusSubscriberOnMessage is a handler used for msg-bus subscriber OnMessage method.
func HandleMsgBusSubscriberOnMessage(
	netService *NetService,
	msgType netPb.NetMsg_MsgType,
	logMsgDescription string,
	message *msgbus.Message) error {
	if netMsg, ok := message.Payload.(*netPb.NetMsg); ok {
		if netMsg.Type.String() != msgType.String() {
			netService.logger.Errorf(
				"[NetService/msg-bus %s subscriber] wrong net msg type(expect %s, got %s)",
				logMsgDescription,
				msgType.String(),
				netMsg.Type.String(),
			)
			return errors.New("wrong net msg type")
		}
		if netMsg.To == "" {
			return handleMsgBusSubscriberOnMessageBroadcast(netService, msgType, logMsgDescription, netMsg)
		}

		return handleMsgBusSubscriberOnMessageSend(netService, msgType, logMsgDescription, netMsg)
	}
	return nil
}

func handleMsgBusSubscriberOnMessageBroadcast(
	netService *NetService,
	msgType netPb.NetMsg_MsgType,
	logMsgDescription string,
	netMsg *netPb.NetMsg) error {
	if (msgType == netPb.NetMsg_TX || msgType == netPb.NetMsg_CONSENSUS_MSG) &&
		!netService.isConsensusNodeIdListEmpty() {
		if err := netService.consensusBroadcastMsg(
			netMsg.GetPayload(),
			CreateFlagWithPrefixAndMsgType(msgBusConsensusTopicPrefix, msgType),
		); err != nil {
			netService.logger.Debugf(
				"[NetService/msg-bus %s subscriber] broadcast failed, %s",
				logMsgDescription,
				err.Error(),
			)
			return err
		}
	} else {
		if err := netService.broadcastMsg(
			netMsg.GetPayload(),
			CreateFlagWithPrefixAndMsgType(msgBusTopicPrefix, msgType),
		); err != nil {
			netService.logger.Debugf(
				"[NetService/msg-bus %s subscriber] broadcast failed, %s",
				logMsgDescription,
				err.Error(),
			)
			return err
		}
	}
	netService.logger.Debugf("[NetService/msg-bus %s subscriber] broadcast ok", logMsgDescription)
	return nil
}

func handleMsgBusSubscriberOnMessageSend(
	netService *NetService,
	msgType netPb.NetMsg_MsgType,
	logMsgDescription string,
	netMsg *netPb.NetMsg) error {
	go func() {
		if err := netService.localNet.SendMsg(
			netService.chainId, netMsg.To, CreateFlagWithPrefixAndMsgType(
				msgBusMsgFlagPrefix,
				msgType,
			),
			netMsg.GetPayload(),
		); err != nil {
			netService.logger.Debugf(
				"[NetService/msg-bus %s subscriber] send msg failed (size:%d) (reason:%s) (to:%s)",
				logMsgDescription,
				proto.Size(netMsg),
				err.Error(),
				netMsg.To,
			)
		} else {
			netService.logger.Debugf(
				"[NetService/msg-bus %s subscriber] send msg ok (size:%d) (to:%s)",
				logMsgDescription,
				proto.Size(netMsg),
				netMsg.To,
			)
		}
	}()
	return nil
}

// ConsensusMsgSubscriber is a subscriber implementation subscribe consensus msg for msgbus.
type ConsensusMsgSubscriber struct {
	netService *NetService
}

func (cms *ConsensusMsgSubscriber) OnMessage(message *msgbus.Message) {
	switch message.Topic {
	case msgbus.SendConsensusMsg:
		go func() {
			err := HandleMsgBusSubscriberOnMessage(
				cms.netService,
				netPb.NetMsg_CONSENSUS_MSG,
				"consensus msg",
				message,
			)
			if err != nil {
				cms.netService.logger.Warnf(
					"[ConsensusMsgSubscriber] handle message failed, %s",
					err.Error(),
				)
			}
		}()
	default:
	}
}

func (cms *ConsensusMsgSubscriber) OnQuit() {
	// do nothing
	//panic("implement me")
}

// TxPoolMsgSubscriber is a subscriber implementation subscribe tx pool msg for msgbus.
type TxPoolMsgSubscriber struct {
	netService *NetService
}

func (cms *TxPoolMsgSubscriber) OnMessage(message *msgbus.Message) {
	switch message.Topic {
	case msgbus.SendTxPoolMsg:
		go func() {
			err := HandleMsgBusSubscriberOnMessage(
				cms.netService,
				netPb.NetMsg_TX,
				"tx_pool msg",
				message,
			)
			if err != nil {
				cms.netService.logger.Warnf(
					"[TxPoolMsgSubscriber] handle message failed, %s",
					err.Error(),
				)
			}
		}()
	default:
	}
}

func (cms *TxPoolMsgSubscriber) OnQuit() {
	// do nothing
	//panic("implement me")
}

// SyncBlockMsgSubscriber is a subscriber implementation subscribe sync block msg for msgbus.
type SyncBlockMsgSubscriber struct {
	netService *NetService
}

func (cms *SyncBlockMsgSubscriber) OnMessage(message *msgbus.Message) {
	switch message.Topic {
	case msgbus.SendSyncBlockMsg:
		go func() {
			err := HandleMsgBusSubscriberOnMessage(
				cms.netService,
				netPb.NetMsg_SYNC_BLOCK_MSG,
				"sync block msg",
				message,
			)
			if err != nil {
				cms.netService.logger.Warnf(
					"[SyncBlockMsgSubscriber] handle message failed, %s",
					err.Error(),
				)
			}
		}()
	default:
	}
}

func (cms *SyncBlockMsgSubscriber) OnQuit() {
	// do nothing
	//panic("implement me")
}

func CreateMsgHandlerForMsgBus(
	netService *NetService,
	topic msgbus.Topic,
	logMsgDescription string,
	msgType netPb.NetMsg_MsgType) func(chainId string, node string, data []byte) error {
	return func(chainId string, node string, data []byte) error {
		netService.logger.Debugf("[NetService/%s handler for msg-bus] receive msg (size:%d) (from:%s)",
			logMsgDescription,
			len(data),
			node,
		)
		if netService.chainId != chainId {
			netService.logger.Warnf("[NetService/%s handler for msg-bus] wrong chain-id(chain-id:%s), ignored.",
				logMsgDescription,
				chainId,
			)
			return nil
		}
		pbMsg := NewNetMsg(data, msgType, node)
		netService.msgBus.Publish(topic, pbMsg)
		return nil
	}
}

// bindMsgBus bind a msgbus.MessageBus.
func (ns *NetService) bindMsgBus(bus msgbus.MessageBus) error {
	if ns.msgBus != nil {
		return ErrorChainMsgBusBeenBound
	}
	ns.msgBus = bus
	return nil
}

func (ns *NetService) initBindMsgBus() error {
	if ns.msgBus == nil {
		ns.logger.Warnf("[NetService] msg-bus not bound.")
		return nil
	}

	// bind msg bus
	// ===========================================

	// for consensus module
	// receive consensus msg from net then publish to msg-bus
	consensusMsgHandler := CreateMsgHandlerForMsgBus(
		ns,
		msgbus.RecvConsensusMsg,
		"consensus msg",
		netPb.NetMsg_CONSENSUS_MSG,
	)
	if err := ns.receiveMsgForMsgBus(
		consensusMsgHandler,
		CreateFlagWithPrefixAndMsgType(
			msgBusMsgFlagPrefix,
			netPb.NetMsg_CONSENSUS_MSG,
		),
	); err != nil {
		return err
	}
	if err := ns.receiveMsgForMsgBus(
		consensusMsgHandler,
		CreateFlagWithPrefixAndMsgType(
			msgBusConsensusTopicPrefix,
			netPb.NetMsg_CONSENSUS_MSG,
		),
	); err != nil {
		return err
	}

	// subscribe a consensus msg subscriber for receiving consensus msg
	// from msg-bus then broadcast the msg to consensus nodes.
	cmSubscriber := &ConsensusMsgSubscriber{
		netService: ns,
	}
	ns.msgBus.Register(msgbus.SendConsensusMsg, cmSubscriber)

	// ===========================================

	// for tx pool module
	// receive tx msg from net then publish to msg-bus
	txPoolMsgHandler := CreateMsgHandlerForMsgBus(
		ns,
		msgbus.RecvTxPoolMsg,
		"tx_pool msg",
		netPb.NetMsg_TX,
	)
	if err := ns.receiveMsgForMsgBus(
		txPoolMsgHandler,
		CreateFlagWithPrefixAndMsgType(
			msgBusMsgFlagPrefix,
			netPb.NetMsg_TX,
		),
	); err != nil {
		return err
	}
	if err := ns.receiveMsgForMsgBus(
		txPoolMsgHandler,
		CreateFlagWithPrefixAndMsgType(
			msgBusConsensusTopicPrefix,
			netPb.NetMsg_TX,
		),
	); err != nil {
		return err
	}
	// subscribe the topic that ths spv node broadcast to
	if err := ns.subscribeTopicForMsgBus(
		txPoolMsgHandler,
		CreateFlagWithPrefixAndMsgType(
			topicNamePrefix,
			netPb.NetMsg_TX,
		),
	); err != nil {
		return err
	}

	// subscribe a tx pool msg subscriber for receiving tx msg from msg-bus then broadcast the msg to consensus nodes.
	txPoolSubscriber := &TxPoolMsgSubscriber{
		netService: ns,
	}
	ns.msgBus.Register(msgbus.SendTxPoolMsg, txPoolSubscriber)

	// ===========================================

	// for sync module
	// receive sync block msg from net then publish to msg-bus
	sbmHandler := CreateMsgHandlerForMsgBus(
		ns,
		msgbus.RecvSyncBlockMsg,
		"sync block msg",
		netPb.NetMsg_SYNC_BLOCK_MSG,
	)
	if err := ns.receiveMsgForMsgBus(
		sbmHandler,
		CreateFlagWithPrefixAndMsgType(
			msgBusMsgFlagPrefix,
			netPb.NetMsg_SYNC_BLOCK_MSG,
		),
	); err != nil {
		return err
	}
	if err := ns.subscribeTopicForMsgBus(
		sbmHandler,
		CreateFlagWithPrefixAndMsgType(
			msgBusTopicPrefix,
			netPb.NetMsg_SYNC_BLOCK_MSG,
		),
	); err != nil {
		return err
	}
	// subscribe a sync block msg subscriber for receiving
	// sync block msg from msg-bus then broadcast the msg to consensus nodes.
	sbmSubscriber := &SyncBlockMsgSubscriber{
		netService: ns,
	}
	ns.msgBus.Register(msgbus.SendSyncBlockMsg, sbmSubscriber)
	ns.logger.Infof("[NetService] init bind msg-bus ok")
	return nil
}

func (ns *NetService) setFlagPriority() {
	ns.localNet.SetMsgPriority(netPb.NetMsg_CONSENSUS_MSG.String(), uint8(priorityblocker.PriorityLevel9))
	ns.localNet.SetMsgPriority(netPb.NetMsg_BLOCK.String(), uint8(priorityblocker.PriorityLevel8))
	ns.localNet.SetMsgPriority(netPb.NetMsg_BLOCKS.String(), uint8(priorityblocker.PriorityLevel8))
	ns.localNet.SetMsgPriority(netPb.NetMsg_TX.String(), uint8(priorityblocker.PriorityLevel7))
	ns.localNet.SetMsgPriority(netPb.NetMsg_TXS.String(), uint8(priorityblocker.PriorityLevel7))
	ns.localNet.SetMsgPriority(netPb.NetMsg_SYNC_BLOCK_MSG.String(), uint8(priorityblocker.PriorityLevel5))

	ns.localNet.SetMsgPriority(
		CreateFlagWithPrefixAndMsgType(msgBusMsgFlagPrefix, netPb.NetMsg_CONSENSUS_MSG),
		uint8(priorityblocker.PriorityLevel9),
	)
	ns.localNet.SetMsgPriority(
		CreateFlagWithPrefixAndMsgType(msgBusConsensusTopicPrefix, netPb.NetMsg_CONSENSUS_MSG),
		uint8(priorityblocker.PriorityLevel9),
	)
	ns.localNet.SetMsgPriority(
		CreateFlagWithPrefixAndMsgType(msgBusMsgFlagPrefix, netPb.NetMsg_TX),
		uint8(priorityblocker.PriorityLevel7),
	)
	ns.localNet.SetMsgPriority(
		CreateFlagWithPrefixAndMsgType(msgBusConsensusTopicPrefix, netPb.NetMsg_TX),
		uint8(priorityblocker.PriorityLevel7),
	)
	ns.localNet.SetMsgPriority(
		CreateFlagWithPrefixAndMsgType(topicNamePrefix, netPb.NetMsg_TX),
		uint8(priorityblocker.PriorityLevel7),
	)
	ns.localNet.SetMsgPriority(
		CreateFlagWithPrefixAndMsgType(msgBusMsgFlagPrefix, netPb.NetMsg_SYNC_BLOCK_MSG),
		uint8(priorityblocker.PriorityLevel5),
	)
	ns.localNet.SetMsgPriority(
		CreateFlagWithPrefixAndMsgType(msgBusTopicPrefix, netPb.NetMsg_SYNC_BLOCK_MSG),
		uint8(priorityblocker.PriorityLevel5),
	)
}
