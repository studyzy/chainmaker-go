/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package chainconfigmgr

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"chainmaker.org/chainmaker-go/chainconf"
	"chainmaker.org/chainmaker-go/utils"
	"chainmaker.org/chainmaker-go/vm/native/common"
	"chainmaker.org/chainmaker/common/sortedmap"
	acPb "chainmaker.org/chainmaker/pb-go/accesscontrol"
	commonPb "chainmaker.org/chainmaker/pb-go/common"
	configPb "chainmaker.org/chainmaker/pb-go/config"
	"chainmaker.org/chainmaker/pb-go/syscontract"
	"chainmaker.org/chainmaker/protocol"
	"github.com/gogo/protobuf/proto"
)

const (
	paramNameOrgId     = "org_id"
	paramNameRoot      = "root"
	paramNameNodeIds   = "node_ids"
	paramNameNodeId    = "node_id"
	paramNameNewNodeId = "new_node_id"

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
	chainConfigContractName = syscontract.SystemContract_CHAIN_CONFIG.String()
	keyChainConfig          = chainConfigContractName
)

type ChainConfigContract struct {
	methods map[string]common.ContractFunc
	log     protocol.Logger
}

func NewChainConfigContract(log protocol.Logger) *ChainConfigContract {
	return &ChainConfigContract{
		log:     log,
		methods: registerChainConfigContractMethods(log),
	}
}

func (c *ChainConfigContract) GetMethod(methodName string) common.ContractFunc {
	return c.methods[methodName]
}

func registerChainConfigContractMethods(log protocol.Logger) map[string]common.ContractFunc {
	methodMap := make(map[string]common.ContractFunc, 64)
	// [core]
	coreRuntime := &ChainCoreRuntime{log: log}

	methodMap[syscontract.ChainConfigFunction_CORE_UPDATE.String()] = coreRuntime.CoreUpdate

	// [block]
	blockRuntime := &ChainBlockRuntime{log: log}
	methodMap[syscontract.ChainConfigFunction_BLOCK_UPDATE.String()] = blockRuntime.BlockUpdate

	// [trust_root]
	trustRootsRuntime := &ChainTrustRootsRuntime{log: log}
	methodMap[syscontract.ChainConfigFunction_TRUST_ROOT_ADD.String()] = trustRootsRuntime.TrustRootAdd
	methodMap[syscontract.ChainConfigFunction_TRUST_ROOT_UPDATE.String()] = trustRootsRuntime.TrustRootUpdate
	methodMap[syscontract.ChainConfigFunction_TRUST_ROOT_DELETE.String()] = trustRootsRuntime.TrustRootDelete

	// [consensus]
	consensusRuntime := &ChainConsensusRuntime{log: log}
	methodMap[syscontract.ChainConfigFunction_NODE_ID_ADD.String()] = consensusRuntime.NodeIdAdd
	methodMap[syscontract.ChainConfigFunction_NODE_ID_UPDATE.String()] = consensusRuntime.NodeIdUpdate
	methodMap[syscontract.ChainConfigFunction_NODE_ID_DELETE.String()] = consensusRuntime.NodeIdDelete
	methodMap[syscontract.ChainConfigFunction_NODE_ORG_ADD.String()] = consensusRuntime.NodeOrgAdd
	methodMap[syscontract.ChainConfigFunction_NODE_ORG_UPDATE.String()] = consensusRuntime.NodeOrgUpdate
	methodMap[syscontract.ChainConfigFunction_NODE_ORG_DELETE.String()] = consensusRuntime.NodeOrgDelete
	methodMap[syscontract.ChainConfigFunction_CONSENSUS_EXT_ADD.String()] = consensusRuntime.ConsensusExtAdd
	methodMap[syscontract.ChainConfigFunction_CONSENSUS_EXT_UPDATE.String()] = consensusRuntime.ConsensusExtUpdate
	methodMap[syscontract.ChainConfigFunction_CONSENSUS_EXT_DELETE.String()] = consensusRuntime.ConsensusExtDelete

	// [permission]
	methodMap[syscontract.ChainConfigFunction_PERMISSION_ADD.String()] = consensusRuntime.ResourcePolicyAdd
	methodMap[syscontract.ChainConfigFunction_PERMISSION_UPDATE.String()] = consensusRuntime.ResourcePolicyUpdate
	methodMap[syscontract.ChainConfigFunction_PERMISSION_DELETE.String()] = consensusRuntime.ResourcePolicyDelete

	// [chainConfig]
	ChainConfigRuntime := &ChainConfigRuntime{log: log}
	methodMap[syscontract.ChainConfigFunction_GET_CHAIN_CONFIG.String()] = ChainConfigRuntime.GetChainConfig
	methodMap[syscontract.ChainConfigFunction_GET_CHAIN_CONFIG_AT.String()] = ChainConfigRuntime.GetChainConfigFromBlockHeight

	//// [archive]
	//archiveStoreRuntime := &ArchiveStoreRuntime{log: log}
	//methodMap[commonPb.ArchiveStoreContractFunction_ARCHIVE_BLOCK.String()] = archiveStoreRuntime.ArchiveBlock
	//methodMap[commonPb.ArchiveStoreContractFunction_RESTORE_BLOCKS.String()] = archiveStoreRuntime.RestoreBlock

	return methodMap
}

func getChainConfig(txSimContext protocol.TxSimContext, params map[string][]byte) (*configPb.ChainConfig, error) {
	if params == nil {
		return nil, common.ErrParamsEmpty
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

	err = chainconf.HandleCompatibility(&chainConfig)
	if err != nil {
		msg := fmt.Errorf("compatibility handle failed, contractName %s err: %+v", chainConfigContractName, err)
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
	log protocol.Logger
}

// GetChainConfig get newest chain config
func (r *ChainConfigRuntime) GetChainConfig(txSimContext protocol.TxSimContext, params map[string][]byte) (result []byte, err error) {
	chainConfig, err := getChainConfig(txSimContext, params)
	if err != nil {
		return nil, err
	}
	bytes, err := proto.Marshal(chainConfig)
	if err != nil {
		return nil, fmt.Errorf("proto marshal chain config failed, err: %s", err.Error())
	}
	return bytes, nil
}

// GetChainConfigAt get chain config from less than or equal to block height
func (r *ChainConfigRuntime) GetChainConfigFromBlockHeight(txSimContext protocol.TxSimContext, params map[string][]byte) (result []byte, err error) {
	if params == nil {
		r.log.Error(common.ErrParamsEmpty)
		return nil, common.ErrParamsEmpty
	}
	blockHeightStr := params[paramNameChainBlockHeight]
	blockHeight, err := strconv.ParseUint(string(blockHeightStr), 10, 0)
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
	log protocol.Logger
}

func (r *ChainCoreRuntime) CoreUpdate(txSimContext protocol.TxSimContext, params map[string][]byte) (result []byte, err error) {
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
		parseUint, err := strconv.ParseUint(string(txSchedulerTimeout), 10, 0)
		if err != nil {
			r.log.Error(err)
			return nil, err
		}
		if parseUint > defaultConfigMaxValidateTimeout {
			r.log.Error(common.ErrOutOfRange)
			return nil, common.ErrOutOfRange
		}
		chainConfig.Core.TxSchedulerTimeout = parseUint
		changed = true
	}
	// [0, 60] tx_scheduler_validate_timeout
	if txSchedulerValidateTimeout, ok := params[paramNameTxSchedulerValidateTimeout]; ok {
		parseUint, err := strconv.ParseUint(string(txSchedulerValidateTimeout), 10, 0)
		if err != nil {
			r.log.Error(err)
			return nil, err
		}
		if parseUint > defaultConfigMaxSchedulerTimeout {
			r.log.Error(common.ErrOutOfRange)
			return nil, common.ErrOutOfRange
		}
		chainConfig.Core.TxSchedulerValidateTimeout = parseUint
		changed = true
	}

	if !changed {
		r.log.Error(common.ErrParams)
		return nil, common.ErrParams
	}
	// [end]
	result, err = setChainConfig(txSimContext, chainConfig)
	if err != nil {
		r.log.Errorf("core update update fail, %s, params %+v", err.Error(), params)
	} else {
		r.log.Infof("core update success, params %+v", params)
	}
	return result, err
}

// [block]
type ChainBlockRuntime struct {
	log protocol.Logger
}

func (r *ChainBlockRuntime) BlockUpdate(txSimContext protocol.TxSimContext, params map[string][]byte) (result []byte, err error) {
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
		r.log.Error(common.ErrParams)
		return nil, common.ErrParams
	}
	// [end]
	result, err = setChainConfig(txSimContext, chainConfig)
	if err != nil {
		r.log.Errorf("block update fail, %s, params %+v", err.Error(), params)
	} else {
		r.log.Infof("block update success, param ", params)
	}
	return result, err
}

// [trust_root]
type ChainTrustRootsRuntime struct {
	log protocol.Logger
}

// TrustRootAdd add trustRoot
func (r *ChainTrustRootsRuntime) TrustRootAdd(txSimContext protocol.TxSimContext, params map[string][]byte) (result []byte, err error) {
	// [start]
	chainConfig, err := getChainConfig(txSimContext, params)
	if err != nil {
		r.log.Error(err)
		return nil, err
	}

	orgId := string(params[paramNameOrgId])
	rootCaCrt := string(params[paramNameRoot])
	if utils.IsAnyBlank(orgId, rootCaCrt) {
		err = fmt.Errorf("%s, add trust root cert require param [%s, %s] not found", common.ErrParams.Error(), paramNameOrgId, paramNameRoot)
		r.log.Error(err)
		return nil, err
	}

	chainConfig.TrustRoots = append(chainConfig.TrustRoots, &configPb.TrustRootConfig{
		OrgId: orgId,
		Root:  rootCaCrt,
	})
	result, err = setChainConfig(txSimContext, chainConfig)
	if err != nil {
		r.log.Errorf("trust root add fail, %s, orgId[%s] cert[%s]", err.Error(), orgId, rootCaCrt)
	} else {
		r.log.Infof("trust root add success. orgId[%s] cert[%s]", orgId, rootCaCrt)
	}
	return result, err
}

// TrustRootUpdate update the trustRoot
func (r *ChainTrustRootsRuntime) TrustRootUpdate(txSimContext protocol.TxSimContext, params map[string][]byte) (result []byte, err error) {
	// [start]
	chainConfig, err := getChainConfig(txSimContext, params)
	if err != nil {
		r.log.Error(err)
		return nil, err
	}

	orgId := string(params[paramNameOrgId])
	rootCaCrt := string(params[paramNameRoot])
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
			if err != nil {
				r.log.Errorf("trust root update fail, %s, orgId[%s] cert[%s]", err.Error(), orgId, rootCaCrt)
			} else {
				r.log.Infof("trust root update success. orgId[%s] cert[%s]", orgId, rootCaCrt)
			}
			return result, err
		}
	}

	err = fmt.Errorf("%s can not found orgId[%s]", common.ErrParams.Error(), orgId)
	r.log.Error(err)
	return nil, err
}

// TrustRootDelete delete trustRoot
func (r *ChainTrustRootsRuntime) TrustRootDelete(txSimContext protocol.TxSimContext, params map[string][]byte) (result []byte, err error) {
	// [start]
	chainConfig, err := getChainConfig(txSimContext, params)
	if err != nil {
		r.log.Error(err)
		return nil, err
	}

	orgId := string(params[paramNameOrgId])
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
	if err != nil {
		r.log.Errorf("trust root delete fail, %s, orgId[%s] ", err.Error(), orgId)
	} else {
		r.log.Infof("trust root delete success. orgId[%s]", orgId)
	}
	return result, err
}

// [consensus]
type ChainConsensusRuntime struct {
	log protocol.Logger
}

// NodeIdAdd add nodeId
func (r *ChainConsensusRuntime) NodeIdAdd(txSimContext protocol.TxSimContext, params map[string][]byte) (result []byte, err error) {
	// [start]
	chainConfig, err := getChainConfig(txSimContext, params)
	if err != nil {
		r.log.Error(err)
		return nil, err
	}

	// verify params
	orgId := string(params[paramNameOrgId])
	nodeIdsStr := string(params[paramNameNodeIds]) // The addresses are separated by ","

	if utils.IsAnyBlank(orgId, nodeIdsStr) {
		err = fmt.Errorf("add node id failed, require param [%s, %s], but not found", paramNameOrgId, paramNameNodeIds)
		r.log.Error(err)
		return nil, err
	}

	nodeIdStrs := strings.Split(nodeIdsStr, ",")
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
		err = fmt.Errorf("add node id failed, org[%s] not found from nodes, you should call NodeOrgAdd first", orgId)
		r.log.Error(err)
		return nil, err
	}

	changed := false
	for _, nid := range nodeIdStrs {
		nid = strings.TrimSpace(nid)
		nodeIds := nodeConf.NodeId
		nodeIds = append(nodeIds, nid)
		nodeConf.NodeId = nodeIds
		nodes[index] = nodeConf
		chainConfig.Consensus.Nodes = nodes
		changed = true
	}

	if !changed {
		r.log.Error(common.ErrParams)
		return nil, common.ErrParams
	}
	// [end]
	result, err = setChainConfig(txSimContext, chainConfig)
	if err != nil {
		r.log.Errorf("node id add fail, %s, orgId[%s] nodeIdsStr[%s]", err.Error(), orgId, nodeIdsStr)
	} else {
		r.log.Infof("node id add success. orgId[%s] nodeIdsStr[%s]", orgId, nodeIdsStr)
	}
	return result, err
}

// NodeIdUpdate update nodeId
func (r *ChainConsensusRuntime) NodeIdUpdate(txSimContext protocol.TxSimContext, params map[string][]byte) (result []byte, err error) {
	// [start]
	chainConfig, err := getChainConfig(txSimContext, params)
	if err != nil {
		r.log.Error(err)
		return nil, err
	}

	// verify params
	orgId := string(params[paramNameOrgId])
	nodeId := string(params[paramNameNodeId])       // origin node id
	newNodeId := string(params[paramNameNewNodeId]) // new node id

	if utils.IsAnyBlank(orgId, nodeId, newNodeId) {
		err = fmt.Errorf("update node id failed, require param [%s, %s, %s], but not found", paramNameOrgId, paramNameNodeId, paramNameNewNodeId)
		r.log.Error(err)
		return nil, err
	}

	nodes := chainConfig.Consensus.Nodes
	nodeId = strings.TrimSpace(nodeId)
	newNodeId = strings.TrimSpace(newNodeId)

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
		err = fmt.Errorf("update node id failed, param orgId[%s] not found from nodes", orgId)
		r.log.Error(err)
		return nil, err
	}

	for j, nid := range nodeConf.NodeId {
		if nodeId == nid {
			nodeConf.NodeId[j] = newNodeId
			nodes[index] = nodeConf
			chainConfig.Consensus.Nodes = nodes
			result, err = setChainConfig(txSimContext, chainConfig)
			if err != nil {
				r.log.Errorf("node id update fail, %s, orgId[%s] addr[%s] newAddr[%s]", err.Error(), orgId, nid, newNodeId)
			} else {
				r.log.Infof("node id update success. orgId[%s] addr[%s] newAddr[%s]", orgId, nid, newNodeId)
			}
			return result, err
		}
	}

	err = fmt.Errorf("update node id failed, param orgId[%s] addr[%s] not found from nodes", orgId, nodeId)
	r.log.Error(err)
	return nil, err
}

// NodeIdDelete delete nodeId
func (r *ChainConsensusRuntime) NodeIdDelete(txSimContext protocol.TxSimContext, params map[string][]byte) (result []byte, err error) {
	// [start]
	chainConfig, err := getChainConfig(txSimContext, params)
	if err != nil {
		r.log.Error(err)
		return nil, err
	}
	// verify params
	orgId := string(params[paramNameOrgId])
	nodeId := string(params[paramNameNodeId])

	if utils.IsAnyBlank(orgId, nodeId) {
		err = fmt.Errorf("delete node id failed, require param [%s, %s], but not found", paramNameOrgId, paramNameNodeId)
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
		err = fmt.Errorf("delete node id failed, param orgId[%s] not found from nodes", orgId)
		r.log.Error(err)
		return nil, err
	}

	nodeIds := nodeConf.NodeId
	for j, nid := range nodeIds {
		if nodeId == nid {
			nodeConf.NodeId = append(nodeIds[:j], nodeIds[j+1:]...)
			nodes[index] = nodeConf
			chainConfig.Consensus.Nodes = nodes
			result, err = setChainConfig(txSimContext, chainConfig)
			if err != nil {
				r.log.Errorf("node id delete fail, %s, orgId[%s] addr[%s]", err.Error(), orgId, nid)
			} else {
				r.log.Infof("node id delete success. orgId[%s] addr[%s]", orgId, nid)
			}
			return result, err
		}
	}

	err = fmt.Errorf("delete node id failed, param orgId[%s] addr[%s] not found from nodes", orgId, nodeId)
	r.log.Error(err)
	return nil, err
}

// NodeOrgAdd add nodeOrg
func (r *ChainConsensusRuntime) NodeOrgAdd(txSimContext protocol.TxSimContext, params map[string][]byte) (result []byte, err error) {
	// [start]
	chainConfig, err := getChainConfig(txSimContext, params)
	if err != nil {
		r.log.Error(err)
		return nil, err
	}

	// verify params
	orgId := string(params[paramNameOrgId])
	nodeIdsStr := string(params[paramNameNodeIds])

	if utils.IsAnyBlank(orgId, nodeIdsStr) {
		err = fmt.Errorf("add node org failed, require param [%s, %s], but not found", paramNameOrgId, paramNameNodeIds)
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
		OrgId:  orgId,
		NodeId: make([]string, 0),
	}

	nodeIds := strings.Split(nodeIdsStr, ",")
	for _, nid := range nodeIds {
		nid = strings.TrimSpace(nid)
		if nid != "" {
			org.NodeId = append(org.NodeId, nid)
		}
	}
	if len(org.NodeId) > 0 {
		chainConfig.Consensus.Nodes = append(chainConfig.Consensus.Nodes, org)

		result, err = setChainConfig(txSimContext, chainConfig)
		if err != nil {
			r.log.Errorf("node org add fail, %s, orgId[%s] nodeIdsStr[%s]", err.Error(), orgId, nodeIdsStr)
		} else {
			r.log.Infof("node org add success. orgId[%s] nodeIdsStr[%s]", orgId, nodeIdsStr)
		}
		return result, err
	}

	r.log.Error(common.ErrParams)
	return nil, common.ErrParams
}

// NodeOrgUpdate update nodeOrg
func (r *ChainConsensusRuntime) NodeOrgUpdate(txSimContext protocol.TxSimContext, params map[string][]byte) (result []byte, err error) {
	// [start]
	chainConfig, err := getChainConfig(txSimContext, params)
	if err != nil {
		r.log.Error(err)
		return nil, err
	}

	// verify params
	changed := false
	orgId := string(params[paramNameOrgId])
	nodeIdsStr := string(params[paramNameNodeIds])

	if utils.IsAnyBlank(orgId, nodeIdsStr) {
		err = fmt.Errorf("update node org failed, require param [%s, %s], but not found", paramNameOrgId, paramNameNodeIds)
		r.log.Error(err)
		return nil, err
	}

	nodeIds := strings.Split(nodeIdsStr, ",")
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

	nodeConf.NodeId = []string{}
	for _, nid := range nodeIds {
		nid = strings.TrimSpace(nid)
		if nid != "" {
			nodeConf.NodeId = append(nodeConf.NodeId, nid)
			nodes[index] = nodeConf
			chainConfig.Consensus.Nodes = nodes
			changed = true
		}
	}

	if !changed {
		r.log.Error(common.ErrParams)
		return nil, common.ErrParams
	}
	// [end]
	result, err = setChainConfig(txSimContext, chainConfig)
	if err != nil {
		r.log.Errorf("node org update fail, %s, orgId[%s] nodeIdsStr[%s]", err.Error(), orgId, nodeIdsStr)
	} else {
		r.log.Infof("node org update success. orgId[%s] nodeIdsStr[%s]", orgId, nodeIdsStr)
	}
	return result, err
}

// NodeOrgDelete delete nodeOrg
func (r *ChainConsensusRuntime) NodeOrgDelete(txSimContext protocol.TxSimContext, params map[string][]byte) (result []byte, err error) {
	// [start]
	chainConfig, err := getChainConfig(txSimContext, params)
	if err != nil {
		r.log.Error(err)
		return nil, err
	}

	// verify params
	orgId := string(params[paramNameOrgId])

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
			if err != nil {
				r.log.Errorf("node org delete fail, %s, orgId[%s]", err.Error(), orgId)
			} else {
				r.log.Infof("node org delete success. orgId[%s]", orgId)
			}
			return result, err
		}
	}

	err = fmt.Errorf("delete node org failed, param orgId[%s] not found from nodes", orgId)
	r.log.Error(err)
	return nil, err
}

// ConsensusExtAdd add consensus extra
func (r *ChainConsensusRuntime) ConsensusExtAdd(txSimContext protocol.TxSimContext, params map[string][]byte) (result []byte, err error) {
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
		extConfigMap[v.Key] = string(v.Value)
	}

	// map is out of order, in order to ensure that each execution sequence is consistent, we need to sort
	sortedParams := sortedmap.NewStringKeySortedMapWithBytesData(params)
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
			Value: []byte(value),
		})
		chainConfig.Consensus.ExtConfig = extConfig
		changed = true
		return true
	})
	if parseParamErr != nil {
		return nil, parseParamErr
	}

	if !changed {
		r.log.Error(common.ErrParams)
		return nil, common.ErrParams
	}
	// [end]
	result, err = setChainConfig(txSimContext, chainConfig)
	if err != nil {
		r.log.Errorf("consensus ext add fail, %s, params %+v", err.Error(), params)
	} else {
		r.log.Infof("consensus ext add success. params %+v", params)
	}
	return result, err
}

// ConsensusExtUpdate update consensus extra
func (r *ChainConsensusRuntime) ConsensusExtUpdate(txSimContext protocol.TxSimContext, params map[string][]byte) (result []byte, err error) {
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
		extConfigMap[v.Key] = string(v.Value)
	}

	for key, val := range params {
		if _, ok := extConfigMap[key]; !ok {
			continue
		}
		for i, config := range extConfig {
			if key == config.Key {
				extConfig[i] = &commonPb.KeyValuePair{
					Key:   key,
					Value: []byte(val),
				}
				chainConfig.Consensus.ExtConfig = extConfig
				changed = true
				break
			}
		}
	}

	if !changed {
		r.log.Error(common.ErrParams)
		return nil, common.ErrParams
	}
	// [end]
	result, err = setChainConfig(txSimContext, chainConfig)
	if err != nil {
		r.log.Errorf("consensus ext update fail, %s, params %+v", err.Error(), params)
	} else {
		r.log.Infof("consensus ext update success. params %+v", params)
	}
	return result, err
}

// ConsensusExtDelete delete consensus extra
func (r *ChainConsensusRuntime) ConsensusExtDelete(txSimContext protocol.TxSimContext, params map[string][]byte) (result []byte, err error) {
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
		extConfigMap[v.Key] = string(v.Value)
	}

	for key := range params {
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
		r.log.Error(common.ErrParams)
		return nil, common.ErrParams
	}
	// [end]
	result, err = setChainConfig(txSimContext, chainConfig)
	if err != nil {
		r.log.Errorf("consensus ext delete fail, %s, params %+v", err.Error(), params)
	} else {
		r.log.Infof("consensus ext delete success. params %+v", params)
	}
	return result, err
}

// [permissions]
type ChainPermissionRuntime struct {
	log protocol.Logger
}

// ResourcePolicyAdd add permission
func (r *ChainConsensusRuntime) ResourcePolicyAdd(txSimContext protocol.TxSimContext, params map[string][]byte) (result []byte, err error) {
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

	sortedParams := sortedmap.NewStringKeySortedMapWithBytesData(params)
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
		r.log.Error(common.ErrParams)
		return nil, common.ErrParams
	}
	// [end]
	result, err = setChainConfig(txSimContext, chainConfig)
	if err != nil {
		r.log.Errorf("resource policy add fail, %s, params %+v", err.Error(), params)
	} else {
		r.log.Infof("resource policy add success. params %+v", params)
	}
	return result, err
}

// ResourcePolicyUpdate update resource policy
func (r *ChainConsensusRuntime) ResourcePolicyUpdate(txSimContext protocol.TxSimContext, params map[string][]byte) (result []byte, err error) {
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

	sortedParams := sortedmap.NewStringKeySortedMapWithBytesData(params)
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
		r.log.Error(common.ErrParams)
		return nil, common.ErrParams
	}
	// [end]
	result, err = setChainConfig(txSimContext, chainConfig)
	if err != nil {
		r.log.Errorf("resource policy update fail, %s, params %+v", err.Error(), params)
	} else {
		r.log.Infof("resource policy update success. params %+v", params)
	}
	return result, err
}

// ResourcePolicyDelete delete permission
func (r *ChainConsensusRuntime) ResourcePolicyDelete(txSimContext protocol.TxSimContext, params map[string][]byte) (result []byte, err error) {
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
	sortedParams := sortedmap.NewStringKeySortedMapWithBytesData(params)
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
		r.log.Error(common.ErrParams)
		return nil, common.ErrParams
	}
	// [end]
	result, err = setChainConfig(txSimContext, chainConfig)
	if err != nil {
		r.log.Errorf("resource policy delete fail, %s, params %+v", err.Error(), params)
	} else {
		r.log.Infof("resource policy delete success. params %+v", params)
	}
	return result, err
}
