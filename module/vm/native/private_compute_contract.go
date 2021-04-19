/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package native

import (
    "chainmaker.org/chainmaker-go/logger"
    commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
    "chainmaker.org/chainmaker-go/protocol"
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

func registerPrivateComputeContractMethods(log *logger.CMLogger) map[string]ContractFunc {
    queryMethodMap := make(map[string]ContractFunc, 64)
    // cert manager
    privateComputeContract := &PrivateComputeContract{log: log}
    queryMethodMap[commonPb.PrivateComputeContractFunction_GET_CONTRACT.String()] = privateComputeContract.GetContract
    queryMethodMap[commonPb.PrivateComputeContractFunction_GET_DATA.String()] = privateComputeContract.GetData
    queryMethodMap[commonPb.PrivateComputeContractFunction_SAVE_CERT.String()] = privateComputeContract.SaveCert
    queryMethodMap[commonPb.PrivateComputeContractFunction_SAVE_DIR.String()] = privateComputeContract.SaveDir
    queryMethodMap[commonPb.PrivateComputeContractFunction_SAVA_DATA.String()] = privateComputeContract.SaveData

    return queryMethodMap
}

func (r *PrivateComputeContract) GetContract(context protocol.TxSimContext, params map[string]string) ([]byte, error) {
    return nil, nil
}

func (r *PrivateComputeContract) GetData(context protocol.TxSimContext, params map[string]string) ([]byte, error) {
    return nil, nil
}
func (r *PrivateComputeContract) SaveCert(context protocol.TxSimContext, params map[string]string) ([]byte, error) {
    return nil, nil
}

func (r *PrivateComputeContract) SaveDir(context protocol.TxSimContext, params map[string]string) ([]byte, error) {
    return nil, nil
}

func (r *PrivateComputeContract) SaveData(context protocol.TxSimContext, params map[string]string) ([]byte, error) {
    return nil, nil
}