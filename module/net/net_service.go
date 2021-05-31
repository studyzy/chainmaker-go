/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package net

import (
	"errors"
	"fmt"
	"io/ioutil"
	"sync"

	commonPb "chainmaker.org/chainmaker/pb-go/common"
	configPb "chainmaker.org/chainmaker/pb-go/config"
	netPb "chainmaker.org/chainmaker/pb-go/net"

	"chainmaker.org/chainmaker/common/msgbus"
	"chainmaker.org/chainmaker-go/localconf"
	rootLog "chainmaker.org/chainmaker-go/logger"
	"chainmaker.org/chainmaker-go/protocol"
	"github.com/gogo/protobuf/proto"
)

// ErrorChainMsgBusBeenBound
var ErrorChainMsgBusBeenBound = errors.New("chain msg bus has been bound")

// ErrorChainMsgBusNotBeenBound
var ErrorChainMsgBusNotBeenBound = errors.New("chain msg bus has not been bound")

// ErrorNetNotRunning
var ErrorNetNotRunning = errors.New("net instance is not running")

const (
	topicNameTemplate            = "topic_%s"
	consensusTopicNameTemplate   = "consensus_topic_%s"
	msgBusTopicTemplate          = "msgbus_topic_%s"
	msgBusConsensusTopicTemplate = "msgbus_consensus_topic_%s"
	msgBusMsgFlagTemplate        = "msgbus_%s"
)

var _ protocol.NetService = (*NetService)(nil)

// NetService provide a net service for modules.
type NetService struct {
	chainId              string
	localNet             Net
	msgBus               msgbus.MessageBus
	logger               *rootLog.CMLogger
	configWatcher        *ConfigWatcher
	consensusNodeIds     map[string]struct{}
	consensusNodeIdsLock sync.Mutex

	ac            protocol.AccessControlProvider
	revokeNodeIds sync.Map // node id of node cert revoked , map[string]struct{}
	vmWatcher     *VmWatcher
}

// NewNetService create a new net service instance.
func NewNetService(chainId string, localNet Net, ac protocol.AccessControlProvider) *NetService {
	logger := rootLog.GetLoggerByChain(rootLog.MODULE_NET, chainId)
	ns := &NetService{chainId: chainId, localNet: localNet, consensusNodeIds: make(map[string]struct{}), ac: ac, logger: logger}
	return ns
}

// BroadcastMsg broadcast a net msg to other nodes belongs to the same chain.
func (ns *NetService) BroadcastMsg(msg []byte, msgType netPb.NetMsg_MsgType) error {
	pbMsg := NewNetMsg(msg, msgType, "")
	err := ns.broadcastMsg(pbMsg, fmt.Sprintf(topicNameTemplate, pbMsg.Type.String()))
	if err != nil {
		return err
	}
	return nil
}

func (ns *NetService) broadcastMsg(pbMsg *netPb.NetMsg, topic string) error {
	return ns.localNet.BroadcastWithChainId(ns.chainId, topic, pbMsg)
}

// Subscribe a pub-sub topic for receiving the msg that be broadcast by the other node.
func (ns *NetService) Subscribe(msgType netPb.NetMsg_MsgType, handler protocol.MsgHandler) error {
	err := ns.subscribe(handler, fmt.Sprintf(topicNameTemplate, msgType.String()))
	if err != nil {
		return err
	}
	return nil
}

func (ns *NetService) subscribe(handler protocol.MsgHandler, topic string) error {
	h := func(publisher string, msg *netPb.NetMsg) error {
		return handler(publisher, msg.GetPayload(), msg.GetType())
	}
	return ns.localNet.SubscribeWithChainId(ns.chainId, topic, h)
}

// CancelSubscribe stop receiving the msg from the pub-sub topic subscribed.
func (ns *NetService) CancelSubscribe(msgType netPb.NetMsg_MsgType) error {
	return ns.localNet.CancelSubscribeWithChainId(ns.chainId, fmt.Sprintf(topicNameTemplate, msgType.String()))
}

func (ns *NetService) getConsensusNodeIdList() []string {
	result := make([]string, 0)
	ns.consensusNodeIdsLock.Lock()
	defer ns.consensusNodeIdsLock.Unlock()
	if ns.consensusNodeIds != nil {
		for node := range ns.consensusNodeIds {
			result = append(result, node)
		}
	}
	return result
}

func (ns *NetService) consensusBroadcastMsg(pbMsg *netPb.NetMsg, topic string) error {
	for _, to := range ns.getConsensusNodeIdList() {
		if to == ns.localNet.GetNodeUid() {
			continue
		}
		msg := NewNetMsg(pbMsg.Payload, pbMsg.Type, to)
		go func() {
			if err := ns.sendMsg(msg, topic); err != nil {
				ns.logger.Warnf("[NetService] send consensus broadcast msg failed, %s", err.Error())
			}
		}()
	}
	return nil
}

// ConsensusBroadcastMsg only broadcast a net msg to other consensus nodes belongs to the same chain.
func (ns *NetService) ConsensusBroadcastMsg(msg []byte, msgType netPb.NetMsg_MsgType) error {
	pbMsg := NewNetMsg(msg, msgType, "")
	return ns.consensusBroadcastMsg(pbMsg, fmt.Sprintf(consensusTopicNameTemplate, pbMsg.Type.String()))
}

// ConsensusSubscribe create a listener for receiving the msg which type is the given netPb.NetMsg_MsgType that be broadcast with ConsensusBroadcastMsg method by the other consensus node.
func (ns *NetService) ConsensusSubscribe(msgType netPb.NetMsg_MsgType, handler protocol.MsgHandler) error {
	err := ns.receiveMsg(handler, fmt.Sprintf(consensusTopicNameTemplate, msgType.String()))
	if err != nil {
		return err
	}
	return nil
}

// CancelConsensusSubscribe stop receiving the msg which type is the given netPb.NetMsg_MsgType that be broadcast with ConsensusBroadcastMsg method by the other consensus node.
func (ns *NetService) CancelConsensusSubscribe(msgType netPb.NetMsg_MsgType) error {
	return ns.cancelReceiveMsg(fmt.Sprintf(consensusTopicNameTemplate, msgType.String()))
}

// SendMsg send a net msg to the nodes which node ids are the given strings.
func (ns *NetService) SendMsg(msg []byte, msgType netPb.NetMsg_MsgType, to ...string) error {
	msgFlag := msgType.String()
	for _, n := range to {
		if n == ns.localNet.GetNodeUid() {
			continue
		}
		pbMsg := NewNetMsg(msg, msgType, n)
		err := ns.sendMsg(pbMsg, msgFlag)
		if err != nil {
			return err
		}
	}
	return nil
}

func (ns *NetService) sendMsg(pbMsg *netPb.NetMsg, flag string) error {
	err := ns.localNet.SendMsg(ns.chainId, pbMsg.To, flag, pbMsg)
	if err != nil {
		ns.logger.Debugf("[NetService] send msg failed(to:%s, flag:%s), %s", pbMsg.To, flag, err.Error())
		return err
	}
	return nil
}

// ReceiveMsg create a listener for receiving the msg which type is the given netPb.NetMsg_MsgType that be sended with ConsensusBroadcastMsg method by the other consensus node.
func (ns *NetService) ReceiveMsg(msgType netPb.NetMsg_MsgType, handler protocol.MsgHandler) error {
	msgFlag := msgType.String()
	if err := ns.receiveMsg(handler, msgFlag); err != nil {
		return err
	}
	return nil
}

func (ns *NetService) receiveMsg(handler protocol.MsgHandler, flag string) error {
	h := func(from string, netMsg *netPb.NetMsg) error {
		err := handler(from, netMsg.GetPayload(), netMsg.GetType())
		if err != nil {
			return err
		}
		return nil
	}
	if err := ns.localNet.DirectMsgHandle(ns.chainId, flag, h); err != nil {
		return err
	}
	return nil
}

func (ns *NetService) cancelReceiveMsg(flag string) error {
	if err := ns.localNet.CancelDirectMsgHandle(ns.chainId, flag); err != nil {
		return err
	}
	return nil
}

// MsgForMsgBusHandler is a handler function that receive the msg from net than publish to msg-bus.
type MsgForMsgBusHandler func(chainId string, from string, msg *netPb.NetMsg) error

func (ns *NetService) receiveMsgForMsgBus(handler MsgForMsgBusHandler, flag string) error {
	h := func(from string, netMsg *netPb.NetMsg) error {
		err := handler(ns.chainId, from, netMsg)
		if err != nil {
			return err
		}
		return nil
	}
	if err := ns.localNet.DirectMsgHandle(ns.chainId, flag, h); err != nil {
		return err
	}
	return nil
}

func (ns *NetService) subscribeTopicForMsgBus(handler MsgForMsgBusHandler, topic string) error {
	h := func(from string, netMsg *netPb.NetMsg) error {
		err := handler(ns.chainId, from, netMsg)
		if err != nil {
			return err
		}
		return nil
	}
	if err := ns.localNet.SubscribeWithChainId(ns.chainId, topic, h); err != nil {
		return err
	}
	return nil
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

	// init pubsub
	if err := ns.localNet.InitPubsub(ns.chainId, 0); err != nil {
		ns.logger.Errorf("[NetService] init pubsub failed, %s", err.Error())
		return err
	}
	if err := ns.initBindMsgBus(); err != nil {
		return err
	}
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
	return "NetService"
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
	// 2.refresh trust roots
	// 2.1 get all new roots
	newCerts := make([][]byte, 0)
	for _, root := range chainConfig.TrustRoots {
		newCerts = append(newCerts, []byte(root.Root))
	}
	// load custom chain trust roots
	for _, chainTrustRoots := range localconf.ChainMakerConfig.NetConfig.CustomChainTrustRoots {
		if chainTrustRoots.ChainId != cw.ns.chainId {
			continue
		}
		for _, roots := range chainTrustRoots.TrustRoots {
			rootBytes, err := ioutil.ReadFile(roots.Root)
			if err != nil {
				cw.ns.logger.Errorf("[NetService] load custom chain trust roots failed, %s", err.Error())
				return err
			}
			newCerts = append(newCerts, rootBytes)
		}
		cw.ns.logger.Infof("[NetService] load custom chain trust roots ok")
	}
	// 2.2 rebuild cert pool
	if err := cw.ns.localNet.RefreshTrustRoots(cw.ns.chainId, newCerts); err != nil {
		cw.ns.logger.Errorf("[NetService] refresh root certs pool failed ,%s", err.Error())
		return err
	}
	cw.ns.logger.Infof("[NetService] refresh root certs pool ok")
	// 2.3 verify trust root again
	cw.ns.localNet.ReVerifyTrustRoots(cw.ns.chainId)
	cw.ns.logger.Infof("[NetService] re-verify trust roots ok")
	cw.ns.logger.Infof("[NetService] refresh chain config ok")
	return nil
}

// VmWatcher return a implementation of protocol.VmWatcher. It is used for refreshing revoked peer which use revoked tls cert.
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
	return "NetService"
}

func (v *VmWatcher) ContractNames() []string {
	return []string{commonPb.ContractName_SYSTEM_CONTRACT_CERT_MANAGE.String()}
}

func (v *VmWatcher) Callback(contractName string, payloadBytes []byte) error {
	switch contractName {
	case commonPb.ContractName_SYSTEM_CONTRACT_CERT_MANAGE.String():
		return v.ns.localNet.CheckRevokeTlsCerts(v.ns.ac, payloadBytes)
	default:
		return nil
	}
}

// HandleMsgBusSubscriberOnMessage is a handler used for msg-bus subscriber OnMessage method.
func HandleMsgBusSubscriberOnMessage(netService *NetService, msgType netPb.NetMsg_MsgType, logMsgDescription string, message *msgbus.Message) error {
	if netMsg, ok := message.Payload.(*netPb.NetMsg); ok {
		if netMsg.Type.String() != msgType.String() {
			netService.logger.Errorf("[NetService/msg-bus %s subscriber] wrong net msg type(expect %s, got %s)", logMsgDescription, msgType.String(), netMsg.Type.String())
			return errors.New("wrong net msg type")
		}
		if netMsg.To == "" {
			return handleMsgBusSubscriberOnMessageBroadcast(netService, msgType, logMsgDescription, netMsg)
		} else {
			return handleMsgBusSubscriberOnMessageSend(netService, msgType, logMsgDescription, netMsg)
		}
	}
	return nil
}

func handleMsgBusSubscriberOnMessageBroadcast(netService *NetService, msgType netPb.NetMsg_MsgType, logMsgDescription string, netMsg *netPb.NetMsg) error {
	if msgType == netPb.NetMsg_TX || msgType == netPb.NetMsg_CONSENSUS_MSG {
		if err := netService.consensusBroadcastMsg(netMsg, fmt.Sprintf(msgBusConsensusTopicTemplate, msgType.String())); err != nil {
			netService.logger.Debugf("[NetService/msg-bus %s subscriber] broadcast failed, %s", logMsgDescription, err.Error())
			return err
		}
	} else {
		if err := netService.broadcastMsg(netMsg, fmt.Sprintf(msgBusTopicTemplate, msgType.String())); err != nil {
			netService.logger.Debugf("[NetService/msg-bus %s subscriber] broadcast failed, %s", logMsgDescription, err.Error())
			return err
		}
	}
	netService.logger.Debugf("[NetService/msg-bus %s subscriber] broadcast ok", logMsgDescription)
	return nil
}

func handleMsgBusSubscriberOnMessageSend(netService *NetService, msgType netPb.NetMsg_MsgType, logMsgDescription string, netMsg *netPb.NetMsg) error {
	go func() {
		if err := netService.sendMsg(netMsg, fmt.Sprintf(msgBusMsgFlagTemplate, msgType.String())); err != nil {
			netService.logger.Debugf("[NetService/msg-bus %s subscriber] send msg failed (size:%d) (reason:%s) (to:%s)", logMsgDescription, proto.Size(netMsg), err.Error(), netMsg.To)
		} else {
			netService.logger.Debugf("[NetService/msg-bus %s subscriber] send msg ok (size:%d) (to:%s)", logMsgDescription, proto.Size(netMsg), netMsg.To)
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
			err := HandleMsgBusSubscriberOnMessage(cms.netService, netPb.NetMsg_CONSENSUS_MSG, "consensus msg", message)
			if err != nil {
				cms.netService.logger.Warnf("[ConsensusMsgSubscriber] handle message failed, %s", err.Error())
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
			err := HandleMsgBusSubscriberOnMessage(cms.netService, netPb.NetMsg_TX, "tx_pool msg", message)
			if err != nil {
				cms.netService.logger.Warnf("[TxPoolMsgSubscriber] handle message failed, %s", err.Error())
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
			err := HandleMsgBusSubscriberOnMessage(cms.netService, netPb.NetMsg_SYNC_BLOCK_MSG, "sync block msg", message)
			if err != nil {
				cms.netService.logger.Warnf("[SyncBlockMsgSubscriber] handle message failed, %s", err.Error())
			}
		}()
	default:
	}
}

func (cms *SyncBlockMsgSubscriber) OnQuit() {
	// do nothing
	//panic("implement me")
}

func CreateMsgHandlerForMsgBus(netService *NetService, topic msgbus.Topic, logMsgDescription string) func(chainId string, node string, pbMsg *netPb.NetMsg) error {
	return func(chainId string, node string, pbMsg *netPb.NetMsg) error {
		netService.logger.Debugf("[NetService/%s handler for msg-bus] receive msg (size:%d) (from:%s)", logMsgDescription, proto.Size(pbMsg), node)
		if netService.chainId != chainId {
			netService.logger.Warnf("[NetService/%s handler for msg-bus] wrong chain-id(chain-id:%s), ignored.", logMsgDescription, chainId)
			return nil
		}
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
	consensusMsgHandler := CreateMsgHandlerForMsgBus(ns, msgbus.RecvConsensusMsg, "consensus msg")
	if err := ns.receiveMsgForMsgBus(consensusMsgHandler, fmt.Sprintf(msgBusMsgFlagTemplate, netPb.NetMsg_CONSENSUS_MSG.String())); err != nil {
		return err
	}
	if err := ns.receiveMsgForMsgBus(consensusMsgHandler, fmt.Sprintf(msgBusConsensusTopicTemplate, netPb.NetMsg_CONSENSUS_MSG.String())); err != nil {
		return err
	}

	// subscribe a consensus msg subscriber for receiving consensus msg from msg-bus then broadcast the msg to consensus nodes.
	cmSubscriber := &ConsensusMsgSubscriber{
		netService: ns,
	}
	ns.msgBus.Register(msgbus.SendConsensusMsg, cmSubscriber)

	// ===========================================

	// for tx pool module
	// receive tx msg from net then publish to msg-bus
	txPoolMsgHandler := CreateMsgHandlerForMsgBus(ns, msgbus.RecvTxPoolMsg, "tx_pool msg")
	if err := ns.receiveMsgForMsgBus(txPoolMsgHandler, fmt.Sprintf(msgBusMsgFlagTemplate, netPb.NetMsg_TX.String())); err != nil {
		return err
	}
	if err := ns.receiveMsgForMsgBus(txPoolMsgHandler, fmt.Sprintf(msgBusConsensusTopicTemplate, netPb.NetMsg_TX.String())); err != nil {
		return err
	}
	// subscribe the topic that ths spv node broadcast to
	if err := ns.subscribeTopicForMsgBus(txPoolMsgHandler, fmt.Sprintf(topicNameTemplate, netPb.NetMsg_TX.String())); err != nil {
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
	sbmHandler := CreateMsgHandlerForMsgBus(ns, msgbus.RecvSyncBlockMsg, "sync block msg")
	if err := ns.receiveMsgForMsgBus(sbmHandler, fmt.Sprintf(msgBusMsgFlagTemplate, netPb.NetMsg_SYNC_BLOCK_MSG.String())); err != nil {
		return err
	}
	if err := ns.subscribeTopicForMsgBus(sbmHandler, fmt.Sprintf(msgBusTopicTemplate, netPb.NetMsg_SYNC_BLOCK_MSG.String())); err != nil {
		return err
	}
	// subscribe a sync block msg subscriber for receiving sync block msg from msg-bus then broadcast the msg to consensus nodes.
	sbmSubscriber := &SyncBlockMsgSubscriber{
		netService: ns,
	}
	ns.msgBus.Register(msgbus.SendSyncBlockMsg, sbmSubscriber)
	ns.logger.Infof("[NetService] init bind msg-bus ok")
	return nil
}
