/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package payload

import (
	"encoding/pem"
	"fmt"
	"io/ioutil"

	"github.com/gogo/protobuf/proto"
	"github.com/spf13/cobra"

	"chainmaker.org/chainmaker/common/v2/crypto"
	"chainmaker.org/chainmaker/common/v2/crypto/asym"
	bcx509 "chainmaker.org/chainmaker/common/v2/crypto/x509"
	sdkPbAc "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	sdkPbCommon "chainmaker.org/chainmaker/pb-go/v2/common"
)

var (
	signInput  string
	signOutput string
)

func signCMD() *cobra.Command {
	signCmd := &cobra.Command{
		Use:   "sign",
		Short: "Sign pb file command",
		Long:  "Sign pb file command",
	}

	flags := signCmd.PersistentFlags()
	flags.StringVarP(&signInput, "input", "i", "./collect.pb", "specify input file")
	flags.StringVarP(&signOutput, "output", "o", "./collect-signed.pb", "specify output file")
	flags.StringVarP(&orgId, "org-id", "O", "wx-org1.chainmaker.org", "specify organization identity")
	flags.StringVarP(&adminKeyPath, "admin-key-path", "k", "./admin1.sign.key", "specify admin key path")
	flags.StringVarP(&adminCertPath, "admin-crt-path", "c", "./admin1.sign.crt", "specify admin certificate path")

	signCmd.AddCommand(signSystemContractPayloadCMD())
	signCmd.AddCommand(signContractMgmtPayloadCMD())

	return signCmd
}

func signSystemContractPayloadCMD() *cobra.Command {
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Config command",
		Long:  "Config command",
		RunE: func(_ *cobra.Command, _ []string) error {
			return signPayload()
		},
	}
	return configCmd
}

func signContractMgmtPayloadCMD() *cobra.Command {
	contractCmd := &cobra.Command{
		Use:   "contract",
		Short: "Contract command",
		Long:  "Contract command",
		RunE: func(_ *cobra.Command, _ []string) error {
			return signPayload()
		},
	}
	return contractCmd
}

const LOAD_FILE_ERROR_FORMAT = "Load file %s error: %s"

func signPayload() error {
	raw, err := ioutil.ReadFile(signInput)
	if err != nil {
		return fmt.Errorf(LOAD_FILE_ERROR_FORMAT, signInput, err)
	}

	payload := &sdkPbCommon.Payload{}
	if err := proto.Unmarshal(raw, payload); err != nil {
		return fmt.Errorf("SystemContractPayload unmarshal error: %s", err)
	}

	entry, err := sign(raw)
	if err != nil {
		return err
	}
	tx := &sdkPbCommon.TxRequest{Payload: payload}
	tx.Sender = entry

	bytes, err := proto.Marshal(tx)
	if err != nil {
		return fmt.Errorf("SystemContractPayload marshal error: %s", err)
	}

	if err = ioutil.WriteFile(signOutput, bytes, 0600); err != nil {
		return fmt.Errorf("Write to file %s error: %s", signOutput, err)
	}

	return nil
}

//func getSigner(sk3 crypto.PrivateKey, sender *sdkPbCommon.Member) (protocol.SigningMember, error) {
//	skPEM, err := sk3.String()
//	if err != nil {
//		return nil, err
//	}
//
//	m, err := accesscontrol.MockAccessControl().NewMember(sender.OrgId, string(sender.MemberInfo))
//	if err != nil {
//		return nil, err
//	}
//
//	signer, err := accesscontrol.MockAccessControl().NewSigningMember(m, skPEM, "")
//	if err != nil {
//		return nil, err
//	}
//	return signer, nil
//}

func sign(msg []byte) (*sdkPbCommon.EndorsementEntry, error) {
	keyFile, err := ioutil.ReadFile(adminKeyPath)
	if err != nil {
		return nil, fmt.Errorf(LOAD_FILE_ERROR_FORMAT, adminKeyPath, err)
	}
	sk3, err := asym.PrivateKeyFromPEM(keyFile, nil)
	if err != nil {
		return nil, fmt.Errorf("Load private key error: %s", err)
	}

	certFile, err := ioutil.ReadFile(adminCertPath)
	if err != nil {
		return nil, fmt.Errorf(LOAD_FILE_ERROR_FORMAT, adminCertPath, err)
	}

	userCrt, err := ParseCert(certFile)
	if err != nil {
		return nil, fmt.Errorf("ParseCert failed, %s", err.Error())
	}

	sig, err := SignTx(sk3, userCrt, msg)
	if err != nil {
		return nil, fmt.Errorf("SignTx failed, %s", err)
	}

	sender := &sdkPbAc.Member{
		OrgId:      orgId,
		MemberInfo: certFile,
		//IsFullCert: true,
	}

	return &sdkPbCommon.EndorsementEntry{
		Signer:    sender,
		Signature: sig,
	}, nil

	//signer, err := getSigner(sk3, sender)
	//if err != nil {
	//	return nil, fmt.Errorf("Get signer error: %s", err)
	//}
	//
	//sig, err := signer.Sign("SHA256", msg)
	//if err != nil {
	//	return nil, fmt.Errorf("Sign error: %s", err)
	//}
	//
	//signerSerial, err := signer.GetMember(true)
	//if err != nil {
	//	return nil, fmt.Errorf("GetMember error: %s", err)
	//}
	//
	//return &sdkPbCommon.EndorsementEntry{
	//	Signer:    signerSerial,
	//	Signature: sig,
	//}, nil
}

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

func SignTx(privateKey crypto.PrivateKey, cert *bcx509.Certificate, msg []byte) ([]byte, error) {
	var opts crypto.SignOpts
	hashalgo, err := bcx509.GetHashFromSignatureAlgorithm(cert.SignatureAlgorithm)
	if err != nil {
		return nil, fmt.Errorf("invalid algorithm: %v", err)
	}

	opts.Hash = hashalgo
	opts.UID = crypto.CRYPTO_DEFAULT_UID

	return privateKey.SignWithOpts(msg, &opts)
}
