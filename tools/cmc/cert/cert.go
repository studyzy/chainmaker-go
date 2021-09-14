/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cert

import (
	"crypto/rand"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"chainmaker.org/chainmaker/common/v2/cert"
	"chainmaker.org/chainmaker/common/v2/crypto"
	"chainmaker.org/chainmaker/common/v2/crypto/asym"
	"chainmaker.org/chainmaker/common/v2/crypto/x509"
	"chainmaker.org/chainmaker/common/v2/random/uuid"
)

const (
	expireYear = 8
	c          = "CN"
	l          = "Beijing"
	p          = "Beijing"
)

var (
	sans = []string{"localhost", "chainmaker.org", "127.0.0.1"}
)

func CertCMD() *cobra.Command {
	certCmd := &cobra.Command{
		Use:   "cert",
		Short: "ChainMaker cert command",
		Long:  "ChainMaker cert command",
	}
	certCmd.AddCommand(caCMD())
	certCmd.AddCommand(csrCMD())
	certCmd.AddCommand(issueCMD())
	certCmd.AddCommand(createCertCrlCMD())
	certCmd.AddCommand(nodeIdCMD())
	certCmd.AddCommand(addrCMD())
	certCmd.AddCommand(certToUserAddrInStake())
	return certCmd
}

func caCMD() *cobra.Command {
	caCmd := &cobra.Command{
		Use:   "ca",
		Short: "Create certificate authority crtificate",
		Long:  "Create certificate authority crtificate",
		RunE: func(_ *cobra.Command, _ []string) error {
			return createCACertificate()
		},
	}

	attachFlags(caCmd, []string{
		"key-path", "hash", "path",
		"name", "org", "cn", "ou",
	})

	return caCmd
}

func csrCMD() *cobra.Command {
	csrCmd := &cobra.Command{
		Use:   "csr",
		Short: "Create certificate request",
		Long:  "Create certificate request",
		RunE: func(_ *cobra.Command, _ []string) error {
			return createCSR()
		},
	}

	attachFlags(csrCmd, []string{
		"key-path", "path", "name",
		"org", "cn", "ou",
	})

	return csrCmd
}

func issueCMD() *cobra.Command {
	issueCmd := &cobra.Command{
		Use:   "issue",
		Short: "Issue certificate",
		Long:  "Issue certificate",
		RunE: func(_ *cobra.Command, _ []string) error {
			return issueCertificate()
		},
	}

	attachFlags(issueCmd, []string{
		"hash", "is-ca", "ca-key-path",
		"ca-cert-path", "csr-path", "path", "name",
	})

	return issueCmd
}

func createCACertificate() error {
	privKey, err := loadPrivateKey(keyPath)
	if err != nil {
		return err
	}
	hashType := crypto.HashAlgoMap[strings.ToUpper(hash)]
	return cert.CreateCACertificate(&cert.CACertificateConfig{PrivKey: privKey, HashType: hashType,
		CertPath: path, CertFileName: name, Country: c, Locality: l, Province: p, OrganizationalUnit: ou,
		Organization: org, CommonName: cn, ExpireYear: expireYear, Sans: sans})
}

func createCSR() error {
	privKey, err := loadPrivateKey(keyPath)
	if err != nil {
		return err
	}
	return cert.CreateCSR(&cert.CSRConfig{PrivKey: privKey, CsrPath: path, CsrFileName: name,
		Country: c, Locality: l, Province: p, OrganizationalUnit: ou, Organization: org, CommonName: cn})
}

func issueCertificate() error {
	hashType := crypto.HashAlgoMap[strings.ToUpper(hash)]
	return cert.IssueCertificate(&cert.IssueCertificateConfig{HashType: hashType, IsCA: isCA,
		IssuerPrivKeyFilePath: caKeyPath, IssuerCertFilePath: caCertPath,
		CsrFilePath: csrPath, CertPath: path, CertFileName: name,
		ExpireYear: expireYear, Sans: sans, Uuid: uuid.GetUUID()})
}

func loadPrivateKey(path string) (crypto.PrivateKey, error) {
	raw, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return asym.PrivateKeyFromPEM(raw, nil)
}

func createCertCrlCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "crl",
		Short: "create cert crl",
		Long:  "create cert crl",
		RunE: func(_ *cobra.Command, _ []string) error {
			return createCrl()
		},
	}

	attachFlags(cmd, []string{
		flagCrlPath, flagCrtPath, flagCaKeyPath, flagCaCertPath,
	})

	cmd.MarkFlagRequired(flagCrlPath)
	cmd.MarkFlagRequired(flagCrtPath)
	cmd.MarkFlagRequired(flagCaKeyPath)
	cmd.MarkFlagRequired(flagCaCertPath)

	return cmd
}

func createCrl() error {
	revokedCert, err := cert.ParseCertificate(crtPath)
	if err != nil {
		return fmt.Errorf("parse cert failed, %s", err.Error())
	}
	issuerPrivKeyRaw, err := ioutil.ReadFile(caKeyPath)
	if err != nil {
		return fmt.Errorf("read ca key file failed, %s", err.Error())
	}
	issuerCert, err := cert.ParseCertificate(caCertPath)
	if err != nil {
		return fmt.Errorf("parse ca cert file failed, %s", err.Error())
	}
	block, _ := pem.Decode(issuerPrivKeyRaw)
	if block == nil {
		return errors.New("pem.Decode failed, invalid cert")
	}
	issuerPrivKey, err := asym.PrivateKeyFromDER(block.Bytes)
	if err != nil {
		return fmt.Errorf("load private key from der failed, %s", err.Error())
	}
	var revokedCerts []pkix.RevokedCertificate
	var revoked pkix.RevokedCertificate
	certSn := revokedCert.SerialNumber
	revoked.SerialNumber = big.NewInt(certSn.Int64())
	revoked.RevocationTime = time.Unix(1711206185, 0)
	revokedCerts = append(revokedCerts, revoked)
	now := time.Now()
	next := now.Add(time.Duration(4) * time.Hour) //撤销列表过期时间（4小时候这个撤销列表就不是最新的了）
	crlBytes, err := x509.CreateCRL(rand.Reader, issuerCert, issuerPrivKey.ToStandardKey(), revokedCerts, now, next)
	if err != nil {
		return fmt.Errorf("create crl failed, %s", err.Error())
	}
	err = ioutil.WriteFile(crlPath, pem.EncodeToMemory(&pem.Block{Type: "X509 CRL", Bytes: crlBytes}), os.ModePerm)
	if err != nil {
		return fmt.Errorf("write crl file failed, %s", err.Error())
	}
	return nil
}
