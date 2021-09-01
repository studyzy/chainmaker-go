/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package utils

import (
	"encoding/json"

	commonpb "chainmaker.org/chainmaker/pb-go/v2/common"
)

//UnmarshalJsonStrKV2KVPairs  传入一个json字符串，该字符串key value都是string，转换为系统需要的KeyValuePair列表
func UnmarshalJsonStrKV2KVPairs(jsonStr string) ([]*commonpb.KeyValuePair, error) {
	var stringKVs []*stringKV
	err := json.Unmarshal([]byte(jsonStr), &stringKVs)
	if err != nil {
		return nil, err
	}
	result := make([]*commonpb.KeyValuePair, len(stringKVs))
	for i, kv := range stringKVs {
		result[i] = &commonpb.KeyValuePair{Key: kv.Key, Value: []byte(kv.Value)}
	}
	return result, nil
}

type stringKV struct {
	Key, Value string
}
