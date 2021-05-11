/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package native

import (
    "chainmaker.org/chainmaker-go/logger"
    commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
    "chainmaker.org/chainmaker-go/protocol"
    "chainmaker.org/chainmaker-go/utils"
    "crypto/sha256"
    "fmt"
)

const(
    COMPUTE_RESULT = "private_compute_result"
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

    return queryMethodMap
}

type PrivateComputeRuntime struct {
    log *logger.CMLogger
}

func (r *PrivateComputeRuntime) SaveContract(context protocol.TxSimContext, params map[string]string) ([]byte, error) {
    name := params["contract_name"]
    if utils.IsAnyBlank(name) {
        err := fmt.Errorf("%s, param[contract_name] of save contract  not found", ErrParams.Error())
        r.log.Errorf(err.Error())
        return nil, err
    }

    code := params["contract_code"]
    if utils.IsAnyBlank(code) {
        err := fmt.Errorf("%s, param[contract_code] of save contract  not found", ErrParams.Error())
        r.log.Errorf(err.Error())
        return nil, err
    }
    //TODO: contract code check

    hash := params["code_hash"]
    if utils.IsAnyBlank(hash) {
        err := fmt.Errorf("%s, param[code_hash] of save contract  not found", ErrParams.Error())
        r.log.Errorf(err.Error())
        return nil, err
    }

    calHash := sha256.Sum256([]byte(code))
    if string(calHash[:]) != hash {
        err := fmt.Errorf("%s, param[code_hash] != hash of param[contract_code] in save contract interface", ErrParams.Error())
        r.log.Errorf(err.Error())
        return nil, err
    }

    combinationName := commonPb.ContractName_SYSTEM_CONTRACT_PRIVATE_COMPUTE.String() + name
    version, ok := params["version"]
    if ok != true {
        v, err := context.Get(combinationName, []byte(protocol.ContractVersion))
        if err == nil && len(v) > 0 {
            if string(v) == version {
                err := fmt.Errorf("%s, param[code_hash] != hash of param[contract_code] in save contract interface", ErrParams.Error())
                r.log.Errorf(err.Error())
                return nil, err
            }
        } else {
            version = "1"
        }
    }

    // save versioned byteCode
    if err := context.Put(combinationName, []byte(protocol.ContractVersion), []byte(version)); err != nil {
        r.log.Errorf("Put contract version into DB failed while save contract, err: %s", err.Error())
        return nil, err
    }

    key := append([]byte(protocol.ContractByteCode), version...)
    if err := context.Put(combinationName, key, []byte(code)); err != nil {
        r.log.Errorf("Put private dir failed, err: %s", err.Error())
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

    if len(contractCode) == 0 {
        r.log.Errorf("Contract[%s] byte code is empty.", name)
        return nil, err
    }

    result.ContractCode = contractCode
    result. GasLimit = protocol.GasLimit

    calHash := sha256.Sum256(result.ContractCode)
    if string(calHash[:]) != hash {
        err := fmt.Errorf("%s, param[code_hash] != hash of contract code in get contract interface", ErrParams.Error())
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

func (r *PrivateComputeRuntime) SaveData(context protocol.TxSimContext, params map[string]string) ([]byte, error) {
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

    combinationName := commonPb.ContractName_SYSTEM_CONTRACT_PRIVATE_COMPUTE.String() + params["contract_name"]
    if err := context.Put(combinationName, []byte(COMPUTE_RESULT), cRes); err != nil {
        r.log.Errorf("Write compute result:%s failed, err: %s", cRes, err.Error())
        return nil, err
    }

    //reportSign := []byte(params["report_sign"])
    //TODO:check report sign

    if result.Code != commonPb.ContractResultCode_OK {
        r.log.Infof("Compute result code != ok, return")
        return nil, nil
    }

    rwSetStr := params["rw_set"]
    if utils.IsAnyBlank(rwSetStr) {
        err := fmt.Errorf("%s, param[rw_set] of save data not found", ErrParams.Error())
        r.log.Errorf(err.Error())
        return nil, err
    }

    var rwSet commonPb.TxRWSet
    if err := rwSet.Unmarshal([]byte(rwSetStr)); err != nil{
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
