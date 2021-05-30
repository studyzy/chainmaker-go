/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package localconf

import (
	"chainmaker.org/chainmaker-go/logger"
)

type nodeConfig struct {
	Type            string       `mapstructure:"type"`
	CertFile        string       `mapstructure:"cert_file"`
	PrivKeyFile     string       `mapstructure:"priv_key_file"`
	PrivKeyPassword string       `mapstructure:"priv_key_password"`
	AuthType        string       `mapstructure:"auth_type"`
	P11Config       pkcs11Config `mapstructure:"pkcs11"`
	NodeId          string       `mapstructure:"node_id"`
	OrgId           string       `mapstructure:"org_id"`
	SignerCacheSize int          `mapstructure:"signer_cache_size"`
	CertCacheSize   int          `mapstructure:"cert_cache_size"`
}

type netConfig struct {
	Provider                string            `mapstructure:"provider"`
	ListenAddr              string            `mapstructure:"listen_addr"`
	PeerStreamPoolSize      int               `mapstructure:"peer_stream_pool_size"`
	MaxPeerCountAllow       int               `mapstructure:"max_peer_count_allow"`
	PeerEliminationStrategy int               `mapstructure:"peer_elimination_strategy"`
	Seeds                   []string          `mapstructure:"seeds"`
	TLSConfig               netTlsConfig      `mapstructure:"tls"`
	BlackList               blackList         `mapstructure:"blacklist"`
	CustomChainTrustRoots   []chainTrustRoots `mapstructure:"custom_chain_trust_roots"`
}

type netTlsConfig struct {
	Enabled     bool   `mapstructure:"enabled"`
	PrivKeyFile string `mapstructure:"priv_key_file"`
	CertFile    string `mapstructure:"cert_file"`
}

type pkcs11Config struct {
	Enabled          bool   `mapstructure:"enabled"`
	Library          string `mapstructure:"library"`
	Label            string `mapstructure:"label"`
	Password         string `mapstructure:"password"`
	SessionCacheSize int    `mapstructure:"session_cache_size"`
	Hash             string `mapstructure:"hash"`
}

type blackList struct {
	Addresses []string `mapstructure:"addresses"`
	NodeIds   []string `mapstructure:"node_ids"`
}

type chainTrustRoots struct {
	ChainId    string       `mapstructure:"chain_id"`
	TrustRoots []trustRoots `mapstructure:"trust_roots"`
}

type trustRoots struct {
	OrgId string `mapstructure:"org_id"`
	Root  string `mapstructure:"root"`
}

type rpcConfig struct {
	Provider                               string           `mapstructure:"provider"`
	Port                                   int              `mapstructure:"port"`
	TLSConfig                              tlsConfig        `mapstructure:"tls"`
	RateLimitConfig                        rateLimitConfig  `mapstructure:"ratelimit"`
	SubscriberConfig                       subscriberConfig `mapstructure:"subscriber"`
	CheckChainConfTrustRootsChangeInterval int              `mapstructure:"check_chain_conf_trust_roots_change_interval"`
}

type tlsConfig struct {
	Mode                  string `mapstructure:"mode"`
	PrivKeyFile           string `mapstructure:"priv_key_file"`
	CertFile              string `mapstructure:"cert_file"`
	TestClientPrivKeyFile string `mapstructure:"test_client_priv_key_file"`
	TestClientCertFile    string `mapstructure:"test_client_cert_file"`
}

type rateLimitConfig struct {
	TokenPerSecond  int `mapstructure:"token_per_second"`
	TokenBucketSize int `mapstructure:"token_bucket_size"`
}

type subscriberConfig struct {
	RateLimitConfig rateLimitConfig `mapstructure:"ratelimit"`
}

type debugConfig struct {
	IsCliOpen           bool `mapstructure:"is_cli_open"`
	IsHttpOpen          bool `mapstructure:"is_http_open"`
	IsProposer          bool `mapstructure:"is_proposer"`
	IsNotRWSetCheck     bool `mapstructure:"is_not_rwset_check"`
	IsConcurPropose     bool `mapstructure:"is_concur_propose"`
	IsConcurVerify      bool `mapstructure:"is_concur_verify"`
	IsSolo              bool `mapstructure:"is_solo"`
	IsHaltPropose       bool `mapstructure:"is_halt_propose"`
	IsSkipAccessControl bool `mapstructure:"is_skip_access_control"` // true: minimize access control; false: use full access control
	IsTraceMemoryUsage  bool `mapstructure:"is_trace_memory_usage"`  // true for trace memory usage information periodically

	IsProposeDuplicately          bool `mapstructure:"is_propose_duplicately"`           // Simulate a node which would propose duplicate after it has proposed Proposal
	IsProposeMultiNodeDuplicately bool `mapstructure:"is_propose_multinode_duplicately"` // Simulate a malicious node which would propose duplicate proposals
	IsProposalOldHeight           bool `mapstructure:"is_proposal_old_height"`
	IsPrevoteDuplicately          bool `mapstructure:"is_prevote_duplicately"`   // Simulate a malicious node which would prevote duplicately
	IsPrevoteOldHeight            bool `mapstructure:"is_prevote_old_height"`    // Simulate a malicious node which would prevote for oldheight
	IsPrevoteLost                 bool `mapstructure:"is_prevote_lost"`          //prevote vote lost
	IsPrecommitDuplicately        bool `mapstructure:"is_precommit_duplicately"` //Simulate a malicious node which would propose duplicate precommits
	IsPrecommitOldHeight          bool `mapstructure:"is_precommit_old_height"`  // Simulate a malicious node which would Precommit a lower height than current height

	IsProposeLost    bool `mapstructure:"is_propose_lost"`     //proposal vote lost
	IsProposeDelay   bool `mapstructure:"is_propose_delay"`    //proposal lost
	IsPrevoteDelay   bool `mapstructure:"is_prevote_delay"`    //network problem resulting in preovote lost
	IsPrecommitLost  bool `mapstructure:"is_precommit_lost"`   //precommit vote lost
	IsPrecommitDelay bool `mapstructure:"is_prevcommit_delay"` //network problem resulting in precommit lost

	IsCommitWithoutPublish bool `mapstructure:"is_commit_without_publish"` //if the node committing block without publishing, TRUE；else, FALSE
	IsPrevoteInvalid       bool `mapstructure:"is_prevote_invalid"`        //simulate a node which sends an invalid prevote(hash=nil)
	IsPrecommitInvalid     bool `mapstructure:"is_precommit_invalid"`      //simulate a node which sends an invalid precommit(hash=nil)

	IsModifyTxPayload    bool `mapstructure:"is_modify_tx_payload"`
	IsExtreme            bool `mapstructure:"is_extreme"` //extreme fast mode
	UseNetMsgCompression bool `mapstructure:"use_net_msg_compression"`
	IsNetInsecurity      bool `mapstructure:"is_net_insecurity"`
}

type blockchainConfig struct {
	ChainId string
	Genesis string
}

type StorageConfig struct {
	//默认的Leveldb配置，如果每个DB有不同的设置，可以在自己的DB中进行设置
	StorePath            string `mapstructure:"store_path"`
	DbPrefix             string `mapstructure:"db_prefix"`
	WriteBufferSize      int    `mapstructure:"write_buffer_size"`
	BloomFilterBits      int    `mapstructure:"bloom_filter_bits"`
	BlockWriteBufferSize int    `mapstructure:"block_write_buffer_size"`
	//数据库模式：light只存区块头,normal存储区块头和交易以及生成的State,full存储了区块头、交易、状态和交易收据（读写集、日志等）
	//Mode string `mapstructure:"mode"`
	DisableHistoryDB       bool      `mapstructure:"disable_historydb"`
	DisableResultDB        bool      `mapstructure:"disable_resultdb"`
	DisableContractEventDB bool      `mapstructure:"disable_contract_eventdb"`
	LogDBWriteAsync        bool      `mapstructure:"logdb_write_async"`
	BlockDbConfig          *DbConfig `mapstructure:"blockdb_config"`
	StateDbConfig          *DbConfig `mapstructure:"statedb_config"`
	HistoryDbConfig        *DbConfig `mapstructure:"historydb_config"`
	ResultDbConfig         *DbConfig `mapstructure:"resultdb_config"`
	ContractEventDbConfig  *DbConfig `mapstructure:"contract_eventdb_config"`
}

func (config *StorageConfig) setDefault() {
	if config.DbPrefix != "" {
		if config.BlockDbConfig != nil && config.BlockDbConfig.SqlDbConfig != nil && config.BlockDbConfig.SqlDbConfig.DbPrefix == "" {
			config.BlockDbConfig.SqlDbConfig.DbPrefix = config.DbPrefix
		}
		if config.StateDbConfig != nil && config.StateDbConfig.SqlDbConfig != nil && config.StateDbConfig.SqlDbConfig.DbPrefix == "" {
			config.StateDbConfig.SqlDbConfig.DbPrefix = config.DbPrefix
		}
		if config.HistoryDbConfig != nil && config.HistoryDbConfig.SqlDbConfig != nil && config.HistoryDbConfig.SqlDbConfig.DbPrefix == "" {
			config.HistoryDbConfig.SqlDbConfig.DbPrefix = config.DbPrefix
		}
		if config.ResultDbConfig != nil && config.ResultDbConfig.SqlDbConfig != nil && config.ResultDbConfig.SqlDbConfig.DbPrefix == "" {
			config.ResultDbConfig.SqlDbConfig.DbPrefix = config.DbPrefix
		}
		if config.ContractEventDbConfig != nil && config.ContractEventDbConfig.SqlDbConfig != nil && config.ContractEventDbConfig.SqlDbConfig.DbPrefix == "" {
			config.ContractEventDbConfig.SqlDbConfig.DbPrefix = config.DbPrefix
		}
	}
}
func (config *StorageConfig) GetBlockDbConfig() *DbConfig {
	if config.BlockDbConfig == nil {
		return config.GetDefaultDBConfig()
	}
	config.setDefault()
	return config.BlockDbConfig
}
func (config *StorageConfig) GetStateDbConfig() *DbConfig {
	if config.StateDbConfig == nil {
		return config.GetDefaultDBConfig()
	}
	config.setDefault()
	return config.StateDbConfig
}
func (config *StorageConfig) GetHistoryDbConfig() *DbConfig {
	if config.HistoryDbConfig == nil {
		return config.GetDefaultDBConfig()
	}
	config.setDefault()
	return config.HistoryDbConfig
}
func (config *StorageConfig) GetResultDbConfig() *DbConfig {
	if config.ResultDbConfig == nil {
		return config.GetDefaultDBConfig()
	}
	config.setDefault()
	return config.ResultDbConfig
}
func (config *StorageConfig) GetContractEventDbConfig() *DbConfig {
	if config.ContractEventDbConfig == nil {
		return config.GetDefaultDBConfig()
	}
	config.setDefault()
	return config.ContractEventDbConfig
}
func (config *StorageConfig) GetDefaultDBConfig() *DbConfig {
	lconfig := &LevelDbConfig{
		StorePath:            config.StorePath,
		WriteBufferSize:      config.WriteBufferSize,
		BloomFilterBits:      config.BloomFilterBits,
		BlockWriteBufferSize: config.WriteBufferSize,
	}
	return &DbConfig{
		Provider:      "leveldb",
		LevelDbConfig: lconfig,
	}
}

//根据配置的DisableDB的情况，确定当前配置活跃的数据库数量
func (config *StorageConfig) GetActiveDBCount() int {
	count := 5
	if config.DisableContractEventDB {
		count--
	}
	if config.DisableHistoryDB {
		count--
	}
	if config.DisableResultDB {
		count--
	}
	return count
}

type DbConfig struct {
	//leveldb,rocksdb,sql
	Provider      string         `mapstructure:"provider"`
	LevelDbConfig *LevelDbConfig `mapstructure:"leveldb_config"`
	SqlDbConfig   *SqlDbConfig   `mapstructure:"sqldb_config"`
}

func (dbc *DbConfig) IsKVDB() bool {
	return dbc.Provider == "leveldb" || dbc.Provider == "rocksdb"
}
func (dbc *DbConfig) IsSqlDB() bool {
	return dbc.Provider == "sql" || dbc.Provider == "mysql" || dbc.Provider == "rdbms"
}

type LevelDbConfig struct {
	StorePath            string `mapstructure:"store_path"`
	WriteBufferSize      int    `mapstructure:"write_buffer_size"`
	BloomFilterBits      int    `mapstructure:"bloom_filter_bits"`
	BlockWriteBufferSize int    `mapstructure:"block_write_buffer_size"`
}
type SqlDbConfig struct {
	//mysql, sqlite, postgres, sqlserver
	SqlDbType       string `mapstructure:"sqldb_type"`
	Dsn             string `mapstructure:"dsn"`
	MaxIdleConns    int    `mapstructure:"max_idle_conns"`
	MaxOpenConns    int    `mapstructure:"max_open_conns"`
	ConnMaxLifeTime int    `mapstructure:"conn_max_lifetime"` //second
	SqlLogMode      string `mapstructure:"sqllog_mode"`       //Silent,Error,Warn,Info
	SqlVerifier     string `mapstructure:"sql_verifier"`      //simple,safe
	DbPrefix        string `mapstructure:"db_prefix"`
}

type txPoolConfig struct {
	PoolType            string `mapstructure:"pool_type"`
	MaxTxPoolSize       uint32 `mapstructure:"max_txpool_size"`
	MaxConfigTxPoolSize uint32 `mapstructure:"max_config_txpool_size"`
	IsMetrics           bool   `mapstructure:"is_metrics"`
	Performance         bool   `mapstructure:"performance"`
	BatchMaxSize        int    `mapstructure:"batch_max_size"`
	BatchCreateTimeout  int64  `mapstructure:"batch_create_timeout"`
	CacheFlushTicker    int64  `mapstructure:"cache_flush_ticker"`
	CacheThresholdCount int64  `mapstructure:"cache_threshold_count"`
	CacheFlushTimeOut   int64  `mapstructure:"cache_flush_timeout"`
	AddTxChannelSize    int64  `mapstructure:"add_tx_channel_size"`
}

type syncConfig struct {
	BroadcastTime             uint32  `mapstructure:"broadcast_time"`
	BlockPoolSize             uint32  `mapstructure:"block_pool_size"`
	WaitTimeOfBlockRequestMsg uint32  `mapstructure:"wait_time_requested"`
	BatchSizeFromOneNode      uint32  `mapstructure:"batch_Size_from_one_node"`
	ProcessBlockTick          float64 `mapstructure:"process_block_tick"`
	NodeStatusTick            float64 `mapstructure:"node_status_tick"`
	LivenessTick              float64 `mapstructure:"liveness_tick"`
	SchedulerTick             float64 `mapstructure:"scheduler_tick"`
	ReqTimeThreshold          float64 `mapstructure:"req_time_threshold"`
	DataDetectionTick         float64 `mapstructure:"data_detection_tick"`
}

type spvConfig struct {
	RefreshReqCacheMills     int64 `mapstructure:"refresh_reqcache_mils"`
	MessageCacheSize         int64 `mapstructure:"message_cahche_size"`
	ReSyncCheckIntervalMills int64 `mapstructure:"resync_check_interval_mils"`
	SyncTimeoutMills         int64 `mapstructure:"sync_timeout_mils"`
	ReqSyncBlockNum          int64 `mapstructure:"reqsync_blocknum"`
	MaxReqSyncBlockNum       int64 `mapstructure:"max_reqsync_blocknum"`
	PeerActiveTime           int64 `mapstructure:"peer_active_time"`
}

type monitorConfig struct {
	Enabled bool `mapstructure:"enabled"`
	Port    int  `mapstructure:"port"`
}

type pprofConfig struct {
	Enabled bool `mapstructure:"enabled"`
	Port    int  `mapstructure:"port"`
}

type redisConfig struct {
	Url          string `mapstructure:"url"`
	Auth         string `mapstructure:"auth"`
	DB           int    `mapstructure:"db"`
	MaxIdle      int    `mapstructure:"max_idle"`
	MaxActive    int    `mapstructure:"max_active"`
	IdleTimeout  int    `mapstructure:"idle_timeout"`
	CacheTimeout int    `mapstructure:"cache_timeout"`
}

type clientConfig struct {
	OrgId           string `mapstructure:"org_id"`
	UserKeyFilePath string `mapstructure:"user_key_file_path"`
	UserCrtFilePath string `mapstructure:"user_crt_file_path"`
	HashType        string `mapstructure:"hash_type"`
}

type schedulerConfig struct {
	RWSetLog int `mapstructure:"rwset_log"`
}

type coreConfig struct {
	Evidence        bool            `mapstructure:"evidence"`
	SchedulerConfig schedulerConfig `mapstructure:"scheduler"`
}

// CMConfig - Local config struct
type CMConfig struct {
	LogConfig        logger.LogConfig   `mapstructure:"log"`
	NetConfig        netConfig          `mapstructure:"net"`
	NodeConfig       nodeConfig         `mapstructure:"node"`
	RpcConfig        rpcConfig          `mapstructure:"rpc"`
	BlockChainConfig []blockchainConfig `mapstructure:"blockchain"`
	StorageConfig    StorageConfig      `mapstructure:"storage"`
	TxPoolConfig     txPoolConfig       `mapstructure:"txpool"`
	SyncConfig       syncConfig         `mapstructure:"sync"`
	SpvConfig        spvConfig          `mapstructure:"spv"`

	// 开发调试使用
	DebugConfig   debugConfig   `mapstructure:"debug"`
	PProfConfig   pprofConfig   `mapstructure:"pprof"`
	MonitorConfig monitorConfig `mapstructure:"monitor"`
	CoreConfig    coreConfig    `mapstructure:"core"`
}

// GetBlockChains - get blockchain config list
func (c *CMConfig) GetBlockChains() []blockchainConfig {
	return c.BlockChainConfig
}
