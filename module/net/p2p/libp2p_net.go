/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package p2p

import (
	"bufio"
	"context"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"chainmaker.org/chainmaker-go/net/p2p/libp2pgmtls"
	"chainmaker.org/chainmaker-go/net/p2p/libp2ptls"
	cmx509 "chainmaker.org/chainmaker/common/v2/crypto/x509"
	"chainmaker.org/chainmaker/common/v2/helper"
	pbac "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	netPb "chainmaker.org/chainmaker/pb-go/v2/net"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"
	"chainmaker.org/chainmaker/utils/v2"
	"github.com/gogo/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/network"

	api "chainmaker.org/chainmaker/protocol/v2"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	libP2pPubSub "github.com/libp2p/go-libp2p-pubsub"
)

// ErrorPubSubNotExist
var ErrorPubSubNotExist = errors.New("pubsub not exist")

// ErrorPubSubExisted
var ErrorPubSubExisted = errors.New("pubsub existed")

// ErrorTopicSubscribed
var ErrorTopicSubscribed = errors.New("topic has been subscribed")

// ErrorSendMsgIncompletely
var ErrorSendMsgIncompletely = errors.New("send msg incompletely")

// ErrorNotConnected
var ErrorNotConnected = errors.New("node not connected")

// ErrorNotBelongToChain
var ErrorNotBelongToChain = errors.New("node not belong to chain")

// MsgPID is the protocol.ID of chainmaker net msg.
const MsgPID = protocol.ID("/ChainMakerNetMsg/1.0.0/")

// DefaultMessageSendTimeout is the default timeout for sending msg.
const DefaultMessageSendTimeout = 3 * time.Second

// compressThreshold is the default threshold value for enable compress net msg bytes. Default value is 1M.
const compressThreshold = 1024 * 1024

const pubsubWhiteListChanCap = 50
const pubsubWhiteListChanQuitCheckDelay = 10

// LibP2pNet is an implementation of net.Net interface.
type LibP2pNet struct {
	compressMsgBytes          bool
	lock                      sync.RWMutex
	startUp                   bool
	netType                   api.NetType
	ctx                       context.Context // ctx context.Context
	libP2pHost                *LibP2pHost     // libP2pHost is a LibP2pHost instance.
	messageHandlerDistributor *MessageHandlerDistributor

	pubSubs          sync.Map                      // pubSubs mapping the chainId to the LibP2pPubSub . Same as map[string]*LibP2pPubSub .
	subscribedTopics map[string]*topicSubscription // subscribedTopics mapping the chainId to a map which mapping the topic name to the Subscription.
	subscribeLock    sync.Mutex

	prepare *LibP2pNetPrepare // prepare contains the base info for the net starting.
}

func (ln *LibP2pNet) SetCompressMsgBytes(enable bool) {
	ln.compressMsgBytes = enable
}

type topicSubscription struct {
	m map[string]*libP2pPubSub.Subscription
}

func (ln *LibP2pNet) peerChainIdsRecorder() *PeerIdChainIdsRecorder {
	return ln.libP2pHost.peerChainIdsRecorder
}

// NewLibP2pNet create a new LibP2pNet instance.
func NewLibP2pNet() (*LibP2pNet, error) {
	ctx := context.Background()
	host := NewLibP2pHost(ctx)
	net := &LibP2pNet{
		startUp:                   false,
		netType:                   api.Libp2p,
		ctx:                       ctx,
		libP2pHost:                host,
		messageHandlerDistributor: newMessageHandlerDistributor(),
		pubSubs:                   sync.Map{},
		subscribedTopics:          make(map[string]*topicSubscription),

		prepare: &LibP2pNetPrepare{
			listenAddr:               DefaultLibp2pListenAddress,
			bootstrapsPeers:          make(map[string]struct{}),
			chainTrustRootCertsBytes: make(map[string][][]byte, 0),
			maxPeerCountAllow:        DefaultMaxPeerCountAllow,
			peerEliminationStrategy:  int(LIFO),

			blackAddresses: make(map[string]struct{}),
			blackPeerIds:   make(map[string]struct{}),
		},
	}
	return net, nil
}

func (ln *LibP2pNet) Prepare() *LibP2pNetPrepare {
	return ln.prepare
}

// GetNodeUid is the unique id of node.
func (ln *LibP2pNet) GetNodeUid() string {
	return ln.libP2pHost.Host().ID().Pretty()
}

// isSubscribed return true if the given topic given has subscribed.Otherwise return false.
func (ln *LibP2pNet) isSubscribed(chainId string, topic string) bool {
	topics, ok := ln.subscribedTopics[chainId]
	if !ok {
		return false
	}
	_, ok = topics.m[topic]
	return ok
}

// getPubsub return the LibP2pPubSub instance which uid equal the given chainId .
func (ln *LibP2pNet) getPubsub(chainId string) (*LibP2pPubSub, bool) {
	ps, ok := ln.pubSubs.Load(chainId)
	var pubsub *LibP2pPubSub = nil
	if ok {
		pubsub = ps.(*LibP2pPubSub)
	}
	return pubsub, ok
}

// InitPubsub will create new LibP2pPubSub instance for LibP2pNet with setting pubsub uid to the given chainId .
func (ln *LibP2pNet) InitPubsub(chainId string, maxMessageSize int) error {
	_, ok := ln.getPubsub(chainId)
	if ok {
		return ErrorPubSubExisted
	}
	if maxMessageSize <= 0 {
		maxMessageSize = DefaultLibp2pPubSubMaxMessageSize
	}
	ps, err := NewPubsub(chainId, ln.libP2pHost, maxMessageSize)
	if err != nil {
		logger.Errorf("[Net] new pubsub failed, %s", err.Error())
		return err
	}
	ln.pubSubs.Store(chainId, ps)
	if ln.startUp {
		if err = ps.Start(); err != nil {
			return err
		}
		ln.reloadChainPubSubWhiteList(chainId)
	}
	return nil
}

// BroadcastWithChainId broadcast a msg to the given topic of the target chain which id is the given chainId .
func (ln *LibP2pNet) BroadcastWithChainId(chainId string, topic string, netMsg *netPb.NetMsg) error {
	topic = chainId + "_" + topic
	msg := NewMsg(netMsg, chainId, "")
	bytes, err := proto.Marshal(msg)
	if err != nil {
		logger.Errorf("[Net] marshal net pb msg failed, %s", err.Error())
		return err
	}
	pubsub, ok := ln.getPubsub(chainId)
	if !ok {
		return ErrorPubSubNotExist
	}
	return pubsub.Publish(topic, bytes) //publish msg
}

// getSubscribeTopicMap
func (ln *LibP2pNet) getSubscribeTopicMap(chainId string) *topicSubscription {
	topics, ok := ln.subscribedTopics[chainId]
	if !ok {
		ln.subscribedTopics[chainId] = &topicSubscription{
			m: make(map[string]*libP2pPubSub.Subscription),
		}
		topics = ln.subscribedTopics[chainId]
	}
	return topics
}

// SubscribeWithChainId subscribe the given topic of the target chain which id is the given chainId with the given sub-msg handler function.
func (ln *LibP2pNet) SubscribeWithChainId(chainId string, topic string, handler api.PubsubMsgHandler) error {
	ln.subscribeLock.Lock()
	defer ln.subscribeLock.Unlock()
	topic = chainId + "_" + topic
	// whether pubsub existed
	pubsub, ok := ln.getPubsub(chainId)
	if !ok {
		return ErrorPubSubNotExist
	}
	// whether has subscribed
	if ln.isSubscribed(chainId, topic) { //检查topic是否已被订阅
		return ErrorTopicSubscribed
	}
	topicSub, err := pubsub.Subscribe(topic) // subscribe the topic
	if err != nil {
		return err
	}
	// add subscribe info
	topics := ln.getSubscribeTopicMap(chainId)
	topics.m[topic] = topicSub
	// run a new goroutine to handle the msg from the topic subscribed.
	go func() {
		defer func() {
			if err := recover(); err != nil {
				if !ln.isSubscribed(chainId, topic) {
					return
				}
				logger.Errorf("[Net] subscribe goroutine recover err, %s", err)
				logger.Error(debug.Stack())
			}
		}()
		ln.topicSubLoop(topicSub, topic, handler)
	}()
	return nil
}

func (ln *LibP2pNet) topicSubLoop(topicSub *libP2pPubSub.Subscription, topic string, handler api.PubsubMsgHandler) {
	for {
		message, err := topicSub.Next(ln.ctx)
		if err != nil {
			if err.Error() == "subscription cancelled" {
				logger.Warn("[Net] ", err)
				break
			}
			//logger
			logger.Errorf("[Net] subscribe next failed, %s", err.Error())
		}
		if message == nil {
			return
		}
		// if author of the msg is myself , just skip and continue
		if message.ReceivedFrom == ln.libP2pHost.host.ID() || message.GetFrom() == ln.libP2pHost.host.ID() {
			continue
		}
		bytes := message.GetData()
		logger.Debugf("[Net] receive subscribed msg(topic:%s), data size:%d", topic, len(bytes))
		msg := &netPb.Msg{}
		err = proto.Unmarshal(bytes, msg)
		if err != nil {
			logger.Errorf("[Net] unmarshal net pb msg failed, %s", err)
		}
		// call handler
		if err := handler(message.GetFrom().Pretty(), msg.GetMsg()); err != nil {
			logger.Errorf("[Net] call subscribe handler failed, %s ", err)
		}
	}
}

// CancelSubscribeWithChainId cancel subscribing the given topic of the target chain which id is the given chainId.
func (ln *LibP2pNet) CancelSubscribeWithChainId(chainId string, topic string) error {
	ln.subscribeLock.Lock()
	defer ln.subscribeLock.Unlock()
	topic = chainId + "_" + topic
	//检查pubsub是否存在
	_, ok := ln.getPubsub(chainId)
	if !ok {
		return ErrorPubSubNotExist
	}
	//取消订阅、移除订阅信息
	topics := ln.getSubscribeTopicMap(chainId)
	if topicSub, ok := topics.m[topic]; ok {
		topicSub.Cancel()
		delete(topics.m, topic)
	}
	return nil
}

func (ln *LibP2pNet) isConnected(node string) (bool, peer.ID, error) {
	isConnected := false
	peerId := node
	if ln.libP2pHost.certPeerIdMapper != nil {
		var err error
		peerIdNew, err := ln.libP2pHost.certPeerIdMapper.findPeerIdByCertId(node)
		if err == nil && peerIdNew != "" {
			// not found
			peerId = peerIdNew
		}
	}
	pid, err := peer.Decode(peerId) // peerId
	if err != nil {
		return false, pid, err
	}
	isConnected = ln.libP2pHost.HasConnected(pid)
	return isConnected, pid, nil

}

// SendMsg send a msg to the given node belong to the given chain.
func (ln *LibP2pNet) SendMsg(chainId string, node string, msgFlag string, netMsg *netPb.NetMsg) error {
	if node == ln.GetNodeUid() {
		logger.Warn("[Net] can not send msg to self")
		return nil
	}
	isConnected, pid, _ := ln.isConnected(node)
	if !isConnected { // is peer connected
		return ErrorNotConnected // node not connected
	}
	// is peer belong to this chain
	if !ln.prepare.isInsecurity && !ln.libP2pHost.peerChainIdsRecorder.isPeerBelongToChain(pid.Pretty(), chainId) {
		return ErrorNotBelongToChain
	}
	stream, err := ln.libP2pHost.peerStreamManager.borrowPeerStream(pid)
	if err != nil {
		return err
	}

	msg := NewMsg(netMsg, chainId, msgFlag)
	bytes, err := proto.Marshal(msg)
	if err != nil {
		logger.Errorf("[Net] marshal net pb msg failed, %s", err.Error())
		ln.libP2pHost.peerStreamManager.returnPeerStream(pid, stream)
		return err
	}
	// whether compress bytes
	isCompressed := []byte{byte(0)}
	if ln.compressMsgBytes && len(bytes) > compressThreshold {
		lengthBeforeCompress := len(bytes)
		bytes, err = utils.GZipCompressBytes(bytes)
		if err != nil {
			return err
		}
		isCompressed[0] = byte(1)
		lengthAfterCompress := len(bytes)
		logger.Debugf("[Net] compress net msg bytes ok(length before/after compress: %d/%d)", lengthBeforeCompress, lengthAfterCompress)
	}

	lengthBytes := IntToBytes(len(bytes))

	writeBytes := append(append(lengthBytes, isCompressed...), bytes...)
	size, err := stream.Write(writeBytes) // send data
	if err != nil {
		ln.libP2pHost.peerStreamManager.dropPeerStream(pid, stream)
		return err
	}
	if size < len(writeBytes) { //发送不完整
		ln.libP2pHost.peerStreamManager.dropPeerStream(pid, stream)
		return ErrorSendMsgIncompletely
	}
	ln.libP2pHost.peerStreamManager.returnPeerStream(pid, stream)
	return nil
}

func (ln *LibP2pNet) registerMsgHandle() error {
	var streamReadHandler = func(stream network.Stream) {
		streamReadHandlerFunc := NewStreamReadHandlerFunc(ln.messageHandlerDistributor)
		go streamReadHandlerFunc(stream)

		// if you want to use two-way stream , open this
		//ln.libP2pHost.peerStreamManager.addPeerStream(stream.Conn().RemotePeer(), stream)
	}
	ln.libP2pHost.host.SetStreamHandler(MsgPID, streamReadHandler) // set stream handler for libP2pHost.
	return nil
}

// NewStreamReadHandlerFunc create new function for listening stream reading.
func NewStreamReadHandlerFunc(mhd *MessageHandlerDistributor) func(stream network.Stream) {
	return func(stream network.Stream) {
		id := stream.Conn().RemotePeer().Pretty() // sender peer id
		reader := bufio.NewReader(stream)
		for {
			length := readMsgLength(reader, stream)
			if length == -1 {
				break
			} else if length == -2 {
				continue
			}

			isCompressed := readMsgIsCompressed(reader, stream)
			if isCompressed == -1 {
				break
			} else if isCompressed == -2 {
				continue
			}

			data, ret := readMsgReadDataRealWanted(reader, stream, length)
			if ret == -1 {
				break
			} else if ret == -2 {
				continue
			}
			if len(data) == 0 {
				time.Sleep(500 * time.Millisecond)
				continue
			}
			if isCompressed == 1 {
				var err error
				data, err = utils.GZipDeCompressBytes(data)
				if err != nil {
					logger.Error("[Net] decompress msg bytes failed, %s", err.Error())
					continue
				}
			}
			var msg netPb.Msg
			err := proto.Unmarshal(data, &msg)
			if err != nil {
				logger.Error("[Net] unmarshal net pb msg failed, %s", err.Error())
				continue
			}
			handler := mhd.handler(msg.GetChainId(), msg.GetFlag())
			if handler == nil {
				logger.Warnf("[Net] handler not registered. drop message. (chainId:%s, flag:%s)", msg.GetChainId(), msg.GetFlag())
				continue
			}
			readMsgCallHandler(id, msg.GetMsg(), handler)
		}
	}
}

func readData(reader *bufio.Reader, length int) ([]byte, error) {
	batchSize := 4000
	result := make([]byte, 0)
	for length > 0 {
		if length < batchSize {
			batchSize = length
		}
		bytes := make([]byte, batchSize)
		c, err := reader.Read(bytes)
		if err != nil {
			return nil, err
		}
		length = length - c
		result = append(result, bytes[0:c]...)
	}
	return result, nil
}

func readMsgReadDataErrCheck(err error, stream network.Stream) int {
	if strings.Contains(err.Error(), "stream reset") {
		_ = stream.Reset()
		return -1
	}
	logger.Errorf("[Net] read stream failed, %s", err.Error())
	return -2
}

func readMsgLength(reader *bufio.Reader, stream network.Stream) int {
	lengthBytes := make([]byte, 8)
	_, err := reader.Read(lengthBytes)
	if err != nil {
		return readMsgReadDataErrCheck(err, stream)
	}
	length := BytesToInt(lengthBytes)
	return length
}

func readMsgIsCompressed(reader *bufio.Reader, stream network.Stream) int {
	isCompressedBytes, err := readData(reader, 1)
	if err != nil {
		return readMsgReadDataErrCheck(err, stream)
	}
	if isCompressedBytes[0] == byte(1) {
		return 1
	}
	return 0
}

func readMsgReadDataRealWanted(reader *bufio.Reader, stream network.Stream, length int) ([]byte, int) {
	data, err := readData(reader, length)
	if err != nil {
		return nil, readMsgReadDataErrCheck(err, stream)
	}
	return data, 0
}

func readMsgCallHandler(id string, netMsg *netPb.NetMsg, handler api.DirectMsgHandler) {
	go func(id string, netMsg *netPb.NetMsg, handler api.DirectMsgHandler) {
		defer func() {
			if err := recover(); err != nil {
				logger.Error("[Net] stream read handler func call handler recover failed, %s", err)
			}
		}()
		err := handler(id, netMsg) // call handler
		if err != nil {
			logger.Error("[Net] stream read handler func call handler failed, %s", err.Error())
		}
	}(id, netMsg, handler)
}

// DirectMsgHandle register a DirectMsgHandler for handling msg received.
func (ln *LibP2pNet) DirectMsgHandle(chainId string, msgFlag string, handler api.DirectMsgHandler) error {
	return ln.messageHandlerDistributor.registerHandler(chainId, msgFlag, handler)
}

// CancelDirectMsgHandle unregister a DirectMsgHandler for handling msg received.
func (ln *LibP2pNet) CancelDirectMsgHandle(chainId string, msgFlag string) error {
	ln.messageHandlerDistributor.cancelRegisterHandler(chainId, msgFlag) // remove stream handler for libP2pHost.
	return nil
}

// AddSeed add a seed node address. It can be a consensus node address.
func (ln *LibP2pNet) AddSeed(seed string) error {
	newSeedsAddrInfos, err := ParseAddrInfo([]string{seed})
	if err != nil {
		return err
	}
	for _, info := range newSeedsAddrInfos {
		ln.libP2pHost.connManager.AddAsHighLevelPeer(info.ID)
	}

	if ln.startUp {
		seedPid, err := helper.GetNodeUidFromAddr(seed)
		if err != nil {
			return err
		}
		oldSeedsAddrInfos := ln.libP2pHost.connSupervisor.getPeerAddrInfos()
		for _, ai := range oldSeedsAddrInfos {
			if ai.ID.Pretty() == seedPid {
				logger.Warn("[Net] seed already exists. ignored.")
				return nil
			}
		}

		oldSeedsAddrInfos = append(oldSeedsAddrInfos, newSeedsAddrInfos...)
		ln.libP2pHost.connSupervisor.refreshPeerAddrInfos(oldSeedsAddrInfos)
		return nil
	}
	ln.prepare.AddBootstrapsPeer(seed)
	return nil
}

// RefreshSeeds reset addresses of seed nodes with given.
func (ln *LibP2pNet) RefreshSeeds(seeds []string) error {
	newSeedsAddrInfos, err := ParseAddrInfo(seeds)
	if err != nil {
		return err
	}
	ln.libP2pHost.connManager.ClearHighLevelPeer()
	for _, info := range newSeedsAddrInfos {
		ln.libP2pHost.connManager.AddAsHighLevelPeer(info.ID)
	}
	if ln.startUp {
		ln.libP2pHost.connSupervisor.refreshPeerAddrInfos(newSeedsAddrInfos)
		return nil
	}
	for _, seed := range seeds {
		ln.prepare.AddBootstrapsPeer(seed)
	}
	return nil
}

// AddTrustRoot add a root cert for chain.
func (ln *LibP2pNet) AddTrustRoot(chainId string, rootCertByte []byte) error {
	if ln.startUp {
		if ln.libP2pHost.isGmTls {
			_, err := libp2pgmtls.AppendNewCertsToTrustRoots(ln.libP2pHost.gmTlsChainTrustRoots, chainId, rootCertByte)
			if err != nil {
				logger.Errorf("[Net] add trust root failed. %s", err.Error())
				return err
			}
		} else if ln.libP2pHost.isTls {
			_, err := libp2ptls.AppendNewCertsToTrustRoots(ln.libP2pHost.tlsChainTrustRoots, chainId, rootCertByte)
			if err != nil {
				logger.Errorf("[Net] add trust root failed. %s", err.Error())
				return err
			}
		}
		return nil
	}
	ln.prepare.AddTrustRootCert(chainId, rootCertByte)
	return nil
}

func (ln *LibP2pNet) ReVerifyTrustRoots(chainId string) {
	if ln.startUp {
		peerIdTlsCertMap := ln.libP2pHost.peerIdTlsCertStore.storeCopy()
		if len(peerIdTlsCertMap) == 0 {
			return
		}
		if ln.libP2pHost.isGmTls {
			chainTrustRoots := ln.libP2pHost.gmTlsChainTrustRoots
			for pid, bytes := range peerIdTlsCertMap {
				cert, err := cmx509.ParseCertificate(bytes)
				if err != nil {
					logger.Errorf("[Net] parse tls cert failed. %s", err.Error())
					continue
				}
				if chainTrustRoots.VerifyCertOfChain(chainId, cert) {
					ln.libP2pHost.peerChainIdsRecorder.addPeerChainId(pid, chainId)
					logger.Infof("[Net] add peer to chain, (pid: %s, chain id: %s)", pid, chainId)
				}
			}
		} else if ln.libP2pHost.isTls {
			chainTrustRoots := ln.libP2pHost.tlsChainTrustRoots
			for pid, bytes := range peerIdTlsCertMap {
				cert, err := x509.ParseCertificate(bytes)
				if err != nil {
					logger.Errorf("[Net] parse tls cert failed. %s", err.Error())
					continue
				}
				if chainTrustRoots.VerifyCertOfChain(chainId, cert) {
					ln.libP2pHost.peerChainIdsRecorder.addPeerChainId(pid, chainId)
					logger.Infof("[Net] add peer to chain, (pid: %s, chain id: %s)", pid, chainId)
				}
			}
		}
		ln.reloadChainPubSubWhiteList(chainId)
	}
}

func (ln *LibP2pNet) reloadChainPubSubWhiteList(chainId string) {
	if ln.startUp {
		v, ok := ln.pubSubs.Load(chainId)
		if ok {
			ps := v.(*LibP2pPubSub)
			for _, pidStr := range ln.libP2pHost.peerChainIdsRecorder.peerIdsOfChain(chainId) {
				pid, err := peer.Decode(pidStr)
				if err != nil {
					logger.Errorf("[Net] parse peer id string to pid failed. %s", err.Error())
					continue
				}
				err = ps.AddWhitelistPeer(pid)
				if err != nil {
					logger.Errorf("[Net] add pubsub white list failed. %s (pid: %s, chain id: %s)", err.Error(), pid, chainId)
					continue
				}
				logger.Infof("[Net] add peer to chain pubsub white list, (pid: %s, chain id: %s)", pid, chainId)
				_ = ps.TryToReloadPeer(pid)
			}

		}
	}
}

// RefreshTrustRoots reset all root certs for chain.
func (ln *LibP2pNet) RefreshTrustRoots(chainId string, rootsCertsBytes [][]byte) error {
	if ln.startUp {
		if !ln.libP2pHost.isTls {
			logger.Warn("[Net] tls disabled. ignored.")
			return nil
		}
		if ln.libP2pHost.isGmTls {
			if !ln.libP2pHost.gmTlsChainTrustRoots.RefreshRootsFromPem(chainId, rootsCertsBytes) {
				return errors.New("refresh trust roots failed")
			}
			return nil
		}
		if !ln.libP2pHost.tlsChainTrustRoots.RefreshRootsFromPem(chainId, rootsCertsBytes) {
			return errors.New("refresh trust roots failed")
		}
		return nil
	}
	for _, certsByte := range rootsCertsBytes {
		ln.prepare.AddTrustRootCert(chainId, certsByte)
	}
	return nil
}

// IsRunning
func (ln *LibP2pNet) IsRunning() bool {
	ln.lock.RLock()
	defer ln.lock.RUnlock()
	return ln.startUp
}

// ChainNodesInfo
func (ln *LibP2pNet) ChainNodesInfo(chainId string) ([]*api.ChainNodeInfo, error) {
	result := make([]*api.ChainNodeInfo, 0)
	if ln.libP2pHost.isTls {
		// 1.find all peerIds of chain
		peerIds := make([]string, 0)
		peerIds = append(peerIds, ln.libP2pHost.host.ID().Pretty())
		peerIds = append(peerIds, ln.libP2pHost.peerChainIdsRecorder.peerIdsOfChain(chainId)...)
		for _, peerId := range peerIds {
			// 2.find addr
			pid, _ := peer.Decode(peerId)
			addrs := make([]string, 0)
			if pid == ln.libP2pHost.host.ID() {
				for _, multiaddr := range ln.libP2pHost.host.Addrs() {
					addrs = append(addrs, multiaddr.String())
				}
			} else {
				conn := ln.libP2pHost.connManager.GetConn(pid)
				if conn == nil || conn.RemoteMultiaddr() == nil {
					continue
				}
				addrs = append(addrs, conn.RemoteMultiaddr().String())
			}

			// 3.find cert
			cert := ln.libP2pHost.peerIdTlsCertStore.getCertByPeerId(peerId)
			result = append(result, &api.ChainNodeInfo{
				NodeUid:     peerId,
				NodeAddress: addrs,
				NodeTlsCert: cert,
			})
		}
	}
	return result, nil
}

// GetNodeUidByCertId
func (ln *LibP2pNet) GetNodeUidByCertId(certId string) (string, error) {
	nodeUid, err := ln.libP2pHost.certPeerIdMapper.findPeerIdByCertId(certId)
	if err != nil {
		return "", err
	}
	return nodeUid, nil
}

func (ln *LibP2pNet) handlePubSubWhiteList() {
	ln.handlePubSubWhiteListOnAddC()
	ln.handlePubSubWhiteListOnRemoveC()
}

func (ln *LibP2pNet) handlePubSubWhiteListOnAddC() {
	go func() {
		onAddC := make(chan string, pubsubWhiteListChanCap)
		ln.libP2pHost.peerChainIdsRecorder.onAddNotifyC(onAddC)
		go func() {
			for ln.IsRunning() {
				time.Sleep(time.Duration(pubsubWhiteListChanQuitCheckDelay) * time.Second)
			}
			close(onAddC)
		}()

		for str := range onAddC {
			//logger.Debugf("[Net] handling pubsub white list on add chan,get %s", str)
			peerIdAndChainId := strings.Split(str, "<-->")
			ps, ok := ln.pubSubs.Load(peerIdAndChainId[1])
			if ok {
				pubsub := ps.(*LibP2pPubSub)
				pid, err := peer.Decode(peerIdAndChainId[0])
				if err != nil {
					logger.Errorf("[Net] peer decode failed, %s", err.Error())
				}
				logger.Infof("[Net] add to pubsub white list(peer-id:%s, chain-id:%s)", peerIdAndChainId[0], peerIdAndChainId[1])
				err = pubsub.AddWhitelistPeer(pid)
				if err != nil {
					logger.Errorf("[Net] add to pubsub white list(peer-id:%s, chain-id:%s) failed, %s", peerIdAndChainId[0], peerIdAndChainId[1], err.Error())
				}
			}
		}
	}()
}

func (ln *LibP2pNet) handlePubSubWhiteListOnRemoveC() {
	go func() {
		onRemoveC := make(chan string, pubsubWhiteListChanCap)
		ln.libP2pHost.peerChainIdsRecorder.onRemoveNotifyC(onRemoveC)
		go func() {
			for ln.IsRunning() {
				time.Sleep(time.Duration(pubsubWhiteListChanQuitCheckDelay) * time.Second)
			}
			close(onRemoveC)
		}()
		for str := range onRemoveC {
			peerIdAndChainId := strings.Split(str, "<-->")
			ps, ok := ln.pubSubs.Load(peerIdAndChainId[1])
			if ok {
				pubsub := ps.(*LibP2pPubSub)
				pid, err := peer.Decode(peerIdAndChainId[0])
				if err != nil {
					logger.Errorf("[Net] peer decode failed, %s", err.Error())
					continue
				}
				logger.Debugf("[Net] remove from pubsub white list(peer-id:%s, chain-id:%s)", peerIdAndChainId[0], peerIdAndChainId[1])
				err = pubsub.RemoveWhitelistPeer(pid)
				if err != nil {
					logger.Errorf("[Net] remove from pubsub white list(peer-id:%s, chain-id:%s) failed, %s", peerIdAndChainId[0], peerIdAndChainId[1], err.Error())
				}
			}
		}
	}()
}

// Start
func (ln *LibP2pNet) Start() error {
	ln.lock.Lock()
	defer ln.lock.Unlock()
	if ln.startUp {
		logger.Warn("[Net] net is running.")
		return nil
	}
	var err error
	// prepare blacklist
	err = ln.prepareBlackList()
	if err != nil {
		return err
	}
	// create libp2p options
	ln.libP2pHost.opts, err = ln.createLibp2pOptions()
	if err != nil {
		return err
	}
	// set max size for conn manager
	ln.libP2pHost.connManager.SetMaxSize(ln.prepare.maxPeerCountAllow)
	// set elimination strategy for conn manager
	ln.libP2pHost.connManager.SetStrategy(ln.prepare.peerEliminationStrategy)
	// start libP2pHost
	if err := ln.libP2pHost.Start(); err != nil {
		return err
	}
	ln.initPeerStreamManager()
	if err := ln.registerMsgHandle(); err != nil {
		return err
	}
	ln.startUp = true

	// start handling NewTlsPeerChainIdsNotifyC
	if ln.libP2pHost.isTls && ln.libP2pHost.peerChainIdsRecorder != nil {
		ln.handlePubSubWhiteList()
	}
	// setup discovery
	adis := make([]string, 0)
	for bp := range ln.prepare.bootstrapsPeers {
		adis = append(adis, bp)
	}
	if err := SetupDiscovery(ln.libP2pHost, true, adis); err != nil {
		return err
	}
	// start pubsub
	var psErr error = nil
	ln.pubSubs.Range(func(_, value interface{}) bool {
		pubsub := value.(*LibP2pPubSub)
		if err := pubsub.Start(); err != nil {
			psErr = err
			return false
		}
		return true
	})
	if psErr != nil {
		return psErr
	}
	return nil
}

// Stop
func (ln *LibP2pNet) Stop() error {
	ln.lock.Lock()
	defer ln.lock.Unlock()
	if !ln.startUp {
		logger.Warn("[Net] net is not running.")
		return nil
	}
	err := ln.libP2pHost.Stop()
	if err != nil {
		return err
	}
	ln.startUp = false

	return nil
}

func (ln *LibP2pNet) AddAC(chainId string, ac api.AccessControlProvider) {
	ln.libP2pHost.revokedValidator.AddAC(chainId, ac)
}

func (ln *LibP2pNet) CheckRevokeTlsCerts(ac api.AccessControlProvider, certManageSystemContractPayload []byte) error {
	var payload commonPb.Payload
	err := proto.Unmarshal(certManageSystemContractPayload, &payload)
	if err != nil {
		return fmt.Errorf("resolve payload failed: %v", err)
	}
	switch payload.Method {
	case syscontract.CertManageFunction_CERTS_REVOKE.String():
		return ln.checkRevokeTlsCertsCertsRevokeMethod(ac, &payload)
	default:
		return nil
	}
}

func (ln *LibP2pNet) checkRevokeTlsCertsCertsRevokeMethod(ac api.AccessControlProvider, payload *commonPb.Payload) error {
	// get all node tls cert
	peerIdCertBytesMap := ln.libP2pHost.peerIdTlsCertStore.storeCopy()
	if len(peerIdCertBytesMap) == 0 {
		return nil
	}
	peerIdCertMap, err := parsePeerIdCertBytesMapToPeerIdCertMap(peerIdCertBytesMap)
	if err != nil {
		return err
	}
	return ln.checkRevokeTlsCertsCertsRevokeMethodRevokePeerId(ac, payload, peerIdCertMap)
}

func parsePeerIdCertBytesMapToPeerIdCertMap(peerIdCertBytesMap map[string][]byte) (map[string]*cmx509.Certificate, error) {
	peerIdCertMap := make(map[string]*cmx509.Certificate)
	for pid := range peerIdCertBytesMap {
		certBytes := peerIdCertBytesMap[pid]
		cert, err := cmx509.ParseCertificate(certBytes)
		if err != nil {
			logger.Errorf("[Net] parse chainmaker certificate failed, %s", err.Error())
			return nil, err
		}
		peerIdCertMap[pid] = cert
	}
	return peerIdCertMap, nil
}

func (ln *LibP2pNet) checkRevokeTlsCertsCertsRevokeMethodRevokePeerId(ac api.AccessControlProvider, payload *commonPb.Payload, peerIdCertMap map[string]*cmx509.Certificate) error {
	for _, param := range payload.Parameters {
		if param.Key == "cert_crl" {

			crlStr := strings.Replace(string(param.Value), ",", "\n", -1)
			_, err := ac.VerifyRelatedMaterial(pbac.VerifyType_CRL, []byte(crlStr))
			if err != nil {
				logger.Errorf("[Net] validate crl failed, %s", err.Error())
				return err
			}

			var crls []*pkix.CertificateList

			crl, err := x509.ParseCRL([]byte(crlStr))
			if err != nil {
				logger.Errorf("[Net] validate crl failed, %s", err.Error())
				return err
			}
			crls = append(crls, crl)
			if err != nil {
				logger.Errorf("[Net] validate crl failed, %s", err.Error())
				return err
			}
			revokedPeerIds := ln.findRevokedPeerIdsByCRLs(crls, peerIdCertMap)
			if err := ln.closeRevokedPeerConnection(revokedPeerIds); err != nil {
				return err
			}
			return nil
		}
	}
	return nil
}

func (ln *LibP2pNet) findRevokedPeerIdsByCRLs(crls []*pkix.CertificateList, peerIdCertMap map[string]*cmx509.Certificate) []string {
	revokedPeerIds := make([]string, 0)
	for _, crl := range crls {
		for _, rc := range crl.TBSCertList.RevokedCertificates {
			for pid := range peerIdCertMap {
				cert := peerIdCertMap[pid]
				if rc.SerialNumber.Cmp(cert.SerialNumber) == 0 {
					revokedPeerIds = append(revokedPeerIds, pid)
				}
			}
		}
	}
	return revokedPeerIds
}

func (ln *LibP2pNet) closeRevokedPeerConnection(revokedPeerIds []string) error {
	for idx := range revokedPeerIds {
		pid := revokedPeerIds[idx]
		logger.Infof("[Net] revoked peer found(pid: %s)", pid)
		peerId, err := peer.Decode(pid)
		if err != nil {
			logger.Errorf("[Net] decode peer id failed, %s", err.Error())
			return err
		}
		ln.libP2pHost.revokedValidator.AddPeerId(pid)
		if ln.libP2pHost.connManager.IsConnected(peerId) {
			conn := ln.libP2pHost.connManager.GetConn(peerId)
			_ = conn.Close()
			logger.Infof("[Net] closing revoked peer connection(pid: %s)", pid)
		}
	}
	return nil
}
