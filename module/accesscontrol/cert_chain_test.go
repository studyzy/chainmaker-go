/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package accesscontrol

import (
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"testing"
	"time"

	bcx509 "chainmaker.org/chainmaker/common/v2/crypto/x509"
	"github.com/stretchr/testify/require"
)

const (
	rawChainTemplate    = "raw chain: %v\n"
	sortedChainTemplate = "sorted chain: %v\n"
)

type certificatePair struct {
	certificate string
	sk          string
}

var (
	sans = []string{"127.0.0.1", "localhost", "chainmaker.org", "8.8.8.8"}
)

var rootCert = certificatePair{
	certificate: `-----BEGIN CERTIFICATE-----
MIICFTCCAbugAwIBAgIIDyXoqXE5qO4wCgYIKoZIzj0EAwIwYjELMAkGA1UEBhMC
Q04xEDAOBgNVBAgTB0JlaWppbmcxEDAOBgNVBAcTB0JlaWppbmcxETAPBgNVBAoT
CG9yZy1yb290MQ0wCwYDVQQLEwRyb290MQ0wCwYDVQQDEwRyb290MB4XDTIxMTEx
MDEyNTcwM1oXDTIzMTExMDEyNTcwM1owYjELMAkGA1UEBhMCQ04xEDAOBgNVBAgT
B0JlaWppbmcxEDAOBgNVBAcTB0JlaWppbmcxETAPBgNVBAoTCG9yZy1yb290MQ0w
CwYDVQQLEwRyb290MQ0wCwYDVQQDEwRyb290MFkwEwYHKoZIzj0CAQYIKoZIzj0D
AQcDQgAEARH59W0nG0cP1hC3qX3h/yR7ZTd3vgkiQYN0fg4kzIGrvgxitRqWHAlI
pm3lQRBXsCtZiV3bMPeiUdgmyJ7A3qNbMFkwDgYDVR0PAQH/BAQDAgEGMA8GA1Ud
EwEB/wQFMAMBAf8wKQYDVR0OBCIEIHPIhP3krHqJ28NvRxNHiZY7/F00eMvSd8/T
ZMwvHtPdMAsGA1UdEQQEMAKCADAKBggqhkjOPQQDAgNIADBFAiEA8pDmJLORvRPA
EbK1soTy+1NIPiVOmud1nZ37vnXOGGwCIAtyYCvZufvGrDiCoM4fjqHKqeJTqM8d
LHRJcBq7K+8/
-----END CERTIFICATE-----`,
	sk: `-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIJK+5Jakx6gk0tXVzPDw3Oj2AwTQxaK/YGn11DRh4OLAoAoGCCqGSM49
AwEHoUQDQgAEARH59W0nG0cP1hC3qX3h/yR7ZTd3vgkiQYN0fg4kzIGrvgxitRqW
HAlIpm3lQRBXsCtZiV3bMPeiUdgmyJ7A3g==
-----END EC PRIVATE KEY-----`,
}
var intermediateCert = certificatePair{
	certificate: `-----BEGIN CERTIFICATE-----
MIICZzCCAgygAwIBAgIIKZ4swBOaRVowCgYIKoZIzj0EAwIwYjELMAkGA1UEBhMC
Q04xEDAOBgNVBAgTB0JlaWppbmcxEDAOBgNVBAcTB0JlaWppbmcxETAPBgNVBAoT
CG9yZy1yb290MQ0wCwYDVQQLEwRyb290MQ0wCwYDVQQDEwRyb290MB4XDTIxMTEx
MDEyNTcwM1oXDTIzMTExMDEyNTcwM1owgYMxCzAJBgNVBAYTAkNOMRAwDgYDVQQI
EwdCZWlqaW5nMRAwDgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmcxLmNo
YWlubWFrZXIub3JnMQswCQYDVQQLEwJjYTEiMCAGA1UEAxMZY2Etd3gtb3JnMS5j
aGFpbm1ha2VyLm9yZzBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABFb/8TENIxlS
PnCCXKt7KMEDMtZhnIEdruuCvLgLAM4OgzKNekmBj1yXYKLJxmA4irTZhfse2mXF
LjiaDrLGZXGjgYkwgYYwDgYDVR0PAQH/BAQDAgEGMA8GA1UdEwEB/wQFMAMBAf8w
KQYDVR0OBCIEICynoTa2/an7Yxhb4JLhGW433/OU5EErUPCDZjXSnm4FMCsGA1Ud
IwQkMCKAIHPIhP3krHqJ28NvRxNHiZY7/F00eMvSd8/TZMwvHtPdMAsGA1UdEQQE
MAKCADAKBggqhkjOPQQDAgNJADBGAiEAr32KGYO4eb6DRv3FXUlXaT7CnXJr/zfd
1aVw/Afz0jACIQC1sIop9VKnK/c3qvRSdK2FcV/v3aHvAlBD82QX3YrsEQ==
-----END CERTIFICATE-----`,
	sk: `-----BEGIN EC PRIVATE KEY-----
MHcCAQEEINGqjsCE7rd9sS/LhU0fSPVgZZS19P4UKVq96da/C6SxoAoGCCqGSM49
AwEHoUQDQgAEVv/xMQ0jGVI+cIJcq3sowQMy1mGcgR2u64K8uAsAzg6DMo16SYGP
XJdgosnGYDiKtNmF+x7aZcUuOJoOssZlcQ==
-----END EC PRIVATE KEY-----`,
}
var leafCert = certificatePair{
	certificate: `-----BEGIN CERTIFICATE-----
MIIClzCCAj2gAwIBAgIIF6NZozbc5lkwCgYIKoZIzj0EAwIwgYMxCzAJBgNVBAYT
AkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAwDgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQK
ExZ3eC1vcmcxLmNoYWlubWFrZXIub3JnMQswCQYDVQQLEwJjYTEiMCAGA1UEAxMZ
Y2Etd3gtb3JnMS5jaGFpbm1ha2VyLm9yZzAeFw0yMTEyMDcwMzQyMzdaFw0yMzEy
MDcwMzQyMzdaMHsxCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAwDgYD
VQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmcxLmNoYWlubWFrZXIub3JnMRIw
EAYDVQQLEwljb25zZW5zdXMxEzARBgNVBAMTCmNvbnNlbnN1czEwWTATBgcqhkjO
PQIBBggqhkjOPQMBBwNCAAQ8Da9Lolmhwr+1K1BKPzlGJteCYJGfR1InYxGbLV7v
4ulGR4FsTao/4iwYClLK4cCnRkS9k/c/aFuNdxE/Qn7Lo4GhMIGeMA4GA1UdDwEB
/wQEAwID+DAdBgNVHSUEFjAUBggrBgEFBQcDAgYIKwYBBQUHAwEwKQYDVR0OBCIE
IGqcFT2jm6yfyDROcHNjMbQedZr0WPHLZ0gYYOR5OXyOMCsGA1UdIwQkMCKAICyn
oTa2/an7Yxhb4JLhGW433/OU5EErUPCDZjXSnm4FMBUGA1UdEQQOMAyCCmNvbnNl
bnN1czEwCgYIKoZIzj0EAwIDSAAwRQIhAP0Cb5DBch9o1e3RmZV2cfwu4Za0O9vW
ObOE/b+dTV0aAiAr5ugbxeO51V932kzd8YAZyOyR5s7XJ4dHpkp326DarQ==
-----END CERTIFICATE-----`,
	sk: `-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIEnlYWSJpYkSAMo0wt3e7Nk1UolzBKh9x68Ob4hBADY1oAoGCCqGSM49
AwEHoUQDQgAEPA2vS6JZocK/tStQSj85RibXgmCRn0dSJ2MRmy1e7+LpRkeBbE2q
P+IsGApSyuHAp0ZEvZP3P2hbjXcRP0J+yw==
-----END EC PRIVATE KEY-----`,
}

func TestCertChainFunction(t *testing.T) {
	{
		fmt.Printf("sans is unused: [sans: %s]\n", sans)
	}
	blockCA, _ := pem.Decode([]byte(rootCert.certificate))
	certRootCA, err := bcx509.ParseCertificate(blockCA.Bytes)
	require.Nil(t, err)
	blockIntermediate, _ := pem.Decode([]byte(intermediateCert.certificate))
	certIntermediate, err := bcx509.ParseCertificate(blockIntermediate.Bytes)
	require.Nil(t, err)
	blockLeaf, _ := pem.Decode([]byte(leafCert.certificate))
	certLeaf, err := bcx509.ParseCertificate(blockLeaf.Bytes)
	require.Nil(t, err)
	rootCertPool := bcx509.NewCertPool()
	rootCertPool.AddCert(certRootCA)
	intermediateCertPool := bcx509.NewCertPool()
	intermediateCertPool.AddCert(certIntermediate)
	chains, err := certIntermediate.Verify(bcx509.VerifyOptions{
		DNSName:                   "",
		Intermediates:             bcx509.NewCertPool(),
		Roots:                     rootCertPool,
		CurrentTime:               time.Time{},
		KeyUsages:                 []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
		MaxConstraintComparisions: 0,
	})
	require.Nil(t, err)
	require.NotNil(t, chains)
	chains, err = certLeaf.Verify(bcx509.VerifyOptions{
		DNSName:                   "",
		Intermediates:             intermediateCertPool,
		Roots:                     rootCertPool,
		CurrentTime:               time.Time{},
		KeyUsages:                 []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
		MaxConstraintComparisions: 0,
	})
	require.Nil(t, err)
	require.NotNil(t, chains)
	chains, err = certLeaf.Verify(bcx509.VerifyOptions{
		DNSName:                   "",
		Intermediates:             bcx509.NewCertPool(),
		Roots:                     intermediateCertPool,
		CurrentTime:               time.Time{},
		KeyUsages:                 []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
		MaxConstraintComparisions: 0,
	})
	require.Nil(t, err)
	require.NotNil(t, chains)
	allPool := bcx509.NewCertPool()
	allPool.AddCert(certRootCA)
	allPool.AddCert(certIntermediate)
	allPool.AddCert(certLeaf)
	chains, err = certLeaf.Verify(bcx509.VerifyOptions{
		DNSName:                   "",
		Intermediates:             allPool,
		Roots:                     rootCertPool,
		CurrentTime:               time.Time{},
		KeyUsages:                 []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
		MaxConstraintComparisions: 0,
	})
	require.Nil(t, err)
	require.NotNil(t, chains)
	fmt.Printf("%v\n", chains)
	chains, err = certIntermediate.Verify(bcx509.VerifyOptions{
		DNSName:                   "",
		Intermediates:             allPool,
		Roots:                     rootCertPool,
		CurrentTime:               time.Time{},
		KeyUsages:                 []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
		MaxConstraintComparisions: 0,
	})
	require.Nil(t, err)
	require.NotNil(t, chains)
	fmt.Printf("%v\n", chains)
	chains, err = certRootCA.Verify(bcx509.VerifyOptions{
		DNSName:                   "",
		Intermediates:             allPool,
		Roots:                     rootCertPool,
		CurrentTime:               time.Time{},
		KeyUsages:                 []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
		MaxConstraintComparisions: 0,
	})
	require.Nil(t, err)
	require.NotNil(t, chains)
	fmt.Printf("%v\n", chains)
	rootCertAllPool := bcx509.NewCertPool()
	rootCertAllPool.AddCert(certRootCA)
	rootCertAllPool.AddCert(certIntermediate)
	chains, err = certLeaf.Verify(bcx509.VerifyOptions{
		DNSName:                   "",
		Intermediates:             allPool,
		Roots:                     rootCertAllPool,
		CurrentTime:               time.Time{},
		KeyUsages:                 []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
		MaxConstraintComparisions: 0,
	})
	require.Nil(t, err)
	require.NotNil(t, chains)
	fmt.Printf("%v\n", chains)
	rawChain := []*bcx509.Certificate{certIntermediate, certRootCA, certLeaf}
	sortedChain := bcx509.BuildCertificateChain(rawChain)
	require.NotNil(t, sortedChain)
	fmt.Printf(rawChainTemplate, rawChain)
	fmt.Printf(sortedChainTemplate, sortedChain)
	rawChain = []*bcx509.Certificate{certIntermediate}
	sortedChain = bcx509.BuildCertificateChain(rawChain)
	require.NotNil(t, sortedChain)
	fmt.Printf(rawChainTemplate, rawChain)
	fmt.Printf(sortedChainTemplate, sortedChain)
	rawChain = []*bcx509.Certificate{certIntermediate, certRootCA}
	sortedChain = bcx509.BuildCertificateChain(rawChain)
	require.NotNil(t, sortedChain)
	fmt.Printf(rawChainTemplate, rawChain)
	fmt.Printf(sortedChainTemplate, sortedChain)
	rawChain = []*bcx509.Certificate{certRootCA, certIntermediate}
	sortedChain = bcx509.BuildCertificateChain(rawChain)
	require.NotNil(t, sortedChain)
	fmt.Printf(rawChainTemplate, rawChain)
	fmt.Printf(sortedChainTemplate, sortedChain)
	rawChain = []*bcx509.Certificate{certLeaf}
	sortedChain = bcx509.BuildCertificateChain(rawChain)
	require.NotNil(t, sortedChain)
	fmt.Printf(rawChainTemplate, rawChain)
	fmt.Printf(sortedChainTemplate, sortedChain)
	fmt.Printf("root: %s, %p\n", hex.EncodeToString(certRootCA.SubjectKeyId), certRootCA)
	fmt.Printf("intermediate: %s, %p\n", hex.EncodeToString(certIntermediate.SubjectKeyId), certIntermediate)
	fmt.Printf("leaf: %s, %p\n", hex.EncodeToString(certLeaf.SubjectKeyId), certLeaf)
}
