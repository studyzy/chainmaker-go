/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

// description: chainmaker-go
//
// @author: xwc1125
// @date: 2020/11/24
package main

import (
	commonPb "chainmaker.org/chainmaker/pb-go/common"
	"encoding/json"
	"fmt"
	"io/ioutil"

	"chainmaker.org/chainmaker-go/utils"
	"github.com/gogo/protobuf/proto"
	"github.com/spf13/cobra"
)

var (
	certHashes string
	certs      string
	certCrl    string
)

const (
	certHash   = "cert_hashes"
	certCrlStr = "cert_crl"
)

func CertManageAddCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "certManageAdd",
		Short: "Cert manage add",
		Long:  "Cert manage add",
		RunE: func(_ *cobra.Command, _ []string) error {
			return certAdd()
		},
	}

	return cmd
}

func CertManageDeleteCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "certManageDelete",
		Short: "Cert manage delete",
		Long:  "Cert manage delete",
		RunE: func(_ *cobra.Command, _ []string) error {
			return certDelete()
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&certHashes, certHash, "", "cert_hashes,use `,` separate multiple hashes")

	return cmd
}

func CertManageQueryCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "certManageQuery",
		Short: "Cert manage query",
		Long:  "Cert manage query",
		RunE: func(_ *cobra.Command, _ []string) error {
			return certQuery()
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&certHashes, certHash, "", "cert_hashes,use `,` separate multiple hashes")
	flags.StringVar(&hashAlgo, "hash-algorithm", "SHA256", "hash algorithm set in chain configuration")

	return cmd
}

func CertManageFrozenCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "certManageFrozen",
		Short: "Cert manage frozen",
		Long:  "Cert manage frozen(org-ids,admin-sign-keys,admin-sign-crts,certs)",
		RunE: func(_ *cobra.Command, _ []string) error {
			return certFrozen()
		},
	}
	flags := cmd.Flags()
	flags.StringVar(&certs, "certs", "", "certs, use `,` separate multiple certs")
	return cmd
}

func CertManageUnfrozenCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "certManageUnfrozen",
		Short: "Cert manage unfrozen",
		Long:  "Cert manage unfrozen(org-ids,admin-sign-keys,admin-sign-crts,certs,cert_hashes)",
		RunE: func(_ *cobra.Command, _ []string) error {
			return certUnfrozen()
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&certs, "certs", "", "certs, use `,` separate multiple certs")
	flags.StringVar(&certHashes, certHash, "", "cert_hashes, use `,` separate multiple hashes")
	return cmd
}

func CertManageRevocationCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "certManageRevocation",
		Short: "Cert manage revocation",
		Long:  "Cert manage revocation(org-ids,admin-sign-keys,admin-sign-crts,cert_crl)",
		RunE: func(_ *cobra.Command, _ []string) error {
			return certRevocation()
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&certCrl, certCrlStr, "", certCrlStr)
	return cmd
}

func certAdd() error {
	// 构造Payload
	txId := utils.GetRandTxId()
	pairs := make([]*commonPb.KeyValuePair, 0)

	payload := &commonPb.SystemContractPayload{
		ChainId:      chainId,
		ContractName: commonPb.ContractName_SYSTEM_CONTRACT_CERT_MANAGE.String(),
		Method:       commonPb.CertManageFunction_CERT_ADD.String(),
		Parameters:   pairs,
		Sequence:     seq,
	}
	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		return err
	}
	resp, err := proposalRequest(sk3, client, commonPb.TxType_INVOKE_SYSTEM_CONTRACT,
		chainId, txId, payloadBytes)
	if err != nil {
		return err
	}

	file, err := ioutil.ReadFile(userCrtPath)
	certId, err := utils.GetCertificateIdHex(file, hashAlgo)
	if err != nil {
		return err
	}
	result := &Result{
		Code:      resp.Code,
		Message:   resp.Message,
		TxId:      txId,
		ShortCert: certId,
	}
	bytes, err := json.Marshal(result)
	if err != nil {
		return err
	}
	fmt.Println(string(bytes))

	return nil
}

func certDelete() error {
	// 构造Payload
	txId := utils.GetRandTxId()
	pairs := make([]*commonPb.KeyValuePair, 0)
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   certHash,
		Value: certHashes,
	})

	payload := &commonPb.SystemContractPayload{
		ChainId:      chainId,
		ContractName: commonPb.ContractName_SYSTEM_CONTRACT_CERT_MANAGE.String(),
		Method:       commonPb.CertManageFunction_CERTS_DELETE.String(),
		Parameters:   pairs,
		Sequence:     seq,
	}
	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := proposalRequest(sk3, client, commonPb.TxType_INVOKE_SYSTEM_CONTRACT,
		chainId, txId, payloadBytes)
	if err != nil {
		return err
	}

	result := &Result{
		Code:    resp.Code,
		Message: resp.Message,
		TxId:    txId,
	}
	bytes, err := json.Marshal(result)
	if err != nil {
		return err
	}
	fmt.Println(string(bytes))

	return nil
}

func certQuery() error {
	// 构造Payload
	pairs := make([]*commonPb.KeyValuePair, 0)
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   certHash,
		Value: certHashes,
	})
	payloadBytes, err := constructPayload(commonPb.ContractName_SYSTEM_CONTRACT_CERT_MANAGE.String(), commonPb.CertManageFunction_CERTS_QUERY.String(), pairs)
	if err != nil {
		return err
	}
	resp, err = proposalRequest(sk3, client, commonPb.TxType_QUERY_SYSTEM_CONTRACT,
		chainId, txId, payloadBytes)
	if err != nil {
		return err
	}

	certInfos := &commonPb.CertInfos{}
	err = proto.Unmarshal(resp.ContractResult.Result, certInfos)
	if err != nil {
		return err
	}

	bytes, err := json.Marshal(certInfos)
	if err != nil {
		return err
	}
	fmt.Println(string(bytes))

	return nil
}

func certFrozen() error {
	// 构造Payload
	txId := utils.GetRandTxId()
	pairs := make([]*commonPb.KeyValuePair, 0)
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "certs",
		Value: certs,
	})

	resp, txId, err := configUpdateRequest(sk3, client, &InvokerMsg{txType: commonPb.TxType_INVOKE_SYSTEM_CONTRACT, chainId: chainId,
		contractName: commonPb.ContractName_SYSTEM_CONTRACT_CERT_MANAGE.String(), method: commonPb.CertManageFunction_CERTS_FREEZE.String(), pairs: pairs, oldSeq: seq})
	if err != nil {
		return err
	}

	if err != nil {
		return err
	}

	file, err := ioutil.ReadFile(userCrtPath)
	certId, err := utils.GetCertificateIdHex(file, hashAlgo)
	if err != nil {
		return err
	}
	result := &Result{
		Code:      resp.Code,
		Message:   resp.Message,
		TxId:      txId,
		ShortCert: certId,
	}
	bytes, err := json.Marshal(result)
	if err != nil {
		return err
	}
	fmt.Println(string(bytes))

	return nil
}

func certUnfrozen() error {
	// 构造Payload
	txId := utils.GetRandTxId()
	pairs := make([]*commonPb.KeyValuePair, 0)
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "certs",
		Value: certs,
	})
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   certHash,
		Value: certHashes,
	})

	resp, txId, err := configUpdateRequest(sk3, client, &InvokerMsg{txType: commonPb.TxType_INVOKE_SYSTEM_CONTRACT, chainId: chainId,
		contractName: commonPb.ContractName_SYSTEM_CONTRACT_CERT_MANAGE.String(), method: commonPb.CertManageFunction_CERTS_UNFREEZE.String(), pairs: pairs, oldSeq: seq})
	if err != nil {
		return err
	}
	file, err := ioutil.ReadFile(userCrtPath)
	certId, err := utils.GetCertificateIdHex(file, hashAlgo)
	if err != nil {
		return err
	}
	result := &Result{
		Code:      resp.Code,
		Message:   resp.Message,
		TxId:      txId,
		ShortCert: certId,
	}
	bytes, err := json.Marshal(result)
	if err != nil {
		return err
	}
	fmt.Println(string(bytes))

	return nil
}

func certRevocation() error {
	// 构造Payload
	txId := utils.GetRandTxId()
	pairs := make([]*commonPb.KeyValuePair, 0)
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   certCrlStr,
		Value: certCrl,
	})

	resp, txId, err := configUpdateRequest(sk3, client, &InvokerMsg{txType: commonPb.TxType_INVOKE_SYSTEM_CONTRACT, chainId: chainId,
		contractName: commonPb.ContractName_SYSTEM_CONTRACT_CERT_MANAGE.String(), method: commonPb.CertManageFunction_CERTS_REVOKE.String(), pairs: pairs, oldSeq: seq})
	if err != nil {
		return err
	}

	if err != nil {
		return err
	}

	file, err := ioutil.ReadFile(userCrtPath)
	certId, err := utils.GetCertificateIdHex(file, hashAlgo)
	if err != nil {
		return err
	}
	result := &Result{
		Code:      resp.Code,
		Message:   resp.Message,
		TxId:      txId,
		ShortCert: certId,
	}
	bytes, err := json.Marshal(result)
	if err != nil {
		return err
	}
	fmt.Println(string(bytes))

	return nil
}
