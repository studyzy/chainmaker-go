/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package blockchain

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"chainmaker.org/chainmaker-go/net"
	"chainmaker.org/chainmaker-go/subscriber"
	"chainmaker.org/chainmaker/common/v2/helper"
	"chainmaker.org/chainmaker/common/v2/msgbus"
	"chainmaker.org/chainmaker/localconf/v2"
	"chainmaker.org/chainmaker/logger/v2"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/protocol/v2"
)

var log = logger.GetLogger(logger.MODULE_BLOCKCHAIN)

const chainIdNotFoundErrorTemplate = "chain id %s not found"

// ChainMakerServer manage all blockchains
type ChainMakerServer struct {
	// net shared by all chains
	net net.Net

	// blockchains known by this node
	blockchains sync.Map // map[string]*Blockchain
}

// NewChainMakerServer create a new ChainMakerServer instance.
func NewChainMakerServer() *ChainMakerServer {
	return &ChainMakerServer{}
}

// Init ChainMakerServer.
func (server *ChainMakerServer) Init() error {
	var err error
	log.Debug("begin init chain maker server...")
	// 1) init net
	if err = server.initNet(); err != nil {
		return err
	}
	// 2) init blockchains
	if err = server.initBlockchains(); err != nil {
		return err
	}
	log.Info("init chain maker server success!")
	return nil
}

func (server *ChainMakerServer) initNet() error {
	var netType protocol.NetType
	var err error
	// load net type
	provider := localconf.ChainMakerConfig.NetConfig.Provider
	log.Infof("load net provider: %s", provider)
	switch strings.ToLower(provider) {
	case "libp2p":
		netType = protocol.Libp2p
	default:
		return errors.New("unsupported net provider")
	}
	// load tls key and cert path
	keyPath := localconf.ChainMakerConfig.NetConfig.TLSConfig.PrivKeyFile
	if !filepath.IsAbs(keyPath) {
		keyPath, err = filepath.Abs(keyPath)
		if err != nil {
			return err
		}
	}
	log.Infof("load net tls key file path: %s", keyPath)
	certPath := localconf.ChainMakerConfig.NetConfig.TLSConfig.CertFile
	if !filepath.IsAbs(certPath) {
		certPath, err = filepath.Abs(certPath)
		if err != nil {
			return err
		}
	}
	log.Infof("load net tls cert file path: %s", certPath)
	// new net
	var netFactory net.NetFactory
	server.net, err = netFactory.NewNet(
		netType,
		net.WithListenAddr(localconf.ChainMakerConfig.NetConfig.ListenAddr),
		net.WithCrypto(keyPath, certPath),
		net.WithPeerStreamPoolSize(localconf.ChainMakerConfig.NetConfig.PeerStreamPoolSize),
		net.WithMaxPeerCountAllow(localconf.ChainMakerConfig.NetConfig.MaxPeerCountAllow),
		net.WithPeerEliminationStrategy(localconf.ChainMakerConfig.NetConfig.PeerEliminationStrategy),
		net.WithSeeds(localconf.ChainMakerConfig.NetConfig.Seeds...),
		net.WithBlackAddresses(localconf.ChainMakerConfig.NetConfig.BlackList.Addresses...),
		net.WithBlackNodeIds(localconf.ChainMakerConfig.NetConfig.BlackList.NodeIds...),
		net.WithMsgCompression(localconf.ChainMakerConfig.DebugConfig.UseNetMsgCompression),
		net.WithInsecurity(localconf.ChainMakerConfig.DebugConfig.IsNetInsecurity),
	)
	if err != nil {
		errMsg := fmt.Sprintf("new net failed, %s", err.Error())
		log.Error(errMsg)
		return errors.New(errMsg)
	}

	// read tls cert, then set the NodeId of local config
	file, err := ioutil.ReadFile(certPath)
	if err != nil {
		return err
	}
	nodeId, err := helper.GetLibp2pPeerIdFromCert(file)
	if err != nil {
		return err
	}
	localconf.ChainMakerConfig.SetNodeId(nodeId)

	// load custom chain trust roots
	for _, chainTrustRoots := range localconf.ChainMakerConfig.NetConfig.CustomChainTrustRoots {
		for _, roots := range chainTrustRoots.TrustRoots {
			rootBytes, err := ioutil.ReadFile(roots.Root)
			if err != nil {
				log.Errorf("load custom chain trust roots failed, %s", err.Error())
				return err
			}
			err = server.net.AddTrustRoot(chainTrustRoots.ChainId, rootBytes)
			if err != nil {
				log.Errorf("add custom chain trust roots failed, %s", err.Error())
				return err
			}
		}
	}
	return nil
}

func (server *ChainMakerServer) initBlockchains() error {
	server.blockchains = sync.Map{}
	for _, chain := range localconf.ChainMakerConfig.GetBlockChains() {
		chainId := chain.ChainId
		if err := server.initBlockchain(chainId, chain.Genesis); err != nil {
			log.Error(err.Error())
			continue
		}
	}
	go server.newBlockchainTaskListener()
	return nil
}

func (server *ChainMakerServer) newBlockchainTaskListener() {
	for newChainId := range localconf.FindNewBlockChainNotifyC {
		_, ok := server.blockchains.Load(newChainId)
		if ok {
			log.Errorf("new block chain found existed(chain-id: %s)", newChainId)
			continue
		}
		log.Infof("new block chain found(chain-id: %s), start to init new block chain.", newChainId)
		for _, chain := range localconf.ChainMakerConfig.GetBlockChains() {
			chainId := chain.ChainId
			if chainId != newChainId {
				continue
			}
			if err := server.initBlockchain(chainId, chain.Genesis); err != nil {
				log.Error(err.Error())
				continue
			}
			newBlockchain, _ := server.blockchains.Load(newChainId)
			go startBlockchain(newBlockchain.(*Blockchain))

		}
	}
}

func (server *ChainMakerServer) initBlockchain(chainId, genesis string) error {
	if !filepath.IsAbs(genesis) {
		var err error
		genesis, err = filepath.Abs(genesis)
		if err != nil {
			return err
		}
	}
	log.Infof("load genesis file path of chain[%s]: %s", chainId, genesis)
	blockchain := NewBlockchain(genesis, chainId, msgbus.NewMessageBus(), server.net)

	if err := blockchain.Init(); err != nil {
		errMsg := fmt.Sprintf("init blockchain[%s] failed, %s", chainId, err.Error())
		return errors.New(errMsg)
	}
	server.blockchains.Store(chainId, blockchain)
	log.Infof("init blockchain[%s] success!", chainId)
	return nil
}

func startBlockchain(chain *Blockchain) {
	if err := chain.Start(); err != nil {
		log.Errorf("[Core] start blockchain[%s] failed, %s", chain.chainId, err.Error())
		os.Exit(-1)
	}
	log.Infof("[Core] start blockchain[%s] success", chain.chainId)
}

// Start ChainMakerServer.
func (server *ChainMakerServer) Start() error {
	// 1) start Net
	if err := server.net.Start(); err != nil {
		log.Errorf("[Net] start failed, %s", err.Error())
		return err
	}
	log.Infof("[Net] start success!")

	// 2) start blockchains
	server.blockchains.Range(func(_, value interface{}) bool {
		chain, _ := value.(*Blockchain)
		go startBlockchain(chain)
		return true
	})

	return nil
}

// Stop ChainMakerServer.
func (server *ChainMakerServer) Stop() {
	// stop all blockchains
	var wg sync.WaitGroup
	server.blockchains.Range(func(_, value interface{}) bool {
		chain, _ := value.(*Blockchain)
		wg.Add(1)
		go func(chain *Blockchain) {
			defer wg.Done()
			chain.Stop()
		}(chain)
		return false
	})
	wg.Wait()
	log.Info("ChainMaker server is stopped!")

	// stop net
	if err := server.net.Stop(); err != nil {
		log.Errorf("stop net failed, %s", err.Error())
	}
	log.Info("net is stopped!")

}

// AddTx add a transaction.
func (server *ChainMakerServer) AddTx(chainId string, tx *common.Transaction, source protocol.TxSource) error {
	if blockchain, ok := server.blockchains.Load(chainId); ok {
		return blockchain.(*Blockchain).txPool.AddTx(tx, source)
	}
	return fmt.Errorf(chainIdNotFoundErrorTemplate, chainId)
}

// GetStore get the store instance of chain which id is the given.
func (server *ChainMakerServer) GetStore(chainId string) (protocol.BlockchainStore, error) {
	if blockchain, ok := server.blockchains.Load(chainId); ok {
		return blockchain.(*Blockchain).store, nil
	}

	return nil, fmt.Errorf(chainIdNotFoundErrorTemplate, chainId)
}

// GetChainConf get protocol.ChainConf of chain which id is the given.
func (server *ChainMakerServer) GetChainConf(chainId string) (protocol.ChainConf, error) {
	if blockchain, ok := server.blockchains.Load(chainId); ok {
		return blockchain.(*Blockchain).chainConf, nil
	}

	return nil, fmt.Errorf(chainIdNotFoundErrorTemplate, chainId)
}

// GetAllChainConf get all protocol.ChainConf of all the chains.
func (server *ChainMakerServer) GetAllChainConf() ([]protocol.ChainConf, error) {
	var chainConfs []protocol.ChainConf
	server.blockchains.Range(func(_, value interface{}) bool {
		blockchain, _ := value.(*Blockchain)
		chainConfs = append(chainConfs, blockchain.chainConf)
		return true
	})

	if len(chainConfs) == 0 {
		return nil, fmt.Errorf("all chain not found")
	}

	return chainConfs, nil
}

// GetVmManager get protocol.VmManager of chain which id is the given.
func (server *ChainMakerServer) GetVmManager(chainId string) (protocol.VmManager, error) {
	if blockchain, ok := server.blockchains.Load(chainId); ok {
		return blockchain.(*Blockchain).vmMgr, nil
	}

	return nil, fmt.Errorf(chainIdNotFoundErrorTemplate, chainId)
}

// GetEventSubscribe get subscriber.EventSubscriber of chain which id is the given.
func (server *ChainMakerServer) GetEventSubscribe(chainId string) (*subscriber.EventSubscriber, error) {
	if blockchain, ok := server.blockchains.Load(chainId); ok {
		return blockchain.(*Blockchain).eventSubscriber, nil
	}

	return nil, fmt.Errorf(chainIdNotFoundErrorTemplate, chainId)
}

// GetNetService get protocol.NetService of chain which id is the given.
func (server *ChainMakerServer) GetNetService(chainId string) (protocol.NetService, error) {
	if blockchain, ok := server.blockchains.Load(chainId); ok {
		return blockchain.(*Blockchain).netService, nil
	}

	return nil, fmt.Errorf(chainIdNotFoundErrorTemplate, chainId)
}

// GetBlockchain get Blockchain of chain which id is the given.
func (server *ChainMakerServer) GetBlockchain(chainId string) (*Blockchain, error) {
	if blockchain, ok := server.blockchains.Load(chainId); ok {
		return blockchain.(*Blockchain), nil
	}

	return nil, fmt.Errorf(chainIdNotFoundErrorTemplate, chainId)
}

// GetAllAC get all protocol.AccessControlProvider of all the chains.
func (server *ChainMakerServer) GetAllAC() ([]protocol.AccessControlProvider, error) {
	var accessControls []protocol.AccessControlProvider
	server.blockchains.Range(func(_, value interface{}) bool {
		blockchain := value.(*Blockchain)
		accessControls = append(accessControls, blockchain.GetAccessControl())
		return true
	})

	if len(accessControls) == 0 {
		return nil, fmt.Errorf("all chain not found")
	}

	return accessControls, nil
}

// Version of chainmaker.
func (server *ChainMakerServer) Version() string {
	return fmt.Sprintf("%d", protocol.DefaultBlockVersion)
}
