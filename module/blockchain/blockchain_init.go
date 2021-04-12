/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package blockchain

import (
	"chainmaker.org/chainmaker-go/accesscontrol"
	"chainmaker.org/chainmaker-go/chainconf"
	"chainmaker.org/chainmaker-go/common/helper"
	"chainmaker.org/chainmaker-go/consensus"
	"chainmaker.org/chainmaker-go/core"
	"chainmaker.org/chainmaker-go/core/cache"
	"chainmaker.org/chainmaker-go/localconf"
	"chainmaker.org/chainmaker-go/net"
	consensusPb "chainmaker.org/chainmaker-go/pb/protogo/consensus"
	storePb "chainmaker.org/chainmaker-go/pb/protogo/store"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/snapshot"
	"chainmaker.org/chainmaker-go/store"
	"chainmaker.org/chainmaker-go/subscriber"
	blockSync "chainmaker.org/chainmaker-go/sync"
	"chainmaker.org/chainmaker-go/txpool"
	"chainmaker.org/chainmaker-go/utils"
	"chainmaker.org/chainmaker-go/vm"
	"encoding/hex"
	"fmt"
	"path/filepath"
)

// Init all the modules.
func (bc *Blockchain) Init() (err error) {
	var (
		moduleNameSubscriber    = "Subscriber"
		moduleNameStore         = "Store"
		moduleNameLedger        = "Ledger"
		moduleNameChainConf     = "ChainConf"
		moduleNameAccessControl = "AccessControl"
		moduleNameNetService    = "NetService"
		moduleNameVM            = "VM"
		moduleNameTxPool        = "TxPool"
		moduleNameCore          = "Core"
		moduleNameConsensus     = "Consensus"
		moduleNameSync          = "Sync"
	)

	baseModules := []map[string]func() error{
		// init Subscriber
		{moduleNameSubscriber: bc.initSubscriber},
		// init store module
		{moduleNameStore: bc.initStore},
		// init ledger module
		{moduleNameLedger: bc.initCache},
		// init chain config , must latter than store module
		{moduleNameChainConf: bc.initChainConf},
	}

	if err := bc.initBaseModules(baseModules); err != nil {
		return err
	}

	var extModules []map[string]func() error

	if bc.getConsensusType() == consensusPb.ConsensusType_SOLO {
		// solo
		extModules = []map[string]func() error{
			// init access control
			{moduleNameAccessControl: bc.initAC},
			// init vm instances and module
			{moduleNameVM: bc.initVM},

			// init transaction pool
			{moduleNameTxPool: bc.initTxPool},
			// init core engine
			{moduleNameCore: bc.initCore},
			// init consensus module
			{moduleNameConsensus: bc.initConsensus},
		}
	} else {
		// not solo
		extModules = []map[string]func() error{
			// init access control
			{moduleNameAccessControl: bc.initAC},
			// init net service
			{moduleNameNetService: bc.initNetService},
			// init vm instances and module
			{moduleNameVM: bc.initVM},

			// init transaction pool
			{moduleNameTxPool: bc.initTxPool},
			// init core engine
			{moduleNameCore: bc.initCore},
			// init consensus module
			{moduleNameConsensus: bc.initConsensus},
			// init sync service module
			{moduleNameSync: bc.initSync},
		}
	}

	bc.log.Debug("start to init blockchain ...")

	if err := bc.initExtModules(extModules); err != nil {
		return err
	}

	return nil
}

func (bc *Blockchain) initBaseModules(baseModules []map[string]func() error) (err error) {
	moduleNum := len(baseModules)
	for idx, baseModule := range baseModules {
		for name, initFunc := range baseModule {
			if err := initFunc(); err != nil {
				bc.log.Errorf("init module[%s] failed, %s", name, err)
				return err
			}
			bc.log.Infof("BASE INIT STEP (%d/%d) => init base[%s] success :)", idx+1, moduleNum, name)
		}
	}
	return
}

func (bc *Blockchain) initExtModules(extModules []map[string]func() error) (err error) {
	moduleNum := len(extModules)
	for idx, initModule := range extModules {
		for name, initFunc := range initModule {
			if err := initFunc(); err != nil {
				bc.log.Errorf("init module[%s] failed, %s", name, err)
				return err
			}
			bc.log.Infof("MODULE INIT STEP (%d/%d) => init module[%s] success :)", idx+1, moduleNum, name)
		}
	}
	return
}

func (bc *Blockchain) initNetService() (err error) {
	var netServiceFactory net.NetServiceFactory
	if bc.netService, err = netServiceFactory.NewNetService(bc.net, bc.chainId, bc.ac, bc.chainConf, net.WithMsgBus(bc.msgBus)); err != nil {
		bc.log.Errorf("new net service failed, %s", err)
		return
	}
	return
}

func (bc *Blockchain) initStore() (err error) {
	var storeFactory store.Factory
	if bc.store, err = storeFactory.NewStore(bc.chainId, &localconf.ChainMakerConfig.StorageConfig); err != nil {
		bc.log.Errorf("new store failed, %s", err.Error())
		return err
	}
	return
}

func (bc *Blockchain) initChainConf() (err error) {
	bc.chainConf, err = chainconf.NewChainConf(
		chainconf.WithChainId(bc.chainId),
		chainconf.WithMsgBus(bc.msgBus),
		chainconf.WithBlockchainStore(bc.store),
	)
	if err != nil {
		bc.log.Errorf("new chain config failed, %s", err.Error())
		return err
	}
	err = bc.chainConf.Init()
	if err != nil {
		bc.log.Errorf("init chain config failed, %s", err)
		return err
	}
	bc.chainNodeList, err = bc.chainConf.GetConsensusNodeIdList()
	if err != nil {
		bc.log.Errorf("load node list of chain config failed, %s", err)
		return err
	}
	return
}

func (bc *Blockchain) initCache() error {
	var err error
	// create genesis block
	// 1) if not exist on chain, create it
	// 2) if exist on chain, load the config in genesis, it will be changed to load the config in config transactions in the future
	bc.lastBlock, err = bc.store.GetLastBlock()
	if err != nil { //可能是全新数据库没有任何数据，而且还没创世，所以可能报错，不返回错误，继续进行创世操作即可
		bc.log.Infof("get last block failed, %s", err.Error())
	}

	if bc.lastBlock != nil {
		bc.log.Infof("get last block [chainId:%s]/[height:%d]/[blockhash:%s] success, no need to create genesis block",
			bc.lastBlock.GetHeader().ChainId, bc.lastBlock.GetHeader().BlockHeight, hex.EncodeToString(bc.lastBlock.GetHeader().BlockHash))
	} else {
		chainConfig, err := chainconf.Genesis(bc.genesis)
		if err != nil {
			bc.log.Errorf("invoke chain config genesis failed, %s", err)
			return err
		}
		genesisBlock, rwSetList, err := utils.CreateGenesis(chainConfig)
		if err != nil {
			return fmt.Errorf("create chain [%s] genesis failed, %s", bc.chainId, err.Error())
		}
		if err = bc.store.InitGenesis(&storePb.BlockWithRWSet{genesisBlock, rwSetList}); err != nil {
			return fmt.Errorf("put chain[%s] genesis block failed, %s", bc.chainId, err.Error())
		}

		bc.lastBlock = genesisBlock
	}

	//// load chain config with genesis block info
	//if err := ledger.ChainConfigBlock2CMConf(*cc, genesisBlock); err != nil {
	//	return fmt.Errorf("chainConfigBlock2CMConf failed, %s", err.Error())
	//}

	// cache the lasted config block
	bc.ledgerCache = cache.NewLedgerCache(bc.chainId)
	bc.ledgerCache.SetLastCommittedBlock(bc.lastBlock)
	bc.proposalCache = cache.NewProposalCache(bc.chainConf, bc.ledgerCache)
	bc.log.Debugf("go last block: %+v", bc.lastBlock)
	return nil
}

func (bc *Blockchain) initAC() (err error) {
	// initialize access control: policy list and resource-policy mapping
	nodeConfig := localconf.ChainMakerConfig.NodeConfig
	skFile := nodeConfig.PrivKeyFile
	if !filepath.IsAbs(skFile) {
		skFile, err = filepath.Abs(skFile)
		if err != nil {
			return err
		}
	}
	certFile := nodeConfig.CertFile
	if !filepath.IsAbs(certFile) {
		certFile, err = filepath.Abs(certFile)
		if err != nil {
			return err
		}
	}

	bc.ac, err = accesscontrol.NewAccessControlWithChainConfig(skFile, nodeConfig.PrivKeyPassword, certFile, bc.chainConf, nodeConfig.OrgId, bc.store)
	if err != nil {
		bc.log.Errorf("get organization information failed, %s", err.Error())
		return
	}

	bc.identity = bc.ac.GetLocalSigningMember()
	return
}

func (bc *Blockchain) initTxPool() (err error) {
	// init transaction pool
	var (
		txPoolFactory txpool.TxPoolFactory
		txType        = txpool.SINGLE
	)
	if localconf.ChainMakerConfig.DebugConfig.UseBatchTxPool {
		txType = txpool.BATCH
	}
	bc.txPool, err = txPoolFactory.NewTxPool(
		txType,
		txpool.WithNodeId(localconf.ChainMakerConfig.NodeConfig.NodeId),
		txpool.WithMsgBus(bc.msgBus),
		txpool.WithChainId(bc.chainId),
		txpool.WithNetService(bc.netService),
		txpool.WithBlockchainStore(bc.store),
		txpool.WithSigner(bc.identity),
		txpool.WithChainConf(bc.chainConf),
		txpool.WithAccessControl(bc.ac),
	)
	if err != nil {
		bc.log.Errorf("new tx pool failed, %s", err)
		return err
	}
	return nil
}

func (bc *Blockchain) initVM() (err error) {
	// init VM
	var snapshotFactory snapshot.Factory
	var vmFactory vm.Factory
	bc.snapshotManager = snapshotFactory.NewSnapshotManager(bc.store)
	if bc.netService == nil {
		bc.vmMgr = vmFactory.NewVmManager(localconf.ChainMakerConfig.StorageConfig.StorePath, bc.snapshotManager, bc.chainId, bc.ac, nil)
	} else {
		bc.vmMgr = vmFactory.NewVmManager(localconf.ChainMakerConfig.StorageConfig.StorePath, bc.snapshotManager, bc.chainId, bc.ac, bc.netService.GetChainNodesInfoProvider())
	}
	return
}

func (bc *Blockchain) initCore() (err error) {
	// init core engine
	var coreFactory core.CoreFactory
	bc.coreEngine, err = coreFactory.NewCoreWithOptions(
		core.WithMsgBus(bc.msgBus),
		core.WithTxPool(bc.txPool),
		core.WithSnapshotManager(bc.snapshotManager),
		core.WithBlockchainStore(bc.store),
		core.WithVmMgr(bc.vmMgr),
		core.WithSigningMember(bc.identity),
		core.WithLedgerCache(bc.ledgerCache),
		core.WithChainId(bc.chainId),
		core.WithChainConf(bc.chainConf),
		core.WithAccessControl(bc.ac),
		core.WithSubscriber(bc.eventSubscriber),
		core.WithProposalCache(bc.proposalCache),
	)
	if err != nil {
		bc.log.Errorf("new core engine failed, %s", err.Error())
		return err
	}
	return
}

func (bc *Blockchain) initConsensus() (err error) {
	// init consensus module
	var consensusFactory consensus.Factory
	id := localconf.ChainMakerConfig.NodeConfig.NodeId
	nodes := bc.chainConf.ChainConfig().Consensus.Nodes
	nodeIds := make([]string, len(nodes))
	for i, node := range nodes {
		for _, addr := range node.Address {
			uid, err := helper.GetNodeUidFromAddr(addr)
			if err != nil {
				return err
			}
			nodeIds[i] = uid
		}
	}
	dbHandle := bc.store.GetDBHandle(protocol.ConsensusDBName)
	bc.consensus, err = consensusFactory.NewConsensusEngine(
		bc.getConsensusType(),
		bc.chainId,
		id,
		nodeIds,
		bc.identity,
		bc.ac,
		dbHandle,
		bc.ledgerCache,
		bc.proposalCache,
		bc.coreEngine.BlockVerifier,
		bc.coreEngine.BlockCommitter,
		bc.netService,
		bc.msgBus,
		bc.chainConf,
		bc.store)
	if err != nil {
		bc.log.Errorf("new consensus engine failed, %s", err)
		return err
	}
	return
}

func (bc *Blockchain) initSync() (err error) {
	// init sync service module
	bc.syncServer = blockSync.NewBlockChainSyncServer(
		bc.chainId,
		bc.netService,
		bc.msgBus,
		bc.store,
		bc.ledgerCache,
		bc.coreEngine.BlockVerifier,
		bc.coreEngine.BlockCommitter,
	)

	return
}

func (bc *Blockchain) initSubscriber() error {
	bc.eventSubscriber = subscriber.NewSubscriber(bc.msgBus)
	return nil
}
