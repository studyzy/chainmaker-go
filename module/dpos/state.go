package dpos

import (
	"bytes"

	"chainmaker.org/chainmaker/pb-go/common"
)

func (impl *DPoSImpl) getState(contractName string, key []byte, block *common.Block, blockTxRwSet map[string]*common.TxRWSet) ([]byte, error) {
	for i := len(block.Txs) - 1; i >= 0; i-- {
		rwSets := blockTxRwSet[block.Txs[i].Header.TxId]
		for _, txWrite := range rwSets.TxWrites {
			if txWrite.ContractName == contractName && bytes.Equal(txWrite.Key, key) {
				return txWrite.Value, nil
			}
		}
	}

	val, err := impl.stateDB.ReadObject(contractName, key)
	if err != nil {
		impl.log.Errorf("query user balance failed, reason: %s", err)
		return nil, err
	}
	return val, nil
}
