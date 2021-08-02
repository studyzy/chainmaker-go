package util

import "chainmaker.org/chainmaker/pb-go/common"

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
