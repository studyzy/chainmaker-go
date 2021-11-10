/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package vm

import (
	"strings"

	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/protocol/v2"
)

type Provider func(chainId string, config map[string]interface{}) (protocol.VmInstancesManager, error)

var vmProviders = make(map[string]Provider)

func RegisterVmProvider(t string, f Provider) {
	vmProviders[strings.ToUpper(t)] = f
}

func GetVmProvider(t string) Provider {
	provider, ok := vmProviders[strings.ToUpper(t)]
	if !ok {
		return nil
	}
	return provider
}

const (
	VmTypeGasm   = "GASM"
	VmTypeWasmer = "WASMER"
	VmTypeEvm    = "EVM"
	VmTypeWxvm   = "WXVM"
)

var VmTypeToRunTimeType = map[string]common.RuntimeType{
	"GASM":       common.RuntimeType_GASM,
	"WASMER":     common.RuntimeType_WASMER,
	"WXVM":       common.RuntimeType_WXVM,
	"EVM":        common.RuntimeType_EVM,
	"DOCKERGO":   common.RuntimeType_DOCKER_GO,
	"DOCKERJAVA": common.RuntimeType_DOCKER_JAVA,
}

var RunTimeTypeToVmType = map[common.RuntimeType]string{
	common.RuntimeType_GASM:        "GASM",
	common.RuntimeType_WASMER:      "WASMER",
	common.RuntimeType_WXVM:        "WXVM",
	common.RuntimeType_EVM:         "EVM",
	common.RuntimeType_DOCKER_GO:   "DOCKERGO",
	common.RuntimeType_DOCKER_JAVA: "DOCKERJAVA",
}
