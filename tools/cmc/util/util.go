package util

import (
	"encoding/hex"

	"chainmaker.org/chainmaker/common/evmutils"
	"chainmaker.org/chainmaker/pb-go/common"
)

func MaxInt(i, j int) int {
	if j > i {
		return j
	}
	return i
}

func ConvertParameters(pars map[string]string) []*common.KeyValuePair {
	var kvp []*common.KeyValuePair
	for k, v := range pars {
		kvp = append(kvp, &common.KeyValuePair{
			Key:   k,
			Value: []byte(v),
		})
	}
	return kvp
}

func CalcEvmContractName(contractName string) string {
	return hex.EncodeToString(evmutils.Keccak256([]byte(contractName)))[24:]
}
