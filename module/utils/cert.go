/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package utils

import (
	"encoding/pem"
	"fmt"

	"chainmaker.org/chainmaker/common/v2/crypto/hash"
	bcx509 "chainmaker.org/chainmaker/common/v2/crypto/x509"
	acPb "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
)

// GetCertHash get certificate hash
func GetCertHash(orgId string, userCrtPEM []byte, hashType string) ([]byte, error) {
	member := &acPb.Member{
		OrgId:      orgId,
		MemberInfo: userCrtPEM,
		MemberType: acPb.MemberType_CERT,
	}

	certHash, err := getCertificateId(member.GetMemberInfo(), hashType)
	if err != nil {
		return nil, fmt.Errorf("calc cert hash failed, %s", err.Error())
	}

	return certHash, nil
}

func getCertificateId(certPEM []byte, hashType string) ([]byte, error) {
	if certPEM == nil {
		return nil, fmt.Errorf("get cert certPEM == nil")
	}

	certDer, _ := pem.Decode(certPEM)
	if certDer == nil {
		return nil, fmt.Errorf("invalid certificate")
	}

	return getCertificateIdFromDER(certDer.Bytes, hashType)
}

func getCertificateIdFromDER(certDER []byte, hashType string) ([]byte, error) {
	if certDER == nil {
		return nil, fmt.Errorf("get cert from der certDER == nil")
	}

	id, err := hash.GetByStrType(hashType, certDER)
	if err != nil {
		return nil, err
	}

	return id, nil
}

// ParseCert convert bytearray to certificate
func ParseCert(crtPEM []byte) (*bcx509.Certificate, error) {
	certBlock, _ := pem.Decode(crtPEM)
	if certBlock == nil {
		return nil, fmt.Errorf("decode pem failed, invalid certificate")
	}

	cert, err := bcx509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("x509 parse cert failed, %s", err)
	}

	return cert, nil
}
