/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"errors"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/gogo/protobuf/proto"
	"github.com/spf13/cobra"

	"chainmaker.org/chainmaker-go/common/crypto"
	"chainmaker.org/chainmaker-go/common/crypto/asym"
	bcx509 "chainmaker.org/chainmaker-go/common/crypto/x509"
	sdk "chainmaker.org/chainmaker-sdk-go"
	"chainmaker.org/chainmaker-sdk-go/pb/protogo/accesscontrol"
	"chainmaker.org/chainmaker-sdk-go/pb/protogo/common"
)

const (
	addTrustRoot = iota
	removeTrustRoot
	updateTrustRoot
)

func configTrustRootCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trustroot",
		Short: "trust root command",
		Long:  "trust root command",
	}
	cmd.AddCommand(addTrustRootCMD())
	cmd.AddCommand(removeTrustRootCMD())
	cmd.AddCommand(updateTrustRootCMD())

	return cmd
}

func addTrustRootCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "add trust root ca cert",
		Long:  "add trust root ca cert",
		RunE: func(_ *cobra.Command, _ []string) error {
			return configTrustRoot(addTrustRoot)
		},
	}

	attachFlags(cmd, []string{
		flagSdkConfPath, flagOrgId, flagEnableCertHash, flagTrustRootCrtPath, flagTrustRootOrgId,
		flagAdminCrtFilePaths, flagAdminKeyFilePaths, flagClientCrtFilePaths, flagClientKeyFilePaths,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagAdminCrtFilePaths)
	cmd.MarkFlagRequired(flagAdminKeyFilePaths)
	cmd.MarkFlagRequired(flagTrustRootOrgId)
	cmd.MarkFlagRequired(flagTrustRootCrtPath)

	return cmd
}

func removeTrustRootCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove",
		Short: "remove trust root ca cert",
		Long:  "remove trust root ca cert",
		RunE: func(_ *cobra.Command, _ []string) error {
			return configTrustRoot(removeTrustRoot)
		},
	}

	attachFlags(cmd, []string{
		flagSdkConfPath, flagOrgId, flagEnableCertHash, flagTrustRootCrtPath, flagTrustRootOrgId,
		flagAdminCrtFilePaths, flagAdminKeyFilePaths, flagClientCrtFilePaths, flagClientKeyFilePaths,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagAdminCrtFilePaths)
	cmd.MarkFlagRequired(flagAdminKeyFilePaths)
	cmd.MarkFlagRequired(flagTrustRootOrgId)
	cmd.MarkFlagRequired(flagTrustRootCrtPath)

	return cmd
}

func updateTrustRootCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "update trust root ca cert",
		Long:  "update trust root ca cert",
		RunE: func(_ *cobra.Command, _ []string) error {
			return configTrustRoot(updateTrustRoot)
		},
	}

	attachFlags(cmd, []string{
		flagSdkConfPath, flagOrgId, flagEnableCertHash, flagTrustRootCrtPath, flagTrustRootOrgId,
		flagAdminCrtFilePaths, flagAdminKeyFilePaths, flagClientCrtFilePaths, flagClientKeyFilePaths,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagAdminCrtFilePaths)
	cmd.MarkFlagRequired(flagAdminKeyFilePaths)
	cmd.MarkFlagRequired(flagTrustRootOrgId)
	cmd.MarkFlagRequired(flagTrustRootCrtPath)

	return cmd
}

func configTrustRoot(op int) error {
	adminKeys := strings.Split(adminKeyFilePaths, ",")
	adminCrts := strings.Split(adminCrtFilePaths, ",")
	if len(adminKeys) == 0 || len(adminCrts) == 0 || len(adminKeys) != len(adminCrts) {
		return fmt.Errorf(ADMIN_KEY_AND_CERT_NOT_ENOUGH_FORMAT, len(adminKeys), len(adminCrts))
	}

	client, err := createClientWithConfig()
	if err != nil {
		return fmt.Errorf(CREATE_USER_CLIENT_FAILED_FORMAT, err)
	}
	defer client.Stop()

	var trustRootBytes []byte
	if op == addTrustRoot || op == updateTrustRoot {
		if trustRootPath == "" {
			return fmt.Errorf("please specify trust root path")
		}
		trustRootBytes, err = ioutil.ReadFile(trustRootPath)
		if err != nil {
			return err
		}
	}

	var payloadBytes []byte
	switch op {
	case addTrustRoot:
		payloadBytes, err = client.CreateChainConfigTrustRootAddPayload(trustRootOrgId, string(trustRootBytes))
	case removeTrustRoot:
		payloadBytes, err = client.CreateChainConfigTrustRootDeletePayload(trustRootOrgId)
	case updateTrustRoot:
		payloadBytes, err = client.CreateChainConfigTrustRootUpdatePayload(trustRootOrgId, string(trustRootBytes))
	default:
		err = errors.New("invalid trust root operation")
	}
	if err != nil {
		return err
	}

	signedPayloads := make([][]byte, len(adminKeys))
	baseOrgId := "wx-org%d.chainmaker.org"
	for i := range adminKeys {
		_, privKey, err := dealUserKey(adminKeys[i])
		if err != nil {
			return err
		}
		crtBytes, crt, err := dealUserCrt(adminCrts[i])
		if err != nil {
			return err
		}

		signedPayload, err := signChainConfigPayload(payloadBytes, crtBytes, privKey, crt, fmt.Sprintf(baseOrgId, i+1))
		if err != nil {
			return err
		}
		signedPayloads[i] = signedPayload
	}

	mergedSignedPayloadBytes, err := client.MergeChainConfigSignedPayload(signedPayloads)
	if err != nil {
		return err
	}

	resp, err := client.SendChainConfigUpdateRequest(mergedSignedPayloadBytes)
	if err != nil {
		return err
	}
	err = checkProposalRequestResp(resp, true)
	if err != nil {
		return err
	}
	fmt.Printf("trustroot response %+v\n", resp)
	return nil
}

func signChainConfigPayload(payloadBytes, userCrtBytes []byte, privateKey crypto.PrivateKey, userCrt *bcx509.Certificate, orgId string) ([]byte, error) {
	payload := &common.SystemContractPayload{}
	if err := proto.Unmarshal(payloadBytes, payload); err != nil {
		return nil, fmt.Errorf("unmarshal config update payload failed, %s", err)
	}

	signBytes, err := signTx(privateKey, userCrt, payloadBytes)
	if err != nil {
		return nil, fmt.Errorf("SignPayload failed, %s", err)
	}

	sender := &accesscontrol.SerializedMember{
		OrgId:      orgId,
		MemberInfo: userCrtBytes,
		IsFullCert: true,
	}

	entry := &common.EndorsementEntry{
		Signer:    sender,
		Signature: signBytes,
	}

	payload.Endorsement = []*common.EndorsementEntry{
		entry,
	}

	signedPayloadBytes, err := proto.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal config update sigend payload failed, %s", err)
	}

	return signedPayloadBytes, nil
}

func signTx(privateKey crypto.PrivateKey, cert *bcx509.Certificate, msg []byte) ([]byte, error) {
	var opts crypto.SignOpts
	hashalgo, err := bcx509.GetHashFromSignatureAlgorithm(cert.SignatureAlgorithm)
	if err != nil {
		return nil, fmt.Errorf("invalid algorithm: %v", err)
	}

	opts.Hash = hashalgo
	opts.UID = crypto.CRYPTO_DEFAULT_UID

	return privateKey.SignWithOpts(msg, &opts)
}

func dealUserCrt(userCrtFilePath string) (userCrtBytes []byte, userCrt *bcx509.Certificate, err error) {

	// 读取用户证书
	userCrtBytes, err = ioutil.ReadFile(userCrtFilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("read user crt file failed, %s", err)
	}

	// 将证书转换为证书对象
	userCrt, err = sdk.ParseCert(userCrtBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("ParseCert failed, %s", err)
	}
	return
}

func dealUserKey(userKeyFilePath string) (userKeyBytes []byte, privateKey crypto.PrivateKey, err error) {

	// 从私钥文件读取用户私钥，转换为privateKey对象
	userKeyBytes, err = ioutil.ReadFile(userKeyFilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("read user key file failed, %s", err)
	}

	privateKey, err = asym.PrivateKeyFromPEM(userKeyBytes, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("parse user key file to privateKey obj failed, %s", err)
	}
	return
}
