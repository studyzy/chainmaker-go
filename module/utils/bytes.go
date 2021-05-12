/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package utils

import (
	"bytes"
	"encoding/binary"
)

// BytesToInt le bytes to int32, little endian
func BytesToInt(b []byte) int32 {
	bytesBuffer := bytes.NewBuffer(b)

	var x int32
	binary.Read(bytesBuffer, binary.LittleEndian, &x)

	return int32(x)
}

// BytesToInt le bytes to int64, little endian
func BytesToInt64(b []byte) int64 {
	bytesBuffer := bytes.NewBuffer(b)
	var x int64
	binary.Read(bytesBuffer, binary.LittleEndian, &x)
	return int64(x)
}

// IntToBytes int32 to le bytes, little endian
func IntToBytes(x int32) []byte {
	bytesBuffer := bytes.NewBuffer([]byte{})
	binary.Write(bytesBuffer, binary.LittleEndian, x)
	return bytesBuffer.Bytes()
}

// Int64ToBytes int64 to le bytes, little endian
func Int64ToBytes(x int64) []byte {
	bytesBuffer := bytes.NewBuffer([]byte{})
	binary.Write(bytesBuffer, binary.LittleEndian, x)
	return bytesBuffer.Bytes()
}
