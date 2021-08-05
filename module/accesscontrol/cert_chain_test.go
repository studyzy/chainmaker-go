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

	bcx509 "chainmaker.org/chainmaker/common/crypto/x509"
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
MIICJzCCAc6gAwIBAgIEAKWqRTAKBggqhkjOPQQDAjBZMQswCQYDVQQGEwJDTjES
MBAGA1UECAwJR3VhbmdEb25nMRIwEAYDVQQHDAlTaGVuIFpoZW4xEDAOBgNVBAoM
B1RlbmNlbnQxEDAOBgNVBAMMB1Jvb3QgQ0EwHhcNMjAxMjA3MDcyMDM4WhcNNDAx
MjAyMDcyMDM4WjBZMQswCQYDVQQGEwJDTjESMBAGA1UECAwJR3VhbmdEb25nMRIw
EAYDVQQHDAlTaGVuIFpoZW4xEDAOBgNVBAoMB1RlbmNlbnQxEDAOBgNVBAMMB1Jv
b3QgQ0EwWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAARZTkoNibOu8VBZbzmkvz9X
YQpKR22IZWZp5fVXI1EQeyJeahVNuTILGqGv87jksqIjeCNtFHKGQn9eWInUQOhi
o4GDMIGAMB0GA1UdDgQWBBQ8nk0ePMIfh4QgmgExojsXQE2GajAfBgNVHSMEGDAW
gBQ8nk0ePMIfh4QgmgExojsXQE2GajAPBgNVHRMBAf8EBTADAQH/MA4GA1UdDwEB
/wQEAwIBBjAdBgNVHREEFjAUgRJrcmFiYXRAdGVuY2VudC5jb20wCgYIKoZIzj0E
AwIDRwAwRAIgBTD1XlcJWNFp3PIpIpVJM6QTR3UeLQRH8fHFbAaTMl8CICZh0/SF
KYTt/Wg75K2HPy/gxiApg1fjLBhH87/tbq8w
-----END CERTIFICATE-----`,
	sk: `-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIFvv+MKo7KfS3dQRVg+X6I3WU64In8BUVoshOLtGozfJoAoGCCqGSM49
AwEHoUQDQgAEWU5KDYmzrvFQWW85pL8/V2EKSkdtiGVmaeX1VyNREHsiXmoVTbky
Cxqhr/O45LKiI3gjbRRyhkJ/XliJ1EDoYg==
-----END EC PRIVATE KEY-----`,
}
var intermediateCert = certificatePair{
	certificate: `-----BEGIN CERTIFICATE-----
MIICHjCCAcSgAwIBAgIIE/jsHHFlkm0wCgYIKoZIzj0EAwIwWTELMAkGA1UEBhMC
Q04xEjAQBgNVBAgMCUd1YW5nRG9uZzESMBAGA1UEBwwJU2hlbiBaaGVuMRAwDgYD
VQQKDAdUZW5jZW50MRAwDgYDVQQDDAdSb290IENBMB4XDTIwMTIwNzA4MTAwOFoX
DTMwMTIwNTA4MTAwOFowSDELMAkGA1UEBhMCQ04xEjAQBgNVBAgMCUd1YW5nRG9u
ZzEQMA4GA1UECgwHVGVuY2VudDETMBEGA1UEAwwKU2lnbmluZyBDQTBZMBMGByqG
SM49AgEGCCqGSM49AwEHA0IABGD0MA3oFq8Nsq0gsYwk3grZAA2Znsm35N6kemRO
oHksh2av7Cv8NFFfSR+lWv9cFKEI7VPv/wQWmXwSq9sB0yGjgYYwgYMwHQYDVR0O
BBYEFAUj6AEg1N6Sobbq8AgAQ3WcjpNWMB8GA1UdIwQYMBaAFDyeTR48wh+HhCCa
ATGiOxdATYZqMBIGA1UdEwEB/wQIMAYBAf8CAQAwDgYDVR0PAQH/BAQDAgEGMB0G
A1UdEQQWMBSBEmtyYWJhdEB0ZW5jZW50LmNvbTAKBggqhkjOPQQDAgNIADBFAiBu
5gA7EhnoG3Qi0kFOZ4DvPyuvEXHLBktvrYLvTqxU4QIhAKfPaJaHewk69QuJvXLZ
cNBHNPAt4HdPiRgUaZ5V9vjB
-----END CERTIFICATE-----`,
	sk: `-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgqVwWV9Edvu/GJRtY
iWYt584FVwh1moYWM85YoIwLb1mhRANCAARg9DAN6BavDbKtILGMJN4K2QANmZ7J
t+TepHpkTqB5LIdmr+wr/DRRX0kfpVr/XBShCO1T7/8EFpl8EqvbAdMh
-----END PRIVATE KEY-----`,
}
var leafCert = certificatePair{
	certificate: `-----BEGIN CERTIFICATE-----
MIIC3jCCAoSgAwIBAgIILD85b0PnKPQwCgYIKoZIzj0EAwIwSDELMAkGA1UEBhMC
Q04xEjAQBgNVBAgMCUd1YW5nRG9uZzEQMA4GA1UECgwHVGVuY2VudDETMBEGA1UE
AwwKU2lnbmluZyBDQTAeFw0yMDEyMDcwODUyMTlaFw0yMTEyMTcwODUyMTlaMEcx
CzAJBgNVBAYTAkNOMRIwEAYDVQQIDAlHdWFuZ0RvbmcxEjAQBgNVBAcMCVNoZW4g
WmhlbjEQMA4GA1UECgwHVGVuY2VudDBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IA
BCHUGXzfFu554qNb1ujafzj0NFj0ItsPkeOPtcT8UZeQjho0pbUFhBQ+lIA/jbkQ
MXkUsX6IYtwpA3x35U0kBM2jggFXMIIBUzAJBgNVHRMEAjAAMBEGCWCGSAGG+EIB
AQQEAwIGQDAzBglghkgBhvhCAQ0EJhYkT3BlblNTTCBHZW5lcmF0ZWQgU2VydmVy
IENlcnRpZmljYXRlMB0GA1UdDgQWBBSDtRiPDwRLb0ubj2GJdS/aGWanFTCBiQYD
VR0jBIGBMH+AFAUj6AEg1N6Sobbq8AgAQ3WcjpNWoV2kWzBZMQswCQYDVQQGEwJD
TjESMBAGA1UECAwJR3VhbmdEb25nMRIwEAYDVQQHDAlTaGVuIFpoZW4xEDAOBgNV
BAoMB1RlbmNlbnQxEDAOBgNVBAMMB1Jvb3QgQ0GCCBP47BxxZZJtMA4GA1UdDwEB
/wQEAwIFoDATBgNVHSUEDDAKBggrBgEFBQcDATAuBgNVHREEJzAlgg93d3cuZXhh
bXBsZS5jb22BEmtyYWJhdEB0ZW5jZW50LmNvbTAKBggqhkjOPQQDAgNIADBFAiEA
wTwF7hyCXDsHf6sOWsaA2mDZWNp186gu1K5FzVZnz+cCIBr7vb7uw7paSa4Sl2c0
MpAwyD3Knjd8vf2CXXLweN0N
-----END CERTIFICATE-----`,
	sk: `-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgpMXAqtcXUknLkY93
291zN/0CkQx8qdsIYO4N46N+C6WhRANCAAQh1Bl83xbueeKjW9bo2n849DRY9CLb
D5Hjj7XE/FGXkI4aNKW1BYQUPpSAP425EDF5FLF+iGLcKQN8d+VNJATN
-----END PRIVATE KEY-----`,
}

func TestCertChainFunction(t *testing.T) {
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
