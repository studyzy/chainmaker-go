package main

import (
	"chainmaker.org/chainmaker-go/txpool"
	batch "chainmaker.org/chainmaker/txpool-batch/v2"
	single "chainmaker.org/chainmaker/txpool-single/v2"
)

func init() {
	// txPool
	txpool.RegisterTxPoolProvider(single.TxPoolType, single.NewTxPoolImpl)
	txpool.RegisterTxPoolProvider(batch.TxPoolType, batch.NewBatchTxPool)
}
