/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package utils

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"chainmaker.org/chainmaker/common/v2/crypto/hash"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"github.com/gogo/protobuf/proto"
)

// CalcRWSetRoot calculate txs' read-write set root hash, following the tx order in txs
func CalcRWSetRoot(hashType string, txs []*commonPb.Transaction) ([]byte, error) {
	// calculate read-write set hash following the order in txs
	// if txId does not exist in txRWSetMap, fill in a default one
	if len(txs) == 0 {
		return nil, nil
	}
	rwSetHashes := make([][]byte, len(txs))
	for i, tx := range txs {
		rwSetHashes[i] = tx.Result.RwSetHash
	}

	// calculate the merkle root
	root, err := hash.GetMerkleRoot(hashType, rwSetHashes)
	return root, err
}

// CalcRWSetHash calculate read-write set hash
// return (nil, nil) if read-write set is nil
func CalcRWSetHash(hashType string, set *commonPb.TxRWSet) ([]byte, error) {
	if set == nil {
		return nil, fmt.Errorf("calc rwset hash set == nil")
	}

	setBytes, err := proto.Marshal(set)
	if err != nil {
		return nil, err
	}

	hashByte, err := hash.GetByStrType(hashType, setBytes)
	return hashByte, err
}

// FormatRWSet format rwset
func FormatRWSet(set *commonPb.TxRWSet) string {
	serializedRWSet := bytes.Buffer{}
	serializedRWSet.WriteString(set.TxId)
	serializedRWSet.WriteString(" - Reads{")
	for _, txRead := range set.TxReads {
		serializedRWSet.WriteString(fmt.Sprintf("[%s, %s] ", string(txRead.Key), hex.EncodeToString(txRead.Value)))
	}
	serializedRWSet.WriteString("}Write:{")
	for _, txWrite := range set.TxWrites {
		serializedRWSet.WriteString(fmt.Sprintf("[%s, %s]", string(txWrite.Key), hex.EncodeToString(txWrite.Value)))
	}
	serializedRWSet.WriteString("}")
	return serializedRWSet.String()
}
