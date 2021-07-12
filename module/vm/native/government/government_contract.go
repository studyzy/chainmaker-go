/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package government

import (
	"chainmaker.org/chainmaker-go/logger"
	"chainmaker.org/chainmaker-go/vm/native/common"
	commonPb "chainmaker.org/chainmaker/pb-go/common"
	"fmt"

	"chainmaker.org/chainmaker/protocol"
)

const (
	GovernmentContractName = "government_contract"
)

type GovernmentContract struct {
	methods map[string]common.ContractFunc
	log     *logger.CMLogger
}

func NewGovernmentContract(log *logger.CMLogger) *GovernmentContract {
	return &GovernmentContract{
		log:     log,
		methods: registerGovernmentContractMethods(log),
	}
}

func (c *GovernmentContract) GetMethod(methodName string) common.ContractFunc {
	return c.methods[methodName]
}

func registerGovernmentContractMethods(log *logger.CMLogger) map[string]common.ContractFunc {
	methodMap := make(map[string]common.ContractFunc, 64)
	// cert manager
	governmentRuntime := &GovernmentRuntime{log: log}
	methodMap[commonPb.QueryFunction_GET_GOVERNANCE_CONTRACT.String()] = governmentRuntime.GetGovernmentContract
	return methodMap
}

type GovernmentRuntime struct {
	log *logger.CMLogger
}

func (r *GovernmentRuntime) GetGovernmentContract(txSimContext protocol.TxSimContext, parameters map[string][]byte) ([]byte, error) {
	store := txSimContext.GetBlockchainStore()
	governmentContractName := GovernmentContractName
	bytes, err := store.ReadObject(governmentContractName, []byte(governmentContractName))
	if err != nil {
		r.log.Errorw("ReadObject.Get err", "governmentContractName", governmentContractName, "err", err)
		return nil, err
	}

	if len(bytes) == 0 {
		r.log.Errorw("ReadObject.Get empty", "governmentContractName", governmentContractName)
		return nil, fmt.Errorf("bytes is empty")
	}

	return bytes, nil
}
