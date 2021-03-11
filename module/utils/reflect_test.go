/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package utils

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
)

// 区块配置
type BlockConfig struct {
	TxTimestampVerify bool   `protobuf:"varint,1,opt,name=tx_timestamp_verify,json=txTimestampVerify,proto3" json:"tx_timestamp_verify,omitempty"`
	TxTimeout         uint32 `protobuf:"varint,2,opt,name=tx_timeout,json=txTimeout,proto3" json:"tx_timeout,omitempty"`
	BlockTxCapacity   uint32 `protobuf:"varint,3,opt,name=block_tx_capacity,json=blockTxCapacity,proto3" json:"block_tx_capacity,omitempty"`
	BlockSize         uint32 `protobuf:"varint,4,opt,name=block_size,json=blockSize,proto3" json:"block_size,omitempty"`
	BlockInterval     uint32 `protobuf:"varint,5,opt,name=block_interval,json=blockInterval,proto3" json:"block_interval,omitempty"`
}

func TestUpdateChainConfigReflect2(t *testing.T) {
	params := make(map[string]string, 0)
	params["block_interval"] = "2"
	params["block_size"] = "3"
	params["block_tx_capacity"] = "4"
	params["tx_timestamp_verify"] = "trues"
	params["tx_timestamp_verify2"] = "trues"

	config := &BlockConfig{}
	fmt.Println("config1", config)
	changed, err := UpdateField(params, "block_interval", config)
	require.Nil(t, err, err)
	changed, err = UpdateField(params, "tx_timestamp_verify2", config)
	require.NotNil(t, err, err)
	fmt.Println("config1", config, changed)
}
