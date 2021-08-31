/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package utils

import (
	"encoding/hex"
	"encoding/pem"
	"fmt"

	"chainmaker.org/chainmaker/common/v2/crypto/hash"
	bcx509 "chainmaker.org/chainmaker/common/v2/crypto/x509"
)

// CheckRootCertificate check the root certificate
func CheckRootCertificate(certPEM string) (bool, error) {
	if certPEM == "" {
		return false, fmt.Errorf("check root cert certPEM == nil")
	}
	pemBlock, _ := pem.Decode([]byte(certPEM))
	if pemBlock == nil {
		return false, fmt.Errorf("invalid PEM string for certificate")
	}
	cert, err := bcx509.ParseCertificate(pemBlock.Bytes)
	if err != nil {
		return false, fmt.Errorf("fail to parse certificate: [%v]", err)
	}
	if !cert.IsCA {
		return false, fmt.Errorf("not a root certificate (X509v3 extension, IsCA field)")
	}

	return true, nil
}

// GetCertificateIdHex on input a certificate in PEM format, a hash algorithm(should be the one in chain configuration),
//output the identity of the certificate in the form of a string (under hexadecimal encoding)
func GetCertificateIdHex(certPEM []byte, hashType string) (string, error) {
	id, err := GetCertificateId(certPEM, hashType)
	if err != nil {
		return "", nil
	}
	idHex := hex.EncodeToString(id)
	return idHex, nil
}

// GetCertificateId get certificate id
func GetCertificateId(certPEM []byte, hashType string) ([]byte, error) {
	if certPEM == nil {
		return nil, fmt.Errorf("get cert certPEM == nil")
	}
	certDer, _ := pem.Decode(certPEM)
	if certDer == nil {
		return nil, fmt.Errorf("invalid certificate")
	}
	return GetCertificateIdFromDER(certDer.Bytes, hashType)
}

// GetCertificateIdFromDER get certificate id from DER
func GetCertificateIdFromDER(certDER []byte, hashType string) ([]byte, error) {
	if certDER == nil {
		return nil, fmt.Errorf("get cert from der certDER == nil")
	}
	id, err := hash.GetByStrType(hashType, certDER)
	if err != nil {
		return nil, err
	}
	return id, nil
}
