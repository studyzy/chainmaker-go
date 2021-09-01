/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"encoding/hex"
	"fmt"

	"chainmaker.org/chainmaker/common/v2/crypto"
	"chainmaker.org/chainmaker/common/v2/crypto/hash"
)

var passStr = "KeyType:%s , test pass.\n"

func testHash() {
	fmt.Println("===== start to test Hash =====")
	defer fmt.Println()
	data := []byte("js")
	if testGetHash(crypto.HASH_TYPE_SM3, data, "036df5686d99cd847e9a2974d7bcb287fcdc6df004f1735cdf31089c8505b6f5") {
		fmt.Printf(passStr, crypto.CRYPTO_ALGO_SM3)
	}
	if testGetHash(crypto.HASH_TYPE_SHA256, data, "16cedf80ade01c62bdd1ae931d0492330c0b62bf294c08c095ce2fab21a9298d") {
		fmt.Printf(passStr, crypto.CRYPTO_ALGO_SHA256)
	}
	if testGetHash(crypto.HASH_TYPE_SHA3_256, data, "7b942617fa4d27ad9cab6c175035827f53570353586583b648e4fa58b7221126") {
		fmt.Printf(passStr, crypto.CRYPTO_ALGO_SHA3_256)
	}
}

func testGetHash(hashType crypto.HashType, data []byte, expect string) bool {
	bytes, _ := hash.Get(hashType, data)
	return expect == hex.EncodeToString(bytes)
}
