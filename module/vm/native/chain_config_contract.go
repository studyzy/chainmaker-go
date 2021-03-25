/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package native

import (
	"chainmaker.org/chainmaker-go/chainconf"
	"chainmaker.org/chainmaker-go/common/helper"
	"chainmaker.org/chainmaker-go/common/sortedmap"
	"chainmaker.org/chainmaker-go/logger"
	acPb "chainmaker.org/chainmaker-go/pb/protogo/accesscontrol"
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	configPb "chainmaker.org/chainmaker-go/pb/protogo/config"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/utils"
	"errors"
	"fmt"
	"github.com/gogo/protobuf/proto"
	"strconv"
	"strings"
)

const (
	paramNameOrgId      = "org_id"
	paramNameRoot       = "root"
	paramNameAddresses  = "addresses"
	paramNameAddress    = "address"
	paramNameNewAddress = "new_address"

	paramNameTxSchedulerTimeout         = "tx_scheduler_timeout"
	paramNameTxSchedulerValidateTimeout = "tx_scheduler_validate_timeout"

	paramNameTxTimestampVerify    = "tx_timestamp_verify"
	paramNameTxTimeout            = "tx_timeout"
	paramNameChainBlockTxCapacity = "block_tx_capacity"
	paramNameChainBlockSize       = "block_size"
	paramNameChainBlockInterval   = "block_interval"
	paramNameChainBlockHeight     = "block_height"

	defaultConfigMaxValidateTimeout  = 60
	defaultConfigMaxSchedulerTimeout = 60
)

var (
	chainConfigContractName = commonPb.ContractName_SYSTEM_CONTRACT_CHAIN_CONFIG.String()
	keyChainConfig          = chainConfigContractName
)

type ChainConfigContract struct {
	methods map[string]ContractFunc
	log     *logger.CMLogger
}

func newChainConfigContract(log *logger.CMLogger) *ChainConfigContract {
	return &ChainConfigContract{
		log:     log,
		methods: registerChainConfigContractMethods(log),
	}
}

func (c *ChainConfigContract) getMethod(methodName string) ContractFunc {
	return c.methods[methodName]
}

func registerChainConfigContractMethods(log *logger.CMLogger) map[string]ContractFunc {
	methodMap := make(map[string]ContractFunc, 64)
	// [core]
	coreRuntime := &ChainCoreRuntime{log: log}
	methodMap[commonPb.ConfigFunction_CORE_UPDATE.String()] = coreRuntime.CoreUpdate

	// [block]
	blockRuntime := &ChainBlockRuntime{log: log}
	methodMap[commonPb.ConfigFunction_BLOCK_UPDATE.String()] = blockRuntime.BlockUpdate

	// [trust_root]
	trustRootsRuntime := &ChainTrustRootsRuntime{log: log}
	methodMap[commonPb.ConfigFunction_TRUST_ROOT_ADD.String()] = trustRootsRuntime.TrustRootAdd
	methodMap[commonPb.ConfigFunction_TRUST_ROOT_UPDATE.String()] = trustRootsRuntime.TrustRootUpdate
	methodMap[commonPb.ConfigFunction_TRUST_ROOT_DELETE.String()] = trustRootsRuntime.TrustRootDelete

	// [consensus]
	consensusRuntime := &ChainConsensusRuntime{log: log}
	methodMap[commonPb.ConfigFunction_NODE_ADDR_ADD.String()] = consensusRuntime.NodeAddrAdd
	methodMap[commonPb.ConfigFunction_NODE_ADDR_UPDATE.String()] = consensusRuntime.NodeAddrUpdate
	methodMap[commonPb.ConfigFunction_NODE_ADDR_DELETE.String()] = consensusRuntime.NodeAddrDelete
	methodMap[commonPb.ConfigFunction_NODE_ORG_ADD.String()] = consensusRuntime.NodeOrgAdd
	methodMap[commonPb.ConfigFunction_NODE_ORG_UPDATE.String()] = consensusRuntime.NodeOrgUpdate
	methodMap[commonPb.ConfigFunction_NODE_ORG_DELETE.String()] = consensusRuntime.NodeOrgDelete
	methodMap[commonPb.ConfigFunction_CONSENSUS_EXT_ADD.String()] = consensusRuntime.ConsensusExtAdd
	methodMap[commonPb.ConfigFunction_CONSENSUS_EXT_UPDATE.String()] = consensusRuntime.ConsensusExtUpdate
	methodMap[commonPb.ConfigFunction_CONSENSUS_EXT_DELETE.String()] = consensusRuntime.ConsensusExtDelete

	// [permission]
	methodMap[commonPb.ConfigFunction_PERMISSION_ADD.String()] = consensusRuntime.ResourcePolicyAdd
	methodMap[commonPb.ConfigFunction_PERMISSION_UPDATE.String()] = consensusRuntime.ResourcePolicyUpdate
	methodMap[commonPb.ConfigFunction_PERMISSION_DELETE.String()] = consensusRuntime.ResourcePolicyDelete

	// [chainConfig]
	ChainConfigRuntime := &ChainConfigRuntime{log: log}
	methodMap[commonPb.ConfigFunction_GET_CHAIN_CONFIG.String()] = ChainConfigRuntime.GetChainConfig
	methodMap[commonPb.ConfigFunction_GET_CHAIN_CONFIG_AT.String()] = ChainConfigRuntime.GetChainConfigFromBlockHeight

	return methodMap
}

func getChainConfig(txSimContext protocol.TxSimContext, params map[string]string) (*configPb.ChainConfig, error) {
	if params == nil {
		return nil, ErrParamsEmpty
	}
	bytes, err := txSimContext.Get(chainConfigContractName, []byte(keyChainConfig))
	if err != nil {
		msg := fmt.Errorf("txSimContext get failed, name[%s] key[%s] err: %+v", chainConfigContractName, keyChainConfig, err)
		return nil, msg
	}

	var chainConfig configPb.ChainConfig
	err = proto.Unmarshal(bytes, &chainConfig)
	if err != nil {
		msg := fmt.Errorf("unmarshal chainConfig failed, contractName %s err: %+v", chainConfigContractName, err)
		return nil, msg
	}

	return &chainConfig, nil
}

func setChainConfig(txSimContext protocol.TxSimContext, chainConfig *configPb.ChainConfig) ([]byte, error) {
	_, err := chainconf.VerifyChainConfig(chainConfig)
	if err != nil {
		return nil, err
	}

	chainConfig.Sequence = chainConfig.Sequence + 1
	pbccPayload, err := proto.Marshal(chainConfig)
	if err != nil {
		return nil, fmt.Errorf("proto marshal pbcc failed, err: %s", err.Error())
	}
	// 如果不存在对应的
	err = txSimContext.Put(chainConfigContractName, []byte(keyChainConfig), pbccPayload)
	if err != nil {
		return nil, fmt.Errorf("txSimContext put failed, err: %s", err.Error())
	}

	return pbccPayload, nil
}

// [core]
type ChainConfigRuntime struct {
	log *logger.CMLogger
}

// GetChainConfig get newest chain config
func (r *ChainConfigRuntime) GetChainConfig(txSimContext protocol.TxSimContext, params map[string]string) (result []byte, err error) {
	if params == nil {
		r.log.Error(ErrParamsEmpty)
		return nil, ErrParamsEmpty
	}
	bytes, err := txSimContext.Get(chainConfigContractName, []byte(keyChainConfig))
	if err != nil {
		r.log.Errorf("txSimContext get failed, name[%s] key[%s] err: %s", chainConfigContractName, keyChainConfig, err)
		return nil, err
	}
	return bytes, nil
}

// GetChainConfigAt get chain config from less than or equal to block height
func (r *ChainConfigRuntime) GetChainConfigFromBlockHeight(txSimContext protocol.TxSimContext, params map[string]string) (result []byte, err error) {
	if params == nil {
		r.log.Error(ErrParamsEmpty)
		return nil, ErrParamsEmpty
	}
	blockHeightStr := params[paramNameChainBlockHeight]
	blockHeight, err := strconv.ParseInt(blockHeightStr, 10, 0)
	if err != nil {
		r.log.Error(err)
		return nil, err
	}
	chainConfig, err := chainconf.GetChainConfigAt(r.log, nil, nil, txSimContext.GetBlockchainStore(), blockHeight)
	if err != nil {
		r.log.Error("get chain config from block height failed, err: %s", err)
		return nil, err
	}
	bytes, err := proto.Marshal(chainConfig)
	if err != nil {
		r.log.Error(err)
		return nil, err
	}
	return bytes, nil
}

// [core]
type ChainCoreRuntime struct {
	log *logger.CMLogger
}

func (r *ChainCoreRuntime) CoreUpdate(txSimContext protocol.TxSimContext, params map[string]string) (result []byte, err error) {
	chainConfig, err := getChainConfig(txSimContext, params)
	if err != nil {
		r.log.Error(err)
		return nil, err
	}
	changed := false
	if chainConfig.Core == nil {
		chainConfig.Core = &configPb.CoreConfig{}
	}

	// [0, 60] tx_scheduler_timeout
	if txSchedulerTimeout, ok := params[paramNameTxSchedulerTimeout]; ok {
		parseUint, err := strconv.ParseUint(txSchedulerTimeout, 10, 0)
		if err != nil {
			r.log.Error(err)
			return nil, err
		}
		if parseUint > defaultConfigMaxValidateTimeout {
			r.log.Error(ErrOutOfRange)
			return nil, ErrOutOfRange
		}
		chainConfig.Core.TxSchedulerTimeout = parseUint
		changed = true
	}
	// [0, 60] tx_scheduler_validate_timeout
	if txSchedulerValidateTimeout, ok := params[paramNameTxSchedulerValidateTimeout]; ok {
		parseUint, err := strconv.ParseUint(txSchedulerValidateTimeout, 10, 0)
		if err != nil {
			r.log.Error(err)
			return nil, err
		}
		if parseUint > defaultConfigMaxSchedulerTimeout {
			r.log.Error(ErrOutOfRange)
			return nil, ErrOutOfRange
		}
		chainConfig.Core.TxSchedulerValidateTimeout = parseUint
		changed = true
	}

	if !changed {
		r.log.Error(ErrParams)
		return nil, ErrParams
	}
	// [end]
	result, err = setChainConfig(txSimContext, chainConfig)
	r.log.Infof("core update success, params ", params)
	return result, err
}

// [block]
type ChainBlockRuntime struct {
	log *logger.CMLogger
}

func (r *ChainBlockRuntime) BlockUpdate(txSimContext protocol.TxSimContext, params map[string]string) (result []byte, err error) {
	// [start]start verify
	chainConfig, err := getChainConfig(txSimContext, params)
	if err != nil {
		r.log.Error(err)
		return nil, err
	}

	// tx_timestamp_verify
	changed1, err := utils.UpdateField(params, paramNameTxTimestampVerify, chainConfig.Block)
	if err != nil {
		return nil, err
	}
	// tx_timeout,(second)
	changed2, err := utils.UpdateField(params, paramNameTxTimeout, chainConfig.Block)
	if err != nil {
		return nil, err
	}
	// block_tx_capacity
	changed3, err := utils.UpdateField(params, paramNameChainBlockTxCapacity, chainConfig.Block)
	if err != nil {
		return nil, err
	}
	// block_size,(MB)
	changed4, err := utils.UpdateField(params, paramNameChainBlockSize, chainConfig.Block)
	if err != nil {
		return nil, err
	}
	// block_interval,(ms)
	changed5, err := utils.UpdateField(params, paramNameChainBlockInterval, chainConfig.Block)
	if err != nil {
		return nil, err
	}

	if !(changed1 || changed2 || changed3 || changed4 || changed5) {
		r.log.Error(ErrParams)
		return nil, ErrParams
	}
	// [end]
	result, err = setChainConfig(txSimContext, chainConfig)
	r.log.Infof("block update success, param ", params)
	return result, err
}

// [trust_root]
type ChainTrustRootsRuntime struct {
	log *logger.CMLogger
}

// TrustRootAdd add trustRoot
func (r *ChainTrustRootsRuntime) TrustRootAdd(txSimContext protocol.TxSimContext, params map[string]string) (result []byte, err error) {
	// [start]
	chainConfig, err := getChainConfig(txSimContext, params)
	if err != nil {
		r.log.Error(err)
		return nil, err
	}

	orgId := params[paramNameOrgId]
	rootCaCrt := params[paramNameRoot]
	if utils.IsAnyBlank(orgId, rootCaCrt) {
		err = fmt.Errorf("%s, add trust root cert require param [%s, %s] not found", ErrParams.Error(), paramNameOrgId, paramNameRoot)
		r.log.Error(err)
		return nil, err
	}

	chainConfig.TrustRoots = append(chainConfig.TrustRoots, &configPb.TrustRootConfig{
		OrgId: orgId,
		Root:  rootCaCrt,
	})
	result, err = setChainConfig(txSimContext, chainConfig)
	r.log.Infof("trust root add success. orgId[%s] cert[%s]", orgId, rootCaCrt)
	return result, err
}

// TrustRootUpdate update the trustRoot
func (r *ChainTrustRootsRuntime) TrustRootUpdate(txSimContext protocol.TxSimContext, params map[string]string) (result []byte, err error) {
	// [start]
	chainConfig, err := getChainConfig(txSimContext, params)
	if err != nil {
		r.log.Error(err)
		return nil, err
	}

	orgId := params[paramNameOrgId]
	rootCaCrt := params[paramNameRoot]
	if utils.IsAnyBlank(orgId, rootCaCrt) {
		err = fmt.Errorf("update trust root cert failed, require param [%s, %s] but not found", paramNameOrgId, paramNameRoot)
		r.log.Error(err)
		return nil, err
	}

	trustRoots := chainConfig.TrustRoots
	for i, root := range trustRoots {
		if orgId == root.OrgId {
			trustRoots[i] = &configPb.TrustRootConfig{
				OrgId: orgId,
				Root:  rootCaCrt,
			}
			result, err = setChainConfig(txSimContext, chainConfig)
			r.log.Infof("trust root update success. orgId[%s] cert[%s]", orgId, rootCaCrt)
			return result, err
		}
	}

	err = fmt.Errorf("%s can not found orgId[%s]", ErrParams.Error(), orgId)
	r.log.Error(err)
	return nil, err
}

// TrustRootDelete delete trustRoot
func (r *ChainTrustRootsRuntime) TrustRootDelete(txSimContext protocol.TxSimContext, params map[string]string) (result []byte, err error) {
	// [start]
	chainConfig, err := getChainConfig(txSimContext, params)
	if err != nil {
		r.log.Error(err)
		return nil, err
	}

	orgId := params[paramNameOrgId]
	if utils.IsAnyBlank(orgId) {
		err = fmt.Errorf("delete trust root cert failed, require param [%s], but not found", paramNameOrgId)
		r.log.Error(err)
		return nil, err
	}

	index := -1
	trustRoots := chainConfig.TrustRoots
	for i, root := range trustRoots {
		if orgId == root.OrgId {
			index = i
			break
		}
	}

	if index == -1 {
		err = fmt.Errorf("delete trust root cert failed, param [%s] not found from TrustRoot", orgId)
		r.log.Error(err)
		return nil, err
	}

	trustRoots = append(trustRoots[:index], trustRoots[index+1:]...)
	nodes := chainConfig.Consensus.Nodes
	for i := len(nodes) - 1; i >= 0; i-- {
		if orgId == nodes[i].OrgId {
			nodes = append(nodes[:i], nodes[i+1:]...)
		}
	}
	chainConfig.Consensus.Nodes = nodes
	chainConfig.TrustRoots = trustRoots
	result, err = setChainConfig(txSimContext, chainConfig)
	r.log.Infof("trust root delete success. orgId[%s]", orgId)
	return result, err
}

// [consensus]
type ChainConsensusRuntime struct {
	log *logger.CMLogger
}

// NodeAddrAdd add nodeAddr
func (r *ChainConsensusRuntime) NodeAddrAdd(txSimContext protocol.TxSimContext, params map[string]string) (result []byte, err error) {
	// [start]
	chainConfig, err := getChainConfig(txSimContext, params)
	if err != nil {
		r.log.Error(err)
		return nil, err
	}

	// verify params
	orgId := params[paramNameOrgId]
	addrStr := params[paramNameAddresses] // The addresses are separated by ","

	if utils.IsAnyBlank(orgId, addrStr) {
		err = fmt.Errorf("add node addr failed, require param [%s, %s], but not found", paramNameOrgId, paramNameAddresses)
		r.log.Error(err)
		return nil, err
	}

	addresses := strings.Split(addrStr, ",")
	nodes := chainConfig.Consensus.Nodes

	index := -1
	var nodeConf *configPb.OrgConfig
	for i, node := range nodes {
		if orgId == node.OrgId {
			index = i
			nodeConf = node
			break
		}
	}

	if index == -1 {
		err = fmt.Errorf("add node addr failed, param [%s] not found from nodes", orgId)
		r.log.Error(err)
		return nil, err
	}

	changed := false
	for _, addr := range addresses {
		addr = strings.TrimSpace(addr)
		if !helper.P2pAddressFormatVerify(addr) {
			err = fmt.Errorf("add node addr failed, address[%s] format error", addr)
			r.log.Error(err)
			return nil, err
		}
		address := nodeConf.Address
		address = append(address, addr)
		nodeConf.Address = address
		nodes[index] = nodeConf
		chainConfig.Consensus.Nodes = nodes
		changed = true
	}

	if !changed {
		r.log.Error(ErrParams)
		return nil, ErrParams
	}
	// [end]
	result, err = setChainConfig(txSimContext, chainConfig)
	r.log.Infof("node addr add success. orgId[%s] addrStr[%s]", orgId, addrStr)
	return result, err
}

// NodeAddrUpdate update nodeAddr
func (r *ChainConsensusRuntime) NodeAddrUpdate(txSimContext protocol.TxSimContext, params map[string]string) (result []byte, err error) {
	// [start]
	chainConfig, err := getChainConfig(txSimContext, params)
	if err != nil {
		r.log.Error(err)
		return nil, err
	}

	// verify params
	orgId := params[paramNameOrgId]
	addr := params[paramNameAddress]       // origin address
	newAddr := params[paramNameNewAddress] // new address

	if utils.IsAnyBlank(orgId, addr, newAddr) {
		err = fmt.Errorf("update node addr failed, require param [%s, %s, %s], but not found", paramNameOrgId, paramNameAddress, paramNameNewAddress)
		r.log.Error(err)
		return nil, err
	}

	nodes := chainConfig.Consensus.Nodes
	addr = strings.TrimSpace(addr)
	newAddr = strings.TrimSpace(newAddr)

	if !helper.P2pAddressFormatVerify(newAddr) {
		err = fmt.Errorf("update node addr failed, address[%s] format error", newAddr)
		r.log.Error(err)
		return nil, err
	}

	index := -1
	var nodeConf *configPb.OrgConfig
	for i, node := range nodes {
		if orgId == node.OrgId {
			index = i
			nodeConf = node
			break
		}
	}

	if index == -1 {
		err = fmt.Errorf("update node addr failed, param orgId[%s] not found from nodes", orgId)
		r.log.Error(err)
		return nil, err
	}

	for j, address := range nodeConf.Address {
		if addr == address {
			nodeConf.Address[j] = newAddr
			nodes[index] = nodeConf
			chainConfig.Consensus.Nodes = nodes
			result, err = setChainConfig(txSimContext, chainConfig)
			r.log.Infof("node addr update success. orgId[%s] addr[%s] newAddr[%s]", orgId, addr, newAddr)
			return result, err
		}
	}

	err = fmt.Errorf("update node addr failed, param orgId[%s] addr[%s] not found from nodes", orgId, addr)
	r.log.Error(err)
	return nil, err
}

// NodeAddrDelete delete nodeAddr
func (r *ChainConsensusRuntime) NodeAddrDelete(txSimContext protocol.TxSimContext, params map[string]string) (result []byte, err error) {
	// [start]
	chainConfig, err := getChainConfig(txSimContext, params)
	if err != nil {
		r.log.Error(err)
		return nil, err
	}
	// verify params
	orgId := params[paramNameOrgId]
	addr := params[paramNameAddress]

	if utils.IsAnyBlank(orgId, addr) {
		err = fmt.Errorf("delete node addr failed, require param [%s, %s], but not found", paramNameOrgId, paramNameAddress)
		r.log.Error(err)
		return nil, err
	}

	nodes := chainConfig.Consensus.Nodes
	index := -1
	var nodeConf *configPb.OrgConfig
	for i, node := range nodes {
		if orgId == node.OrgId {
			index = i
			nodeConf = node
			break
		}
	}

	if index == -1 {
		err = fmt.Errorf("delete node addr failed, param orgId[%s] not found from nodes", orgId)
		r.log.Error(err)
		return nil, err
	}

	addresses := nodeConf.Address
	for j, address := range addresses {
		if address == addr {
			nodeConf.Address = append(addresses[:j], addresses[j+1:]...)
			nodes[index] = nodeConf
			chainConfig.Consensus.Nodes = nodes
			result, err = setChainConfig(txSimContext, chainConfig)
			r.log.Infof("node addr delete success. orgId[%s] addr[%s]", orgId, addr)
			return result, err
		}
	}

	err = fmt.Errorf("delete node addr failed, param orgId[%s] addr[%s] not found from nodes", orgId, addr)
	r.log.Error(err)
	return nil, err
}

// NodeOrgAdd add nodeOrg
func (r *ChainConsensusRuntime) NodeOrgAdd(txSimContext protocol.TxSimContext, params map[string]string) (result []byte, err error) {
	// [start]
	chainConfig, err := getChainConfig(txSimContext, params)
	if err != nil {
		r.log.Error(err)
		return nil, err
	}

	// verify params
	orgId := params[paramNameOrgId]
	addrStr := params[paramNameAddresses]

	if utils.IsAnyBlank(orgId, addrStr) {
		err = fmt.Errorf("add node org failed, require param [%s, %s], but not found", paramNameOrgId, paramNameAddress)
		r.log.Error(err)
		return nil, err
	}
	nodes := chainConfig.Consensus.Nodes
	for _, node := range nodes {
		if orgId == node.OrgId {
			return nil, errors.New(paramNameOrgId + " is exist")
		}
	}
	org := &configPb.OrgConfig{
		OrgId:   orgId,
		Address: make([]string, 0),
	}

	addresses := strings.Split(addrStr, ",")
	for _, address := range addresses {
		address = strings.TrimSpace(address)
		if address != "" {
			org.Address = append(org.Address, address)
		}
	}
	if len(org.Address) > 0 {
		chainConfig.Consensus.Nodes = append(chainConfig.Consensus.Nodes, org)

		result, err = setChainConfig(txSimContext, chainConfig)
		r.log.Infof("node org add success. orgId[%s] addrStr[%s]", orgId, addrStr)
		return result, err
	}

	r.log.Error(ErrParams)
	return nil, ErrParams
}

// NodeOrgUpdate update nodeOrg
func (r *ChainConsensusRuntime) NodeOrgUpdate(txSimContext protocol.TxSimContext, params map[string]string) (result []byte, err error) {
	// [start]
	chainConfig, err := getChainConfig(txSimContext, params)
	if err != nil {
		r.log.Error(err)
		return nil, err
	}

	// verify params
	changed := false
	orgId := params[paramNameOrgId]
	addrStr := params[paramNameAddresses]

	if utils.IsAnyBlank(orgId, addrStr) {
		err = fmt.Errorf("update node org failed, require param [%s, %s], but not found", paramNameOrgId, paramNameAddress)
		r.log.Error(err)
		return nil, err
	}

	addresses := strings.Split(addrStr, ",")
	nodes := chainConfig.Consensus.Nodes
	index := -1
	var nodeConf *configPb.OrgConfig
	for i, node := range nodes {
		if orgId == node.OrgId {
			index = i
			nodeConf = node
			break
		}
	}

	if index == -1 {
		err = fmt.Errorf("update node org failed, param orgId[%s] not found from nodes", orgId)
		r.log.Error(err)
		return nil, err
	}

	nodeConf.Address = []string{}
	for _, addr := range addresses {
		addr = strings.TrimSpace(addr)
		if addr != "" {
			nodeConf.Address = append(nodeConf.Address, addr)
			nodes[index] = nodeConf
			chainConfig.Consensus.Nodes = nodes
			changed = true
		}
	}

	if !changed {
		r.log.Error(ErrParams)
		return nil, ErrParams
	}
	// [end]
	result, err = setChainConfig(txSimContext, chainConfig)
	r.log.Infof("node org update success. orgId[%s] addrStr[%s]", orgId, addrStr)
	return result, err
}

// NodeOrgDelete delete nodeOrg
func (r *ChainConsensusRuntime) NodeOrgDelete(txSimContext protocol.TxSimContext, params map[string]string) (result []byte, err error) {
	// [start]
	chainConfig, err := getChainConfig(txSimContext, params)
	if err != nil {
		r.log.Error(err)
		return nil, err
	}

	// verify params
	orgId := params[paramNameOrgId]

	if utils.IsAnyBlank(orgId) {
		err = fmt.Errorf("delete node org failed, require param [%s], but not found", paramNameOrgId)
		r.log.Error(err)
		return nil, err
	}

	nodes := chainConfig.Consensus.Nodes
	if len(nodes) == 1 {
		err := fmt.Errorf("there is at least one org")
		r.log.Error(err)
		return nil, err
	}
	for i, node := range nodes {
		if orgId == node.OrgId {
			nodes = append(nodes[:i], nodes[i+1:]...)
			chainConfig.Consensus.Nodes = nodes

			result, err = setChainConfig(txSimContext, chainConfig)
			r.log.Infof("node org delete success. orgId[%s]", orgId)
			return result, err
		}
	}

	err = fmt.Errorf("delete node org failed, param orgId[%s] not found from nodes", orgId)
	r.log.Error(err)
	return nil, err
}

// ConsensusExtAdd add consensus extra
func (r *ChainConsensusRuntime) ConsensusExtAdd(txSimContext protocol.TxSimContext, params map[string]string) (result []byte, err error) {
	// [start]
	chainConfig, err := getChainConfig(txSimContext, params)
	if err != nil {
		r.log.Error(err)
		return nil, err
	}

	// verify params
	changed := false
	extConfig := chainConfig.Consensus.ExtConfig
	if extConfig == nil {
		extConfig = make([]*commonPb.KeyValuePair, 0)
	}

	extConfigMap := make(map[string]string)
	for _, v := range extConfig {
		extConfigMap[v.Key] = v.Value
	}

	// map is out of order, in order to ensure that each execution sequence is consistent, we need to sort
	sortedParams := sortedmap.NewStringKeySortedMapWithData(params)
	var parseParamErr error
	sortedParams.Range(func(key string, val interface{}) (isContinue bool) {
		value := val.(string)
		if _, ok := extConfigMap[key]; ok {
			parseParamErr = fmt.Errorf("ext_config key[%s] is exist", key)
			r.log.Error(parseParamErr.Error())
			return false
		}
		extConfig = append(extConfig, &commonPb.KeyValuePair{
			Key:   key,
			Value: value,
		})
		chainConfig.Consensus.ExtConfig = extConfig
		changed = true
		return true
	})
	if parseParamErr != nil {
		return nil, parseParamErr
	}

	if !changed {
		r.log.Error(ErrParams)
		return nil, ErrParams
	}
	// [end]
	result, err = setChainConfig(txSimContext, chainConfig)
	r.log.Infof("consensus ext add success. params ", params)
	return result, err
}

// ConsensusExtUpdate update consensus extra
func (r *ChainConsensusRuntime) ConsensusExtUpdate(txSimContext protocol.TxSimContext, params map[string]string) (result []byte, err error) {
	// [start]
	chainConfig, err := getChainConfig(txSimContext, params)
	if err != nil {
		r.log.Error(err)
		return nil, err
	}

	// verify params
	changed := false
	extConfig := chainConfig.Consensus.ExtConfig
	if extConfig == nil {
		extConfig = make([]*commonPb.KeyValuePair, 0)
	}

	extConfigMap := make(map[string]string)
	for _, v := range extConfig {
		extConfigMap[v.Key] = v.Value
	}

	for key, val := range params {
		if _, ok := extConfigMap[key]; !ok {
			continue
		}
		for i, config := range extConfig {
			if key == config.Key {
				extConfig[i] = &commonPb.KeyValuePair{
					Key:   key,
					Value: val,
				}
				chainConfig.Consensus.ExtConfig = extConfig
				changed = true
				break
			}
		}
	}

	if !changed {
		r.log.Error(ErrParams)
		return nil, ErrParams
	}
	// [end]
	result, err = setChainConfig(txSimContext, chainConfig)
	r.log.Infof("consensus ext update success. params ", params)
	return result, err
}

// ConsensusExtDelete delete consensus extra
func (r *ChainConsensusRuntime) ConsensusExtDelete(txSimContext protocol.TxSimContext, params map[string]string) (result []byte, err error) {
	// [start]
	chainConfig, err := getChainConfig(txSimContext, params)
	if err != nil {
		r.log.Error(err)
		return nil, err
	}

	// verify params
	changed := false
	extConfig := chainConfig.Consensus.ExtConfig
	if extConfig == nil {
		return nil, errors.New("ext_config is empty")
	}
	extConfigMap := make(map[string]string)
	for _, v := range extConfig {
		extConfigMap[v.Key] = v.Value
	}

	for key, _ := range params {
		if _, ok := extConfigMap[key]; !ok {
			continue
		}

		for i, config := range extConfig {
			if key == config.Key {
				extConfig = append(extConfig[:i], extConfig[i+1:]...)
				changed = true
				break
			}
		}
	}
	chainConfig.Consensus.ExtConfig = extConfig
	if !changed {
		r.log.Error(ErrParams)
		return nil, ErrParams
	}
	// [end]
	result, err = setChainConfig(txSimContext, chainConfig)
	r.log.Infof("consensus ext delete success. params ", params)
	return result, err
}

// [permissions]
type ChainPermissionRuntime struct {
	log *logger.CMLogger
}

// ResourcePolicyAdd add permission
func (r *ChainConsensusRuntime) ResourcePolicyAdd(txSimContext protocol.TxSimContext, params map[string]string) (result []byte, err error) {
	// [start]
	chainConfig, err := getChainConfig(txSimContext, params)
	if err != nil {
		r.log.Error(err)
		return nil, err
	}

	// verify params
	changed := false
	resourcePolicies := chainConfig.ResourcePolicies
	if resourcePolicies == nil {
		resourcePolicies = make([]*configPb.ResourcePolicy, 0)
	}

	resourceMap := make(map[string]interface{})
	for _, p := range resourcePolicies {
		resourceMap[p.ResourceName] = struct{}{}
	}

	sortedParams := sortedmap.NewStringKeySortedMapWithData(params)
	var parseParamErr error
	sortedParams.Range(func(key string, val interface{}) (isContinue bool) {
		value := val.(string)

		_, ok := resourceMap[key]
		if ok {
			parseParamErr = fmt.Errorf("permission resource_name[%s] is exist", key)
			r.log.Errorf(parseParamErr.Error())
			return false
		}

		policy := &acPb.Policy{}
		err := proto.Unmarshal([]byte(value), policy)
		if err != nil {
			parseParamErr = fmt.Errorf("policy Unmarshal err:%s", err)
			r.log.Errorf(parseParamErr.Error())
			return false
		}

		resourcePolicy := &configPb.ResourcePolicy{
			ResourceName: key,
			Policy:       policy,
		}

		ac, err := txSimContext.GetAccessControl()
		if err != nil {
			parseParamErr = fmt.Errorf("add resource policy GetAccessControl err:%s", err)
			r.log.Errorf(parseParamErr.Error())
			return false
		}

		b := ac.ValidateResourcePolicy(resourcePolicy)
		if !b {
			parseParamErr = fmt.Errorf("add resource policy failed this resourcePolicy is restricted CheckPrincipleValidity err resourcePolicy[%s]", resourcePolicy)
			r.log.Errorf(parseParamErr.Error())
			return false
		}
		resourcePolicies = append(resourcePolicies, resourcePolicy)

		chainConfig.ResourcePolicies = resourcePolicies
		changed = true
		return true
	})

	if parseParamErr != nil {
		return nil, parseParamErr
	}
	if !changed {
		r.log.Error(ErrParams)
		return nil, ErrParams
	}
	// [end]
	result, err = setChainConfig(txSimContext, chainConfig)
	r.log.Infof("resource policy add success. params ", params)
	return result, err
}

// ResourcePolicyUpdate update resource policy
func (r *ChainConsensusRuntime) ResourcePolicyUpdate(txSimContext protocol.TxSimContext, params map[string]string) (result []byte, err error) {
	// [start]
	chainConfig, err := getChainConfig(txSimContext, params)
	if err != nil {
		r.log.Error(err)
		return nil, err
	}

	// verify params
	changed := false
	resourcePolicies := chainConfig.ResourcePolicies
	if resourcePolicies == nil {
		resourcePolicies = make([]*configPb.ResourcePolicy, 0)
	}

	resourceMap := make(map[string]interface{})
	for _, p := range resourcePolicies {
		resourceMap[p.ResourceName] = struct{}{}
	}

	sortedParams := sortedmap.NewStringKeySortedMapWithData(params)
	var parseParamErr error
	sortedParams.Range(func(key string, val interface{}) (isContinue bool) {
		value := val.(string)
		_, ok := resourceMap[key]
		if !ok {
			parseParamErr = fmt.Errorf("permission resource name does not exist resource_name[%s]", value)
			r.log.Errorf(parseParamErr.Error())
			return false
		}
		policy := &acPb.Policy{}
		err := proto.Unmarshal([]byte(value), policy)
		if err != nil {
			parseParamErr = fmt.Errorf("policy Unmarshal err:%s", err)
			r.log.Errorf(parseParamErr.Error())
			return false
		}
		for i, resourcePolicy := range resourcePolicies {
			if resourcePolicy.ResourceName != key {
				continue
			}
			rp := &configPb.ResourcePolicy{
				ResourceName: key,
				Policy:       policy,
			}
			ac, err := txSimContext.GetAccessControl()
			if err != nil {
				parseParamErr = fmt.Errorf("GetAccessControl, err:%s", err)
				r.log.Errorf(parseParamErr.Error())
				return false
			}
			b := ac.ValidateResourcePolicy(rp)
			if !b {
				parseParamErr = fmt.Errorf("update resource policy this resourcePolicy is restricted. CheckPrincipleValidity err resourcePolicy %+v", rp)
				r.log.Errorf(parseParamErr.Error())
				return false
			}
			resourcePolicies[i] = rp
			chainConfig.ResourcePolicies = resourcePolicies
			changed = true
		}
		return true
	})

	if parseParamErr != nil {
		return nil, parseParamErr
	}

	if !changed {
		r.log.Error(ErrParams)
		return nil, ErrParams
	}
	// [end]
	result, err = setChainConfig(txSimContext, chainConfig)
	r.log.Infof("resource policy update success. params ", params)
	return result, err
}

// ResourcePolicyDelete delete permission
func (r *ChainConsensusRuntime) ResourcePolicyDelete(txSimContext protocol.TxSimContext, params map[string]string) (result []byte, err error) {
	// [start]
	chainConfig, err := getChainConfig(txSimContext, params)
	if err != nil {
		r.log.Error(err)
		return nil, err
	}

	// verify params
	changed := false
	resourcePolicies := chainConfig.ResourcePolicies
	if resourcePolicies == nil {
		resourcePolicies = make([]*configPb.ResourcePolicy, 0)
	}

	resourceMap := make(map[string]interface{})
	for _, p := range resourcePolicies {
		resourceMap[p.ResourceName] = struct{}{}
	}

	// map is out of order, in order to ensure that each execution sequence is consistent, we need to sort
	sortedParams := sortedmap.NewStringKeySortedMapWithData(params)
	var parseParamErr error
	sortedParams.Range(func(key string, val interface{}) (isContinue bool) {
		_, ok := resourceMap[key]
		if !ok {
			parseParamErr = fmt.Errorf("permission resource name does not exist resource_name[%s]", key)
			r.log.Error(parseParamErr.Error())
			return false
		}
		resourcePolicy := &configPb.ResourcePolicy{
			ResourceName: key,
			Policy: &acPb.Policy{
				Rule:     string(protocol.RuleDelete),
				OrgList:  nil,
				RoleList: nil,
			},
		}
		ac, err := txSimContext.GetAccessControl()
		if err != nil {
			parseParamErr = fmt.Errorf("delete resource policy GetAccessControl err:%s", err)
			r.log.Error(parseParamErr.Error())
			return false
		}
		b := ac.ValidateResourcePolicy(resourcePolicy)
		if !b {
			parseParamErr = fmt.Errorf("delete resource policy this resourcePolicy is restricted, CheckPrincipleValidity err resourcePolicy %+v", resourcePolicy)
			r.log.Error(parseParamErr.Error())
			return false
		}

		for i, rp := range resourcePolicies {
			if rp.ResourceName == key {
				resourcePolicies = append(resourcePolicies[:i], resourcePolicies[i+1:]...)
				chainConfig.ResourcePolicies = resourcePolicies
				changed = true
				break
			}
		}
		return true
	})
	if parseParamErr != nil {
		return nil, parseParamErr
	}

	if !changed {
		r.log.Error(ErrParams)
		return nil, ErrParams
	}
	// [end]
	result, err = setChainConfig(txSimContext, chainConfig)
	r.log.Infof("resource policy delete success. params ", params)
	return result, err
}
