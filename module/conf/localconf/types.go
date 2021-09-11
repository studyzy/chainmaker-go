/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package localconf

import (
	"chainmaker.org/chainmaker/logger/v2"
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
	IsCliOpen       bool `mapstructure:"is_cli_open"`
	IsHttpOpen      bool `mapstructure:"is_http_open"`
	IsProposer      bool `mapstructure:"is_proposer"`
	IsNotRWSetCheck bool `mapstructure:"is_not_rwset_check"`
	IsConcurPropose bool `mapstructure:"is_concur_propose"`
	IsConcurVerify  bool `mapstructure:"is_concur_verify"`
	IsSolo          bool `mapstructure:"is_solo"`
	IsHaltPropose   bool `mapstructure:"is_halt_propose"`
	// true: minimize access control; false: use full access control
	IsSkipAccessControl bool `mapstructure:"is_skip_access_control"`
	// true for trace memory usage information periodically
	IsTraceMemoryUsage bool `mapstructure:"is_trace_memory_usage"`
	// Simulate a node which would propose duplicate after it has proposed Proposal
	IsProposeDuplicately bool `mapstructure:"is_propose_duplicately"`
	// Simulate a malicious node which would propose duplicate proposals
	IsProposeMultiNodeDuplicately bool `mapstructure:"is_propose_multinode_duplicately"`
	IsProposalOldHeight           bool `mapstructure:"is_proposal_old_height"`
	// Simulate a malicious node which would prevote duplicately
	IsPrevoteDuplicately bool `mapstructure:"is_prevote_duplicately"`
	// Simulate a malicious node which would prevote for oldheight
	IsPrevoteOldHeight bool `mapstructure:"is_prevote_old_height"`
	IsPrevoteLost      bool `mapstructure:"is_prevote_lost"` //prevote vote lost
	//Simulate a malicious node which would propose duplicate precommits
	IsPrecommitDuplicately bool `mapstructure:"is_precommit_duplicately"`
	// Simulate a malicious node which would Precommit a lower height than current height
	IsPrecommitOldHeight bool `mapstructure:"is_precommit_old_height"`

	IsProposeLost    bool `mapstructure:"is_propose_lost"`     //proposal vote lost
	IsProposeDelay   bool `mapstructure:"is_propose_delay"`    //proposal lost
	IsPrevoteDelay   bool `mapstructure:"is_prevote_delay"`    //network problem resulting in preovote lost
	IsPrecommitLost  bool `mapstructure:"is_precommit_lost"`   //precommit vote lost
	IsPrecommitDelay bool `mapstructure:"is_prevcommit_delay"` //network problem resulting in precommit lost
	//if the node committing block without publishing, TRUE；else, FALSE
	IsCommitWithoutPublish bool `mapstructure:"is_commit_without_publish"`
	//simulate a node which sends an invalid prevote(hash=nil)
	IsPrevoteInvalid bool `mapstructure:"is_prevote_invalid"`
	//simulate a node which sends an invalid precommit(hash=nil)
	IsPrecommitInvalid bool `mapstructure:"is_precommit_invalid"`

	IsModifyTxPayload    bool `mapstructure:"is_modify_tx_payload"`
	IsExtreme            bool `mapstructure:"is_extreme"` //extreme fast mode
	UseNetMsgCompression bool `mapstructure:"use_net_msg_compression"`
	IsNetInsecurity      bool `mapstructure:"is_net_insecurity"`
}

type BlockchainConfig struct {
	ChainId string
	Genesis string
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

type monitorConfig struct {
	Enabled bool `mapstructure:"enabled"`
	Port    int  `mapstructure:"port"`
}

type pprofConfig struct {
	Enabled bool `mapstructure:"enabled"`
	Port    int  `mapstructure:"port"`
}

type raftConfig struct {
	SnapCount    uint64 `mapstructure:"snap_count"`
	AsyncWalSave bool   `mapstructure:"async_wal_save"`
}

type ConsensusConfig struct {
	RaftConfig raftConfig `mapstructure:"raft"`
}

//type redisConfig struct {
//	Url          string `mapstructure:"url"`
//	Auth         string `mapstructure:"auth"`
//	DB           int    `mapstructure:"db"`
//	MaxIdle      int    `mapstructure:"max_idle"`
//	MaxActive    int    `mapstructure:"max_active"`
//	IdleTimeout  int    `mapstructure:"idle_timeout"`
//	CacheTimeout int    `mapstructure:"cache_timeout"`
//}

//type clientConfig struct {
//	OrgId           string `mapstructure:"org_id"`
//	UserKeyFilePath string `mapstructure:"user_key_file_path"`
//	UserCrtFilePath string `mapstructure:"user_crt_file_path"`
//	HashType        string `mapstructure:"hash_type"`
//}

type schedulerConfig struct {
	RWSetLog bool `mapstructure:"rwset_log"`
}

type coreConfig struct {
	Evidence bool `mapstructure:"evidence"`
}

// CMConfig - Local config struct
type CMConfig struct {
	LogConfig        logger.LogConfig       `mapstructure:"log"`
	NetConfig        netConfig              `mapstructure:"net"`
	NodeConfig       nodeConfig             `mapstructure:"node"`
	RpcConfig        rpcConfig              `mapstructure:"rpc"`
	BlockChainConfig []BlockchainConfig     `mapstructure:"blockchain"`
	ConsensusConfig  ConsensusConfig        `mapstructure:"consensus"`
	StorageConfig    map[string]interface{} `mapstructure:"storage"`
	TxPoolConfig     txPoolConfig           `mapstructure:"txpool"`
	SyncConfig       syncConfig             `mapstructure:"sync"`

	// 开发调试使用
	DebugConfig     debugConfig     `mapstructure:"debug"`
	PProfConfig     pprofConfig     `mapstructure:"pprof"`
	MonitorConfig   monitorConfig   `mapstructure:"monitor"`
	CoreConfig      coreConfig      `mapstructure:"core"`
	SchedulerConfig schedulerConfig `mapstructure:"scheduler"`
}

// GetBlockChains - get blockchain config list
func (c *CMConfig) GetBlockChains() []BlockchainConfig {
	return c.BlockChainConfig
}
func (c *CMConfig) GetStorePath() string {
	if path, ok := c.StorageConfig["store_path"]; ok {
		return path.(string)
	}
	return ""
}
