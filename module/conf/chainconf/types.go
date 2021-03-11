/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package chainconf

import (
	"chainmaker.org/chainmaker-go/common/helper"
	"chainmaker.org/chainmaker-go/logger"
	pbac "chainmaker.org/chainmaker-go/pb/protogo/accesscontrol"
	"chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/pb/protogo/config"
	"chainmaker.org/chainmaker-go/pb/protogo/consensus"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/utils"
	"encoding/pem"
	"errors"
	"github.com/golang/protobuf/proto"
	"strings"
)

var (
	chainConfigVerifier = make(map[consensus.ConsensusType]protocol.Verifier, 0)
)

// chainConfig chainConfig struct
type chainConfig struct {
	*config.ChainConfig
	NodeOrgIds       map[string][]string // NodeOrgIds is a map mapped org with consensus nodes addresses.
	NodeAddresses    map[string]string   // NodeAddresses is a map mapped node address with org.
	Ips              map[string]string   // Ips is a map mapped ip with address.
	CaRoots          map[string]struct{} // CaRoots is a map stored org ids.
	ResourcePolicies map[string]struct{} // ResourcePolicies is a map stored resource.
}

// VerifyChainConfig verify the chain config.
func VerifyChainConfig(config *config.ChainConfig) (*chainConfig, error) {
	log := logger.GetLoggerByChain(logger.MODULE_CHAINCONF, config.ChainId)
	// validate params
	if err := validateParams(config); err != nil {
		return nil, err
	}

	mConfig := &chainConfig{
		ChainConfig:      config,
		NodeOrgIds:       make(map[string][]string),
		NodeAddresses:    make(map[string]string),
		Ips:              make(map[string]string),
		CaRoots:          make(map[string]struct{}),
		ResourcePolicies: make(map[string]struct{}),
	}

	if err := verifyChainConfigTrustRoots(config, mConfig); err != nil {
		return nil, err
	}

	if err := verifyChainConfigConsensus(config, mConfig); err != nil {
		return nil, err
	}

	if err := verifyChainConfigResourcePolicies(config, mConfig); err != nil {
		return nil, err
	}

	if len(mConfig.TrustRoots) < 1 {
		log.Errorw("trust roots len is low", "trustRoots len", len(mConfig.TrustRoots))
		return nil, errors.New("trust roots len is low")
	}
	if len(mConfig.NodeAddresses) < 1 {
		log.Errorw("nodeAddresses len is low", "nodeAddresses len", len(mConfig.NodeAddresses))
		return nil, errors.New("node address len is low")
	}
	// block
	if config.Block.TxTimeout < 600 {
		// timeout
		log.Errorw("txTimeout len is low", "txTimeout len", config.Block.TxTimeout)
		return nil, errors.New("tx_time is low")
	}
	if config.Block.BlockTxCapacity < 1 {
		// block tx cap
		log.Errorw("blockTxCapacity is low", "blockTxCapacity", config.Block.BlockTxCapacity)
		return nil, errors.New("block_tx_capacity is low")
	}
	if config.Block.BlockSize < 1 {
		// block size
		log.Errorw("blockSize is low", "blockSize", config.Block.BlockSize)
		return nil, errors.New("blockSize is low")
	}
	if config.Block.BlockInterval < 10 {
		// block interval
		log.Errorw("blockInterval is low", "blockInterval", config.Block.BlockInterval)
		return nil, errors.New("blockInterval is low")
	}
	// verify
	verifier := GetVerifier(config.Consensus.Type)
	if verifier != nil {
		err := verifier.Verify(config.Consensus.Type, config)
		if err != nil {
			log.Errorw("consensus verify is err", "err", err)
			return nil, err
		}
	}

	return mConfig, nil
}

func verifyChainConfigTrustRoots(config *config.ChainConfig, mConfig *chainConfig) error {
	// load all ca root certs
	for _, root := range config.TrustRoots {
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

func verifyChainConfigConsensus(config *config.ChainConfig, mConfig *chainConfig) error {
	// verify consensus
	if config.Consensus != nil && config.Consensus.Nodes != nil {
		for _, node := range config.Consensus.Nodes {
			// org id can not be repeated
			if _, ok := mConfig.NodeOrgIds[node.OrgId]; ok {
				log.Errorf("org id(%s) existed", node.OrgId)
				return errors.New("org id existed")
			}
			// when creating genesis, the org id of node must be exist in CaRoots.
			if _, ok := mConfig.CaRoots[node.OrgId]; !ok {
				log.Errorf("org id(%s) not in trust roots config", node.OrgId)
				return errors.New("org id not in trust roots config")
			}

			mConfig.NodeOrgIds[node.OrgId] = node.Address
			if err := verifyChainConfigConsensusNodesAddress(mConfig, node); err != nil {
				return err
			}
		}
	}
	return nil
}

func verifyChainConfigConsensusNodesAddress(mConfig *chainConfig, node *config.OrgConfig) error {
	for _, address := range node.Address {
		b := helper.P2pAddressFormatVerify(address)
		if !b {
			return errors.New("node address format is error")
		}
		splitAfter := strings.Split(address, "/")
		if splitAfter == nil || len(splitAfter) != 7 {
			log.Errorf("wrong address(%s)", address)
			return errors.New("wrong address")
		}
		realAddr := splitAfter[6]
		ip := splitAfter[2] + ":" + splitAfter[4]
		// node address can not be repeated
		if _, ok := mConfig.NodeAddresses[realAddr]; ok {
			log.Errorf("address(%s) existed", realAddr)
			return errors.New("address existed")
		}
		mConfig.NodeAddresses[realAddr] = node.OrgId
		// ip + port can not be repeated
		if _, ok := mConfig.Ips[ip]; ok {
			log.Errorf("ip(%s) existed", ip)
			return errors.New("ip existed")
		}
		mConfig.Ips[ip] = realAddr
	}
	return nil
}

func verifyChainConfigResourcePolicies(config *config.ChainConfig, mConfig *chainConfig) error {
	if config.ResourcePolicies != nil {
		resourceLen := len(config.ResourcePolicies)
		for _, resourcePolicy := range config.ResourcePolicies {
			mConfig.ResourcePolicies[resourcePolicy.ResourceName] = struct{}{}
			policy := resourcePolicy.Policy
			verifyPolicy(policy)
		}
		resLen := len(mConfig.ResourcePolicies)
		if resourceLen != resLen {
			return errors.New("resource name duplicate")
		}
	}
	return nil
}

func verifyPolicy(policy *pbac.Policy) {
	if policy != nil {
		// to upper
		rule := policy.Rule
		policy.Rule = strings.ToUpper(rule)
		roles := policy.RoleList
		if roles != nil {
			// to upper
			for i, role := range roles {
				role = strings.ToUpper(role)
				roles[i] = role
			}
			policy.RoleList = roles
		}
	}
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
	return nil
}

// RegisterVerifier register a verifier.
func RegisterVerifier(consensusType consensus.ConsensusType, verifier protocol.Verifier) error {
	if _, ok := chainConfigVerifier[consensusType]; ok {
		return errors.New("consensusType verifier is exist")
	}
	chainConfigVerifier[consensusType] = verifier
	return nil
}

// GetVerifier get a verifier if exist.
func GetVerifier(consensusType consensus.ConsensusType) protocol.Verifier {
	verifier, ok := chainConfigVerifier[consensusType]
	if !ok {
		return nil
	}
	return verifier
}

// IsNative whether the contractName is a native
func IsNative(contractName string) bool {
	switch contractName {
	case common.ContractName_SYSTEM_CONTRACT_CHAIN_CONFIG.String(),
		common.ContractName_SYSTEM_CONTRACT_QUERY.String(),
		common.ContractName_SYSTEM_CONTRACT_CERT_MANAGE.String(),
		common.ContractName_SYSTEM_CONTRACT_MULTI_SIGN.String(),
		common.ContractName_SYSTEM_CONTRACT_GOVERNANCE.String():
		return true
	default:
		return false
	}
}

// IsNativeTx whether the transaction is a native
func IsNativeTx(tx *common.Transaction) (contract string, b bool) {
	if tx == nil || tx.Header == nil {
		return "", false
	}
	txType := tx.Header.TxType
	switch txType {
	case common.TxType_INVOKE_SYSTEM_CONTRACT, common.TxType_UPDATE_CHAIN_CONFIG:
		payloadBytes := tx.RequestPayload
		payload := new(common.SystemContractPayload)
		err := proto.Unmarshal(payloadBytes, payload)
		if err != nil {
			return "", false
		}
		return payload.ContractName, IsNative(payload.ContractName)
	case common.TxType_MANAGE_USER_CONTRACT:
		payloadBytes := tx.RequestPayload
		payload := new(common.ContractMgmtPayload)
		err := proto.Unmarshal(payloadBytes, payload)
		if err != nil {
			return "", false
		}
		if payload.ContractId == nil {
			return "", false
		}
		return payload.ContractId.ContractName, IsNative(payload.ContractId.ContractName)
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
