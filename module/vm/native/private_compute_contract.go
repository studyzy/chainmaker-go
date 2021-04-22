/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package native

import (
    "chainmaker.org/chainmaker-go/logger"
    commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
    "chainmaker.org/chainmaker-go/protocol"
    "io/ioutil"
    "os"
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
    queryMethodMap[commonPb.PrivateComputeContractFunction_SAVA_DATA.String()] = privateComputeRuntime.SaveData

    return queryMethodMap
}

type PrivateComputeRuntime struct {
    log *logger.CMLogger
}

func (r *PrivateComputeRuntime) GetContract(context protocol.TxSimContext, params map[string]string) ([]byte, error) {
    //TODO:check user permission
    //byteCert := []byte(params["user_cert"])

    //TODO: verify hash and sign

    //name := params["contract_name"]
    //version, err := context.Get(name, []byte(protocol.ContractVersion))
    //if  err != nil {
    //    r.log.Errorf("Unable to find latest version for contract[%s], system error:%s.", name, err.Error())
    //    return nil, err
    //} else if len(version) == 0 {
    //    r.log.Errorf("The contract does not exist. contract[%s].", name)
    //    return nil, err
    //}

    var result commonPb.PrivateGetContract
    //codeVersionKey := append([]byte(protocol.ContractByteCode), version...)
    //contractCode, err := context.Get(name, codeVersionKey)
    //if err != nil {
    //    r.log.Errorf("Read contract[%s] failed.", name)
    //    return nil, err
    //} else if len(contractCode) == 0 {
    //    r.log.Errorf("Contract[%s] byte code is empty.", name)
    //    return nil, err
    //} else {
    //    result.ContractCode = contractCode
    //    result. GasLimit = protocol.GasLimit
    //}

    //TEMPORARY CODE JUST FOR TEST
    pathname := "/Users/moses/Workspace/chainmaker-go/test/wasm/go-fact-1.0.0.wasm"
    fh, err:= os.Open(pathname)
    if err != nil {
        r.log.Errorf("Open contract code file(%s) failed.", pathname)
        return nil, err
    }
    defer fh.Close()

    code, err := ioutil.ReadAll(fh)
    if err != nil {
        r.log.Errorf("Read contract code file(%s) failed.", pathname)
        return nil, err
    }

    result.ContractCode = code
    result. GasLimit = protocol.GasLimit
    r.log.Infof("Read contract success，code:%s.", code)

    return result.Marshal()
}

func (r *PrivateComputeRuntime) SaveDir(context protocol.TxSimContext, params map[string]string) ([]byte, error) {
    //TODO:check user permission
    //byteCert := []byte(params["user_cert"])
    key := []byte(params["private_dir"])
    value := []byte(params["order_id"])

    //TODO: verify hash and sign

    if err := context.Put(commonPb.ContractName_SYSTEM_CONTRACT_PRIVATE_COMPUTE.String(), value, key); err != nil {
        r.log.Errorf("Put private dir failed, err: %s", err.Error())
        return nil, err
    }

    return nil, nil
}

func (r *PrivateComputeRuntime) SaveData(context protocol.TxSimContext, params map[string]string) ([]byte, error) {
    //TODO:check user permission
    //byteCert := []byte(params["user_cert"])

    //TODO: verify hash and sign
    if params["compute_result"] != "true" {
    	return nil, nil
    }

    byteRWSet := []byte(params["rw_set"])
    var rwSet commonPb.TxRWSet
    if err := rwSet.Unmarshal(byteRWSet); err != nil{
        r.log.Errorf("Unmarshal RWSet failed, err: %s", err.Error())
        return nil, err
    }

    for i := 0; i < len(rwSet.TxReads); i++ {
        key := rwSet.TxReads[i].Key
        val := rwSet.TxReads[i].Value
        if _, err := context.Get(commonPb.ContractName_SYSTEM_CONTRACT_PRIVATE_COMPUTE.String(), key); err != nil {
            r.log.Errorf("Put key: %s, value:%s into read set failed, err: %s", key, val, err.Error())
		}
    }

    name := params["contract_name"]
    for j :=0; j < len(rwSet.TxWrites); j++ {
        key := rwSet.TxWrites[j].Key
        val := rwSet.TxWrites[j].Value
        extKey := append([]byte(name), key...)
        if err := context.Put(commonPb.ContractName_SYSTEM_CONTRACT_PRIVATE_COMPUTE.String(), extKey, val); err != nil {
            r.log.Errorf("Put key: %s, value:%s into write set failed, err: %s", key, val, err.Error())
        }
    }

    //TODO: put events into DB

    return nil, nil
}

func (r *PrivateComputeRuntime) GetData(context protocol.TxSimContext, params map[string]string) ([]byte, error) {
    //TODO:check user permission
    //byteCert := []byte(params["user_cert"])

    //TODO: verify hash and sign
    name := params["contract_name"]
    key := params["private_key"]
    extKey := append([]byte(name), []byte(key)...)
    if value, err := context.Get(commonPb.ContractName_SYSTEM_CONTRACT_PRIVATE_COMPUTE.String(), extKey); err != nil {
        return value, err
    }

    return nil, nil
}

func (r *PrivateComputeRuntime) SaveCert(context protocol.TxSimContext, params map[string]string) ([]byte, error) {
    //TODO:check user permission
    //byteCert := []byte(params["user_cert"])

    teeId := []byte(params["enclave_id"])
    teeCert := []byte(params["enclave_cert"])
    //TODO: verify tee cert

    if err := context.Put(commonPb.ContractName_SYSTEM_CONTRACT_PRIVATE_COMPUTE.String(), teeId, teeCert); err != nil {
        r.log.Errorf("Put enclave:%s cert into chain DB failed, err: %s", teeId, err.Error())
    }

    return nil, nil
}
