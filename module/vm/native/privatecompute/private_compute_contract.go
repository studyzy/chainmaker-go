/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */
package privatecompute

import (
	"bytes"
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

	"chainmaker.org/chainmaker-go/utils"
	"chainmaker.org/chainmaker-go/vm/native/common"
	"chainmaker.org/chainmaker/common/v2/crypto"
	"chainmaker.org/chainmaker/common/v2/crypto/asym"
	"chainmaker.org/chainmaker/common/v2/crypto/asym/rsa"
	"chainmaker.org/chainmaker/common/v2/crypto/hash"
	"chainmaker.org/chainmaker/common/v2/crypto/tee"
	bcx509 "chainmaker.org/chainmaker/common/v2/crypto/x509"
	"chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"
	"chainmaker.org/chainmaker/protocol/v2"
)

const (
	ComputeResult       = "private_compute_result"
	ContractKey         = ":K:"
	ContractByteHeader  = ":H:"
	ContractByteCode    = ":B:"
	ContractVersion     = ":V:"
	ContractRuntimeType = ":R:"
	ContractCreator     = ":C:"
	ContractFreeze      = ":F:"
	ContractRevoke      = ":RV:"
	ContractAddress     = ":A:"
)

type PrivateComputeContract struct {
	methods map[string]common.ContractFunc
	log     protocol.Logger
}

func NewPrivateComputeContact(log protocol.Logger) *PrivateComputeContract {
	return &PrivateComputeContract{
		log:     log,
		methods: registerPrivateComputeContractMethods(log),
	}
}

func (p *PrivateComputeContract) GetMethod(methodName string) common.ContractFunc {
	return p.methods[methodName]
}

func registerPrivateComputeContractMethods(log protocol.Logger) map[string]common.ContractFunc {
	queryMethodMap := make(map[string]common.ContractFunc, 64)
	// cert manager
	privateComputeRuntime := &PrivateComputeRuntime{log: log}

	queryMethodMap[syscontract.PrivateComputeFunction_GET_CONTRACT.String()] = privateComputeRuntime.GetContract
	queryMethodMap[syscontract.PrivateComputeFunction_GET_DATA.String()] = privateComputeRuntime.GetData
	queryMethodMap[syscontract.PrivateComputeFunction_SAVE_CA_CERT.String()] = privateComputeRuntime.SaveEnclaveCACert
	queryMethodMap[syscontract.PrivateComputeFunction_SAVE_DIR.String()] = privateComputeRuntime.SaveDir
	queryMethodMap[syscontract.PrivateComputeFunction_SAVE_DATA.String()] = privateComputeRuntime.SaveData
	queryMethodMap[syscontract.PrivateComputeFunction_SAVE_ENCLAVE_REPORT.String()] = privateComputeRuntime.SaveEnclaveReport
	queryMethodMap[syscontract.PrivateComputeFunction_GET_DIR.String()] = privateComputeRuntime.GetDir
	queryMethodMap[syscontract.PrivateComputeFunction_GET_CA_CERT.String()] = privateComputeRuntime.GetEnclaveCACert
	queryMethodMap[syscontract.PrivateComputeFunction_GET_ENCLAVE_PROOF.String()] = privateComputeRuntime.GetEnclaveProof
	queryMethodMap[syscontract.PrivateComputeFunction_CHECK_CALLER_CERT_AUTH.String()] = privateComputeRuntime.CheckCallerCertAuth
	queryMethodMap[syscontract.PrivateComputeFunction_GET_ENCLAVE_ENCRYPT_PUB_KEY.String()] = privateComputeRuntime.GetEnclaveEncryptPubKey
	queryMethodMap[syscontract.PrivateComputeFunction_GET_ENCLAVE_VERIFICATION_PUB_KEY.String()] = privateComputeRuntime.GetEnclaveVerificationPubKey
	queryMethodMap[syscontract.PrivateComputeFunction_GET_ENCLAVE_REPORT.String()] = privateComputeRuntime.GetEnclaveReport
	queryMethodMap[syscontract.PrivateComputeFunction_GET_ENCLAVE_CHALLENGE.String()] = privateComputeRuntime.GetEnclaveChallenge
	queryMethodMap[syscontract.PrivateComputeFunction_GET_ENCLAVE_SIGNATURE.String()] = privateComputeRuntime.GetEnclaveSignature
	queryMethodMap[syscontract.PrivateComputeFunction_SAVE_REMOTE_ATTESTATION.String()] = privateComputeRuntime.SaveRemoteAttestation

	return queryMethodMap
}

type PrivateComputeRuntime struct {
	log protocol.Logger
}

func (r *PrivateComputeRuntime) VerifyByEnclaveCert(context protocol.TxSimContext, enclaveId []byte, data []byte, sign []byte) (bool, error) {
	enclaveCert, err := context.Get(syscontract.SystemContract_PRIVATE_COMPUTE.String(), enclaveId)
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
		err := fmt.Errorf("%s, key is empty", common.ErrParams.Error())
		r.log.Errorf(err.Error())
		return nil, err
	}

	value, err := context.Get(syscontract.SystemContract_PRIVATE_COMPUTE.String(), []byte(key))
	if err != nil {
		r.log.Errorf("Get key: %s from context failed, err: %s", key, err.Error())
		return nil, err
	}

	return value, nil
}

//
//func (r *PrivateComputeRuntime) SaveContract(context protocol.TxSimContext, params map[string][]byte) ([]byte, error) {
//	name := params["contract_name"]
//	code := params["contract_code"]
//	codeHash := params["code_hash"]
//	version := params["version"]
//	if utils.IsAnyBlank(name, code, codeHash, version) {
//		err := fmt.Errorf("%s, param[contract_name]=%s, param[contract_code]=%s, param[code_hash]=%s, params[version]=%s",
//			common.ErrParams.Error(), name, code, codeHash, version)
//		r.log.Errorf(err.Error())
//		return nil, err
//	}
//
//	calHash := sha256.Sum256([]byte(code))
//	if string(calHash[:]) != codeHash {
//		err := fmt.Errorf("%s, param[code_hash] != codeHash of param[contract_code] in save contract interface", common.ErrParams.Error())
//		r.log.Errorf(err.Error())
//		return nil, err
//	}
//
//	if len(version) > protocol.DefaultVersionLen {
//		err := fmt.Errorf("param[version] string of the contract[%+v] too long, should be less than %d", name, protocol.DefaultVersionLen)
//		r.log.Errorf(err.Error())
//		return nil, err
//	}
//
//	match, err := regexp.MatchString(protocol.DefaultVersionRegex, version)
//	if err != nil || !match {
//		err := fmt.Errorf("param[version] string of the contract[%+v] invalid while invoke user contract, should match [%s]", name, protocol.DefaultVersionRegex)
//		r.log.Errorf(err.Error())
//		return nil, err
//	}
//
//	combinationName := syscontract.SystemContract_PRIVATE_COMPUTE.String() + name
//	versionKey := []byte(protocol.ContractVersion)
//	versionInCtx, err := context.Get(combinationName, versionKey)
//	if err != nil {
//		err := fmt.Errorf("unable to find latest version for contract[%s], system error:%s", name, err.Error())
//		r.log.Errorf(err.Error())
//		return nil, err
//	}
//
//	if versionInCtx != nil {
//		err := fmt.Errorf("the contract already exists. contract[%s], version[%s]", name, string(versionInCtx))
//		r.log.Errorf(err.Error())
//		return nil, err
//	}
//
//	if err := context.Put(combinationName, versionKey, []byte(version)); err != nil {
//		r.log.Errorf("Put contract version into DB failed while save contract, err: %s", err.Error())
//		return nil, err
//	}
//
//	key := append([]byte(protocol.ContractByteCode), []byte(version)...)
//	if err := context.Put(combinationName, key, []byte(code)); err != nil {
//		r.log.Errorf("Put compute contract[%s] failed, err: %s", err.Error(), name)
//		return nil, err
//	}
//
//	return nil, nil
//}

func (r *PrivateComputeRuntime) saveContract(context protocol.TxSimContext, name, version string,
	codeHeader, code []byte, codeHash string) error {
	if utils.IsAnyBlank(name, version, string(codeHeader), string(code), codeHash) {
		err := fmt.Errorf("%s, param[contract_name]=%s, param[contract_code]=%s, param[code_hash]=%s, params[version]=%s",
			common.ErrParams.Error(), name, code, codeHash, version)
		r.log.Errorf(err.Error())
		return err
	}
	headerLen := len(codeHeader)
	fullCodes := make([]byte, headerLen+len(code))
	copy(fullCodes, codeHeader)
	copy(fullCodes[headerLen:], code)

	calHash := sha256.Sum256(fullCodes)
	if string(calHash[:]) != codeHash {
		err := fmt.Errorf("%s, param[code_hash] %x != calculated hash of codes: %x, full codes: %x",
			common.ErrParams.Error(), []byte(codeHash), calHash, fullCodes)
		r.log.Errorf(err.Error())
		return err
	}

	if len(version) > protocol.DefaultVersionLen {
		err := fmt.Errorf("param[version] string of the contract[%+v] too long, should be less than %d", name, protocol.DefaultVersionLen)
		r.log.Errorf(err.Error())
		return err
	}

	match, err := regexp.MatchString(protocol.DefaultVersionRegex, version)
	if err != nil || !match {
		err := fmt.Errorf("param[version] string of the contract[%+v] invalid while invoke user contract, should match [%s]", name, protocol.DefaultVersionRegex)
		r.log.Errorf(err.Error())
		return err
	}

	combinationName := syscontract.SystemContract_PRIVATE_COMPUTE.String() + name
	versionKey := []byte(ContractVersion)
	versionInCtx, err := context.Get(combinationName, versionKey)
	if err != nil {
		err := fmt.Errorf("unable to find latest version for contract[%s], system error:%s", name, err.Error())
		r.log.Errorf(err.Error())
		return err
	}

	if versionInCtx != nil {
		err := fmt.Errorf("the contract already exists. contract[%s], version[%s]", name, string(versionInCtx))
		r.log.Errorf(err.Error())
		return err
	}

	if err := context.Put(combinationName, versionKey, []byte(version)); err != nil {
		r.log.Errorf("Put contract version into DB failed while save contract, err: %s", err.Error())
		return err
	}

	key := append([]byte(ContractByteCode), []byte(version)...)
	if err := context.Put(combinationName, key, []byte(code)); err != nil {
		r.log.Errorf("Put compute contract[%s] failed, err: %s", err.Error(), name)
		return err
	}

	headerKey := append([]byte(ContractByteHeader), []byte(version)...)
	if err := context.Put(combinationName, headerKey, []byte(codeHeader)); err != nil {
		r.log.Errorf("Put compute contract[%s] failed, err: %s", err.Error(), name)
		return err
	}

	return nil
}

//
//func (r *PrivateComputeRuntime) UpdateContract(context protocol.TxSimContext, params map[string][]byte) ([]byte, error) {
//	name := params["contract_name"]
//	code := params["contract_code"]
//	hash := params["code_hash"]
//	version := params["version"]
//	if utils.IsAnyBlank(name, code, hash, version) {
//		err := fmt.Errorf("%s, param[contract_name]=%s, param[contract_code]=%s, param[code_hash]=%s, params[version]=%s",
//			common.ErrParams.Error(), name, code, hash, version)
//		r.log.Errorf(err.Error())
//		return nil, err
//	}
//
//	calHash := sha256.Sum256([]byte(code))
//	if string(calHash[:]) != hash {
//		err := fmt.Errorf("%s, param hash[%v] != param contract_code hash[%v] in save contract interface", common.ErrParams.Error(), []byte(hash), calHash)
//		r.log.Errorf(err.Error())
//		return nil, err
//	}
//
//	if len(version) > protocol.DefaultVersionLen {
//		err := fmt.Errorf("param[version] string of the contract[%+v] too long, should be less than %d", name, protocol.DefaultVersionLen)
//		r.log.Errorf(err.Error())
//		return nil, err
//	}
//
//	match, err := regexp.MatchString(protocol.DefaultVersionRegex, version)
//	if err != nil || !match {
//		err := fmt.Errorf("param[version] string of the contract[%+v] invalid while invoke user contract, should match [%s]", name, protocol.DefaultVersionRegex)
//		r.log.Errorf(err.Error())
//		return nil, err
//	}
//
//	combinationName := syscontract.SystemContract_PRIVATE_COMPUTE.String() + name
//	versionKey := []byte(protocol.ContractVersion)
//	versionInCtx, err := context.Get(combinationName, versionKey)
//	if err != nil {
//		err := fmt.Errorf("unable to find latest version for contract[%s], system error:%s", name, err.Error())
//		r.log.Errorf(err.Error())
//		return nil, err
//	}
//
//	if len(versionInCtx) == 0 {
//		err := fmt.Errorf("the contract[%s] does not exist, update failed", name)
//		r.log.Errorf(err.Error())
//		return nil, err
//	}
//
//	key := append([]byte(protocol.ContractByteCode), []byte(version)...)
//	codeInCtx, err := context.Get(combinationName, key)
//	if err == nil && len(codeInCtx) > 0 {
//		err := fmt.Errorf("the contract version[%s] and code[%s] is already exist", version, codeInCtx)
//		r.log.Errorf(err.Error())
//		return nil, err
//	}
//
//	if err := context.Put(combinationName, versionKey, []byte(version)); err != nil {
//		r.log.Errorf("Put contract version into DB failed while save contract, err: %s", err.Error())
//		return nil, err
//	}
//
//	if err := context.Put(combinationName, key, []byte(code)); err != nil {
//		r.log.Errorf("Put compute contract[%s] failed, err: %s", err.Error(), name)
//		return nil, err
//	}
//
//	return nil, nil
//}

func (r *PrivateComputeRuntime) GetContract(context protocol.TxSimContext, params map[string][]byte) ([]byte, error) {
	name := string(params["contract_name"])
	if utils.IsAnyBlank(name) {
		err := fmt.Errorf("%s, param[contract_name] of get contract not found", common.ErrParams.Error())
		r.log.Errorf(err.Error())
		return nil, err
	}

	codehash := string(params["code_hash"])
	if utils.IsAnyBlank(codehash) {
		err := fmt.Errorf("%s, param[code_hash] of get contract not found", common.ErrParams.Error())
		r.log.Errorf(err.Error())
		return nil, err
	}

	combinationName := syscontract.SystemContract_PRIVATE_COMPUTE.String() + name
	version, err := context.Get(combinationName, []byte(ContractVersion))
	if err != nil {
		r.log.Errorf("Unable to find latest version for contract[%s], system error:%s.", name, err.Error())
		return nil, err
	}

	if len(version) == 0 {
		r.log.Errorf("The contract does not exist. contract[%s].", name)
		return nil, err
	}

	var result commonPb.PrivateGetContract
	key := append([]byte(ContractByteCode), version...)
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

	headerKey := append([]byte(ContractByteHeader), version...)
	headerCode, err := context.Get(combinationName, headerKey)
	if err != nil {
		r.log.Errorf("Read contract code header[%s] failed.", name)
		return nil, err
	}
	r.log.Infof("get contract, name[%s], header code[%v]", name, headerCode)

	if len(headerCode) == 0 {
		r.log.Errorf("Contract[%s] header code is empty.", name)
		return nil, err
	}

	headerLen := len(headerCode)
	fullCodes := make([]byte, headerLen+len(contractCode))
	copy(fullCodes, headerCode)
	copy(fullCodes[headerLen:], contractCode)

	calHash := sha256.Sum256(fullCodes)
	if string(calHash[:]) != codehash {
		err := fmt.Errorf("%s, param codehash[%v] != contract code codehash[%v] in get contract interface",
			common.ErrParams.Error(), []byte(codehash), calHash)
		r.log.Errorf(err.Error())
		return nil, err
	}

	result.ContractCode = contractCode
	result.GasLimit = protocol.GasLimit
	result.Version = string(version)

	return result.Marshal()
}

func (r *PrivateComputeRuntime) SaveDir(context protocol.TxSimContext, params map[string][]byte) ([]byte, error) {
	key := string(params["order_id"])
	if utils.IsAnyBlank(key) {
		err := fmt.Errorf("%s, param[order_id] of save dir  not found", common.ErrParams.Error())
		r.log.Errorf(err.Error())
		return nil, err
	}

	value := string(params["private_dir"])
	if utils.IsAnyBlank(value) {
		err := fmt.Errorf("%s, param[private_key] of save dir not found", common.ErrParams.Error())
		r.log.Errorf(err.Error())
		return nil, err
	}

	if err := context.Put(syscontract.SystemContract_PRIVATE_COMPUTE.String(), []byte(key), []byte(value)); err != nil {
		r.log.Errorf("Put private dir failed, err: %s", err.Error())
		return nil, err
	}

	return nil, nil
}

func (r *PrivateComputeRuntime) GetDir(context protocol.TxSimContext, params map[string][]byte) ([]byte, error) {
	return r.getValue(context, string(params["order_id"]))
}

func (r *PrivateComputeRuntime) SaveData(context protocol.TxSimContext, params map[string][]byte) ([]byte, error) {
	name := string(params["contract_name"])
	version := string(params["version"])
	codeHash := string(params["code_hash"])
	reportHash := string(params["report_hash"])
	userCert := string(params["user_cert"])
	clientSign := string(params["client_sign"])
	orgId := string(params["org_id"])
	isDeployStr := string(params["is_deploy"])
	codeHeader := string(params["code_header"])
	cRes := []byte(params["result"])
	r.log.Debugf("save data received code header len: %d, code header: %x", len(codeHeader), []byte(codeHeader))

	/*get private contract compute result form cRes unmarshal*/
	var result commonPb.ContractResult
	if err := result.Unmarshal(cRes); err != nil {
		r.log.Errorf("Unmarshal ContractResult failed, err: %s", err.Error())
		return nil, err
	}

	if isDeployStr == "" {
		err := fmt.Errorf("is_deploy param should not be empty")
		r.log.Errorf(err.Error())
		return nil, err
	}

	isDeploy, err := strconv.ParseBool(isDeployStr)
	if err != nil {
		r.log.Errorf(err.Error())
		return nil, err
	}

	/*check access control by sign pairs, org ids, payload bytes and ac*/
	ac, err := context.GetAccessControl()
	if err != nil {
		return nil, err
	}

	var signPairs []*syscontract.SignInfo
	var orgIds []string
	var payloadBytes []byte
	var requestBytes []byte
	//r.log.Debugf("Deploy request bytes: %v, isDeploy: %v", params[], isDeploy)
	if isDeploy {
		requestBytes = []byte(params["deploy_req"])
		deployReq, err := r.getDeployRequest(params)
		if err != nil || deployReq.SignPair == nil || deployReq.Payload == nil {
			err := fmt.Errorf("get private deploy request from params failed, err: %v", err)
			r.log.Errorf(err.Error())
			return nil, err
		}

		r.log.Debugf("deployReq: %v", deployReq)
		signPairs = deployReq.SignPair
		orgIds = deployReq.Payload.OrgId
		payloadBytes, err = deployReq.Payload.Marshal()
		if err != nil {
			err := fmt.Errorf("marshal deploy request payload failed, err: %v", err)
			r.log.Errorf(err.Error())
			return nil, err
		}
	} else {
		requestBytes = []byte(params["private_req"])
		req, err := r.getPrivateRequest(params)
		if err != nil || req.SignPair == nil || req.Payload == nil {
			err := fmt.Errorf("get private compute request from params failed, err: %v", err)
			r.log.Errorf(err.Error())
			return nil, err
		}

		signPairs = req.SignPair
		orgIds = req.Payload.OrgId
		payloadBytes, err = req.Payload.Marshal()
		if err != nil {
			err := fmt.Errorf("marshal compute request payload failed, err: %v", err)
			r.log.Errorf(err.Error())
			return nil, err
		}
	}

	auth, err := r.verifyMultiCallerAuth(signPairs, orgIds, payloadBytes, ac)
	if !auth || err != nil {
		err := fmt.Errorf("verify user auth failed, user_cert[%v], signature[%v], request payload[code_hash]=%v",
			params["user_cert"], params["client_sign"], params["payload"])
		r.log.Errorf(err.Error())
		return nil, err
	}

	/*if deploy, save private contract code*/
	if isDeploy && (codeHeader == "" || len(result.Result) == 0) {
		r.log.Errorf("code_header should not be empty when deploying contract")
		return nil, err
	}

	if isDeploy {
		err := r.saveContract(context, name, version, []byte(codeHeader), result.Result, codeHash)
		if err != nil {
			r.log.Errorf("save contract err: %s", err.Error())
			return nil, err
		}
	}

	if utils.IsAnyBlank(name, version, codeHash, reportHash) {
		err := fmt.Errorf(
			"%s, param[contract_name]=%s, params[version]=%s, param[code_hash]=%s, param[report_hash]=%s, "+
				"params[user_cert]=%s, params[client_sign]=%s, params[payload]=%s, params[org_id]=%s,",
			common.ErrParams.Error(), name, version, codeHash, reportHash, userCert, clientSign, requestBytes, orgId)
		r.log.Errorf(err.Error())
		return nil, err
	}

	rwb := []byte(params["rw_set"])
	r.log.Debug("rwset bytes: ", rwb)
	var rwSet commonPb.TxRWSet
	if err := rwSet.Unmarshal(rwb); err != nil {
		r.log.Errorf("Unmarshal RWSet failed, err: %s", err.Error())
		return nil, err
	}

	/* get PEM, pk and construct private contract compute result, then verify sign */
	combinedKey := syscontract.SystemContract_PRIVATE_COMPUTE.String() + "global_enclave_id"
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

	// Code
	if err := binary.Write(evmResultBuffer, binary.LittleEndian, result.Code); err != nil {
		return nil, err
	}
	// Result
	if err := binary.Write(evmResultBuffer, binary.LittleEndian, uint32(len(result.Result))); err != nil {
		return nil, err
	}
	evmResultBuffer.Write(result.Result)
	// Gas
	if err := binary.Write(evmResultBuffer, binary.LittleEndian, uint64(result.GasUsed)); err != nil {
		return nil, err
	}
	// rsets
	if err := binary.Write(evmResultBuffer, binary.LittleEndian, uint32(len(rwSet.TxReads))); err != nil {
		return nil, err
	}
	for i := 0; i < len(rwSet.TxReads); i++ {
		// Key
		if err := binary.Write(evmResultBuffer, binary.LittleEndian, uint32(len(rwSet.TxReads[i].Key))); err != nil {
			return nil, err
		}
		evmResultBuffer.Write(rwSet.TxReads[i].Key)
		// Value
		if err := binary.Write(evmResultBuffer, binary.LittleEndian, uint32(len(rwSet.TxReads[i].Value))); err != nil {
			return nil, err
		}
		evmResultBuffer.Write(rwSet.TxReads[i].Value)
		// Version
		if err := binary.Write(evmResultBuffer, binary.LittleEndian, uint32(0)); err != nil {
			return nil, err
		}
		// evmResultBuffer.Write([]byte(rwSet.TxReads[i].Version.RefTxId))
	}

	// wsets
	if err := binary.Write(evmResultBuffer, binary.LittleEndian, uint32(len(rwSet.TxWrites))); err != nil {
		return nil, err
	}
	for i := 0; i < len(rwSet.TxWrites); i++ {
		// Key
		if err := binary.Write(evmResultBuffer, binary.LittleEndian, uint32(len(rwSet.TxWrites[i].Key))); err != nil {
			return nil, err
		}
		evmResultBuffer.Write(rwSet.TxWrites[i].Key)

		// Value
		if err := binary.Write(evmResultBuffer, binary.LittleEndian, uint32(len(rwSet.TxWrites[i].Value))); err != nil {
			return nil, err
		}
		evmResultBuffer.Write(rwSet.TxWrites[i].Value)
	}

	// name
	if err := binary.Write(evmResultBuffer, binary.LittleEndian, uint32(len(name))); err != nil {
		return nil, err
	}
	evmResultBuffer.Write([]byte(name))
	// version
	if err := binary.Write(evmResultBuffer, binary.LittleEndian, uint32(len(version))); err != nil {
		return nil, err
	}
	evmResultBuffer.Write([]byte(version))
	// code hash
	if err := binary.Write(evmResultBuffer, binary.LittleEndian, uint32(len(codeHash))); err != nil {
		return nil, err
	}
	evmResultBuffer.Write([]byte(codeHash))
	// report hash
	if err := binary.Write(evmResultBuffer, binary.LittleEndian, uint32(len(reportHash))); err != nil {
		return nil, err
	}
	evmResultBuffer.Write([]byte(reportHash))
	// user request
	if err := binary.Write(evmResultBuffer, binary.LittleEndian, uint32(len(requestBytes))); err != nil {
		return nil, err
	}
	evmResultBuffer.Write(requestBytes)
	// code header
	if err := binary.Write(evmResultBuffer, binary.LittleEndian, uint32(len(codeHeader))); err != nil {
		return nil, err
	}
	evmResultBuffer.Write([]byte(codeHeader))
	evmResultBytes := evmResultBuffer.Bytes()
	b, err := pk.VerifyWithOpts(evmResultBytes, []byte(params["sign"]), &crypto.SignOpts{
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

	/* check contract code hash */
	combinationName := syscontract.SystemContract_PRIVATE_COMPUTE.String() + name
	key := append([]byte(ContractByteCode), version...)
	contractCode, err := context.Get(combinationName, key)
	if err != nil || len(contractCode) == 0 {
		r.log.Errorf("Read contract[%s] failed.", name)
		return nil, err
	}

	headerKey := append([]byte(ContractByteHeader), version...)
	headerCode, err := context.Get(combinationName, headerKey)
	if err != nil {
		r.log.Errorf("Save data: read contract code header[%s] failed.", name)
		return nil, err
	}
	r.log.Infof("Save data: contract name[%s], header code[%v]", name, headerCode)

	if len(headerCode) == 0 {
		r.log.Errorf("Contract[%s] header code is empty.", name)
		return nil, err
	}

	headerLen := len(headerCode)
	fullCodes := make([]byte, headerLen+len(contractCode))
	copy(fullCodes, headerCode)
	copy(fullCodes[headerLen:], contractCode)

	calHash := sha256.Sum256(fullCodes)
	if string(calHash[:]) != codeHash {
		err := fmt.Errorf("%s, param[code_hash] != hash of contract code in get contract interface", common.ErrParams.Error())
		r.log.Errorf(err.Error())
		return nil, err
	}

	/*check gas limit*/
	if result.GasUsed > protocol.GasLimit {
		err := fmt.Errorf("gas[%d] expend the limit[%f]", result.GasUsed, protocol.GasLimit)
		r.log.Errorf(err.Error())
		return nil, err
	}

	/*save private contract compute result*/
	if err := context.Put(combinationName, []byte(ComputeResult), cRes); err != nil {
		r.log.Errorf("Write compute result:%s failed, err: %s", cRes, err.Error())
		return nil, err
	}

	if result.Code != 0 {
		r.log.Infof("Compute result code != ok, return")
		return nil, nil
	}

	/*check read set version and save rwSet*/
	for i := 0; i < len(rwSet.TxReads); i++ {
		key := rwSet.TxReads[i].Key
		val := rwSet.TxReads[i].Value
		//version := rwSet.TxReads[i].Version
		chainValue, err := context.Get(combinationName, key)
		if err != nil {
			r.log.Errorf("Get key: %s failed, err: %s", key, err.Error())
			return nil, err
		}
		r.log.Infof("RSet key: %v value: %v, value on chain: %v\n", key, val, chainValue)
		//var rSet commonPb.TxRead
		//if err := rSet.Unmarshal(chainRSetBytes); err != nil {
		//	r.log.Errorf("Unmarshal RSet failed, err: %s", err.Error())
		//	return nil, err
		//}

		if chainValue != nil && len(chainValue) > 0 && !bytes.Equal(val, chainValue) {
			r.log.Errorf("rSet verification failed! key: %v, value: %v; but value on chain: %v\n",
				key, val, chainValue)
			return nil, fmt.Errorf("rSet verification failed! key: %v, value: %v, but value on chain: %v",
				key, val, chainValue)
		}
	}

	for j := 0; j < len(rwSet.TxWrites); j++ {
		key := rwSet.TxWrites[j].Key
		val := rwSet.TxWrites[j].Value
		if err := context.Put(combinationName, key, val); err != nil {
			r.log.Errorf("Put key: %s, value:%s into write set failed, err: %s", key, val, err.Error())
			return nil, err
		}
	}

	return nil, nil
}

func (r *PrivateComputeRuntime) GetData(context protocol.TxSimContext, params map[string][]byte) ([]byte, error) {
	key := []byte(params["key"])
	if utils.IsAnyBlank(string(params["key"])) {
		err := fmt.Errorf("%s,param[private_key] of get data  not found", common.ErrParams.Error())
		r.log.Errorf(err.Error())
		return nil, err
	}

	name := string(params["contract_name"])
	//if res != true {
	//	name = ""
	//}

	combinationName := syscontract.SystemContract_PRIVATE_COMPUTE.String() + name
	value, err := context.Get(combinationName, key)
	if err != nil {
		r.log.Errorf("Get key: %s from context failed, err: %s", key, err.Error())
		return nil, err
	}

	return value, nil
}

func (r *PrivateComputeRuntime) GetEnclaveCACert(context protocol.TxSimContext, params map[string][]byte) ([]byte, error) {

	caCertPEM, err := context.Get(syscontract.SystemContract_PRIVATE_COMPUTE.String(), []byte("ca_cert"))
	if err != nil {
		r.log.Errorf("get enclave ca cert failed: %v", err.Error())
		return nil, err
	}

	return caCertPEM, nil
}

func (r *PrivateComputeRuntime) SaveEnclaveCACert(context protocol.TxSimContext, params map[string][]byte) ([]byte, error) {
	// PEM 格式的证书
	caCertPEM := string(params["ca_cert"])
	if utils.IsAnyBlank(caCertPEM) {
		err := fmt.Errorf("%s,param[ca_cert] does not found", common.ErrParams.Error())
		r.log.Errorf(err.Error())
		return nil, err
	}

	if err := context.Put(syscontract.SystemContract_PRIVATE_COMPUTE.String(), []byte("ca_cert"), []byte(caCertPEM)); err != nil {
		r.log.Errorf("save enclave ca cert failed: %v", err.Error())
		return nil, err
	}

	return nil, nil
}

func (r *PrivateComputeRuntime) SaveRemoteAttestation(context protocol.TxSimContext, params map[string][]byte) ([]byte, error) {
	// get params
	proofDataStr := string(params["proof"])
	r.log.Debug("SaveRemoteAttestation start, proof data: ", proofDataStr)
	if utils.IsAnyBlank(proofDataStr) {
		err := fmt.Errorf("'proof' is nil")
		r.log.Errorf(err.Error())
		return nil, err
	}

	proofData, err := hex.DecodeString(proofDataStr)
	r.log.Debug("SaveRemoteAttestation decoded proof data: ", proofData)
	if err != nil {
		r.log.Errorf(err.Error())
		return nil, err
	}

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
	enclaveIdKey := syscontract.SystemContract_PRIVATE_COMPUTE.String() + enclaveId
	reportFromChain, err := context.Get(enclaveIdKey, []byte("report"))
	if err != nil {
		err := fmt.Errorf("get enclave 'report' from chain error: %v", err)
		r.log.Errorf(err.Error())
		return nil, err
	}

	// get ca_cert from chain
	caCertPem, err := context.Get(syscontract.SystemContract_PRIVATE_COMPUTE.String(), []byte("ca_cert"))
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

func (r *PrivateComputeRuntime) GetEnclaveEncryptPubKey(context protocol.TxSimContext, params map[string][]byte) ([]byte, error) {
	// get params
	enclaveId := string(params["enclave_id"])
	if utils.IsAnyBlank(enclaveId) {
		err := fmt.Errorf("%s,param[ca_cert] of save cert  not found", common.ErrParams.Error())
		r.log.Errorf(err.Error())
		return nil, err
	}

	// get data from chain
	combinedKey := syscontract.SystemContract_PRIVATE_COMPUTE.String() + enclaveId
	pemEncryptPubKey, err := context.Get(combinedKey, []byte("encrypt_pub_key"))
	if err != nil {
		err := fmt.Errorf("get 'encrypt_pub_key' from chain error: %v", err)
		r.log.Errorf(err.Error())
		return nil, err
	}

	return pemEncryptPubKey, nil
}

func (r *PrivateComputeRuntime) GetEnclaveVerificationPubKey(context protocol.TxSimContext, params map[string][]byte) ([]byte, error) {
	// get params
	enclaveId := string(params["enclave_id"])
	if utils.IsAnyBlank(enclaveId) {
		err := fmt.Errorf("%s,param['ca_cert'] of save cert  not found", common.ErrParams.Error())
		r.log.Errorf(err.Error())
		return nil, err
	}

	// get data from chain
	combinedKey := syscontract.SystemContract_PRIVATE_COMPUTE.String() + enclaveId
	pemVerificationPubKey, err := context.Get(combinedKey, []byte("verification_pub_key"))
	if err != nil {
		err := fmt.Errorf("get 'verification_pub_key' from chain error: %v", err)
		r.log.Errorf(err.Error())
		return nil, err
	}

	return pemVerificationPubKey, nil
}

func (r *PrivateComputeRuntime) SaveEnclaveReport(context protocol.TxSimContext, params map[string][]byte) ([]byte, error) {
	// get params
	enclaveId := string(params["enclave_id"])
	report := string(params["report"])
	if utils.IsAnyBlank(enclaveId, report) {
		err := fmt.Errorf("%s,param['enclave_id'] or param['report'] does not found", common.ErrParams.Error())
		r.log.Errorf(err.Error())
		return nil, err
	}
	reportStr, err := hex.DecodeString(report)
	if err != nil {
		r.log.Errorf(err.Error())
		return nil, err
	}
	r.log.Debugf("Save enclave report start, orginal report data: %s, decoded report data: %s", report, reportStr)
	// save report into chain
	enclaveIdKey := syscontract.SystemContract_PRIVATE_COMPUTE.String() + enclaveId
	if err := context.Put(enclaveIdKey, []byte("report"), []byte(reportStr)); err != nil {
		err := fmt.Errorf("save enclave 'report' failed, err: %s", err.Error())
		r.log.Errorf(err.Error())
		return nil, err
	}

	return nil, nil
}

func (r *PrivateComputeRuntime) GetEnclaveReport(context protocol.TxSimContext, params map[string][]byte) ([]byte, error) {
	// get params
	enclaveId := string(params["enclave_id"])
	if utils.IsAnyBlank(enclaveId) {
		err := fmt.Errorf("%s,param['enclave_id'] not found", common.ErrParams.Error())
		r.log.Errorf(err.Error())
		return nil, err
	}

	// get data from chain
	enclaveIdKey := syscontract.SystemContract_PRIVATE_COMPUTE.String() + enclaveId
	report, err := context.Get(enclaveIdKey, []byte("report"))
	if err != nil {
		err := fmt.Errorf("get 'report' from chain error: %v", err)
		r.log.Errorf(err.Error())
		return nil, err
	}

	reportBytes := make([]byte, hex.EncodedLen(len(report)))
	hex.Encode(reportBytes, report)
	return reportBytes, nil
}

func (r *PrivateComputeRuntime) GetEnclaveChallenge(context protocol.TxSimContext, params map[string][]byte) ([]byte, error) {
	// 证书二进制数据
	enclaveId := string(params["enclave_id"])
	if utils.IsAnyBlank(enclaveId) {
		err := fmt.Errorf("%s,param['enclave_id'] not found", common.ErrParams.Error())
		r.log.Errorf(err.Error())
		return nil, err
	}

	// get data from chain
	enclaveIdKey := syscontract.SystemContract_PRIVATE_COMPUTE.String() + enclaveId
	challenge, err := context.Get(enclaveIdKey, []byte("challenge"))
	if err != nil {
		err := fmt.Errorf("get 'challenge' from chain error: %v", err)
		r.log.Errorf(err.Error())
		return nil, err
	}

	return challenge, nil
}

func (r *PrivateComputeRuntime) GetEnclaveSignature(context protocol.TxSimContext, params map[string][]byte) ([]byte, error) {
	// 证书二进制数据
	enclaveId := string(params["enclave_id"])
	if utils.IsAnyBlank(enclaveId) {
		err := fmt.Errorf("%s,param['enclave_id'] not found", common.ErrParams.Error())
		r.log.Errorf(err.Error())
		return nil, err
	}

	// get data from chain
	combinedKey := syscontract.SystemContract_PRIVATE_COMPUTE.String() + enclaveId
	signature, err := context.Get(combinedKey, []byte("signature"))
	if err != nil {
		err := fmt.Errorf("get 'signature' from chain error: %v", err)
		r.log.Errorf(err.Error())
		return nil, err
	}

	return signature, nil
}

func (r *PrivateComputeRuntime) GetEnclaveProof(context protocol.TxSimContext, params map[string][]byte) ([]byte, error) {
	// 证书二进制数据
	enclaveId := string(params["enclave_id"])
	if utils.IsAnyBlank(enclaveId) {
		err := fmt.Errorf("%s,param['enclave_id'] not found", common.ErrParams.Error())
		r.log.Errorf(err.Error())
		return nil, err
	}

	// get data from chain
	combinedKey := syscontract.SystemContract_PRIVATE_COMPUTE.String() + enclaveId
	proof, err := context.Get(combinedKey, []byte("proof"))
	if err != nil {
		err := fmt.Errorf("get 'proof' from chain error: %v", err)
		r.log.Errorf(err.Error())
		return nil, err
	}
	proofBytes := make([]byte, hex.EncodedLen(len(proof)))
	hex.Encode(proofBytes, proof)
	return proofBytes, nil
}

func (r *PrivateComputeRuntime) CheckCallerCertAuth(ctx protocol.TxSimContext, params map[string][]byte) ([]byte, error) {
	ac, err := ctx.GetAccessControl()
	if err != nil {
		return nil, err
	}
	signPairStr := params["sign_pairs"]
	payloadByteStr := params["payload"]
	orgIdStr := params["org_ids"]
	var signPairs []*syscontract.SignInfo
	err = json.Unmarshal([]byte(signPairStr), &signPairs)
	if err != nil {
		return nil, err
	}
	var orgIds []string
	err = json.Unmarshal([]byte(orgIdStr), &orgIds)
	if err != nil {
		return nil, err
	}
	payloadBytes := make([]byte, hex.DecodedLen(len(payloadByteStr)))
	_, err = hex.Decode(payloadBytes, []byte(payloadByteStr))
	if err != nil {
		return nil, err
	}
	//auth, err := r.verifyCallerAuth(params, ctx.GetTx().Header.ChainId, ac)
	auth, err := r.verifyMultiCallerAuth(signPairs, orgIds, payloadBytes, ac)
	if err != nil {
		return nil, err
	}

	return []byte(strconv.FormatBool(auth)), nil
}

func (r *PrivateComputeRuntime) verifyCallerAuth(params map[string][]byte, chainId string, ac protocol.AccessControlProvider) (bool, error) { //todo delete

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

	sender := &accesscontrol.Member{
		OrgId:      orgId,
		MemberInfo: userCertPemBytes,
		MemberType: accesscontrol.MemberType_CERT,
	}

	endorsements := []*commonPb.EndorsementEntry{{
		Signer:    sender,
		Signature: clientSignBytes,
	}}

	principal, err := ac.CreatePrincipal("PRIVATE_COMPUTE", endorsements, payLoadBytes) //todo pb
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

func (r *PrivateComputeRuntime) getOrgId(payLoad []byte) (string, error) { //todo delete
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

func (r *PrivateComputeRuntime) getParamValue(parameters map[string][]byte, key string) (string, error) {
	value, ok := parameters[key]
	if !ok {
		errMsg := fmt.Sprintf("miss params %s", key)
		r.log.Error(errMsg)
		return "", errors.New(errMsg)
	}
	return string(value), nil
}

func (r *PrivateComputeRuntime) verifyMultiCallerAuth(signPairs []*syscontract.SignInfo, orgId []string,
	payloadBytes []byte, ac protocol.AccessControlProvider) (bool, error) {
	for i, certPair := range signPairs {
		clientSignBytes, err := hex.DecodeString(certPair.ClientSign)
		if err != nil {
			r.log.Errorf("sign pair number is: %v ,client sign hex err:%v", i, err.Error())
			return false, err
		}
		fmt.Printf("++++++++++++private clientSignBytges is %v++++++++++", clientSignBytes)

		userCertPemBytes, err := hex.DecodeString(certPair.Cert)
		if err != nil {
			r.log.Errorf("sign pair number is: %v ,user cert pem hex err:%v", i, err.Error())
			return false, err
		}

		sender := &accesscontrol.Member{
			OrgId:      orgId[i],
			MemberInfo: userCertPemBytes,
			MemberType: accesscontrol.MemberType_CERT,
		}

		endorsements := []*commonPb.EndorsementEntry{{
			Signer:    sender,
			Signature: clientSignBytes,
		}}

		principal, err := ac.CreatePrincipal("PRIVATE_COMPUTE", endorsements, payloadBytes) //todo pb
		if err != nil {
			return false, fmt.Errorf("sign pair number is: %v ,fail to construct authentication principal: %s", i, err.Error())
		}

		ok, err := ac.VerifyPrincipal(principal)
		if err != nil {
			return false, fmt.Errorf("sign pair number is: %v ,authentication error, %s", i, err.Error())
		}

		if !ok {
			return false, fmt.Errorf("sign pair number is: %v ,authentication failed", i)
		}
	}
	return true, nil
}

//
//func (r *PrivateComputeRuntime) verifyMultiCallerAuth(params map[string][]byte, ac protocol.AccessControlProvider) (bool, error) {
//
//	req, err := r.getPrivateRequest(params)
//	if err != nil {
//		return false, err
//	}
//
//	for i, certPair := range req.SignPair {
//		payLoadBytes, err := req.Payload.Marshal()
//		if err != nil {
//			r.log.Errorf("sign pair number is: %v ,payload marshal err:%v", i, err.Error())
//			return false, err
//		}
//
//		clientSignBytes, err := hex.DecodeString(certPair.ClientSign)
//		if err != nil {
//			r.log.Errorf("sign pair number is: %v ,client sign hex err:%v", i, err.Error())
//			return false, err
//		}
//		fmt.Printf("++++++++++++private clientSignBytges is %v++++++++++", clientSignBytes)
//
//		userCertPemBytes, err := hex.DecodeString(certPair.Cert)
//		if err != nil {
//			r.log.Errorf("sign pair number is: %v ,user cert pem hex err:%v", i, err.Error())
//			return false, err
//		}
//
//		sender := &accesscontrol.Member{
//			OrgId:      req.Payload.OrgId[i],
//			MemberInfo: userCertPemBytes,
//			//IsFullCert: true,
//		}
//
//		endorsements := []*commonPb.EndorsementEntry{{
//			Signer:    sender,
//			Signature: clientSignBytes,
//		}}
//
//		principal, err := ac.CreatePrincipal("PRIVATE_COMPUTE", endorsements, payLoadBytes) //todo pb
//		if err != nil {
//			return false, fmt.Errorf("sign pair number is: %v ,fail to construct authentication principal: %s", i, err.Error())
//		}
//
//		ok, err := ac.VerifyPrincipal(principal)
//		if err != nil {
//			return false, fmt.Errorf("sign pair number is: %v ,authentication error, %s", i, err.Error())
//		}
//
//		if !ok {
//			return false, fmt.Errorf("sign pair number is: %v ,authentication failed", i)
//		}
//	}
//	return true, nil
//}

func (r *PrivateComputeRuntime) getPrivateRequest(params map[string][]byte) (*syscontract.PrivateComputeRequest, error) {
	privateReq, err := r.getParamValue(params, "private_req")
	if err != nil {
		return nil, err
	}

	//privateReqBytes, err := hex.DecodeString(privateReq)
	req := &syscontract.PrivateComputeRequest{}
	if err := req.Unmarshal([]byte(privateReq)); err != nil {
		return nil, err
	}

	return req, nil
}

func (r *PrivateComputeRuntime) getDeployRequest(params map[string][]byte) (*syscontract.PrivateDeployRequest, error) {
	deployReq, err := r.getParamValue(params, "deploy_req")
	if err != nil {
		return nil, err
	}

	//deployReqBytes, err := hex.DecodeString(deployReq)
	req := &syscontract.PrivateDeployRequest{}
	if err := req.Unmarshal([]byte(deployReq)); err != nil {
		return nil, err
	}

	return req, nil
}
