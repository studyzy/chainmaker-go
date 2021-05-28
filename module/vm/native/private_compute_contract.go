/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package native

import (
    "bytes"
    "chainmaker.org/chainmaker-go/common/crypto/hash"
    bcx509 "chainmaker.org/chainmaker-go/common/crypto/x509"
    "chainmaker.org/chainmaker-go/logger"
    commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
    "chainmaker.org/chainmaker-go/protocol"
    "chainmaker.org/chainmaker-go/utils"
    "crypto/sha256"
    "fmt"
    "regexp"
    "strings"
)

const(
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
    queryMethodMap[commonPb.PrivateComputeContractFunction_SAVE_CERT.String()] = privateComputeRuntime.SaveCert
    queryMethodMap[commonPb.PrivateComputeContractFunction_SAVE_DIR.String()] = privateComputeRuntime.SaveDir
    queryMethodMap[commonPb.PrivateComputeContractFunction_SAVE_DATA.String()] = privateComputeRuntime.SaveData
    queryMethodMap[commonPb.PrivateComputeContractFunction_SAVE_CONTRACT.String()] = privateComputeRuntime.SaveContract
    queryMethodMap[commonPb.PrivateComputeContractFunction_SAVE_QUOTE.String()] = privateComputeRuntime.SaveQuote
    queryMethodMap[commonPb.PrivateComputeContractFunction_GET_DIR.String()] = privateComputeRuntime.GetDir
    queryMethodMap[commonPb.PrivateComputeContractFunction_GET_CERT.String()] = privateComputeRuntime.GetCert
    queryMethodMap[commonPb.PrivateComputeContractFunction_GET_QUOTE.String()] = privateComputeRuntime.GetQuote
    queryMethodMap[commonPb.PrivateComputeContractFunction_UPDATE_CONTRACT.String()] = privateComputeRuntime.UpdateContract

    return queryMethodMap
}

type PrivateComputeRuntime struct {
    log *logger.CMLogger
}

func (r *PrivateComputeRuntime) VerifyByEnclaveCert(context protocol.TxSimContext, enclaveId []byte, data []byte, sign []byte) (bool, error){
    enclaveCert, err:= context.Get(commonPb.ContractName_SYSTEM_CONTRACT_PRIVATE_COMPUTE.String(), enclaveId)
    if  err != nil {
        r.log.Errorf("%s, get enclave cert[%s] failed", err.Error(), enclaveId)
        return false, err
    }

    cert, err := utils.ParseCert(enclaveCert)
    if  err != nil {
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

    return true,  nil
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

func (r *PrivateComputeRuntime) SaveQuote(context protocol.TxSimContext, params map[string]string) ([]byte, error) {
    sign      := params["sign"]
    quote     := params["quote"]
    quoteId   := params["quote_id"]
    enclaveId := params["enclave_id"]
    if utils.IsAllBlank(enclaveId, quoteId, quote, sign) {
        err := fmt.Errorf("%s, param[enclave_id]%s, param[quote_id]%s, param[quote]%s, param[sign]%s ",
            ErrParams.Error(), enclaveId, quoteId, quote, sign)
        r.log.Errorf(err.Error())
        return nil, err
    }

    //if ok, err := r.VerifyByEnclaveCert(context, []byte(enclaveId), []byte(quote), []byte(sign)); !ok {
    //    r.log.Errorf("%s, enclave certificate[%s] verify quote[%s] failed", err.Error(), enclaveId, quoteId)
    //    return nil, err
    //}

    if err := context.Put(commonPb.ContractName_SYSTEM_CONTRACT_PRIVATE_COMPUTE.String(), []byte(quoteId), []byte(quote)); err != nil{
        r.log.Errorf("%s, save quote[%s] failed", err.Error(), quoteId)
        return nil, err
    }

    return nil, nil
}

func (r *PrivateComputeRuntime) GetQuote(context protocol.TxSimContext, params map[string]string) ([]byte, error) {
    return r.getValue(context, params["quote_id"])
}

func (r *PrivateComputeRuntime) SaveContract(context protocol.TxSimContext, params map[string]string) ([]byte, error) {
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
        err := fmt.Errorf("%s, param[code_hash] != hash of param[contract_code] in save contract interface", ErrParams.Error())
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
    if  err != nil {
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
    result. GasLimit = protocol.GasLimit
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
    name := params["contract_name"]
    hash := params["hash"]
    version := params["version"]
    if utils.IsAnyBlank(name, hash, version) {
        err := fmt.Errorf("%s, param[contract_name]=%s, param[contract_code]=%s, param[code_hash]=%s, params[version]=%s",
            ErrParams.Error(), name, hash, version)
        r.log.Errorf(err.Error())
        return nil, err
    }

    combinationName := commonPb.ContractName_SYSTEM_CONTRACT_PRIVATE_COMPUTE.String() + name
    key := append([]byte(protocol.ContractByteCode), version...)
    contractCode, err := context.Get(combinationName, key)
    if err != nil || len(contractCode) == 0 {
        r.log.Errorf("Read contract[%s] failed.", name)
        return nil, err
    }

    calHash := sha256.Sum256(contractCode)
    if string(calHash[:]) != hash {
        err := fmt.Errorf("%s, param[code_hash] != hash of contract code in get contract interface", ErrParams.Error())
        r.log.Errorf(err.Error())
        return nil, err
    }

    var result commonPb.ContractResult
    cRes := []byte(params["result"])
    if err := result.Unmarshal(cRes); err != nil {
        r.log.Errorf("Unmarshal ContractResult failed, err: %s", err.Error())
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

    report := bytes.Join([][]byte{cRes, []byte(params["rw_set"]), []byte(params["events"])}, []byte{})
    ok, err := r.VerifyByEnclaveCert(context, []byte(params["enclave_id"]), report, []byte(params["report_sign"]))
    if !ok{
        r.log.Errorf("%s, enclave certificate[%s] verify quote of save data failed", err.Error(), params["enclave_id"])
        return nil, err
    }

    if result.Code != commonPb.ContractResultCode_OK {
        r.log.Infof("Compute result code != ok, return")
        return nil, nil
    }

    var rwSet commonPb.TxRWSet
    if err := rwSet.Unmarshal([]byte(params["rw_set"])); err != nil{
        r.log.Errorf("Unmarshal RWSet failed, err: %s", err.Error())
        return nil, err
    }

    for i := 0; i < len(rwSet.TxReads); i++ {
        key := rwSet.TxReads[i].Key
        val := rwSet.TxReads[i].Value
        if _, err := context.Get(combinationName, key); err != nil {
            r.log.Errorf("Put key: %s, value:%s into read set failed, err: %s", key, val, err.Error())
		}
    }

    for j :=0; j < len(rwSet.TxWrites); j++ {
        key := rwSet.TxWrites[j].Key
        val := rwSet.TxWrites[j].Value
        if err := context.Put(combinationName, key, val); err != nil {
            r.log.Errorf("Put key: %s, value:%s into write set failed, err: %s", key, val, err.Error())
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
    if res!= true {
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

func (r *PrivateComputeRuntime) SaveCert(context protocol.TxSimContext, params map[string]string) ([]byte, error) {
    teeId := params["enclave_id"]
    if utils.IsAnyBlank(teeId) {
        err := fmt.Errorf("%s,param[enclave_id] of save cert  not found", ErrParams.Error())
        r.log.Errorf(err.Error())
        return nil, err
    }

    teeCert := params["enclave_cert"]
    if utils.IsAnyBlank(teeCert) {
        err := fmt.Errorf("%s,param[enclave_cert] of save cert  not found", ErrParams.Error())
        r.log.Errorf(err.Error())
        return nil, err
    }

    if err := context.Put(commonPb.ContractName_SYSTEM_CONTRACT_PRIVATE_COMPUTE.String(), []byte(teeId), []byte(teeCert)); err != nil {
        r.log.Errorf("Put enclave:%s cert into chain DB failed, err: %s", teeId, err.Error())
    }

    return nil, nil
}

func (r *PrivateComputeRuntime) GetCert(context protocol.TxSimContext, params map[string]string) ([]byte, error) {
    return r.getValue(context, params["enclave_id"])
}