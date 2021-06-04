/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package utils

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
	"unicode"

	"chainmaker.org/chainmaker-go/common/crypto/hash"
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	configPb "chainmaker.org/chainmaker-go/pb/protogo/config"
	"chainmaker.org/chainmaker-go/pb/protogo/consensus"
	dpospb "chainmaker.org/chainmaker-go/pb/protogo/dpos"
	"github.com/gogo/protobuf/proto"
	"github.com/mr-tron/base58/base58"
)

// default timestamp is "2020-11-30 0:0:0"
const (
	defaultTimestamp           = int64(1606669261)
	errMsgMarshalChainConfFail = "proto marshal chain config failed, %s"
	keyERC20Total              = "erc20.total"
	keyERC20Owner              = "erc20.owner"
	keyERC20Decimals           = "erc20.decimals"
	keyStakeMinSelfDelegation  = "stake.minSelfDelegation"
	keyStakeEpochValidatorNum  = "stake.epochValidatorNum"
	keyStakeEpochBlockNum      = "stake.epochBlockNum"
	keyStakeCandidate          = "stake.candidate"
)

// CreateGenesis create genesis block (with read-write set) based on chain config
func CreateGenesis(cc *configPb.ChainConfig) (*commonPb.Block, []*commonPb.TxRWSet, error) {
	var (
		err      error
		tx       *commonPb.Transaction
		rwSet    *commonPb.TxRWSet
		txHash   []byte
		hashType = cc.Crypto.Hash
	)

	// generate config tx, read-write set, and hash
	if tx, err = genConfigTx(cc); err != nil {
		return nil, nil, fmt.Errorf("create genesis config tx failed, %s", err)
	}

	if rwSet, err = genConfigTxRWSet(cc); err != nil {
		return nil, nil, fmt.Errorf("create genesis config tx read-write set failed, %s", err)
	}

	if tx.Result.RwSetHash, err = CalcRWSetHash(cc.Crypto.Hash, rwSet); err != nil {
		return nil, nil, fmt.Errorf("calculate genesis config tx read-write set hash failed, %s", err)
	}

	if txHash, err = CalcTxHash(cc.Crypto.Hash, tx); err != nil {
		return nil, nil, fmt.Errorf("calculate tx hash failed, %s", err)
	}

	// generate genesis block
	genesisBlock := &commonPb.Block{
		Header: &commonPb.BlockHeader{
			ChainId:        cc.ChainId,
			BlockHeight:    0,
			PreBlockHash:   nil,
			BlockHash:      nil,
			PreConfHeight:  0,
			BlockVersion:   []byte(cc.Version),
			DagHash:        nil,
			RwSetRoot:      nil,
			TxRoot:         nil,
			BlockTimestamp: defaultTimestamp,
			Proposer:       nil,
			ConsensusArgs:  nil,
			TxCount:        1,
			Signature:      nil,
		},
		Dag: &commonPb.DAG{
			Vertexes: []*commonPb.DAG_Neighbor{
				&commonPb.DAG_Neighbor{
					Neighbors: nil,
				},
			},
		},
		Txs: []*commonPb.Transaction{tx},
	}

	if genesisBlock.Header.TxRoot, err = hash.GetMerkleRoot(hashType, [][]byte{txHash}); err != nil {
		return nil, nil, fmt.Errorf("calculate genesis block tx root failed, %s", err)
	}

	if genesisBlock.Header.RwSetRoot, err = CalcRWSetRoot(hashType, genesisBlock.Txs); err != nil {
		return nil, nil, fmt.Errorf("calculate genesis block rwset root failed, %s", err)
	}

	if genesisBlock.Header.DagHash, err = CalcDagHash(hashType, genesisBlock.Dag); err != nil {
		return nil, nil, fmt.Errorf("calculate genesis block DAG hash failed, %s", err)
	}

	if genesisBlock.Header.BlockHash, err = CalcBlockHash(hashType, genesisBlock); err != nil {
		return nil, nil, fmt.Errorf("calculate genesis block hash failed, %s", err)
	}

	return genesisBlock, []*commonPb.TxRWSet{rwSet}, nil
}

func genConfigTx(cc *configPb.ChainConfig) (*commonPb.Transaction, error) {
	var (
		err          error
		ccBytes      []byte
		payloadBytes []byte
	)

	if ccBytes, err = proto.Marshal(cc); err != nil {
		return nil, fmt.Errorf(errMsgMarshalChainConfFail, err.Error())
	}

	payload := &commonPb.SystemContractPayload{
		ChainId:      cc.ChainId,
		ContractName: commonPb.ContractName_SYSTEM_CONTRACT_CHAIN_CONFIG.String(),
		Method:       "Genesis",
		Parameters:   make([]*commonPb.KeyValuePair, 0),
		Sequence:     cc.Sequence,
		Endorsement:  nil,
	}
	payload.Parameters = append(payload.Parameters, &commonPb.KeyValuePair{
		Key:   commonPb.ContractName_SYSTEM_CONTRACT_CHAIN_CONFIG.String(),
		Value: cc.String(),
	})

	if payloadBytes, err = proto.Marshal(payload); err != nil {
		return nil, fmt.Errorf(errMsgMarshalChainConfFail, err.Error())
	}

	tx := &commonPb.Transaction{
		Header: &commonPb.TxHeader{
			ChainId:        cc.ChainId,
			Sender:         nil,
			TxType:         commonPb.TxType_UPDATE_CHAIN_CONFIG,
			TxId:           GetTxIdWithSeed(int64(defaultTimestamp)),
			Timestamp:      defaultTimestamp,
			ExpirationTime: -1,
		},
		RequestPayload:   payloadBytes,
		RequestSignature: nil,
		Result: &commonPb.Result{
			Code: commonPb.TxStatusCode_SUCCESS,
			ContractResult: &commonPb.ContractResult{
				Code:    commonPb.ContractResultCode_OK,
				Message: commonPb.ContractResultCode_OK.String(),
				Result:  ccBytes,
			},
			RwSetHash: nil,
		},
	}

	return tx, nil
}

func genConfigTxRWSet(cc *configPb.ChainConfig) (*commonPb.TxRWSet, error) {
	var (
		err     error
		ccBytes []byte
	)

	if ccBytes, err = proto.Marshal(cc); err != nil {
		return nil, fmt.Errorf(errMsgMarshalChainConfFail, err.Error())
	}

	var (
		erc20Config *ERC20Config
		stakeConfig *StakeConfig
	)
	if cc.Consensus.Type == consensus.ConsensusType_DPOS {
		// preCheck
		erc20Config, err = loadERC20Config(cc.Consensus.ExtConfig)
		if err != nil {
			return nil, err
		}
		stakeConfig, err = loadStakeConfig(cc.Consensus.ExtConfig)
		if err != nil {
			return nil, err
		}
		// postCheck
	}
	rwSets, err := totalTxRWSet(ccBytes, erc20Config, stakeConfig)
	if err != nil {
		return nil, err
	}
	set := &commonPb.TxRWSet{
		TxId:     GetTxIdWithSeed(int64(defaultTimestamp)),
		TxReads:  nil,
		TxWrites: rwSets,
	}
	return set, nil
}

// ERC20Config for DPoS
type ERC20Config struct {
	total    string
	owner    string
	decimals string
}

// legal check field is legal
func (e *ERC20Config) legal() error {
	// total and decimals must be number
	// owner must be base58 encode
	_, err := base58.Decode(e.owner)
	if err != nil {
		return fmt.Errorf("config of owner[%s] is not in base58 format", e.owner)
	}
	if !isNumber(e.total) {
		return fmt.Errorf("config of total[%s] is not number", e.total)
	}
	if !isNumber(e.decimals) {
		return fmt.Errorf("config of decimals[%s] is not number", e.decimals)
	}
	return nil
}

// toTxWrites convert to TxWrites
func (e *ERC20Config) toTxWrites() []*commonPb.TxWrite {
	return []*commonPb.TxWrite{
		{
			Key:          []byte("OWN"), // equal with native.KeyOwner
			Value:        []byte(e.owner),
			ContractName: commonPb.ContractName_SYSTEM_CONTRACT_DPOS_ERC20.String(),
		},
		{
			Key:          []byte("DEC"), // equal with native.KeyDecimals
			Value:        []byte(e.decimals),
			ContractName: commonPb.ContractName_SYSTEM_CONTRACT_DPOS_ERC20.String(),
		},
		{
			Key:          []byte("TS"), // equal with native.KeyTotalSupply
			Value:        []byte(e.total),
			ContractName: commonPb.ContractName_SYSTEM_CONTRACT_DPOS_ERC20.String(),
		},
	}
}

// isNumber check str
func isNumber(str string) bool {
	for _, x := range []rune(str) {
		if !unicode.IsDigit(x) {
			return false
		}
	}
	return true
}

// loadERC20Config load config of erc20 contract
func loadERC20Config(consensusExtConfig []*commonPb.KeyValuePair) (*ERC20Config, error) {
	/**
	  erc20合约的配置
	  ext_config: # 扩展字段，记录难度、奖励等其他类共识算法配置
	    - key: erc20.total
	      value: 1000000000000
	    - key: erc20.owner
	      value: 5pQfwDwtyA
	    - key: erc20.decimals
	      value: 18
		- key: erc20.account:<addr1>
		  value: 8000
		- key: erc20.account:<addr2>
		  value: 8000
	*/
	config := &ERC20Config{}
	for i := 0; i < len(consensusExtConfig); i++ {
		keyValuePair := consensusExtConfig[i]
		switch keyValuePair.Key {
		case keyERC20Total:
			config.total = keyValuePair.Value
		case keyERC20Owner:
			config.owner = keyValuePair.Value
		case keyERC20Decimals:
			config.decimals = keyValuePair.Value
		}
	}
	// check config is legal
	if err := config.legal(); err != nil {
		return nil, err
	}
	return config, nil
}

type StakeConfig struct {
	minSelfDelegation string
	validatorNum      uint64
	eachEpochNum      uint64
	candidates        []*dpospb.CandidateInfo
}

func (s *StakeConfig) toTxWrites() ([]*commonPb.TxWrite, error) {
	var (
		valNum   = make([]byte, 8)
		epochNum = make([]byte, 8)
	)
	binary.BigEndian.PutUint64(valNum, s.validatorNum)
	binary.BigEndian.PutUint64(epochNum, s.eachEpochNum)

	// 1. add property in rwSets
	rwSets := []*commonPb.TxWrite{
		{
			ContractName: commonPb.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String(),
			Key:          []byte(commonPb.StakePrefix_Prefix_MinSelfDelegation.String()),
			Value:        []byte(s.minSelfDelegation),
		},
		{
			ContractName: commonPb.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String(),
			Key:          []byte(commonPb.StakePrefix_Prefix_Validator.String()), // todo, will modify validatorNUm
			Value:        valNum,
		},
		{
			ContractName: commonPb.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String(),
			Key:          []byte(commonPb.StakePrefix_Prefix_Validator.String()), // todo, will modify epochNum
			Value:        epochNum,
		},
	}

	// 2. add validatorInfo in rwSet
	validators := make([][]byte, 0, len(s.candidates))
	for _, candidate := range s.candidates {
		bz, err := proto.Marshal(&commonPb.Validator{
			Jailed:                     false,
			Status:                     commonPb.BondStatus_Bonded,
			Tokens:                     candidate.Weight,
			ValidatorAddress:           candidate.PeerID,
			DelegatorShares:            candidate.Weight,
			SelfDelegation:             candidate.Weight,
			UnbondingEpochID:           math.MaxInt64,
			UnbondingCompletionEpochID: math.MaxUint64,
		})
		if err != nil {
			return nil, err
		}
		validators = append(validators, bz)
	}
	for i, validator := range s.candidates {
		rwSets = append(rwSets, &commonPb.TxWrite{
			ContractName: commonPb.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String(),
			Key:          []byte(commonPb.StakePrefix_Prefix_Validator.String() + validator.PeerID), // todo, will modify epochNum
			Value:        validators[i],
		})
	}

	// 3. add delegationInfo in rwSet
	delegations := make([][]byte, 0, len(s.candidates))
	for _, candidate := range s.candidates {
		bz, err := proto.Marshal(&commonPb.Delegation{
			DelegatorAddress: candidate.PeerID,
			ValidatorAddress: candidate.PeerID,
			Shares:           candidate.Weight,
		})
		if err != nil {
			return nil, err
		}
		delegations = append(delegations, bz)
	}
	for i, validator := range s.candidates {
		rwSets = append(rwSets, &commonPb.TxWrite{
			ContractName: commonPb.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String(),
			Key:          []byte(commonPb.StakePrefix_Prefix_Delegation.String() + validator.PeerID + validator.PeerID), // key: prefix|delegator|validator
			Value:        delegations[i],                                                                                // val: delegation info
		})
	}

	// 4. add epoch info
	epochID := make([]byte, 8)
	valAddrs := make([]string, 0, len(s.candidates))
	for _, v := range s.candidates {
		valAddrs = append(valAddrs, v.PeerID)
	}
	epochInfo, err := proto.Marshal(&commonPb.Epoch{
		EpochID:               0,
		ProposerVector:        valAddrs,
		NextEpochCreateHeight: s.eachEpochNum,
	})
	if err != nil {
		return nil, err
	}
	rwSets = append(rwSets, &commonPb.TxWrite{
		ContractName: commonPb.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String(),
		Key:          []byte(commonPb.StakePrefix_Prefix_Curr_Epoch.String()), // key: prefix
		Value:        epochInfo,                                               // val: epochInfo
	})
	rwSets = append(rwSets, &commonPb.TxWrite{
		ContractName: commonPb.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String(),
		Key:          append([]byte(commonPb.StakePrefix_Prefix_Epoch_Record.String()), epochID...), // key: prefix|epochID
		Value:        epochInfo,                                                                     // val: epochInfo
	})
	return rwSets, nil
}

func loadStakeConfig(consensusExtConfig []*commonPb.KeyValuePair) (*StakeConfig, error) {
	/**
	  stake合约的配置
	  ext_config: # 扩展字段，记录难度、奖励等其他类共识算法配置
	    - key: stake.minSelfDelegation
	      value: 1000000000000
	    - key: stake.epochValidatorNum
	      value: 10
	    - key: stake.epochBlockNum
	      value: 2000
		- key: stake.candidate:<addr1>
	      value: 800000
		- key: stake.candidate:<addr2>
	      value: 600000
	*/
	config := StakeConfig{}
	for _, kv := range consensusExtConfig {
		switch kv.Key {
		case keyStakeEpochBlockNum:
			val, err := strconv.ParseUint(kv.Value, 10, 64)
			if err != nil {
				return nil, err
			}
			config.eachEpochNum = val
		case keyStakeEpochValidatorNum:
			val, err := strconv.ParseUint(kv.Value, 10, 64)
			if err != nil {
				return nil, err
			}
			config.validatorNum = val
		case keyStakeMinSelfDelegation:
			if err := isValidBigInt(kv.Value); err != nil {
				return nil, fmt.Errorf("%s error, reason: %s", keyStakeMinSelfDelegation, err)
			}
			config.minSelfDelegation = kv.Value
		default:
			if !strings.HasPrefix(kv.Key, keyStakeCandidate) {
				continue
			}
			values := strings.Split(kv.Key, ":")
			if len(values) != 2 {
				return nil, fmt.Errorf("stake.candidate config error, actual: %s, expect: %s:<addr1>", kv.Key, keyStakeCandidate)
			}
			if err := isValidBigInt(kv.Value); err != nil {
				return nil, fmt.Errorf("stake.candidate amount error, reason: %s", err)
			}
			config.candidates = append(config.candidates, &dpospb.CandidateInfo{
				PeerID: values[1], Weight: kv.Value,
			})
		}
	}
	return &config, nil
}

func isValidBigInt(val string) error {
	_, ok := big.NewInt(0).SetString(val, 10)
	if !ok {
		return fmt.Errorf("parse string to big.Int failed, actual: %s", val)
	}
	return nil
}

func totalTxRWSet(chainConfigBytes []byte, erc20Config *ERC20Config, stakeConfig *StakeConfig) ([]*commonPb.TxWrite, error) {
	txWrites := make([]*commonPb.TxWrite, 0)
	txWrites = append(txWrites, &commonPb.TxWrite{
		Key:          []byte(commonPb.ContractName_SYSTEM_CONTRACT_CHAIN_CONFIG.String()),
		Value:        chainConfigBytes,
		ContractName: commonPb.ContractName_SYSTEM_CONTRACT_CHAIN_CONFIG.String(),
	})
	if erc20Config != nil {
		erc20ConfigTxWrites := erc20Config.toTxWrites()
		txWrites = append(txWrites, erc20ConfigTxWrites...)
	}
	if stakeConfig != nil {
		stakeConfigTxWrites, err := stakeConfig.toTxWrites()
		if err != nil {
			return nil, err
		}
		txWrites = append(txWrites, stakeConfigTxWrites...)
	}
	return txWrites, nil
}
