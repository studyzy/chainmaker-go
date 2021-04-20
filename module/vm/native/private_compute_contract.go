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

func newPrivateComputeContact(log *logger.CMLogger) *BlockContact {
    return &BlockContact{
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
    //var resultVersion string
    //versionedByteCodeKey := append([]byte(protocol.ContractByteCode), []byte(resultVersion)...)

    //name := params["ContractName"]
    //code, err := context.Get(name, versionedByteCodeKey)
    //if err != nil {
    //    r.log.Errorf("Read contract(name:%s) failed.", name)
    //    return nil, nil
    //}

    //TEMPORARY CODE JUST FOR TEST
    pathname := "/Users/moses/Workspace/chainmaker-go/test/wasm/go-fact-1.0.0.wasm"
    fh, err:= os.Open(pathname)
    if err != nil {
        r.log.Errorf("Open contract code file(%s) failed.", pathname)
        return nil, nil
    }
    defer fh.Close()

    code, err := ioutil.ReadAll(fh)
    if err != nil {
        r.log.Errorf("Read contract code file(%s) failed.", pathname)
        return nil, nil
    }

    r.log.Infof("Read contract success，code:%s.", code)
    return code, nil
}

func (r *PrivateComputeRuntime) GetData(context protocol.TxSimContext, params map[string]string) ([]byte, error) {
    return nil, nil
}
func (r *PrivateComputeRuntime) SaveCert(context protocol.TxSimContext, params map[string]string) ([]byte, error) {
    return nil, nil
}

func (r *PrivateComputeRuntime) SaveDir(context protocol.TxSimContext, params map[string]string) ([]byte, error) {
    return nil, nil
}

func (r *PrivateComputeRuntime) SaveData(context protocol.TxSimContext, params map[string]string) ([]byte, error) {
    return nil, nil
}