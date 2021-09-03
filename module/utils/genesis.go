/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package utils

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"sort"
	"strconv"
	"strings"

	"chainmaker.org/chainmaker/pb-go/v2/syscontract"

	"chainmaker.org/chainmaker/protocol/v2"

	"chainmaker.org/chainmaker/common/v2/crypto/hash"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	configPb "chainmaker.org/chainmaker/pb-go/v2/config"
	"chainmaker.org/chainmaker/pb-go/v2/consensus"
	dpospb "chainmaker.org/chainmaker/pb-go/v2/consensus/dpos"
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
	keyERC20Acc                = "erc20.account:"
	keyStakeMinSelfDelegation  = "stake.minSelfDelegation"
	keyStakeEpochValidatorNum  = "stake.epochValidatorNum"
	keyStakeEpochBlockNum      = "stake.epochBlockNum"
	keyStakeUnbondingEpochNum  = "stake.completionUnbondingEpochNum"
	keyStakeCandidate          = "stake.candidate"
	keyStakeConfigNodeID       = "stake.nodeID"

	keyCurrentEpoch         = "CE"
	keyMinSelfDelegation    = "MSD"
	keyEpochFormat          = "E/%s"
	keyDelegationFormat     = "D/%s/%s"
	keyValidatorFormat      = "V/%s"
	keyEpochValidatorNumber = "EVN"
	keyEpochBlockNumber     = "EBN"
	keyUnbondingEpochNumber = "UEN"
	keyNodeIDFormat         = "N/%s"
	keyRevNodeFormat        = "NR/%s"
)

const (
	defaultDPoSMinSelfDelegation            = "250000000000000000000000"
	defaultDPoSEpochBlockNumber             = 1000
	defaultDPoSEpochValidatorNumber         = 4
	defaultDPoSCompletionUnboundingEpochNum = 1
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
			BlockType:      commonPb.BlockType_CONFIG_BLOCK,
			PreBlockHash:   nil,
			BlockHash:      nil,
			PreConfHeight:  0,
			BlockVersion:   protocol.DefaultBlockVersion, //兼容v1.1.0，否则添加新节点会导致创世快不一致。
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
				{
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
		err     error
		ccBytes []byte
		//payloadBytes []byte
	)

	if ccBytes, err = proto.Marshal(cc); err != nil {
		return nil, fmt.Errorf(errMsgMarshalChainConfFail, err.Error())
	}

	payload := &commonPb.Payload{
		ChainId:      cc.ChainId,
		ContractName: syscontract.SystemContract_CHAIN_CONFIG.String(),
		Method:       "Genesis",
		Parameters:   make([]*commonPb.KeyValuePair, 0),
		Sequence:     cc.Sequence,
		TxType:       commonPb.TxType_INVOKE_CONTRACT,
		TxId:         GetTxIdWithSeed(int64(defaultTimestamp)),
		Timestamp:    defaultTimestamp,
	}
	payload.Parameters = append(payload.Parameters, &commonPb.KeyValuePair{
		Key:   syscontract.SystemContract_CHAIN_CONFIG.String(),
		Value: []byte(cc.String()),
	})

	//if payloadBytes, err = proto.Marshal(payload); err != nil {
	//	return nil, fmt.Errorf(errMsgMarshalChainConfFail, err.Error())
	//}

	tx := &commonPb.Transaction{
		Payload: payload,
		Result: &commonPb.Result{
			Code: commonPb.TxStatusCode_SUCCESS,
			ContractResult: &commonPb.ContractResult{
				Code: uint32(0),

				Result: ccBytes,
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
		erc20Config, err = loadERC20Config(cc.Consensus.DposConfig)
		if err != nil {
			return nil, err
		}
		stakeConfig, err = loadStakeConfig(cc.Consensus.DposConfig)
		if err != nil {
			return nil, err
		}
		// check erc20 config
		if err = erc20Config.legal(); err != nil {
			return nil, err
		}
		// check stake's sum with erc20
		stakeContractAddr := getContractAddress()
		tokenInERC20, stackContractToken := erc20Config.loadToken(stakeContractAddr), stakeConfig.getSumToken()
		if tokenInERC20 == nil || stackContractToken == nil {
			return nil, fmt.Errorf("token of stake contract account[%s] is nil", stakeContractAddr)
		}
		if tokenInERC20.Cmp(stackContractToken) != 0 {
			return nil, fmt.Errorf("token of stake contract account[%s] is not equal, erc20[%s] stake[%s]",
				stakeContractAddr, tokenInERC20.String(), stackContractToken)
		}
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
	total    *BigInteger
	owner    string
	decimals *BigInteger
	accounts []*struct {
		address string
		token   *BigInteger
	}
}

func newERC20Config() *ERC20Config {
	return &ERC20Config{
		accounts: make([]*struct {
			address string
			token   *BigInteger
		}, 0),
	}
}

func (e *ERC20Config) addAccount(address string, token *BigInteger) error {
	// 需要判断是否有重复，每个地址只允许配置一次token
	for i := 0; i < len(e.accounts); i++ {
		if e.accounts[i].address == address {
			return fmt.Errorf("token of address[%s] cannot be set more than once", address)
		}
	}
	e.accounts = append(e.accounts, &struct {
		address string
		token   *BigInteger
	}{address: address, token: token})
	return nil
}

// toTxWrites convert to TxWrites
func (e *ERC20Config) toTxWrites() []*commonPb.TxWrite {
	contractName := syscontract.SystemContract_DPOS_ERC20.String()
	txWrites := []*commonPb.TxWrite{
		{
			Key:          []byte("OWN"), // equal with native.KeyOwner
			Value:        []byte(e.owner),
			ContractName: contractName,
		},
		{
			Key:          []byte("DEC"), // equal with native.KeyDecimals
			Value:        []byte(e.decimals.String()),
			ContractName: contractName,
		},
		{
			Key:          []byte("TS"), // equal with native.KeyTotalSupply
			Value:        []byte(e.total.String()),
			ContractName: contractName,
		},
	}

	// 添加accounts的读写集
	sort.SliceStable(e.accounts, func(i, j int) bool {
		return e.accounts[i].address < e.accounts[j].address
	})
	for i := 0; i < len(e.accounts); i++ {
		txWrites = append(txWrites, &commonPb.TxWrite{
			Key:          []byte(fmt.Sprintf("B/%s", e.accounts[i].address)),
			Value:        []byte(e.accounts[i].token.String()),
			ContractName: contractName,
		})
	}
	return txWrites
}

func (e *ERC20Config) loadToken(address string) *BigInteger {
	for i := 0; i < len(e.accounts); i++ {
		if e.accounts[i].address == address {
			return e.accounts[i].token
		}
	}
	return nil
}

func (e *ERC20Config) legal() error {
	if len(e.accounts) == 0 {
		return fmt.Errorf("account's size must more than zero")
	}
	// 其他信息已校验过，当前只需要校验所有账户的token和为total即可
	sum := NewZeroBigInteger()
	for i := 0; i < len(e.accounts); i++ {
		sum.Add(e.accounts[i].token)
	}
	// 比较sum与total
	if sum.Cmp(e.total) != 0 {
		return fmt.Errorf("sum of token is not equal with total, sum[%s] total[%s]", sum.String(), e.total.String())
	}
	return nil
}

// loadERC20Config load config of erc20 contract
func loadERC20Config(consensusExtConfig []*configPb.ConfigKeyValue) (*ERC20Config, error) {
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
	config := newERC20Config()
	for i := 0; i < len(consensusExtConfig); i++ {
		keyValuePair := consensusExtConfig[i]
		switch keyValuePair.Key {
		case keyERC20Total:
			config.total = NewBigInteger(string(keyValuePair.Value))
			if config.total == nil || config.total.Cmp(NewZeroBigInteger()) <= 0 {
				return nil, fmt.Errorf("total config of dpos must more than zero")
			}
		case keyERC20Owner:
			config.owner = string(keyValuePair.Value)
			_, err := base58.Decode(config.owner)
			if err != nil {
				return nil, fmt.Errorf("config of owner[%s] is not in base58 format", config.owner)
			}
		case keyERC20Decimals:
			config.decimals = NewBigInteger(string(keyValuePair.Value))
			if config.decimals == nil || config.decimals.Cmp(NewZeroBigInteger()) < 0 {
				return nil, fmt.Errorf("decimals config of dpos must more than -1")
			}
		default:
			if strings.HasPrefix(keyValuePair.Key, keyERC20Acc) {
				accAddress := keyValuePair.Key[len(keyERC20Acc):]
				if accAddress == syscontract.SystemContract_DPOS_STAKE.String() {
					accAddress = getContractAddress()
				}
				_, err := base58.Decode(accAddress)
				if err != nil {
					return nil, fmt.Errorf("account [%s] is not in base58 format", accAddress)
				}
				token := NewBigInteger(string(keyValuePair.Value))
				if token == nil || token.Cmp(NewZeroBigInteger()) <= 0 {
					return nil, fmt.Errorf("token must more than zero, address[%s] token[%s]",
						accAddress, keyValuePair.Value)
				}
				err = config.addAccount(accAddress, token)
				if err != nil {
					return nil, err
				}
			}
		}
	}
	return config, nil
}

type StakeConfig struct {
	minSelfDelegation string
	validatorNum      uint64
	eachEpochNum      uint64
	unbondingEpochNum uint64
	candidates        []*dpospb.CandidateInfo
	nodeIDs           map[string]string // userAddr => nodeID
}

func (s *StakeConfig) toTxWrites() ([]*commonPb.TxWrite, error) {
	var (
		valNum                = make([]byte, 8)
		epochNum              = make([]byte, 8)
		completeUnboundingNum = make([]byte, 8)
	)
	binary.BigEndian.PutUint64(valNum, s.validatorNum)
	binary.BigEndian.PutUint64(epochNum, s.eachEpochNum)
	binary.BigEndian.PutUint64(completeUnboundingNum, s.unbondingEpochNum)

	// 1. add property in rwSets
	rwSets := []*commonPb.TxWrite{
		{
			ContractName: syscontract.SystemContract_DPOS_STAKE.String(),
			Key:          []byte(keyMinSelfDelegation),
			Value:        []byte(s.minSelfDelegation),
		},
		{
			ContractName: syscontract.SystemContract_DPOS_STAKE.String(),
			Key:          []byte(keyEpochValidatorNumber),
			Value:        valNum,
		},
		{
			ContractName: syscontract.SystemContract_DPOS_STAKE.String(),
			Key:          []byte(keyEpochBlockNumber),
			Value:        epochNum,
		},
		{
			ContractName: syscontract.SystemContract_DPOS_STAKE.String(),
			Key:          []byte(keyUnbondingEpochNumber),
			Value:        completeUnboundingNum,
		},
	}

	// 2. add validatorInfo, delegationInfo in rwSet
	sort.SliceStable(s.candidates, func(i, j int) bool {
		return s.candidates[i].PeerId < s.candidates[j].PeerId
	})
	validators := make([][]byte, 0, len(s.candidates))
	delegations := make([][]byte, 0, len(s.candidates))
	for _, candidate := range s.candidates {
		bz, err := proto.Marshal(&syscontract.Validator{
			Jailed:                     false,
			Status:                     syscontract.BondStatus_BONDED,
			Tokens:                     candidate.Weight,
			ValidatorAddress:           candidate.PeerId,
			DelegatorShares:            candidate.Weight,
			SelfDelegation:             candidate.Weight,
			UnbondingEpochId:           math.MaxInt64,
			UnbondingCompletionEpochId: math.MaxUint64,
		})
		if err != nil {
			return nil, err
		}
		validators = append(validators, bz)

		delegateBz, err := proto.Marshal(&syscontract.Delegation{
			DelegatorAddress: candidate.PeerId,
			ValidatorAddress: candidate.PeerId,
			Shares:           candidate.Weight,
		})
		if err != nil {
			return nil, err
		}
		delegations = append(delegations, delegateBz)
	}
	for i, validator := range s.candidates {
		rwSets = append(rwSets, &commonPb.TxWrite{
			ContractName: syscontract.SystemContract_DPOS_STAKE.String(),
			Key:          []byte(fmt.Sprintf(keyValidatorFormat, validator.PeerId)),
			Value:        validators[i],
		})
		// key: prefix|delegator|validator
		key := []byte(fmt.Sprintf(keyDelegationFormat, validator.PeerId, validator.PeerId))
		rwSets = append(rwSets, &commonPb.TxWrite{
			ContractName: syscontract.SystemContract_DPOS_STAKE.String(),
			Key:          key,
			Value:        delegations[i], // val: delegation info
		})
	}

	// 4. add epoch info
	valAddrs := make([]string, 0, len(s.candidates))
	for _, v := range s.candidates {
		valAddrs = append(valAddrs, v.PeerId)
	}
	epochInfo, err := proto.Marshal(&syscontract.Epoch{
		EpochId:               0,
		ProposerVector:        valAddrs,
		NextEpochCreateHeight: s.eachEpochNum,
	})
	if err != nil {
		return nil, err
	}
	rwSets = append(rwSets, &commonPb.TxWrite{
		ContractName: syscontract.SystemContract_DPOS_STAKE.String(),
		Key:          []byte(keyCurrentEpoch), // key: prefix
		Value:        epochInfo,               // val: epochInfo
	})
	rwSets = append(rwSets, &commonPb.TxWrite{
		ContractName: syscontract.SystemContract_DPOS_STAKE.String(),
		Key:          []byte(fmt.Sprintf(keyEpochFormat, "0")), // key: prefix|epochID
		Value:        epochInfo,                                // val: epochInfo
	})

	for _, addr := range valAddrs {
		rwSets = append(rwSets, &commonPb.TxWrite{
			ContractName: syscontract.SystemContract_DPOS_STAKE.String(),
			Key:          []byte(fmt.Sprintf(keyNodeIDFormat, addr)), // key: prefix|addr
			Value:        []byte(s.nodeIDs[addr]),                    // val: nodeID
		})
		rwSets = append(rwSets, &commonPb.TxWrite{
			ContractName: syscontract.SystemContract_DPOS_STAKE.String(),
			Key:          []byte(fmt.Sprintf(keyRevNodeFormat, s.nodeIDs[addr])), // key: prefix|nodeID
			Value:        []byte(addr),                                           // val: addr
		})
	}
	return rwSets, nil
}

// getContractAddress 返回质押合约地址
func getContractAddress() string {
	bz := sha256.Sum256([]byte(syscontract.SystemContract_DPOS_STAKE.String()))
	return base58.Encode(bz[:])
}

// getSumToken 返回所有token的值
func (s *StakeConfig) getSumToken() *BigInteger {
	sum := NewZeroBigInteger()
	for i := 0; i < len(s.candidates); i++ {
		sum.Add(NewBigInteger(s.candidates[i].Weight))
	}
	return sum
}

func (s *StakeConfig) setCandidate(key, value string) error {
	values := strings.Split(key, ":")
	if len(values) != 2 {
		return fmt.Errorf("stake.candidate config error, actual: %s, expect: %s:<addr1>", key, keyStakeCandidate)
	}
	if err := isValidBigInt(value); err != nil {
		return fmt.Errorf("stake.candidate amount error, reason: %s", err)
	}
	s.candidates = append(s.candidates, &dpospb.CandidateInfo{
		PeerId: values[1], Weight: value,
	})
	return nil
}

func (s *StakeConfig) setNodeID(key, value string) error {
	values := strings.Split(key, ":")
	if len(values) != 2 {
		return fmt.Errorf("stake.nodeIDs config error, actual: %s, expect: %s:<addr1>", key, keyStakeConfigNodeID)
	}
	s.nodeIDs[values[1]] = value
	return nil
}

func loadStakeConfig(consensusExtConfig []*configPb.ConfigKeyValue) (*StakeConfig, error) {
	/**
	  stake合约的配置
	  ext_config: # 扩展字段，记录难度、奖励等其他类共识算法配置
	    - key: stake.minSelfDelegation
	      value: 1000000000000
	    - key: stake.epochValidatorNum
	      value: 10
	    - key: stake.epochBlockNum
	      value: 2000
	    - key: stake.completionUnbondingEpochNum
	      value: 1
		- key: stake.candidate:<addr1>
	      value: 800000
		- key: stake.candidate:<addr2>
	      value: 600000
		- key: stake.nodeID:<addr1>
		  value: nodeID
	*/
	config := &StakeConfig{
		nodeIDs: make(map[string]string),
	}
	for _, kv := range consensusExtConfig {
		switch kv.Key {
		case keyStakeEpochBlockNum:
			val, err := strconv.ParseUint(string(kv.Value), 10, 64)
			if err != nil {
				return nil, err
			}
			config.eachEpochNum = val
		case keyStakeEpochValidatorNum:
			val, err := strconv.ParseUint(string(kv.Value), 10, 64)
			if err != nil {
				return nil, err
			}
			config.validatorNum = val
		case keyStakeMinSelfDelegation:
			if err := isValidBigInt(string(kv.Value)); err != nil {
				return nil, fmt.Errorf("%s error, reason: %s", keyStakeMinSelfDelegation, err)
			}
			config.minSelfDelegation = string(kv.Value)
		case keyStakeUnbondingEpochNum:
			val, err := strconv.ParseUint(string(kv.Value), 10, 64)
			if err != nil {
				return nil, err
			}
			config.unbondingEpochNum = val
		default:
			if strings.HasPrefix(kv.Key, keyStakeCandidate) {
				if err := config.setCandidate(kv.Key, string(kv.Value)); err != nil {
					return nil, err
				}
			}
			if strings.HasPrefix(kv.Key, keyStakeConfigNodeID) {
				if err := config.setNodeID(kv.Key, string(kv.Value)); err != nil {
					return nil, err
				}
			}
		}
	}
	if len(config.nodeIDs) != len(config.candidates) {
		return nil, fmt.Errorf("config nodeIDs and candidates not matched, nodeIDs num: %d, candidates: %d ",
			len(config.nodeIDs), len(config.candidates))
	}
	if len(config.minSelfDelegation) == 0 {
		config.minSelfDelegation = defaultDPoSMinSelfDelegation
	}
	if config.eachEpochNum == 0 {
		config.eachEpochNum = defaultDPoSEpochBlockNumber
	}
	if config.unbondingEpochNum == 0 {
		config.unbondingEpochNum = defaultDPoSCompletionUnboundingEpochNum
	}
	if config.validatorNum == 0 {
		config.validatorNum = defaultDPoSEpochValidatorNumber
	}
	if len(config.candidates) < int(config.validatorNum) {
		return nil, fmt.Errorf("The number of candidates[%d] is less than "+
			"the required number of validator[%d] ", len(config.candidates), config.validatorNum)
	}
	return config, nil
}

func isValidBigInt(val string) error {
	_, ok := big.NewInt(0).SetString(val, 10)
	if !ok {
		return fmt.Errorf("parse string to big.Int failed, actual: %s", val)
	}
	return nil
}

func totalTxRWSet(chainConfigBytes []byte, erc20Config *ERC20Config, stakeConfig *StakeConfig) (
	[]*commonPb.TxWrite, error) {
	txWrites := make([]*commonPb.TxWrite, 0)
	txWrites = append(txWrites, &commonPb.TxWrite{
		Key:          []byte(syscontract.SystemContract_CHAIN_CONFIG.String()),
		Value:        chainConfigBytes,
		ContractName: syscontract.SystemContract_CHAIN_CONFIG.String(),
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
	//初始化系统合约的Contract状态数据
	syscontractKeys := []int{}
	for k := range syscontract.SystemContract_name {
		syscontractKeys = append(syscontractKeys, int(k))
	}
	sort.Ints(syscontractKeys)
	for k := range syscontractKeys {
		name := syscontract.SystemContract_name[int32(k)]
		txWrites = append(txWrites, initSysContractTxWrite(name))
	}
	return txWrites, nil
}
func initSysContractTxWrite(name string) *commonPb.TxWrite {
	contract := &commonPb.Contract{
		Name:        name,
		Version:     "v1",
		RuntimeType: commonPb.RuntimeType_NATIVE,
		Status:      commonPb.ContractStatus_NORMAL,
		Creator:     nil,
	}
	data, _ := contract.Marshal()
	return &commonPb.TxWrite{
		Key:          GetContractDbKey(name),
		Value:        data,
		ContractName: syscontract.SystemContract_CONTRACT_MANAGE.String(),
	}
}
