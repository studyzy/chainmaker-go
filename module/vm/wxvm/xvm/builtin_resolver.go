/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package xvm

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"

	"chainmaker.org/chainmaker-go/wxvm/xvm/exec"
	"chainmaker.org/chainmaker-go/wxvm/xvm/runtime/emscripten"
)

//func touint32(n int32) uint32 {
//	return *(*uint32)(unsafe.Pointer(&n))
//}//unused function(deadcode)

func hashFunc(name string) hash.Hash {
	switch name {
	case "sha256":
		return sha256.New()
	default:
		return nil
	}
}

func wxvmHash(ctx exec.Context,
	nameptr uint32,
	inputptr uint32, inputlen uint32,
	outputptr uint32, outputlen uint32) uint32 {

	codec := exec.NewCodec(ctx)
	name := codec.CString(nameptr)
	input := codec.Bytes(inputptr, inputlen)
	output := codec.Bytes(outputptr, outputlen)

	hasher := hashFunc(name)
	if hasher == nil {
		exec.ThrowMessage(fmt.Sprintf("hash %s not found", name))
	}
	hasher.Write(input)
	out := hasher.Sum(nil)
	copy(output, out[:])
	return 0
}

type codec interface {
	Encode(in []byte) []byte
	Decode(in []byte) ([]byte, error)
}

func getCodec(name string) codec {
	switch name {
	case "hex":
		return hexCodec{}
	default:
		return nil
	}
}

type hexCodec struct{}

func (h hexCodec) Encode(in []byte) []byte {
	out := make([]byte, hex.EncodedLen(len(in)))
	hex.Encode(out, in)
	return out
}
func (h hexCodec) Decode(in []byte) ([]byte, error) {
	out := make([]byte, hex.DecodedLen(len(in)))
	_, err := hex.Decode(out, in)
	return out, err
}

func wxvmEncode(ctx exec.Context,
	nameptr uint32,
	inputptr uint32, inputlen uint32,
	outputpptr uint32, outputLenPtr uint32) uint32 {

	codec := exec.NewCodec(ctx)
	name := codec.CString(nameptr)
	input := codec.Bytes(inputptr, inputlen)

	c := getCodec(name)
	if c == nil {
		exec.ThrowMessage(fmt.Sprintf("codec %s not found", name))
	}
	out := c.Encode(input)

	codec.SetUint32(outputpptr, bytesdup(ctx, out))
	codec.SetUint32(outputLenPtr, uint32(len(out)))
	return 0
}

func wxvmDecode(ctx exec.Context,
	nameptr uint32,
	inputptr uint32, inputlen uint32,
	outputpptr uint32, outputLenPtr uint32) uint32 {

	codec := exec.NewCodec(ctx)
	name := codec.CString(nameptr)
	input := codec.Bytes(inputptr, inputlen)

	c := getCodec(name)
	if c == nil {
		exec.ThrowMessage(fmt.Sprintf("codec %s not found", name))
	}
	out, err := c.Decode(input)
	if err != nil {
		return 1
	}

	codec.SetUint32(outputpptr, bytesdup(ctx, out))
	codec.SetUint32(outputLenPtr, uint32(len(out)))
	return 0
}

// Returns a pointer to a bytes, which is a duplicate of b.
// The returned pointer must be passed to free to avoid a memory leak
func bytesdup(ctx exec.Context, b []byte) uint32 {
	codec := exec.NewCodec(ctx)
	memptr, err := emscripten.Malloc(ctx, len(b))
	if err != nil {
		exec.ThrowError(err)
	}
	mem := codec.Bytes(memptr, uint32(len(b)))
	copy(mem, b)
	return memptr
}

// Returns a pointer to a null-terminated string, which is a duplicate of the string s.
// The returned pointer must be passed to free to avoid a memory leak
//func strdup(ctx exec.Context, s string) uint32 {
//	codec := exec.NewCodec(ctx)
//	memptr, err := emscripten.Malloc(ctx, len(s)+1)
//	if err != nil {
//		exec.ThrowError(err)
//	}
//	mem := codec.Bytes(memptr, uint32(len(s)+1))
//	copy(mem, s)
//	mem[len(s)] = 0
//	return memptr
//}//unused code(deadcode)

//BuiltinResolver export
var BuiltinResolver = exec.MapResolver(map[string]interface{}{
	"env._wxvm_hash":   wxvmHash,
	"env._wxvm_encode": wxvmEncode,
	"env._wxvm_decode": wxvmDecode,
	//"env._wxvm_ecverify":         xvmECVerify,
	//"env._wxvm_make_tx": xvmMakeTx,
	//"env._wxvm_addr_from_pubkey": xvmAddressFromPubkey,
})
