/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package native

import (
	"bytes"
	"chainmaker.org/chainmaker-go/common/crypto"
	"chainmaker.org/chainmaker-go/common/crypto/asym"
	"chainmaker.org/chainmaker-go/common/crypto/asym/rsa"
	"chainmaker.org/chainmaker-go/common/crypto/hash"
	"chainmaker.org/chainmaker-go/common/crypto/tee"
	bcx509 "chainmaker.org/chainmaker-go/common/crypto/x509"
	"chainmaker.org/chainmaker-go/logger"
	"chainmaker.org/chainmaker-go/pb/protogo/accesscontrol"
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/utils"
	"crypto/sha256"
	"crypto/x509"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	ComputeResult = "private_compute_result"
)

type PrivateComputeContract struct {
	methods map[string]ContractFunc
	log     *logger.CMLogger
}

func newPrivateComputeContact(log *logger.CMLogger) *PrivateComputeContract {
	return &PrivateComputeContract{
		log:     log,
		methods: registerPrivateComputeContractMethods(log),
	}
}

func (p *PrivateComputeContract) getMethod(methodName string) ContractFunc {
	return p.methods[methodName]
}

func registerPrivateComputeContractMethods(log *logger.CMLogger) map[string]ContractFunc {
	queryMethodMap := make(map[string]ContractFunc, 64)
	// cert manager
	privateComputeRuntime := &PrivateComputeRuntime{log: log}
	queryMethodMap[commonPb.PrivateComputeContractFunction_GET_CONTRACT.String()] = privateComputeRuntime.GetContract
	queryMethodMap[commonPb.PrivateComputeContractFunction_GET_DATA.String()] = privateComputeRuntime.GetData
	queryMethodMap[commonPb.PrivateComputeContractFunction_SAVE_CA_CERT.String()] = privateComputeRuntime.SaveEnclaveCACert
	queryMethodMap[commonPb.PrivateComputeContractFunction_SAVE_DIR.String()] = privateComputeRuntime.SaveDir
	queryMethodMap[commonPb.PrivateComputeContractFunction_SAVE_DATA.String()] = privateComputeRuntime.SaveData
	queryMethodMap[commonPb.PrivateComputeContractFunction_SAVE_CONTRACT.String()] = privateComputeRuntime.SaveContract
	queryMethodMap[commonPb.PrivateComputeContractFunction_SAVE_ENCLAVE_REPORT.String()] = privateComputeRuntime.SaveEnclaveReport
	queryMethodMap[commonPb.PrivateComputeContractFunction_GET_DIR.String()] = privateComputeRuntime.GetDir
	queryMethodMap[commonPb.PrivateComputeContractFunction_GET_CA_CERT.String()] = privateComputeRuntime.GetEnclaveCACert
	queryMethodMap[commonPb.PrivateComputeContractFunction_GET_ENCLAVE_PROOF.String()] = privateComputeRuntime.QueryEnclaveProof
	queryMethodMap[commonPb.PrivateComputeContractFunction_UPDATE_CONTRACT.String()] = privateComputeRuntime.UpdateContract
	queryMethodMap[commonPb.PrivateComputeContractFunction_CHECK_CALLER_CERT_AUTH.String()] = privateComputeRuntime.CheckCallerCertAuth
	queryMethodMap[commonPb.PrivateComputeContractFunction_GET_ENCLAVE_ENCRYPT_PUB_KEY.String()] = privateComputeRuntime.QueryEnclaveEncryptPubKey
	queryMethodMap[commonPb.PrivateComputeContractFunction_GET_ENCLAVE_VERIFICATION_PUB_KEY.String()] = privateComputeRuntime.QueryEnclaveVerificationPubKey
	queryMethodMap[commonPb.PrivateComputeContractFunction_GET_ENCLAVE_REPORT.String()] = privateComputeRuntime.QueryEnclaveReport
	queryMethodMap[commonPb.PrivateComputeContractFunction_GET_ENCLAVE_CHALLENGE.String()] = privateComputeRuntime.QueryEnclaveChallenge
	queryMethodMap[commonPb.PrivateComputeContractFunction_GET_ENCLAVE_SIGNATURE.String()] = privateComputeRuntime.QueryEnclaveSignature
	queryMethodMap[commonPb.PrivateComputeContractFunction_SAVE_REMOTE_ATTESTATION.String()] = privateComputeRuntime.SaveRemoteAttestation

	return queryMethodMap
}

type PrivateComputeRuntime struct {
	log *logger.CMLogger
}

func (r *PrivateComputeRuntime) VerifyByEnclaveCert(context protocol.TxSimContext, enclaveId []byte, data []byte, sign []byte) (bool, error) {
	enclaveCert, err := context.Get(commonPb.ContractName_SYSTEM_CONTRACT_PRIVATE_COMPUTE.String(), enclaveId)
	if err != nil {
		r.log.Errorf("%s, get enclave cert[%s] failed", err.Error(), enclaveId)
		return false, err
	}

	cert, err := utils.ParseCert(enclaveCert)
	if err != nil {
		r.log.Errorf("%s, parse enclave certificate failed, enclave id[%s], cert bytes[%s]", err.Error(), enclaveId, enclaveCert)
		return false, err
	}

	hashAlgo, err := bcx509.GetHashFromSignatureAlgorithm(cert.SignatureAlgorithm)
	digest, err := hash.Get(hashAlgo, data)
	if err != nil {
		r.log.Errorf("%s, calculate hash of data[%s] failed", err.Error(), data)
		return false, err
	}

	ok, err := cert.PublicKey.Verify(digest, sign)
	if !ok {
		r.log.Errorf("%s, enclave certificate[%s] verify data[%s] failed", err.Error(), enclaveId, data)
		return false, err
	}

	return true, nil
}

func (r *PrivateComputeRuntime) getValue(context protocol.TxSimContext, key string) ([]byte, error) {
	if strings.TrimSpace(key) == "" {
		err := fmt.Errorf("%s, key is empty", ErrParams.Error())
		r.log.Errorf(err.Error())
		return nil, err
	}

	value, err := context.Get(commonPb.ContractName_SYSTEM_CONTRACT_PRIVATE_COMPUTE.String(), []byte(key))
	if err != nil {
		r.log.Errorf("Get key: %s from context failed, err: %s", key, err.Error())
		return nil, err
	}

	return value, nil
}

func (r *PrivateComputeRuntime) SaveContract(context protocol.TxSimContext, params map[string]string) ([]byte, error) {
	name := params["contract_name"]
	code := params["contract_code"]
	codeHash := params["code_hash"]
	version := params["version"]
	if utils.IsAnyBlank(name, code, codeHash, version) {
		err := fmt.Errorf("%s, param[contract_name]=%s, param[contract_code]=%s, param[code_hash]=%s, params[version]=%s",
			ErrParams.Error(), name, code, codeHash, version)
		r.log.Errorf(err.Error())
		return nil, err
	}

	calHash := sha256.Sum256([]byte(code))
	if string(calHash[:]) != codeHash {
		err := fmt.Errorf("%s, param[code_hash] != codeHash of param[contract_code] in save contract interface", ErrParams.Error())
		r.log.Errorf(err.Error())
		return nil, err
	}

	if len(version) > protocol.DefaultVersionLen {
		err := fmt.Errorf("param[version] string of the contract[%+v] too long, should be less than %d", name, protocol.DefaultVersionLen)
		r.log.Errorf(err.Error())
		return nil, err
	}

	match, err := regexp.MatchString(protocol.DefaultVersionRegex, version)
	if err != nil || !match {
		err := fmt.Errorf("param[version] string of the contract[%+v] invalid while invoke user contract, should match [%s]", name, protocol.DefaultVersionRegex)
		r.log.Errorf(err.Error())
		return nil, err
	}

	combinationName := commonPb.ContractName_SYSTEM_CONTRACT_PRIVATE_COMPUTE.String() + name
	versionKey := []byte(protocol.ContractVersion)
	versionInCtx, err := context.Get(combinationName, versionKey)
	if err != nil {
		err := fmt.Errorf("unable to find latest version for contract[%s], system error:%s", name, err.Error())
		r.log.Errorf(err.Error())
		return nil, err
	}

	if versionInCtx != nil {
		err := fmt.Errorf("the contract already exists. contract[%s], version[%s]", name, string(versionInCtx))
		r.log.Errorf(err.Error())
		return nil, err
	}

	if err := context.Put(combinationName, versionKey, []byte(version)); err != nil {
		r.log.Errorf("Put contract version into DB failed while save contract, err: %s", err.Error())
		return nil, err
	}

	key := append([]byte(protocol.ContractByteCode), []byte(version)...)
	if err := context.Put(combinationName, key, []byte(code)); err != nil {
		r.log.Errorf("Put compute contract[%s] failed, err: %s", err.Error(), name)
		return nil, err
	}

	return nil, nil
}

func (r *PrivateComputeRuntime) UpdateContract(context protocol.TxSimContext, params map[string]string) ([]byte, error) {
	name := params["contract_name"]
	code := params["contract_code"]
	hash := params["code_hash"]
	version := params["version"]
	if utils.IsAnyBlank(name, code, hash, version) {
		err := fmt.Errorf("%s, param[contract_name]=%s, param[contract_code]=%s, param[code_hash]=%s, params[version]=%s",
			ErrParams.Error(), name, code, hash, version)
		r.log.Errorf(err.Error())
		return nil, err
	}

	calHash := sha256.Sum256([]byte(code))
	if string(calHash[:]) != hash {
		err := fmt.Errorf("%s, param hash[%v] != param contract_code hash[%v] in save contract interface", ErrParams.Error(), []byte(hash), calHash)
		r.log.Errorf(err.Error())
		return nil, err
	}

	if len(version) > protocol.DefaultVersionLen {
		err := fmt.Errorf("param[version] string of the contract[%+v] too long, should be less than %d", name, protocol.DefaultVersionLen)
		r.log.Errorf(err.Error())
		return nil, err
	}

	match, err := regexp.MatchString(protocol.DefaultVersionRegex, version)
	if err != nil || !match {
		err := fmt.Errorf("param[version] string of the contract[%+v] invalid while invoke user contract, should match [%s]", name, protocol.DefaultVersionRegex)
		r.log.Errorf(err.Error())
		return nil, err
	}

	combinationName := commonPb.ContractName_SYSTEM_CONTRACT_PRIVATE_COMPUTE.String() + name
	versionKey := []byte(protocol.ContractVersion)
	versionInCtx, err := context.Get(combinationName, versionKey)
	if err != nil {
		err := fmt.Errorf("unable to find latest version for contract[%s], system error:%s", name, err.Error())
		r.log.Errorf(err.Error())
		return nil, err
	}

	if len(versionInCtx) == 0 {
		err := fmt.Errorf("the contract[%s] does not exist, update failed", name)
		r.log.Errorf(err.Error())
		return nil, err
	}

	key := append([]byte(protocol.ContractByteCode), []byte(version)...)
	codeInCtx, err := context.Get(combinationName, key)
	if err == nil && len(codeInCtx) > 0 {
		err := fmt.Errorf("the contract version[%s] and code[%s] is already exist", version, codeInCtx)
		r.log.Errorf(err.Error())
		return nil, err
	}

	if err := context.Put(combinationName, versionKey, []byte(version)); err != nil {
		r.log.Errorf("Put contract version into DB failed while save contract, err: %s", err.Error())
		return nil, err
	}

	if err := context.Put(combinationName, key, []byte(code)); err != nil {
		r.log.Errorf("Put compute contract[%s] failed, err: %s", err.Error(), name)
		return nil, err
	}

	return nil, nil
}

func (r *PrivateComputeRuntime) GetContract(context protocol.TxSimContext, params map[string]string) ([]byte, error) {
	name := params["contract_name"]
	if utils.IsAnyBlank(name) {
		err := fmt.Errorf("%s, param[contract_name] of get contract not found", ErrParams.Error())
		r.log.Errorf(err.Error())
		return nil, err
	}

	hash := params["code_hash"]
	if utils.IsAnyBlank(hash) {
		err := fmt.Errorf("%s, param[code_hash] of get contract not found", ErrParams.Error())
		r.log.Errorf(err.Error())
		return nil, err
	}

	combinationName := commonPb.ContractName_SYSTEM_CONTRACT_PRIVATE_COMPUTE.String() + name
	version, err := context.Get(combinationName, []byte(protocol.ContractVersion))
	if err != nil {
		r.log.Errorf("Unable to find latest version for contract[%s], system error:%s.", name, err.Error())
		return nil, err
	}

	if len(version) == 0 {
		r.log.Errorf("The contract does not exist. contract[%s].", name)
		return nil, err
	}

	var result commonPb.PrivateGetContract
	key := append([]byte(protocol.ContractByteCode), version...)
	contractCode, err := context.Get(combinationName, key)
	if err != nil {
		r.log.Errorf("Read contract[%s] failed.", name)
		return nil, err
	}
	r.log.Infof("get contract, name[%s], code[%v]", name, contractCode)

	if len(contractCode) == 0 {
		r.log.Errorf("Contract[%s] byte code is empty.", name)
		return nil, err
	}

	result.ContractCode = contractCode
	result.GasLimit = protocol.GasLimit
	result.Version = string(version)

	calHash := sha256.Sum256(result.ContractCode)
	if string(calHash[:]) != hash {
		err := fmt.Errorf("%s, param hash[%v] != contract code hash[%v] in get contract interface", ErrParams.Error(), []byte(hash), calHash)
		r.log.Errorf(err.Error())
		return nil, err
	}

	return result.Marshal()
}

func (r *PrivateComputeRuntime) SaveDir(context protocol.TxSimContext, params map[string]string) ([]byte, error) {
	key := params["order_id"]
	if utils.IsAnyBlank(key) {
		err := fmt.Errorf("%s, param[order_id] of save dir  not found", ErrParams.Error())
		r.log.Errorf(err.Error())
		return nil, err
	}

	value := params["private_dir"]
	if utils.IsAnyBlank(value) {
		err := fmt.Errorf("%s, param[private_key] of save dir not found", ErrParams.Error())
		r.log.Errorf(err.Error())
		return nil, err
	}

	if err := context.Put(commonPb.ContractName_SYSTEM_CONTRACT_PRIVATE_COMPUTE.String(), []byte(key), []byte(value)); err != nil {
		r.log.Errorf("Put private dir failed, err: %s", err.Error())
		return nil, err
	}

	return nil, nil
}

func (r *PrivateComputeRuntime) GetDir(context protocol.TxSimContext, params map[string]string) ([]byte, error) {
	return r.getValue(context, params["order_id"])
}

func (r *PrivateComputeRuntime) SaveData(context protocol.TxSimContext, params map[string]string) ([]byte, error) {
	ac, err := context.GetAccessControl()
	if err != nil {
		return nil, err
	}

	auth, err := r.verifyCallerAuth(params, context.GetTx().Header.ChainId, ac)
	if !auth || err != nil {
		err := fmt.Errorf("verify user auth failed, user_cert[%v], signature[%v], request payload[code_hash]=%v",
			params["user_cert"], params["client_sign"], params["payload"])
		r.log.Errorf(err.Error())
		return nil, err
	}

	name := params["contract_name"]
	version := params["version"]
	codeHash := params["code_hash"]
	reportHash := params["report_hash"]
	userCert := params["user_cert"]
	clientSign := params["client_sign"]
	payload := params["payload"]
	orgId := params["org_id"]

	if utils.IsAnyBlank(name, version, codeHash, reportHash) {
		err := fmt.Errorf(
			"%s, param[contract_name]=%s, params[version]=%s, param[code_hash]=%s, param[report_hash]=%s, "+
				"params[user_cert]=%s, params[client_sign]=%s, params[payload]=%s, params[org_id]=%s,",
			ErrParams.Error(), name, version, codeHash, reportHash, userCert, clientSign, payload, orgId)
		r.log.Errorf(err.Error())
		return nil, err
	}
	var result commonPb.ContractResult
	cRes := []byte(params["result"])
	if err := result.Unmarshal(cRes); err != nil {
		r.log.Errorf("Unmarshal ContractResult failed, err: %s", err.Error())
		return nil, err
	}
	var rwSet commonPb.TxRWSet
	if err := rwSet.Unmarshal([]byte(params["rw_set"])); err != nil {
		r.log.Errorf("Unmarshal RWSet failed, err: %s", err.Error())
		return nil, err
	}
	// verify sign
	combinedKey := commonPb.ContractName_SYSTEM_CONTRACT_PRIVATE_COMPUTE.String() + "global_enclave_id"
	pkPEM, err := context.Get(combinedKey, []byte("verification_pub_key"))
	if err != nil {
		r.log.Errorf("get verification_pub_key error: %s", err.Error())
		return nil, err
	}
	pk, err := asym.PublicKeyFromPEM(pkPEM)
	if err != nil {
		r.log.Errorf("get pk from PEM error: %s", err.Error())
		return nil, err
	}
	evmResultBuffer := bytes.NewBuffer([]byte{})
	if err := binary.Write(evmResultBuffer, binary.LittleEndian, result.Code); err != nil {
		return nil, err
	}
	evmResultBuffer.Write(result.Result)
	if err := binary.Write(evmResultBuffer, binary.LittleEndian, result.GasUsed); err != nil {
		return nil, err
	}
	for i := 0; i < len(rwSet.TxReads); i++ {
		evmResultBuffer.Write(rwSet.TxReads[i].Key)
		evmResultBuffer.Write(rwSet.TxReads[i].Value)
		//evmResultBuffer.Write([]byte(rwSet.TxReads[i].Version.RefTxId))
	}
	for i := 0; i < len(rwSet.TxWrites); i++ {
		evmResultBuffer.Write(rwSet.TxWrites[i].Key)
		evmResultBuffer.Write(rwSet.TxWrites[i].Value)
	}
	evmResultBuffer.Write([]byte(name))
	evmResultBuffer.Write([]byte(version))
	evmResultBuffer.Write([]byte(codeHash))
	evmResultBuffer.Write([]byte(reportHash))
	evmResultBuffer.Write([]byte(userCert))
	evmResultBuffer.Write([]byte(clientSign))
	evmResultBuffer.Write([]byte(orgId))
	evmResultBuffer.Write([]byte(payload))
	b, err := pk.VerifyWithOpts(evmResultBuffer.Bytes(), []byte(params["report_sign"]), &crypto.SignOpts{
		Hash:         crypto.HASH_TYPE_SHA256,
		UID:          "",
		EncodingType: rsa.RSA_PSS,
	})
	if err != nil {
		r.log.Errorf("verify ContractResult err: %s", err.Error())
		return nil, err
	}
	if !b {
		r.log.Debug("verify ContractResult failed")
		return nil, nil
	}
	r.log.Debug("verify ContractResult success")

	combinationName := commonPb.ContractName_SYSTEM_CONTRACT_PRIVATE_COMPUTE.String() + name
	key := append([]byte(protocol.ContractByteCode), version...)
	contractCode, err := context.Get(combinationName, key)
	if err != nil || len(contractCode) == 0 {
		r.log.Errorf("Read contract[%s] failed.", name)
		return nil, err
	}

	calHash := sha256.Sum256(contractCode)
	if string(calHash[:]) != codeHash {
		err := fmt.Errorf("%s, param[code_hash] != hash of contract code in get contract interface", ErrParams.Error())
		r.log.Errorf(err.Error())
		return nil, err
	}

	if result.GasUsed > protocol.GasLimit {
		err := fmt.Errorf("gas[%d] expend the limit[%f]", result.GasUsed, protocol.GasLimit)
		r.log.Errorf(err.Error())
		return nil, err
	}

	if err := context.Put(combinationName, []byte(ComputeResult), cRes); err != nil {
		r.log.Errorf("Write compute result:%s failed, err: %s", cRes, err.Error())
		return nil, err
	}

	if result.Code != commonPb.ContractResultCode_OK {
		r.log.Infof("Compute result code != ok, return")
		return nil, nil
	}

	for i := 0; i < len(rwSet.TxReads); i++ {
		key := rwSet.TxReads[i].Key
		val := rwSet.TxReads[i].Value
		//version := rwSet.TxReads[i].Version
		chainRSetBytes, err := context.Get(combinationName, key)
		if err != nil {
			r.log.Errorf("Put key: %s, value:%s into read set failed, err: %s", key, val, err.Error())
			return nil, err
		}
		if len(chainRSetBytes) == 0 {
			r.log.Debug("there is not a rSet with key: %s on chain\n", key)
		}
		var rSet commonPb.TxRead
		if err := rSet.Unmarshal(chainRSetBytes); err != nil {
			r.log.Errorf("Unmarshal RSet failed, err: %s", err.Error())
			return nil, err
		}
		//if !bytes.Equal(val, rSet.Value) || version.RefTxId != rSet.Version.RefTxId {
		//	r.log.Errorf("rSet verification failed! key: %s, value: %s, version: %s; but value on chain: %s, version on chain: %s",
		//		key, val, version.RefTxId, rSet.Value, rSet.Version.RefTxId)
		//	return nil, errors.New("rSet verification failed! key: " + string(key) + ", value: " + string(val) +
		//		", version: " + version.RefTxId + "; but value on chain: " + string(rSet.Value) +
		//		", version on chain: " + rSet.Version.RefTxId)
		//}
	}

	for j := 0; j < len(rwSet.TxWrites); j++ {
		key := rwSet.TxWrites[j].Key
		val := rwSet.TxWrites[j].Value
		if err := context.Put(combinationName, key, val); err != nil {
			r.log.Errorf("Put key: %s, value:%s into write set failed, err: %s", key, val, err.Error())
			return nil, err
		}
	}

	//TODO: put events into DB

	return nil, nil
}

func (r *PrivateComputeRuntime) GetData(context protocol.TxSimContext, params map[string]string) ([]byte, error) {
	key := []byte(params["key"])
	if utils.IsAnyBlank(params["key"]) {
		err := fmt.Errorf("%s,param[private_key] of get data  not found", ErrParams.Error())
		r.log.Errorf(err.Error())
		return nil, err
	}

	name, res := params["contract_name"]
	if res != true {
		name = ""
	}

	combinationName := commonPb.ContractName_SYSTEM_CONTRACT_PRIVATE_COMPUTE.String() + name
	value, err := context.Get(combinationName, key)
	if err != nil {
		r.log.Errorf("Get key: %s from context failed, err: %s", key, err.Error())
		return nil, err
	}

	return value, nil
}

func (r *PrivateComputeRuntime) GetEnclaveCACert(context protocol.TxSimContext, params map[string]string) ([]byte, error) {

	caCertPEM, err := context.Get(commonPb.ContractName_SYSTEM_CONTRACT_PRIVATE_COMPUTE.String(), []byte("ca_cert"))
	if err != nil {
		r.log.Errorf("get enclave ca cert failed: %v", err.Error())
		return nil, err
	}

	return caCertPEM, nil
}

func (r *PrivateComputeRuntime) SaveEnclaveCACert(context protocol.TxSimContext, params map[string]string) ([]byte, error) {
	// PEM 格式的证书
	caCertPEM := params["ca_cert"]
	if utils.IsAnyBlank(caCertPEM) {
		err := fmt.Errorf("%s,param[ca_cert] does not found", ErrParams.Error())
		r.log.Errorf(err.Error())
		return nil, err
	}

	if err := context.Put(commonPb.ContractName_SYSTEM_CONTRACT_PRIVATE_COMPUTE.String(), []byte("ca_cert"), []byte(caCertPEM)); err != nil {
		r.log.Errorf("save enclave ca cert failed: %v", err.Error())
		return nil, err
	}

	return nil, nil
}

func (r *PrivateComputeRuntime) SaveRemoteAttestation(context protocol.TxSimContext, params map[string]string) ([]byte, error) {
	// get params
	proofDataStr := params["proof"]
	if utils.IsAnyBlank(proofDataStr) {
		err := fmt.Errorf("'proof' is nil")
		r.log.Errorf(err.Error())
		return nil, err
	}

	proofData := []byte(proofDataStr)

	//
	//
	// 1）extract challenge/report/signing pub key/encrypt pub key/ from proof
	//
	// ok, proof, msg, err := splitProof(proofData)
	// if err != nil || !ok {
	// 	 err := fmt.Errorf("split 'proof' data error: %v", err)
	//	 r.log.Errorf(err.Error())
	//	 return nil, err
	// }

	// 2）construct the enclaveId
	//
	// enclaveData, err := utils.GetCertificateIdFromDER(proof.CertificateDER, bccrypto.CRYPTO_ALGO_SHA3_256)
	//if err != nil {
	//    err := fmt.Errorf("generate enclave_id error: %v", err)
	//    r.log.Errorf(err.Error())
	//    return nil, err
	// }
	// enclaveId := base64.StdEncoding.EncodeToString(enclaveData)
	enclaveId := "global_enclave_id"

	// get report from chain
	enclaveIdKey := commonPb.ContractName_SYSTEM_CONTRACT_PRIVATE_COMPUTE.String() + enclaveId
	reportFromChain, err := context.Get(enclaveIdKey, []byte("report"))
	if err != nil {
		err := fmt.Errorf("get enclave 'report' from chain error: %v", err)
		r.log.Errorf(err.Error())
		return nil, err
	}

	// get ca_cert from chain
	caCertPem, err := context.Get(commonPb.ContractName_SYSTEM_CONTRACT_PRIVATE_COMPUTE.String(), []byte("ca_cert"))
	if err != nil {
		err := fmt.Errorf("get enclave 'ca_cert' from chain error: %v", err)
		r.log.Errorf(err.Error())
		return nil, err
	}
	caCertBlock, _ := pem.Decode(caCertPem)
	if caCertBlock == nil {
		err := fmt.Errorf("decode enclave 'ca_cert' from pem format error: %v", err)
		r.log.Errorf(err.Error())
		return nil, err
	}
	caCert, err := bcx509.ParseCertificate(caCertBlock.Bytes)
	if err != nil {
		err := fmt.Errorf("parse enclave 'ca_cert' error: %v", err)
		r.log.Errorf(err.Error())
		return nil, err
	}

	intermediateCAPool := bcx509.NewCertPool()
	intermediateCAPool.AddCert(caCert)
	verifyOption := bcx509.VerifyOptions{
		DNSName:                   "",
		Roots:                     intermediateCAPool,
		CurrentTime:               time.Time{},
		KeyUsages:                 []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
		MaxConstraintComparisions: 0,
	}
	// verify remote attestation
	passed, proof, err := tee.AttestationVerify(
		proofData,
		verifyOption,
		reportFromChain)
	if err != nil || !passed {
		err := fmt.Errorf("save RemoteAttestation Proof error: %v", err)
		r.log.Errorf(err.Error())
		return nil, err
	}

	// save remote attestation
	if err := context.Put(enclaveIdKey, []byte("proof"), proofData); err != nil {
		err := fmt.Errorf("save RemoteAttestatipn proof failed, err: %s", err.Error())
		r.log.Errorf(err.Error())
		return nil, err
	}
	if err := context.Put(enclaveIdKey, []byte("cert"), proof.CertificateDER); err != nil {
		err := fmt.Errorf("save RemoteAttestatipn attribute 'cert' failed, err: %s", err.Error())
		r.log.Errorf(err.Error())
		return nil, err
	}
	if err := context.Put(enclaveIdKey, []byte("challenge"), proof.Challenge); err != nil {
		err := fmt.Errorf("save RemoteAttestatipn attribute 'challenge' failed, err: %s", err.Error())
		r.log.Errorf(err.Error())
		return nil, err
	}
	if err := context.Put(enclaveIdKey, []byte("signature"), proof.Signature); err != nil {
		err := fmt.Errorf("save RemoteAttestatipn attribute 'challenge' failed, err: %s", err.Error())
		r.log.Errorf(err.Error())
		return nil, err
	}
	if err := context.Put(enclaveIdKey, []byte("verification_pub_key"), proof.VerificationKeyPEM); err != nil {
		err := fmt.Errorf("save remote attestatipn attribute <verification_pub_key> failed, err: %s", err.Error())
		r.log.Errorf(err.Error())
		return nil, err
	}
	if err := context.Put(enclaveIdKey, []byte("encrypt_pub_key"), proof.EncryptionKeyPEM); err != nil {
		err := fmt.Errorf("save remote attestatipn attribute <encrypt_pub_key> failed, err: %s", err.Error())
		r.log.Errorf(err.Error())
		return nil, err
	}

	return []byte(enclaveId), nil
}

func (r *PrivateComputeRuntime) QueryEnclaveEncryptPubKey(context protocol.TxSimContext, params map[string]string) ([]byte, error) {
	// get params
	enclaveId := params["enclave_id"]
	if utils.IsAnyBlank(enclaveId) {
		err := fmt.Errorf("%s,param[ca_cert] of save cert  not found", ErrParams.Error())
		r.log.Errorf(err.Error())
		return nil, err
	}

	// get data from chain
	combinedKey := commonPb.ContractName_SYSTEM_CONTRACT_PRIVATE_COMPUTE.String() + enclaveId
	pemEncryptPubKey, err := context.Get(combinedKey, []byte("encrypt_pub_key"))
	if err != nil {
		err := fmt.Errorf("get 'encrypt_pub_key' from chain error: %v", err)
		r.log.Errorf(err.Error())
		return nil, err
	}

	return pemEncryptPubKey, nil
}

func (r *PrivateComputeRuntime) QueryEnclaveVerificationPubKey(context protocol.TxSimContext, params map[string]string) ([]byte, error) {
	// get params
	enclaveId := params["enclave_id"]
	if utils.IsAnyBlank(enclaveId) {
		err := fmt.Errorf("%s,param['ca_cert'] of save cert  not found", ErrParams.Error())
		r.log.Errorf(err.Error())
		return nil, err
	}

	// get data from chain
	combinedKey := commonPb.ContractName_SYSTEM_CONTRACT_PRIVATE_COMPUTE.String() + enclaveId
	pemVerificationPubKey, err := context.Get(combinedKey, []byte("verification_pub_key"))
	if err != nil {
		err := fmt.Errorf("get 'verification_pub_key' from chain error: %v", err)
		r.log.Errorf(err.Error())
		return nil, err
	}

	return pemVerificationPubKey, nil
}

func (r *PrivateComputeRuntime) SaveEnclaveReport(context protocol.TxSimContext, params map[string]string) ([]byte, error) {
	// get params
	enclaveId := params["enclave_id"]
	report := params["report"]
	if utils.IsAnyBlank(enclaveId, report) {
		err := fmt.Errorf("%s,param['enclave_id'] or param['report'] does not found", ErrParams.Error())
		r.log.Errorf(err.Error())
		return nil, err
	}

	// save report into chain
	enclaveIdKey := commonPb.ContractName_SYSTEM_CONTRACT_PRIVATE_COMPUTE.String() + enclaveId
	if err := context.Put(enclaveIdKey, []byte("report"), []byte(report)); err != nil {
		err := fmt.Errorf("save enclave 'report' failed, err: %s", err.Error())
		r.log.Errorf(err.Error())
		return nil, err
	}

	return nil, nil
}

func (r *PrivateComputeRuntime) QueryEnclaveReport(context protocol.TxSimContext, params map[string]string) ([]byte, error) {
	// get params
	enclaveId := params["enclave_id"]
	if utils.IsAnyBlank(enclaveId) {
		err := fmt.Errorf("%s,param['enclave_id'] not found", ErrParams.Error())
		r.log.Errorf(err.Error())
		return nil, err
	}

	// get data from chain
	enclaveIdKey := commonPb.ContractName_SYSTEM_CONTRACT_PRIVATE_COMPUTE.String() + enclaveId
	report, err := context.Get(enclaveIdKey, []byte("report"))
	if err != nil {
		err := fmt.Errorf("get 'report' from chain error: %v", err)
		r.log.Errorf(err.Error())
		return nil, err
	}

	return report, nil
}

func (r *PrivateComputeRuntime) QueryEnclaveChallenge(context protocol.TxSimContext, params map[string]string) ([]byte, error) {
	// 证书二进制数据
	enclaveId := params["enclave_id"]
	if utils.IsAnyBlank(enclaveId) {
		err := fmt.Errorf("%s,param['enclave_id'] not found", ErrParams.Error())
		r.log.Errorf(err.Error())
		return nil, err
	}

	// get data from chain
	enclaveIdKey := commonPb.ContractName_SYSTEM_CONTRACT_PRIVATE_COMPUTE.String() + enclaveId
	challenge, err := context.Get(enclaveIdKey, []byte("challenge"))
	if err != nil {
		err := fmt.Errorf("get 'challenge' from chain error: %v", err)
		r.log.Errorf(err.Error())
		return nil, err
	}

	return challenge, nil
}

func (r *PrivateComputeRuntime) QueryEnclaveSignature(context protocol.TxSimContext, params map[string]string) ([]byte, error) {
	// 证书二进制数据
	enclaveId := params["enclave_id"]
	if utils.IsAnyBlank(enclaveId) {
		err := fmt.Errorf("%s,param['enclave_id'] not found", ErrParams.Error())
		r.log.Errorf(err.Error())
		return nil, err
	}

	// get data from chain
	combinedKey := commonPb.ContractName_SYSTEM_CONTRACT_PRIVATE_COMPUTE.String() + enclaveId
	signature, err := context.Get(combinedKey, []byte("signature"))
	if err != nil {
		err := fmt.Errorf("get 'signature' from chain error: %v", err)
		r.log.Errorf(err.Error())
		return nil, err
	}

	return signature, nil
}

func (r *PrivateComputeRuntime) QueryEnclaveProof(context protocol.TxSimContext, params map[string]string) ([]byte, error) {
	// 证书二进制数据
	enclaveId := params["enclave_id"]
	if utils.IsAnyBlank(enclaveId) {
		err := fmt.Errorf("%s,param['enclave_id'] not found", ErrParams.Error())
		r.log.Errorf(err.Error())
		return nil, err
	}

	// get data from chain
	combinedKey := commonPb.ContractName_SYSTEM_CONTRACT_PRIVATE_COMPUTE.String() + enclaveId
	proof, err := context.Get(combinedKey, []byte("proof"))
	if err != nil {
		err := fmt.Errorf("get 'proof' from chain error: %v", err)
		r.log.Errorf(err.Error())
		return nil, err
	}

	return proof, nil
}

func (r *PrivateComputeRuntime) CheckCallerCertAuth(ctx protocol.TxSimContext, params map[string]string) ([]byte, error) {
	ac, err := ctx.GetAccessControl()
	if err != nil {
		return nil, err
	}

	auth, err := r.verifyCallerAuth(params, ctx.GetTx().Header.ChainId, ac)
	if err != nil {
		return nil, err
	}

	return []byte(strconv.FormatBool(auth)), nil
}

func (r *PrivateComputeRuntime) verifyCallerAuth(params map[string]string, chainId string, ac protocol.AccessControlProvider) (bool, error) {

	clientSign, err := r.getParamValue(params, "client_sign")
	if err != nil {
		return false, err
	}

	payload, err := r.getParamValue(params, "payload")
	if err != nil {
		return false, err
	}

	payLoadBytes, err := hex.DecodeString(payload)
	if err != nil {
		r.log.Errorf("payload hex err:%v", err.Error())
		return false, err
	}

	clientSignBytes, err := hex.DecodeString(clientSign)
	if err != nil {
		r.log.Errorf("client sign hex err:%v", err.Error())
		return false, err
	}

	fmt.Printf("++++++++++++private clientSignBytges is %v++++++++++", clientSignBytes)
	orgId, err := r.getOrgId(payLoadBytes)
	if err != nil {
		return false, err
	}

	userCertPem, err := r.getParamValue(params, "user_cert")
	if err != nil {
		return false, err
	}

	userCertPemBytes, err := hex.DecodeString(userCertPem)
	if err != nil {
		r.log.Errorf("user cert pem hex err:%v", err.Error())
		return false, err
	}

	sender := &accesscontrol.SerializedMember{
		OrgId:      orgId,
		MemberInfo: userCertPemBytes,
		IsFullCert: true,
	}

	endorsements := []*commonPb.EndorsementEntry{{
		Signer:    sender,
		Signature: clientSignBytes,
	}}

	principal, err := ac.CreatePrincipal("PRIVATE_COMPUTE", endorsements, payLoadBytes)  //todo pb
	if err != nil {
		return false, fmt.Errorf("fail to construct authentication principal: %s", err)
	}

	ok, err := ac.VerifyPrincipal(principal)
	if err != nil {
		return false, fmt.Errorf("authentication error, %s", err)
	}

	if !ok {
		return false, fmt.Errorf("authentication failed")
	}

	return true, nil
}

func (r *PrivateComputeRuntime) getOrgId(payLoad []byte) (string, error) {
	result := make(map[string]string, 0)

	if err := json.Unmarshal(payLoad, &result); err != nil {
		return "", errors.New("unmarshal payload failed err" + err.Error())
	}

	orgId, ok := result["org_id"]
	if ok {
		return orgId, nil
	}

	return "", errors.New("payload miss org_id ")
}

func (r *PrivateComputeRuntime) getParamValue(parameters map[string]string, key string) (string, error) {
	value, ok := parameters[key]
	if !ok {
		errMsg := fmt.Sprintf("miss params %s", key)
		r.log.Error(errMsg)
		return "", errors.New(errMsg)
	}
	return value, nil
}
