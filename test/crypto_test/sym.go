/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"crypto/rand"
	"fmt"

	"chainmaker.org/chainmaker/common/v2/crypto"
	"chainmaker.org/chainmaker/common/v2/crypto/sym"
)

func testSym() error {
	fmt.Println("===== start to test Sym =====")
	defer fmt.Println()
	if err := testSymAES(); err != nil {
		return err
	}
	if err := testSymAESStr(); err != nil {
		return err
	}
	fmt.Printf("KeyType:%s , test pass.\n", crypto.KeyType2NameMap[crypto.AES])

	if err := testSymSM4(); err != nil {
		return err
	}
	if err := testSymSM4Str(); err != nil {
		return err
	}
	fmt.Printf("KeyType:%s , test pass.\n", crypto.KeyType2NameMap[crypto.SM4])
	return nil
}

func testSymAES() error {
	msg := "js"
	key := make([]byte, 16)
	_, err := rand.Read(key)
	if err != nil {
		return err
	}

	aes, err := sym.GenerateSymKey(crypto.AES, key)
	if err != nil {
		return err
	}

	crypt, err := aes.Encrypt([]byte(msg))
	if err != nil {
		return err
	}
	decrypt, err := aes.Decrypt(crypt)
	if err != nil {
		return err
	}
	if string(decrypt) != msg {
		return fmt.Errorf("decrypt mismatching")
	}
	return nil
}

func testSymSM4() error {
	msg := "js"
	key := make([]byte, 16)
	_, err := rand.Read(key)
	if err != nil {
		return err
	}

	sm4, err := sym.GenerateSymKey(crypto.SM4, key)
	if err != nil {
		return err
	}

	crypt, err := sm4.Encrypt([]byte(msg))
	if err != nil {
		return err
	}

	decrypt, err := sm4.Decrypt(crypt)
	if err != nil {
		return err
	}

	if string(decrypt) != msg {
		return fmt.Errorf("decrypt mismatching0")
	}
	return nil
}

func testSymAESStr() error {
	msg := "js"
	keyHex := "43494f2804a3cf33e96077637e45d211"

	aes, err := sym.GenerateSymKeyStr(crypto.AES, keyHex)
	if err != nil {
		return err
	}

	crypt, err := aes.Encrypt([]byte(msg))
	if err != nil {
		return err
	}

	decrypt, err := aes.Decrypt(crypt)
	if err != nil {
		return err
	}

	if string(decrypt) != msg {
		return fmt.Errorf("decrypt mismatching1")
	}
	return nil
}

func testSymSM4Str() error {
	msg := "js"
	keyHex := "43494f2804a3cf33e96077637e45d211"

	sm4, err := sym.GenerateSymKeyStr(crypto.SM4, keyHex)
	if err != nil {
		return err
	}

	crypt, err := sm4.Encrypt([]byte(msg))
	if err != nil {
		return err
	}

	decrypt, err := sm4.Decrypt(crypt)
	if err != nil {
		return err
	}
	if string(decrypt) != msg {
		return fmt.Errorf("decrypt mismatching2")
	}
	return nil
}
