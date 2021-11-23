/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package blockchain

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"chainmaker.org/chainmaker/store/v2"

	componentVm "chainmaker.org/chainmaker-go/vm"

	"chainmaker.org/chainmaker-go/accesscontrol"
	"chainmaker.org/chainmaker-go/consensus"
	"chainmaker.org/chainmaker-go/core"
	"chainmaker.org/chainmaker-go/core/cache"
	providerConf "chainmaker.org/chainmaker-go/core/provider/conf"
	"chainmaker.org/chainmaker-go/net"
	"chainmaker.org/chainmaker-go/snapshot"
	"chainmaker.org/chainmaker-go/subscriber"
	blockSync "chainmaker.org/chainmaker-go/sync"
	"chainmaker.org/chainmaker-go/txpool"
	"chainmaker.org/chainmaker/chainconf/v2"
	"chainmaker.org/chainmaker/common/v2/container"
	"chainmaker.org/chainmaker/localconf/v2"
	"chainmaker.org/chainmaker/logger/v2"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	consensusPb "chainmaker.org/chainmaker/pb-go/v2/consensus"
	storePb "chainmaker.org/chainmaker/pb-go/v2/store"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/store/v2/conf"
	"chainmaker.org/chainmaker/utils/v2"
	"chainmaker.org/chainmaker/vm/v2"
	"github.com/mitchellh/mapstructure"
)

const (
	//PREFIX_dpos_stake_nodeId the nodeId prefix in the dpos config in the chainconf
	PREFIX_dpos_stake_nodeId string = "stake.nodeID"
)

// Init all the modules.
func (bc *Blockchain) Init() (err error) {
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

	return bc.initExtModules(extModules)
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
	_, ok := bc.initModules[moduleNameNetService]
	if ok {
		bc.log.Infof("net service module existed, ignore.")
		return
	}
	var netServiceFactory net.NetServiceFactory
	if bc.netService, err = netServiceFactory.NewNetService(
		bc.net, bc.chainId, bc.ac, bc.chainConf, net.WithMsgBus(bc.msgBus)); err != nil {
		bc.log.Errorf("new net service failed, %s", err)
		return
	}
	bc.initModules[moduleNameNetService] = struct{}{}
	return
}

func (bc *Blockchain) initStore() (err error) {
	_, ok := bc.initModules[moduleNameStore]
	if ok {
		bc.log.Infof("store module existed, ignore.")
		return
	}
	var storeFactory store.Factory // nolint: typecheck
	storeLogger := logger.GetLoggerByChain(logger.MODULE_STORAGE, bc.chainId)
	err = container.Register(func() protocol.Logger { return storeLogger }, container.Name("store"))
	if err != nil {
		return err
	}
	config := &conf.StorageConfig{}
	err = mapstructure.Decode(localconf.ChainMakerConfig.StorageConfig, config)
	if err != nil {
		return err
	}

	//p11Handle, err := localconf.ChainMakerConfig.GetP11Handle()
	err = container.Register(localconf.ChainMakerConfig.GetP11Handle)
	if err != nil {
		return err
	}

	err = container.Register(storeFactory.NewStore,
		container.Parameters(map[int]interface{}{0: bc.chainId, 1: config}),
		container.DependsOn(map[int]string{2: "store"}),
		container.Name(bc.chainId))
	if err != nil {
		return err
	}
	err = container.Resolve(&bc.store, container.ResolveName(bc.chainId))
	if err != nil {
		bc.log.Errorf("new store failed, %s", err.Error())
		return err
	}
	bc.initModules[moduleNameStore] = struct{}{}
	return
}

func (bc *Blockchain) initChainConf() (err error) {
	_, ok := bc.initModules[moduleNameChainConf]
	if ok {
		bc.log.Infof("chain config module existed, ignore.")
		return
	}
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

	authType := strings.ToLower(bc.chainConf.ChainConfig().AuthType)
	if authType == "" {
		authType = protocol.PermissionedWithCert
	}

	if authType == protocol.Identity {
		authType = protocol.PermissionedWithCert
	}

	localAuthType := strings.ToLower(localconf.ChainMakerConfig.AuthType)

	if localAuthType == "" {
		localAuthType = protocol.PermissionedWithCert
	}

	if localAuthType == protocol.Identity {
		localAuthType = protocol.PermissionedWithCert
	}

	if authType != localAuthType {
		return fmt.Errorf("auth type of chain config mismatch the local config")
	}

	bc.chainNodeList, err = bc.chainConf.GetConsensusNodeIdList()
	if err != nil {
		bc.log.Errorf("load node list of chain config failed, %s", err)
		return err
	}
	bc.initModules[moduleNameChainConf] = struct{}{}

	// register myself as config watcher
	bc.chainConf.AddWatch(bc)
	//if localconf.ChainMakerConfig.StorageConfig.StateDbConfig.IsSqlDB() {
	//	panic("init chain conf fail. sql the future feature")
	//}
	return
}

func (bc *Blockchain) initCache() (err error) {
	_, ok := bc.initModules[moduleNameLedger]
	if ok {
		bc.log.Infof("ledger module existed, ignore.")
		return
	}
	// create genesis block
	// 1) if not exist on chain, create it
	// 2) if exist on chain, load the config in genesis
	// it will be changed to load the config in config transactions in the future
	bc.lastBlock, err = bc.store.GetLastBlock()
	if err != nil { //可能是全新数据库没有任何数据，而且还没创世，所以可能报错，不返回错误，继续进行创世操作即可
		bc.log.Infof("get last block failed, if it's a genesis block, ignore this error, %s", err.Error())
	}

	if bc.lastBlock != nil {
		bc.log.Infof(
			"get last block [chainId:%s]/[height:%d]/[blockhash:%s] success, no need to create genesis block",
			bc.lastBlock.GetHeader().ChainId, bc.lastBlock.GetHeader().BlockHeight,
			hex.EncodeToString(bc.lastBlock.GetHeader().BlockHash),
		)
	} else {
		chainConfig, err := chainconf.Genesis(bc.genesis)
		if err != nil {
			bc.log.Errorf("invoke chain config genesis failed, %s", err)
			return err
		}

		authType := strings.ToLower(chainConfig.AuthType)
		if authType == "" {
			authType = protocol.PermissionedWithCert
		}

		if authType == protocol.Identity {
			authType = protocol.PermissionedWithCert
		}

		localAuthType := strings.ToLower(localconf.ChainMakerConfig.AuthType)

		if localAuthType == "" {
			localAuthType = protocol.PermissionedWithCert
		}

		if localAuthType == protocol.Identity {
			localAuthType = protocol.PermissionedWithCert
		}

		if authType != localAuthType {
			return fmt.Errorf("auth type of chain config mismatch the local config")
		}

		genesisBlock, rwSetList, err := utils.CreateGenesis(chainConfig)
		if err != nil {
			return fmt.Errorf("create chain [%s] genesis failed, %s", bc.chainId, err.Error())
		}
		if err = bc.store.InitGenesis(
			&storePb.BlockWithRWSet{Block: genesisBlock, TxRWSets: rwSetList, ContractEvents: nil}); err != nil {
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
	bc.initModules[moduleNameLedger] = struct{}{}
	return nil
}

func (bc *Blockchain) initAC() (err error) {
	_, ok := bc.initModules[moduleNameAccessControl]
	if ok {
		bc.log.Infof("access control module existed, ignore.")
		return
	}
	// initialize access control: policy list and resource-policy mapping
	nodeConfig := localconf.ChainMakerConfig.NodeConfig
	//skFile := nodeConfig.PrivKeyFile
	//if !filepath.IsAbs(skFile) {
	//	skFile, err = filepath.Abs(skFile)
	//	if err != nil {
	//		return err
	//	}
	//}
	//certFile := nodeConfig.CertFile
	//if !filepath.IsAbs(certFile) {
	//	certFile, err = filepath.Abs(certFile)
	//	if err != nil {
	//		return err
	//	}
	//}
	acLog := logger.GetLoggerByChain(logger.MODULE_ACCESS, bc.chainId)
	//bc.ac, err = accesscontrol.NewAccessControlWithChainConfig(
	//	bc.chainConf, nodeConfig.OrgId, bc.store, acLog)
	//if err != nil {
	//	bc.log.Errorf("get organization information failed, %s", err.Error())
	//	return
	//}
	acFactory := accesscontrol.ACFactory()
	bc.ac, err = acFactory.NewACProvider(bc.chainConf, nodeConfig.OrgId, bc.store, acLog)
	if err != nil {
		bc.log.Errorf("new ac provider failed, %s", err.Error())
		return
	}

	switch bc.chainConf.ChainConfig().AuthType {
	case protocol.PermissionedWithCert, protocol.Identity:
		bc.identity, err = accesscontrol.InitCertSigningMember(bc.chainConf.ChainConfig(), nodeConfig.OrgId,
			nodeConfig.PrivKeyFile, nodeConfig.PrivKeyPassword, nodeConfig.CertFile)
		if err != nil {
			bc.log.Errorf("initialize identity failed, %s", err.Error())
			return
		}
	case protocol.PermissionedWithKey, protocol.Public:
		bc.identity, err = accesscontrol.InitPKSigningMember(bc.ac, nodeConfig.OrgId,
			nodeConfig.PrivKeyFile, nodeConfig.PrivKeyPassword)
		if err != nil {
			bc.log.Errorf("initialize identity failed, %s", err.Error())
			return
		}
	default:
		err = fmt.Errorf("auth type doesn't exist")
		bc.log.Errorf("initialize identity failed, %s", err.Error())
		return
	}

	bc.initModules[moduleNameAccessControl] = struct{}{}
	return
}

func (bc *Blockchain) initTxPool() (err error) {
	_, ok := bc.initModules[moduleNameTxPool]
	if ok {
		bc.log.Infof("tx pool module existed, ignore.")
		return
	}

	txPoolType := txpool.TypeDefault

	if value, ok := localconf.ChainMakerConfig.TxPoolConfig["pool_type"]; ok {
		txPoolType, _ = value.(string)
		txPoolType = strings.ToUpper(txPoolType)
	}

	txPoolLogger := logger.GetLoggerByChain(logger.MODULE_TXPOOL, bc.chainId)
	txPoolProvider := txpool.GetTxPoolProvider(txPoolType)
	if txPoolProvider == nil {
		return errors.New("get txPool provider failed, expected txPool not found")
	}

	currentTxPool, err := txPoolProvider(
		localconf.ChainMakerConfig.NodeConfig.NodeId,
		bc.chainId,
		bc.store,
		bc.msgBus,
		bc.chainConf,
		bc.ac,
		txPoolLogger,
		localconf.ChainMakerConfig.MonitorConfig.Enabled,
		localconf.ChainMakerConfig.TxPoolConfig,
	)

	if err != nil {
		bc.log.Errorf("new tx pool failed, %s", err)
		return err
	}

	bc.txPool = currentTxPool
	bc.initModules[moduleNameTxPool] = struct{}{}
	return nil
}

func (bc *Blockchain) initVM() (err error) {
	_, ok := bc.initModules[moduleNameVM]
	if ok {
		bc.log.Infof("vm module existed, ignore.")
		return
	}
	// init VM
	if bc.netService == nil {
		/*
			bc.vmMgr = vm.NewVmManager(
				wasmer.NewVmPoolManager(bc.chainId),
				&evm.InstancesManager{},
				&gasm.InstancesManager{},
				&wxvm.InstancesManager{},
				localconf.ChainMakerConfig.GetStorePath(),
				bc.ac, &soloChainNodesInfoProvider{},
				bc.chainConf,
			)
		*/

		/*
			bc.vmMgr = vm.NewVmManager(
				map[common.RuntimeType]protocol.VmInstancesManager{
					common.RuntimeType_GASM:   &gasm.InstancesManager{},
					common.RuntimeType_WXVM:   &wxvm.InstancesManager{},
					common.RuntimeType_EVM:    &evm.InstancesManager{},
					common.RuntimeType_WASMER: &wasmer.InstancesManager{},
				},
				localconf.ChainMakerConfig.GetStorePath(),
				bc.ac,
				&soloChainNodesInfoProvider{},
				bc.chainConf,
			)
		*/

		chainConfig, err := chainconf.Genesis(bc.genesis)
		if err != nil {
			bc.log.Errorf("invoke chain config genesis failed, %s", err)
			return err
		}

		supportedVmManagerList := make(map[common.RuntimeType]protocol.VmInstancesManager)

		for _, vmType := range chainConfig.Vm.SupportList {
			vmInstancesManagerProvider := componentVm.GetVmProvider(vmType)
			vmInstancesManager, err := vmInstancesManagerProvider(bc.chainId, localconf.ChainMakerConfig.VMConfig)
			if err != nil {
				bc.log.Errorf("create instance manager failed, %v", err)
			}
			supportedVmManagerList[componentVm.VmTypeToRunTimeType[strings.ToUpper(vmType)]] = vmInstancesManager
		}

		bc.vmMgr = vm.NewVmManager(
			supportedVmManagerList,
			localconf.ChainMakerConfig.GetStorePath(),
			bc.ac,
			&soloChainNodesInfoProvider{},
			bc.chainConf,
		)
	} else {
		/*
			bc.vmMgr = vm.NewVmManager(
				wasmer.NewVmPoolManager(bc.chainId),
				&evm.InstancesManager{},
				&gasm.InstancesManager{},
				&wxvm.InstancesManager{},
				localconf.ChainMakerConfig.GetStorePath(),
				bc.ac,
				bc.netService.GetChainNodesInfoProvider(),
				bc.chainConf,
			)
		*/

		/*
			bc.vmMgr = vm.NewVmManager(
				map[common.RuntimeType]protocol.VmInstancesManager{
					common.RuntimeType_GASM: &gasm.InstancesManager{},
					common.RuntimeType_WXVM: &wxvm.InstancesManager{},
					common.RuntimeType_EVM:  &evm.InstancesManager{},
				},
				localconf.ChainMakerConfig.GetStorePath(),
				bc.ac,
				bc.netService.GetChainNodesInfoProvider(),
				bc.chainConf,
			)
		*/

		chainConfig, err := chainconf.Genesis(bc.genesis)
		if err != nil {
			bc.log.Errorf("invoke chain config genesis failed, %s", err)
			return err
		}

		supportedVmManagerList := make(map[common.RuntimeType]protocol.VmInstancesManager)

		for _, vmType := range chainConfig.Vm.SupportList {
			vmInstancesManagerProvider := componentVm.GetVmProvider(vmType)
			vmInstancesManager, err := vmInstancesManagerProvider(bc.chainId, localconf.ChainMakerConfig.VMConfig)
			if err != nil {
				bc.log.Errorf("create instance manager failed, %v", err)
			}
			supportedVmManagerList[componentVm.VmTypeToRunTimeType[strings.ToUpper(vmType)]] = vmInstancesManager
		}

		bc.vmMgr = vm.NewVmManager(
			supportedVmManagerList,
			localconf.ChainMakerConfig.GetStorePath(),
			bc.ac,
			bc.netService.GetChainNodesInfoProvider(),
			bc.chainConf,
		)
	}
	bc.initModules[moduleNameVM] = struct{}{}
	return
}

type soloChainNodesInfoProvider struct{}

func (s *soloChainNodesInfoProvider) GetChainNodesInfo() ([]*protocol.ChainNodeInfo, error) {
	return []*protocol.ChainNodeInfo{}, nil
}

func (bc *Blockchain) initCore() (err error) {
	_, ok := bc.initModules[moduleNameCore]
	if ok {
		bc.log.Infof("core engine module existed, ignore.")
		return
	}
	// create snapshot manager
	var snapshotFactory snapshot.Factory
	if bc.chainConf.ChainConfig().Snapshot != nil && bc.chainConf.ChainConfig().Snapshot.EnableEvidence {
		bc.snapshotManager = snapshotFactory.NewSnapshotEvidenceMgr(bc.store)
	} else {
		bc.snapshotManager = snapshotFactory.NewSnapshotManager(bc.store)
	}

	// init coreEngine module
	coreEngineConfig := &providerConf.CoreEngineConfig{
		ChainId:         bc.chainId,
		TxPool:          bc.txPool,
		SnapshotManager: bc.snapshotManager,
		MsgBus:          bc.msgBus,
		Identity:        bc.identity,
		LedgerCache:     bc.ledgerCache,
		ChainConf:       bc.chainConf,
		AC:              bc.ac,
		BlockchainStore: bc.store,
		Log:             logger.GetLoggerByChain(logger.MODULE_CORE, bc.chainId),
		VmMgr:           bc.vmMgr,
		ProposalCache:   bc.proposalCache,
		Subscriber:      bc.eventSubscriber,
	}

	coreEngineFactory := core.Factory()
	bc.coreEngine, err = coreEngineFactory.NewConsensusEngine(bc.getConsensusType().String(), coreEngineConfig)
	if err != nil {
		bc.log.Errorf("new core engine failed, %s", err.Error())
		return err
	}
	bc.initModules[moduleNameCore] = struct{}{}
	return
}

func (bc *Blockchain) initConsensus() (err error) {
	// init consensus module
	var consensusFactory consensus.Factory
	id := localconf.ChainMakerConfig.NodeConfig.NodeId
	nodes := bc.chainConf.ChainConfig().Consensus.Nodes
	nodeIds := make([]string, len(nodes))
	isConsensusNode := false
	for i, node := range nodes {
		for _, nid := range node.NodeId {
			nodeIds[i] = nid
			if nid == id {
				isConsensusNode = true
			}
		}
	}
	if !isConsensusNode {
		// this node is not a consensus node
		delete(bc.initModules, moduleNameConsensus)
		return nil
	}
	_, ok := bc.initModules[moduleNameConsensus]
	if ok {
		bc.log.Infof("consensus module existed, ignore.")
		return
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
		bc.coreEngine.GetBlockVerifier(),
		bc.coreEngine.GetBlockCommitter(),
		bc.netService,
		bc.msgBus,
		bc.chainConf,
		bc.store,
		bc.coreEngine.GetHotStuffHelper(),
	)
	if err != nil {
		bc.log.Errorf("new consensus engine failed, %s", err)
		return err
	}
	bc.initModules[moduleNameConsensus] = struct{}{}
	return
}

func (bc *Blockchain) initSync() (err error) {
	_, ok := bc.initModules[moduleNameSync]
	if ok {
		bc.log.Infof("sync module existed, ignore.")
		return
	}
	// init sync service module
	bc.syncServer = blockSync.NewBlockChainSyncServer(
		bc.chainId,
		bc.netService,
		bc.msgBus,
		bc.store,
		bc.ledgerCache,
		bc.coreEngine.GetBlockVerifier(),
		bc.coreEngine.GetBlockCommitter(),
	)
	bc.initModules[moduleNameSync] = struct{}{}
	return
}

func (bc *Blockchain) initSubscriber() error {
	_, ok := bc.initModules[moduleNameSubscriber]
	if ok {
		bc.log.Infof("subscriber module existed, ignore.")
		return nil
	}
	bc.eventSubscriber = subscriber.NewSubscriber(bc.msgBus)
	bc.initModules[moduleNameSubscriber] = struct{}{}
	return nil
}

func (bc *Blockchain) isModuleInit(moduleName string) bool {
	_, ok := bc.initModules[moduleName]
	return ok
}
