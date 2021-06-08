package dpos

import (
	"bytes"

	"chainmaker.org/chainmaker-go/pb/protogo/common"
	commonpb "chainmaker.org/chainmaker-go/pb/protogo/common"
)

func (impl *DPoSImpl) getState(key []byte, block *common.Block, blockTxRwSet map[string]*common.TxRWSet) ([]byte, error) {
	for i := len(block.Txs) - 1; i >= 0; i-- {
		rwSets := blockTxRwSet[block.Txs[i].Header.TxId]
		for _, txWrite := range rwSets.TxWrites {
			if bytes.Equal(txWrite.Key, key) {
				return txWrite.Value, nil
			}
		}
	}

	val, err := impl.stateDB.ReadObject(commonpb.ContractName_SYSTEM_CONTRACT_DPOS_ERC20.String(), key)
	if err != nil {
		impl.log.Errorf("query user balance failed, reason: %s", err)
		return nil, err
	}
	return val, nil
}
