/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package native

import (
	"chainmaker.org/chainmaker-go/logger"
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"fmt"

	"chainmaker.org/chainmaker-go/protocol"
)

const (
	GovernmentContractName = "government_contract"
)

type GovernmentContract struct {
	methods map[string]ContractFunc
	log     *logger.CMLogger
}

func newGovernmentContract(log *logger.CMLogger) *GovernmentContract {
	return &GovernmentContract{
		log:     log,
		methods: registerGovernmentContractMethods(log),
	}
}

func (c *GovernmentContract) getMethod(methodName string) ContractFunc {
	return c.methods[methodName]
}

func registerGovernmentContractMethods(log *logger.CMLogger) map[string]ContractFunc {
	methodMap := make(map[string]ContractFunc, 64)
	// cert manager
	governmentRuntime := &GovernmentRuntime{log: log}
	methodMap[commonPb.QueryFunction_GET_GOVERNANCE_CONTRACT.String()] = governmentRuntime.GetGovernmentContract
	return methodMap
}

type GovernmentRuntime struct {
	log *logger.CMLogger
}

func (r *GovernmentRuntime) GetGovernmentContract(txSimContext protocol.TxSimContext, parameters map[string]string) ([]byte, error) {
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
