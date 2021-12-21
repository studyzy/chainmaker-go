/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"chainmaker.org/chainmaker-go/txpool"
	"chainmaker.org/chainmaker-go/vm"
	"chainmaker.org/chainmaker/localconf/v2"
	"chainmaker.org/chainmaker/protocol/v2"
	batch "chainmaker.org/chainmaker/txpool-batch/v2"
	single "chainmaker.org/chainmaker/txpool-single/v2"
	dockergo "chainmaker.org/chainmaker/vm-docker-go/v2"
	evm "chainmaker.org/chainmaker/vm-evm/v2"
	gasm "chainmaker.org/chainmaker/vm-gasm/v2"
	wasmer "chainmaker.org/chainmaker/vm-wasmer/v2"
	wxvm "chainmaker.org/chainmaker/vm-wxvm/v2"
)

func init() {
	// txPool
	txpool.RegisterTxPoolProvider(single.TxPoolType, single.NewTxPoolImpl)
	txpool.RegisterTxPoolProvider(batch.TxPoolType, batch.NewBatchTxPool)

	// vm
	vm.RegisterVmProvider(
		"GASM",
		func(chainId string, configs map[string]interface{}) (protocol.VmInstancesManager, error) {
			return &gasm.InstancesManager{}, nil
		})
	vm.RegisterVmProvider(
		"WASMER",
		func(chainId string, configs map[string]interface{}) (protocol.VmInstancesManager, error) {
			return wasmer.NewInstancesManager(chainId), nil
		})
	vm.RegisterVmProvider(
		"WXVM",
		func(chainId string, configs map[string]interface{}) (protocol.VmInstancesManager, error) {
			return &wxvm.InstancesManager{}, nil
		})
	vm.RegisterVmProvider(
		"EVM",
		func(chainId string, configs map[string]interface{}) (protocol.VmInstancesManager, error) {
			return &evm.InstancesManager{}, nil
		})

	vm.RegisterVmProvider(
		"DOCKERGO",
		func(chainId string, configs map[string]interface{}) (protocol.VmInstancesManager, error) {
			return dockergo.NewDockerManager(chainId, localconf.ChainMakerConfig.VMConfig), nil
		})
}
