/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package chainconf

import (
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
	"sync"

	"chainmaker.org/chainmaker/pb-go/syscontract"

	"chainmaker.org/chainmaker-go/localconf"

	"chainmaker.org/chainmaker-go/logger"
	"chainmaker.org/chainmaker-go/utils"
	"chainmaker.org/chainmaker/common/helper"
	"chainmaker.org/chainmaker/pb-go/common"
	"chainmaker.org/chainmaker/pb-go/config"
	"chainmaker.org/chainmaker/pb-go/consensus"
	"chainmaker.org/chainmaker/protocol"
)

type consensusVerifier map[consensus.ConsensusType]protocol.Verifier

const (
	minTxTimeout       = 600
	minBlockInterval   = 10
	minBlockTxCapacity = 1
	minBlockSize       = 1
	minNodeIds         = 1
	minTrustRoots      = 1
)

var (
	chainConsensusVerifier = make(map[string]consensusVerifier)
	// for multi chain start
	chainConfigVerifierLock = sync.RWMutex{}
)

// ChainConfig ChainConfig struct
type ChainConfig struct {
	*config.ChainConfig
	NodeOrgIds       map[string][]string // NodeOrgIds is a map mapped org with consensus nodes ids.
	NodeIds          map[string]string   // NodeIds is a map mapped node id with org.
	CaRoots          map[string]struct{} // CaRoots is a map stored org ids.
	ResourcePolicies map[string]struct{} // ResourcePolicies is a map stored resource.
}

// VerifyChainConfig verify the chain config.
func VerifyChainConfig(cconfig *config.ChainConfig) (*ChainConfig, error) {
	chainLog := logger.GetLoggerByChain(logger.MODULE_CHAINCONF, cconfig.ChainId)
	// validate params
	if err := validateParams(cconfig); err != nil {
		return nil, err
	}

	mConfig := &ChainConfig{
		ChainConfig:      cconfig,
		NodeOrgIds:       make(map[string][]string),
		NodeIds:          make(map[string]string),
		CaRoots:          make(map[string]struct{}),
		ResourcePolicies: make(map[string]struct{}),
	}

	if err := verifyChainConfigTrustRoots(cconfig, mConfig, chainLog); err != nil {
		return nil, err
	}

	if err := verifyChainConfigConsensus(cconfig, mConfig, chainLog); err != nil {
		return nil, err
	}

	if err := verifyChainConfigResourcePolicies(cconfig, mConfig); err != nil {
		return nil, err
	}

	if len(mConfig.TrustRoots) < minTrustRoots {
		msg := fmt.Sprintf("trustRoots len less than %d, trustRoots len is %d", minTrustRoots, len(mConfig.TrustRoots))
		chainLog.Error(msg)
		return nil, errors.New(msg)
	}

	if len(mConfig.NodeIds) < minNodeIds {
		msg := fmt.Sprintf("nodeIds len less than %d, nodeIds len is %d", minNodeIds, len(mConfig.NodeIds))
		chainLog.Error(msg)
		return nil, errors.New(msg)
	}
	// block
	if cconfig.Block.TxTimeout < minTxTimeout {
		// timeout
		msg := fmt.Sprintf("txTimeout less than %d, txTimeout is %d", minTxTimeout, cconfig.Block.TxTimeout)
		chainLog.Error(msg)
		return nil, errors.New(msg)
	}
	if cconfig.Block.BlockTxCapacity < minBlockTxCapacity {
		// block tx cap
		msg := fmt.Sprintf("blockTxCapacity less than %d, blockTxCapacity is %d",
			minBlockTxCapacity, cconfig.Block.BlockTxCapacity)
		chainLog.Error(msg)
		return nil, errors.New(msg)
	}
	if cconfig.Block.BlockSize < minBlockSize {
		// block size
		msg := fmt.Sprintf("blockSize less than %d, blockSize is %d", minBlockSize, cconfig.Block.BlockSize)
		chainLog.Error(msg)
		return nil, errors.New(msg)
	}
	if cconfig.Block.BlockInterval < minBlockInterval {
		// block interval
		msg := fmt.Sprintf("blockInterval less than %d, blockInterval is %d",
			minBlockInterval, cconfig.Block.BlockInterval)
		chainLog.Error(msg)
		return nil, errors.New(msg)
	}

	if cconfig.Contract == nil {
		cconfig.Contract = &config.ContractConfig{EnableSqlSupport: false} //by default disable sql support
	}

	if cconfig.Contract.EnableSqlSupport {
		provider := localconf.ChainMakerConfig.StorageConfig.StateDbConfig.Provider
		if provider != "sql" {
			chainLog.Errorf("chain config error: chain config sql is enable, expect chainmaker config provider is sql,"+
				" but got %s. current config: storage.statedb_config.provider = %s, contract.enable_sql_support = true",
				provider, provider)
			return nil, errors.New("chain config error")
		}
		if cconfig.Consensus.Type == consensus.ConsensusType_HOTSTUFF {
			chainLog.Errorf("chain config error: chain config sql is enable, expect consensus tbft/raft/solo,"+
				" but got %s. current config: consensus.type = %s, contract.enable_sql_support = true",
				consensus.ConsensusType_HOTSTUFF.String(), consensus.ConsensusType_HOTSTUFF)
			return nil, errors.New("chain config error")
		}
	}
	// verify
	verifier := GetVerifier(cconfig.ChainId, cconfig.Consensus.Type)
	if verifier != nil {
		err := verifier.Verify(cconfig.Consensus.Type, cconfig)
		if err != nil {
			chainLog.Errorw("consensus verify is err", "err", err)
			return nil, err
		}
	}

	return mConfig, nil
}

func verifyChainConfigTrustRoots(config *config.ChainConfig, mConfig *ChainConfig, log protocol.Logger) error {
	// load all ca root certs
	for _, root := range config.TrustRoots {
		if _, ok := mConfig.CaRoots[root.OrgId]; ok {
			err := fmt.Errorf("check root certificate failed, org id [%s] already exists", root.OrgId)
			log.Error(err)
			return err
		}
		// check root cert
		if ok, err := utils.CheckRootCertificate(root.Root); err != nil && !ok {
			log.Errorf("check root certificate failed, %s", err.Error())
			return err
		}
		mConfig.CaRoots[root.OrgId] = struct{}{}
		block, _ := pem.Decode([]byte(root.Root))
		if block == nil {
			return errors.New("root is empty")
		}
	}
	return nil
}

func verifyChainConfigConsensus(config *config.ChainConfig, mConfig *ChainConfig, log protocol.Logger) error {
	// verify consensus
	if config.Consensus != nil && config.Consensus.Nodes != nil {
		if len(config.Consensus.Nodes) == 0 {
			err := fmt.Errorf("there is at least one consensus node")
			log.Error(err.Error())
			return err
		}
		for _, node := range config.Consensus.Nodes {
			// org id can not be repeated
			if _, ok := mConfig.NodeOrgIds[node.OrgId]; ok {
				err := fmt.Errorf("org id(%s) existed", node.OrgId)
				log.Error(err.Error())
				return err
			}
			// when creating genesis, the org id of node must be exist in CaRoots.
			if _, ok := mConfig.CaRoots[node.OrgId]; !ok {
				err := fmt.Errorf("org id(%s) not in trust roots config", node.OrgId)
				log.Error(err.Error())
				return err
			}

			mConfig.NodeOrgIds[node.OrgId] = node.NodeId
			if err := verifyChainConfigConsensusNodesIds(mConfig, node, log); err != nil {
				return err
			}
		}
	}
	return nil
}

func verifyChainConfigConsensusNodesIds(mConfig *ChainConfig, node *config.OrgConfig, log protocol.Logger) error {
	if len(node.NodeId) > 0 {
		for _, nid := range node.NodeId {
			// node id can not be repeated
			if _, ok := mConfig.NodeIds[nid]; ok {
				log.Errorf("node id(%s) existed", nid)
				return errors.New("node id existed")
			}
			mConfig.NodeIds[nid] = node.OrgId
		}
	} else {
		for _, addr := range node.Address {
			nid, err := helper.GetNodeUidFromAddr(addr)
			if err != nil {
				log.Errorf("get node id from addr(%s) failed", addr)
				return err
			}
			// node id can not be repeated
			if _, ok := mConfig.NodeIds[nid]; ok {
				log.Errorf("node id(%s) existed", nid)
				return errors.New("node id existed")
			}
			mConfig.NodeIds[nid] = node.OrgId
		}
	}

	return nil
}

func verifyChainConfigResourcePolicies(config *config.ChainConfig, mConfig *ChainConfig) error {
	if config.ResourcePolicies != nil {
		resourceLen := len(config.ResourcePolicies)
		for _, resourcePolicy := range config.ResourcePolicies {
			mConfig.ResourcePolicies[resourcePolicy.ResourceName] = struct{}{}
			if err := verifyPolicy(resourcePolicy); err != nil {
				return err
			}
		}
		resLen := len(mConfig.ResourcePolicies)
		if resourceLen != resLen {
			return errors.New("resource name duplicate")
		}
	}
	return nil
}

func verifyPolicy(resourcePolicy *config.ResourcePolicy) error {
	policy := resourcePolicy.Policy
	resourceName := resourcePolicy.ResourceName
	if policy != nil {
		// to upper
		rule := policy.Rule
		policy.Rule = strings.ToUpper(rule)

		// self only for NODE_ID_UPDATE or TRUST_ROOT_UPDATE
		if policy.Rule == string(protocol.RuleSelf) {
			if resourceName != syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
				syscontract.ChainConfigFunction_NODE_ID_UPDATE.String() &&
				resourceName != syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
					syscontract.ChainConfigFunction_TRUST_ROOT_UPDATE.String() {
				err := fmt.Errorf("self rule can only be used by NODE_ID_UPDATE or TRUST_ROOT_UPDATE")
				return err
			}
		}

		roles := policy.RoleList
		if roles != nil {
			// to upper
			for i, role := range roles {
				role = strings.ToUpper(role)
				roles[i] = role
				// MAJORITY role allow admin or null
				if policy.Rule == string(protocol.RuleMajority) {
					if len(role) > 0 && role != string(protocol.RoleAdmin) {
						err := fmt.Errorf("config rule[MAJORITY], role can only be admin or null")
						return err
					}
				}
			}
			policy.RoleList = roles
		}
		// MAJORITY  not allowed org_list
		if policy.Rule == string(protocol.RuleMajority) && len(policy.OrgList) > 0 {
			err := fmt.Errorf("config rule[MAJORITY], org_list param not allowed")
			return err
		}
	}
	return nil
}

// validateParams validate the chainconfig
func validateParams(config *config.ChainConfig) error {
	if config.TrustRoots == nil {
		return errors.New("chainconfig trust_roots is nil")
	}
	if config.Consensus == nil {
		return errors.New("chainconfig consensus is nil")
	}
	if config.Block == nil {
		return errors.New("chainconfig block is nil")
	}
	match := utils.CheckChainIdFormat(config.ChainId)
	if !match {
		return fmt.Errorf("chain id[%s] can only consist of numbers,"+
			" letters and underscores and chainId length must less than 30", config.ChainId)
	}
	return nil
}

// RegisterVerifier register a verifier.
func RegisterVerifier(chainId string, consensusType consensus.ConsensusType, verifier protocol.Verifier) error {
	chainConfigVerifierLock.Lock()
	defer chainConfigVerifierLock.Unlock()
	initChainConsensusVerifier(chainId)
	if _, ok := chainConsensusVerifier[chainId][consensusType]; ok {
		return errors.New("consensusType verifier is exist")
	}
	chainConsensusVerifier[chainId][consensusType] = verifier
	return nil
}

// GetVerifier get a verifier if exist.
func GetVerifier(chainId string, consensusType consensus.ConsensusType) protocol.Verifier {
	chainConfigVerifierLock.RLock()
	defer chainConfigVerifierLock.RUnlock()
	initChainConsensusVerifier(chainId)
	verifier, ok := chainConsensusVerifier[chainId][consensusType]
	if !ok {
		return nil
	}
	return verifier
}

func initChainConsensusVerifier(chainId string) {
	if _, ok := chainConsensusVerifier[chainId]; !ok {
		chainConsensusVerifier[chainId] = make(consensusVerifier)
	}
}

// IsNative whether the contractName is a native
func IsNative(contractName string) bool {
	_, ok := syscontract.SystemContract_value[contractName]
	return ok
	//switch contractName {
	//case syscontract.SystemContract_CHAIN_CONFIG.String(),
	//	syscontract.SystemContract_CHAIN_QUERY.String(),
	//	syscontract.SystemContract_CERT_MANAGE.String(),
	//	syscontract.SystemContract_MULTI_SIGN.String(),
	//	syscontract.SystemContract_GOVERNANCE.String(),
	//	syscontract.SystemContract_PRIVATE_COMPUTE.String():
	//	return true
	//default:
	//	return false
	//}
}

// IsNativeTx whether the transaction is a native
func IsNativeTx(tx *common.Transaction) (contract string, b bool) {
	if tx == nil || tx.Payload == nil {
		return "", false
	}
	txType := tx.Payload.TxType
	switch txType {
	case common.TxType_INVOKE_CONTRACT:
		payload := tx.Payload
		return payload.ContractName, IsNative(payload.ContractName)
	//case common.TxType_MANAGE_USER_CONTRACT:
	//	payloadBytes := tx.RequestPayload
	//	payload := new(common.Payload)
	//	err := proto.Unmarshal(payloadBytes, payload)
	//	if err != nil {
	//		return "", false
	//	}
	//	if payload.Contract == nil {
	//		return "", false
	//	}
	//	return payload.Contract.Name, IsNative(payload.Contract.Name)
	default:
		return "", false
	}
}

func IsNativeTxSucc(tx *common.Transaction) (contract string, b bool) {
	if tx.Result == nil || tx.Result.ContractResult == nil || tx.Result.ContractResult.Result == nil {
		return "", false
	}
	contract, b = IsNativeTx(tx)
	if !b {
		return "", false
	}
	if tx.Result.Code != common.TxStatusCode_SUCCESS {
		return "", false
	}
	return contract, true
}
