/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"crypto/sha256"
	"fmt"

	"chainmaker.org/chainmaker/common/v2/crypto"
	"chainmaker.org/chainmaker/common/v2/crypto/asym"
)

var templateStr = "KeyType:%s , test pass.\n"

func testASym() error {
	fmt.Println("===== start to test ASym =====")
	defer fmt.Println()
	if err := testSignAndVerify(crypto.ECC_NISTP256); err != nil {
		return err
	}
	fmt.Printf("KeyType:%s , test pass.\n", crypto.KeyType2NameMap[crypto.ECC_NISTP256])
	if err := testSignAndVerify(crypto.ECC_NISTP384); err != nil {
		return err
	}
	fmt.Printf(templateStr, crypto.KeyType2NameMap[crypto.ECC_NISTP384])
	if err := testSignAndVerify(crypto.ECC_NISTP521); err != nil {
		return err
	}
	fmt.Printf(templateStr, crypto.KeyType2NameMap[crypto.ECC_NISTP521])
	if err := testSignAndVerify(crypto.SM2); err != nil {
		return err
	}
	fmt.Printf(templateStr, crypto.KeyType2NameMap[crypto.SM2])
	if err := testSignAndVerify(crypto.RSA2048); err != nil {
		return err
	}
	fmt.Printf(templateStr, crypto.KeyType2NameMap[crypto.RSA2048])
	if err := testSignAndVerify(crypto.RSA1024); err != nil {
		return err
	}
	fmt.Printf(templateStr, crypto.KeyType2NameMap[crypto.RSA1024])
	if err := testSignAndVerify(crypto.RSA512); err != nil {
		return err
	}
	fmt.Printf(templateStr, crypto.KeyType2NameMap[crypto.RSA512])
	if err := testSignAndVerify(crypto.ECC_Secp256k1); err != nil {
		return err
	}
	fmt.Printf(templateStr, crypto.KeyType2NameMap[crypto.ECC_Secp256k1])
	return nil
}

var failedReasonStr = "verify failed, reason: %s"

func testSignAndVerify(opt crypto.KeyType) error {
	digest := sha256.Sum256([]byte("js"))

	// 方式1：
	sk, pk, err := asym.GenerateKeyPairPEM(opt)
	if err != nil {
		return err
	}
	sign, err := asym.Sign(sk, digest[:])
	if err != nil {
		return err
	}
	ok, err := asym.Verify(pk, digest[:], sign)
	if err != nil || !ok {
		return fmt.Errorf(failedReasonStr, err)
	}

	// 方式2：
	sk2, pk2, err := asym.GenerateKeyPairBytes(opt)
	if err != nil {
		return err
	}
	sign2, err := asym.Sign(sk2, digest[:])
	if err != nil {
		return err
	}
	ok, err = asym.Verify(pk2, digest[:], sign2)
	if err != nil || !ok {
		return fmt.Errorf(failedReasonStr, err)
	}

	// 方式3：
	sk3, err := asym.GenerateKeyPair(opt)
	if err != nil {
		return err
	}
	sign3, err := sk3.Sign(digest[:])
	if err != nil {
		return err
	}
	ok, err = sk3.PublicKey().Verify(digest[:], sign3)
	if err != nil || !ok {
		return fmt.Errorf(failedReasonStr, err)
	}

	// 方式4:
	sk4, err := asym.PrivateKeyFromPEM([]byte(sk), nil)
	if err != nil {
		return err
	}
	pk4, err := asym.PublicKeyFromPEM([]byte(pk))
	if err != nil {
		return err
	}

	sig4, err := sk4.Sign(digest[:])
	if err != nil {
		return err
	}
	ok, err = pk4.Verify(digest[:], sig4)
	if err != nil || !ok {
		return fmt.Errorf(failedReasonStr, err)
	}

	// Cross check:
	ok, err = pk4.Verify(digest[:], sign)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("verify failed")
	}

	ok, err = asym.Verify(pk, digest[:], sig4)
	if err != nil || !ok {
		return fmt.Errorf(failedReasonStr, err)
	}

	return testSignAndVerifyWithOpts(opt)
}

func testSignAndVerifyWithOpts(opt crypto.KeyType) error {
	digest := sha256.Sum256([]byte("js"))

	optSHA256 := &crypto.SignOpts{
		Hash: crypto.HASH_TYPE_SHA256,
		UID:  "",
	}
	optSM3 := &crypto.SignOpts{
		Hash: crypto.HASH_TYPE_SM3,
		UID:  crypto.CRYPTO_DEFAULT_UID,
	}

	// 方式1：
	skPEM, pkPEM, err := asym.GenerateKeyPairPEM(opt)
	if err != nil {
		return err
	}
	sk1, err := asym.PrivateKeyFromPEM([]byte(skPEM), nil)
	if err != nil {
		return err
	}
	pk1, err := asym.PublicKeyFromPEM([]byte(pkPEM))
	if err != nil {
		return err
	}

	sig1, err := sk1.SignWithOpts(digest[:], optSHA256)
	if err != nil {
		return err
	}
	ok, err := pk1.VerifyWithOpts(digest[:], sig1, optSHA256)
	if err != nil || !ok {
		return fmt.Errorf("verify with opts failed, reason0: %s", err)
	}

	sig2, err := sk1.SignWithOpts(digest[:], optSM3)
	if err != nil {
		return err
	}
	ok, err = pk1.VerifyWithOpts(digest[:], sig2, optSM3)
	if err != nil || !ok {
		return fmt.Errorf("verify with opts failed, reason1: %s", err)
	}

	// 方式2：
	sk3, err := asym.GenerateKeyPair(opt)
	if err != nil {
		return err
	}
	sign3, err := sk3.SignWithOpts(digest[:], optSHA256)
	if err != nil {
		return err
	}
	ok, err = sk3.PublicKey().VerifyWithOpts(digest[:], sign3, optSHA256)
	if err != nil || !ok {
		return fmt.Errorf("verify with opts failed, reason2: %s", err)
	}
	sign4, err := sk3.SignWithOpts(digest[:], optSM3)
	if err != nil {
		return err
	}
	ok, err = sk3.PublicKey().VerifyWithOpts(digest[:], sign4, optSM3)
	if err != nil || !ok {
		return fmt.Errorf("verify with opts failed, reason3: %s", err)
	}
	return nil
}
