/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package privatecompute

import (
	"encoding/pem"
	"fmt"

	bccrypto "chainmaker.org/chainmaker/common/v2/crypto"
	"chainmaker.org/chainmaker/common/v2/crypto/asym"
	"chainmaker.org/chainmaker/common/v2/crypto/tee"
	bcx509 "chainmaker.org/chainmaker/common/v2/crypto/x509"
)

func getPubkeyPairFromCert(pemData []byte) (verificationPubKey bccrypto.PublicKey,
	encryptPubKey bccrypto.PublicKey, retErr error) {

	// pem => der
	certBlock, _ := pem.Decode(pemData)
	if certBlock == nil {
		retErr = fmt.Errorf("decode pem failed, invalid certificate")
		return
	}

	// der => cert
	cert, err := bcx509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		retErr = fmt.Errorf("x509 parse cert failed, %s", err)
		return
	}

	// get pem data of crypto public key from cert
	encryptPubkeyPemData, err := bcx509.GetExtByOid(tee.OidKeyBag, cert.Extensions)
	if err != nil {
		retErr = fmt.Errorf("get crypto pubkey by oid error: %v", err)
		return
	}

	// pem => der
	encryptPubkeyBlock, _ := pem.Decode(encryptPubkeyPemData)
	if encryptPubkeyBlock == nil {
		retErr = fmt.Errorf("get crypto pub key block error")
		return
	}

	// der => encrypt public key
	encryptPubKey, err = asym.PublicKeyFromDER(encryptPubkeyBlock.Bytes)
	if err != nil {
		retErr = fmt.Errorf("get crypto pub key error: %v", err)
		return
	}

	// cert => signing verification public key
	verificationPubKey = cert.PublicKey
	return
}
