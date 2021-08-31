/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package utils

import (
	"errors"
	"fmt"

	"chainmaker.org/chainmaker/pb-go/v2/syscontract"

	"chainmaker.org/chainmaker/common/v2/crypto/hash"
	"chainmaker.org/chainmaker/common/v2/random/uuid"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/protocol/v2"
	"github.com/gogo/protobuf/proto"
)

// CalcUnsignedTxBytes calculate unsigned transaction bytes [request payload bytes]
func CalcUnsignedTxBytes(t *commonPb.Transaction) ([]byte, error) {
	if t == nil {
		return nil, errors.New("calc unsigned tx bytes error, tx == nil")
	}
	return t.Payload.Marshal()

}

// CalcUnsignedTxRequestBytes calculate unsigned transaction request bytes
func CalcUnsignedTxRequestBytes(txReq *commonPb.TxRequest) ([]byte, error) {
	if txReq == nil {
		return nil, errors.New("calc unsigned tx request bytes error, tx == nil")
	}
	return txReq.Payload.Marshal()
}

// CalcUnsignedCompleteTxBytes calculate unsigned complete transaction bytearray
//func CalcUnsignedCompleteTxBytes(t *commonPb.Transaction) ([]byte, error) {
//	if t == nil {
//		return nil, errors.New("calc unsigned complete tx bytes error, tx == nil")
//	}
//	headerBytes, err := proto.Marshal(t.Header)
//	if err != nil {
//		return nil, err
//	}
//	resultBytes, err := proto.Marshal(t.Result)
//	if err != nil {
//		return nil, err
//	}
//	completeTxBytes := bytes.Join([][]byte{headerBytes, t.RequestPayload, resultBytes}, []byte{})
//	return completeTxBytes, nil
//}

// CalcTxHash calculate transaction hash, include tx.Payload, tx.signature, tx.Payload, tx.Result
func CalcTxHash(hashType string, t *commonPb.Transaction) ([]byte, error) {
	//txBytes, err := CalcUnsignedCompleteTxBytes(t)
	txBytes, err := t.Marshal()
	if err != nil {
		return nil, err
	}

	hashedTx, err := hash.GetByStrType(hashType, txBytes)
	if err != nil {
		return nil, err
	}
	return hashedTx, nil
}

// CalcTxRequestHash calculate hash of transaction request
func CalcTxRequestHash(hashType string, t *commonPb.Transaction) ([]byte, error) {
	txBytes, err := CalcUnsignedTxBytes(t)
	if err != nil {
		return nil, err
	}

	hashedTx, err := hash.GetByStrType(hashType, txBytes)
	if err != nil {
		return nil, err
	}
	return hashedTx, nil
}

// CalcTxResultHash calculate hash of transaction result
func CalcTxResultHash(hashType string, result *commonPb.Result) ([]byte, error) {
	resultBytes, err := CalcResultBytes(result)
	if err != nil {
		return nil, err
	}
	resultHash, err := hash.GetByStrType(hashType, resultBytes)
	if err != nil {
		return nil, err
	}
	return resultHash, nil
}

// CalcResultBytes get bytearray of result
func CalcResultBytes(result *commonPb.Result) ([]byte, error) {
	if result == nil {
		return nil, errors.New("calculate result bytes error, result == nil")
	}
	tmpGas := result.ContractResult.GasUsed
	result.ContractResult.GasUsed = 0
	resultBytes, err := proto.Marshal(result)
	result.ContractResult.GasUsed = tmpGas
	if err != nil {
		return nil, err
	}
	return resultBytes, nil
}

// IsManageContractAsConfigTx Whether the Manager Contract is considered a configuration transaction
func IsManageContractAsConfigTx(tx *commonPb.Transaction, enableSqlDB bool) bool {
	if tx == nil {
		return false
	}
	return enableSqlDB && IsContractMgmtTx(tx)
}

//IsContractMgmtTx 是否是合约安装、升级的交易
func IsContractMgmtTx(tx *commonPb.Transaction) bool {
	payload := tx.Payload

	return payload.ContractName == syscontract.SystemContract_CONTRACT_MANAGE.String() &&
		(payload.Method == syscontract.ContractManageFunction_INIT_CONTRACT.String() ||
			payload.Method == syscontract.ContractManageFunction_UPGRADE_CONTRACT.String())
}

// IsConfigTx the transaction is a config transaction or not
func IsConfigTx(tx *commonPb.Transaction) bool {
	if tx == nil {
		return false
	}
	return tx.Payload.ContractName == syscontract.SystemContract_CHAIN_CONFIG.String()
}

// IsValidConfigTx the transaction is a valid config transaction or not
func IsValidConfigTx(tx *commonPb.Transaction) bool {
	if tx.Result == nil || tx.Result.ContractResult == nil || tx.Result.ContractResult.Result == nil {
		return false
	}
	if !IsConfigTx(tx) {
		return false
	}
	if tx.Result.Code != commonPb.TxStatusCode_SUCCESS {
		return false
	}
	return true
}

// GetRandTxId return hex string format random transaction id with length = 64
func GetRandTxId() string {
	return uuid.GetUUID() + uuid.GetUUID()
}

// GetTxIdWithSeed return tx-id with seed
func GetTxIdWithSeed(seed int64) string {
	return uuid.GetUUIDWithSeed(seed) + uuid.GetUUIDWithSeed(seed)
}

// CalcTxVerifyWorkers calculate work size of transaction verify
func CalcTxVerifyWorkers(txCount int) int {
	if txCount>>12 > 0 {
		// more than 4095, then use 100 workers
		return 100
	} else if txCount>>11 > 0 {
		// more than 2047, then use 50 workers
		return 50
	} else if txCount>>10 > 0 {
		// more than 1023, then use 20 workers
		return 20
	} else if txCount>>8 > 0 {
		// more than 255, then use 10 workers
		return 10
	} else if txCount>>7 > 0 {
		// more than 127, then use 8 workers
		return 8
	} else if txCount>>5 > 0 {
		// more than 31, then use 5 workers
		return 5
	}
	// else use only 1 worker
	return 1
}

// DispatchTxVerifyTask dispatch transaction verify task
func DispatchTxVerifyTask(txs []*commonPb.Transaction) map[int][]*commonPb.Transaction {
	txCount := len(txs)
	batchCount := CalcTxVerifyWorkers(txCount)
	batchSize := txCount / batchCount
	batch := make(map[int][]*commonPb.Transaction)
	for i := 0; i < batchCount-1; i++ {
		batch[i] = txs[i*batchSize : i*batchSize+batchSize]
	}
	batch[batchCount-1] = txs[(batchCount-1)*batchSize:]
	return batch
}

func GetTxIds(txs []*commonPb.Transaction) []string {
	ret := make([]string, len(txs))
	for i, tx := range txs {
		ret[i] = tx.Payload.TxId
	}
	return ret
}

// VerifyTxWithoutPayload verify a transaction with access control provider.
//The payload of the transaction will not be verified.
func VerifyTxWithoutPayload(tx *commonPb.Transaction, chainId string, ac protocol.AccessControlProvider) error {
	if tx == nil {
		return errors.New("tx is nil")
	}
	if err := verifyTxHeader(tx.Payload, chainId); err != nil {
		return fmt.Errorf("verify tx header failed, %s", err)
	}
	if err := verifyTxAuth(tx, ac); err != nil {
		return fmt.Errorf("verify tx authentation failed, %s", err)
	}
	return nil
}

// verify transaction header
func verifyTxHeader(header *commonPb.Payload, targetChainId string) error {
	defaultTxIdLen := 64 // txId的长度
	// 1. header not null
	if header == nil {
		return errors.New("tx header is nil")
	}
	// 2. chain id matches target chain id
	if header.ChainId != targetChainId {
		return fmt.Errorf("chain id [%s] is incorrect, wanted [%s]", header.ChainId, targetChainId)
	}
	// 3. tx id length is 64
	if len(header.TxId) > defaultTxIdLen {
		return fmt.Errorf("tx id length is incorrect, wanted %d", defaultTxIdLen)
	}
	//only invoke contract tx need check txid
	if header.TxType == commonPb.TxType_INVOKE_CONTRACT {
		// 4. tx id only contains [a-z0-9]
		match := CheckTxIDFormat(header.TxId)
		if !match {
			return errors.New("check tx id failed, only [a-zA-Z0-9_] are allowed, your TxId is:" + header.TxId)
		}
	}
	// 5. timestamp (in seconds) before expiration time
	if header.ExpirationTime != 0 && header.ExpirationTime <= header.Timestamp {
		return fmt.Errorf("tx timestamp %d should be before expiration time %d", header.Timestamp, header.ExpirationTime)
	}
	// 6. sender should not be nil
	//if header.Sender == nil || header.Sender.OrgId == "" || header.Sender.MemberInfo == nil {
	//	return fmt.Errorf("tx sender is nil")
	//}
	return nil
}

// verify transaction sender's authentication (include signature verification,
//cert-chain verification, access verification)
func verifyTxAuth(t *commonPb.Transaction, ac protocol.AccessControlProvider) error {
	var principal protocol.Principal
	var err error
	txBytes, err := CalcUnsignedTxBytes(t)
	if err != nil {
		return err
	}

	endorsements := []*commonPb.EndorsementEntry{t.Sender}
	txType := t.Payload.TxType
	resourceId := t.Payload.ContractName + "-" + t.Payload.Method

	// sender authentication
	_, err = ac.LookUpExceptionalPolicy(resourceId)
	if err == nil {
		principal, err = ac.CreatePrincipal(resourceId, endorsements, txBytes)
		if err != nil {
			return fmt.Errorf("fail to construct authentication principal for %s : %s", resourceId, err)
		}
	} else {
		principal, err = ac.CreatePrincipal(txType.String(), endorsements, txBytes)
		if err != nil {
			return fmt.Errorf("fail to construct authentication principal for %s : %s", txType.String(), err)
		}
	}
	ok, err := ac.VerifyPrincipal(principal)
	if err != nil {
		return fmt.Errorf("authentication error: %s", err)
	}
	if !ok {
		return fmt.Errorf("authentication failed")
	}

	// endorsers authentication for invoke_contract
	if t.Payload.TxType == commonPb.TxType_INVOKE_CONTRACT {
		p, err := ac.LookUpPolicy(resourceId)
		if err != nil {
			return nil
		}
		endorsements := t.Endorsers
		if endorsements == nil {
			return fmt.Errorf("endorsement is nil in verifyTxAuth for resourceId[%s]", resourceId)
		}

		if p.Rule == string(protocol.RuleSelf) {
			var targetOrg string
			parameterPairs := t.Payload.Parameters
			if parameterPairs != nil {
				for i := 0; i < len(parameterPairs); i++ {
					key := parameterPairs[i].Key
					if key == protocol.ConfigNameOrgId {
						targetOrg = string(parameterPairs[i].Value)
						break
					}
				}
				if targetOrg == "" {
					return fmt.Errorf("verification rule is [SELF], but org_id is not set in the parameter")
				}
				principal, err = ac.CreatePrincipalForTargetOrg(resourceId, endorsements, txBytes, targetOrg)
				if err != nil {
					return fmt.Errorf("fail to construct authentication principal with orgId %s for %s-%s: %s",
						targetOrg, t.Payload.ContractName, t.Payload.Method, err)
				}
			}
		} else {
			principal, err = ac.CreatePrincipal(resourceId, endorsements, txBytes)
			if err != nil {
				return fmt.Errorf("fail to construct authentication principal for %s-%s: %s",
					t.Payload.ContractName, t.Payload.Method, err)
			}
		}

		ok, err := ac.VerifyPrincipal(principal)
		if err != nil {
			return fmt.Errorf("authentication error for %s-%s: %s", t.Payload.ContractName, t.Payload.Method, err)
		}
		if !ok {
			return fmt.Errorf("authentication failed for %s-%s", t.Payload.ContractName, t.Payload.Method)
		}
	}
	return nil
}

func GenerateInstallContractPayload(contractName, version string, runtimeType commonPb.RuntimeType, bytecode []byte,
	initParameters []*commonPb.KeyValuePair) (*commonPb.Payload, error) {
	var pairs []*commonPb.KeyValuePair
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   syscontract.InitContract_CONTRACT_NAME.String(),
		Value: []byte(contractName),
	})
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   syscontract.InitContract_CONTRACT_VERSION.String(),
		Value: []byte(version),
	})
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   syscontract.InitContract_CONTRACT_RUNTIME_TYPE.String(),
		Value: []byte(runtimeType.String()),
	})
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   syscontract.InitContract_CONTRACT_BYTECODE.String(),
		Value: bytecode,
	})
	pairs = append(pairs, initParameters...)
	payload := &commonPb.Payload{
		ContractName: syscontract.SystemContract_CONTRACT_MANAGE.String(),
		Method:       syscontract.ContractManageFunction_INIT_CONTRACT.String(),
		Parameters:   pairs,
	}
	return payload, nil
}

func GetRoleFromTx(tx *commonPb.Transaction, ac protocol.AccessControlProvider) (protocol.Role, error) {

	var member protocol.Member
	var err error
	member, err = ac.NewMember(tx.Sender.Signer)

	if err != nil {
		return "", err
	}

	return member.GetRole(), nil
}
