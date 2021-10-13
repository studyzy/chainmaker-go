package main

import (
	"chainmaker.org/chainmaker-go/txpool"
	"chainmaker.org/chainmaker-go/vm"
	"chainmaker.org/chainmaker/protocol/v2"
	batch "chainmaker.org/chainmaker/txpool-batch/v2"
	single "chainmaker.org/chainmaker/txpool-single/v2"
	evm "chainmaker.org/chainmaker/vm-evm"
	gasm "chainmaker.org/chainmaker/vm-gasm"
	wasmer "chainmaker.org/chainmaker/vm-wasmer"
	wxvm "chainmaker.org/chainmaker/vm-wxvm"
)

func init() {
	// txPool
	txpool.RegisterTxPoolProvider(single.TxPoolType, single.NewTxPoolImpl)
	txpool.RegisterTxPoolProvider(batch.TxPoolType, batch.NewBatchTxPool)

	// vm
	vm.RegisterVmProvider("GASM", func() (protocol.VmInstancesManager, error) { return &gasm.InstancesManager{}, nil })
	vm.RegisterVmProvider("WASMER", func() (protocol.VmInstancesManager, error) { return &wasmer.InstancesManager{}, nil })
	vm.RegisterVmProvider("WXVM", func() (protocol.VmInstancesManager, error) { return &wxvm.InstancesManager{}, nil })
	vm.RegisterVmProvider("EVM", func() (protocol.VmInstancesManager, error) { return &evm.InstancesManager{}, nil })
}
