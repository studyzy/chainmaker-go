/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package certmgr

import (
	"bytes"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
	"time"

	"chainmaker.org/chainmaker-go/vm/native/chainconfigmgr"
	configPb "chainmaker.org/chainmaker/pb-go/v2/config"

	pbac "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"

	"chainmaker.org/chainmaker-go/vm/native/common"
	bcx509 "chainmaker.org/chainmaker/common/v2/crypto/x509"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/utils/v2"
	"github.com/gogo/protobuf/proto"
)

const (
	paramNameCertHashes = "cert_hashes"
	paramNameCerts      = "certs"
	paramNameCertCrl    = "cert_crl"
)

type CertManageContract struct {
	methods map[string]common.ContractFunc
	log     protocol.Logger
}

func NewCertManageContract(log protocol.Logger) *CertManageContract {
	return &CertManageContract{
		log:     log,
		methods: registerCertManageContractMethods(log),
	}
}

func (c *CertManageContract) GetMethod(methodName string) common.ContractFunc {
	return c.methods[methodName]
}

func registerCertManageContractMethods(log protocol.Logger) map[string]common.ContractFunc {
	methodMap := make(map[string]common.ContractFunc, 64)
	// cert manager
	certManageRuntime := &CertManageRuntime{log: log}

	methodMap[syscontract.CertManageFunction_CERT_ADD.String()] = certManageRuntime.Add
	methodMap[syscontract.CertManageFunction_CERTS_DELETE.String()] = certManageRuntime.Delete
	methodMap[syscontract.CertManageFunction_CERTS_FREEZE.String()] = certManageRuntime.Freeze
	methodMap[syscontract.CertManageFunction_CERTS_UNFREEZE.String()] = certManageRuntime.Unfreeze
	methodMap[syscontract.CertManageFunction_CERTS_REVOKE.String()] = certManageRuntime.Revoke
	// query
	methodMap[syscontract.CertManageFunction_CERTS_QUERY.String()] = certManageRuntime.Query
	return methodMap
}

type CertManageRuntime struct {
	log protocol.Logger
}

// Add cert add
func (r *CertManageRuntime) Add(txSimContext protocol.TxSimContext, _ map[string][]byte) (
	result []byte, err error) {

	tx := txSimContext.GetTx()
	sender := tx.Sender
	memberInfo := sender.Signer.GetMemberInfo()

	ac, err := txSimContext.GetAccessControl()
	if err != nil {
		r.log.Errorf("txSimContext.GetAccessControl failed, err: %s", err.Error())
		return nil, err
	}

	hashType := ac.GetHashAlg()
	certHash, err := utils.GetCertificateIdHex(memberInfo, hashType)
	if err != nil {
		r.log.Errorf("get certHash failed, err: %s", err.Error())
		return nil, err
	}

	err = txSimContext.Put(syscontract.SystemContract_CERT_MANAGE.String(), []byte(certHash), memberInfo)
	if err != nil {
		r.log.Errorf("certManage add cert failed, err: %s", err.Error())
		return nil, err
	}

	r.log.Infof("certManage add cert success certHash[%s] memberInfo[%s]", certHash, string(memberInfo))
	return []byte(certHash), nil
}

// Delete cert delete
func (r *CertManageRuntime) Delete(txSimContext protocol.TxSimContext, params map[string][]byte) (
	result []byte, err error) {

	// verify params
	certHashesStr := string(params[paramNameCertHashes])

	if utils.IsAnyBlank(certHashesStr) {
		err = fmt.Errorf("%s, delete cert require param [%s] not found", common.ErrParams.Error(), paramNameCertHashes)
		r.log.Error(err)
		return nil, err
	}

	certHashes := strings.Split(certHashesStr, ",")
	for _, certHash := range certHashes {
		bytes, err := txSimContext.Get(syscontract.SystemContract_CERT_MANAGE.String(), []byte(certHash))
		if err != nil {
			r.log.Errorf("certManage delete the certHash failed, certHash[%s], err: %s", certHash, err.Error())
			return nil, err
		}

		if len(bytes) == 0 {
			msg := fmt.Sprintf(
				"certManage delete the certHash failed, certHash[%s], err: certHash is not exist", certHash)
			r.log.Error(msg)
			return nil, errors.New(msg)
		}

		err = txSimContext.Del(syscontract.SystemContract_CERT_MANAGE.String(), []byte(certHash))
		if err != nil {
			r.log.Errorf("certManage txSimContext.Del failed, certHash[%s] err: %s", certHash, err.Error())
			return nil, err
		}
	}

	r.log.Infof("certManage delete success certHashes[%s]", certHashesStr)
	return []byte("Success"), nil
}

// Query certs query
func (r *CertManageRuntime) Query(txSimContext protocol.TxSimContext, params map[string][]byte) ([]byte, error) {

	// verify params
	certHashesStr := string(params[paramNameCertHashes])

	if utils.IsAnyBlank(certHashesStr) {
		err := fmt.Errorf("%s, query cert require param [%s] not found", common.ErrParams.Error(), paramNameCertHashes)
		r.log.Error(err)
		return nil, err
	}

	certHashes := strings.Split(certHashesStr, ",")
	certInfos := make([]*commonPb.CertInfo, len(certHashes))
	for i, certHash := range certHashes {
		certBytes, err := txSimContext.Get(syscontract.SystemContract_CERT_MANAGE.String(), []byte(certHash))
		if err != nil {
			r.log.Errorf("certManage delete the certHash failed, certHash[%s] err: %s", certHash, err.Error())
			return nil, err
		}

		certInfos[i] = &commonPb.CertInfo{
			Hash: certHash,
			Cert: certBytes,
		}
	}

	c := &commonPb.CertInfos{}
	c.CertInfos = certInfos
	certBytes, err := proto.Marshal(c)
	if err != nil {
		r.log.Errorf("certManage query proto.Marshal(c) err certHash[%s] err", certHashesStr, err)
		return nil, err
	}

	r.log.Infof("certManage query success certHashes[%s]", certHashesStr)
	return certBytes, nil
}

// Freeze certs
func (r *CertManageRuntime) Freeze(txSimContext protocol.TxSimContext, params map[string][]byte) ([]byte, error) {
	// verify params
	changed := false

	hashType, freezeKeyArray, err := r.getFreezeKeyArray(txSimContext)
	if err != nil {
		return nil, err
	}

	// the full cert
	var certFullHashes bytes.Buffer
	certsStr := string(params[paramNameCerts])

	if utils.IsAnyBlank(certsStr) {
		err = fmt.Errorf("%s, freeze cert require param [%s] not found", common.ErrParams.Error(), paramNameCerts)
		r.log.Error(err)
		return nil, err
	}

	config, _ := chainconfigmgr.GetChainConfig(txSimContext)

	certs := strings.Split(certsStr, ",")

	for _, cert := range certs {
		if msg := r.checkCert(cert, config.TrustRoots); msg != nil {
			r.log.Warnf("checkCert failed, err: %s", msg)
			return nil, msg
		}
		certHash, err1 := utils.GetCertificateIdHex([]byte(cert), hashType)
		if err1 != nil {
			r.log.Warnf("utils.GetCertificateIdHex failed, err: %s", err1.Error())
			return nil, err1
		}
		certHashKey := protocol.CertFreezeKeyPrefix + certHash
		certHashBytes, err1 := txSimContext.Get(syscontract.SystemContract_CERT_MANAGE.String(), []byte(certHashKey))
		if err1 != nil {
			r.log.Warnf("txSimContext get certHashKey certHashKey[%s], err:", certHashKey, err1.Error())
			return nil, err1
		}

		if len(certHashBytes) > 0 {
			// the certHashKey is exist
			msg := fmt.Errorf("the certHashKey is exist certHashKey[%s]", certHashKey)
			r.log.Warn(msg)
			return nil, msg
		}

		err = txSimContext.Put(syscontract.SystemContract_CERT_MANAGE.String(), []byte(certHashKey), []byte(cert))
		if err != nil {
			r.log.Warnf("txSimContext.Put err, err: %s", err.Error())
			return nil, err
		}

		// add the certHashKey
		freezeKeyArray = append(freezeKeyArray, certHashKey)
		certFullHashes.WriteString(certHash)
		certFullHashes.WriteString(",")
		changed = true
	}

	if !changed {
		r.log.Error(common.ErrParams)
		return nil, common.ErrParams
	}

	marshal, err := json.Marshal(freezeKeyArray)
	if err != nil {
		r.log.Errorf("freezeKeyArray err: ", err.Error())
		return nil, err
	}
	err = txSimContext.Put(syscontract.SystemContract_CERT_MANAGE.String(), []byte(protocol.CertFreezeKey), marshal)
	if err != nil {
		r.log.Errorf("txSimContext put CERT_FREEZE_KEY err ", err.Error())
		return nil, err
	}

	certHashes := strings.TrimRight(certFullHashes.String(), ",")

	r.log.Infof("certManage freeze success certHashes[%s]", certHashes)
	return []byte(certHashes), nil
}

// Unfreeze certs unfreeze
func (r *CertManageRuntime) Unfreeze(txSimContext protocol.TxSimContext, params map[string][]byte) (
	[]byte, error) {
	// verify params
	changed := false

	hashType, freezeKeyArray, err := r.getFreezeKeyArray(txSimContext)
	if err != nil {
		return nil, err
	}

	if len(freezeKeyArray) == 0 {
		r.log.Errorf("no cert need to unfreeze")
		return nil, errors.New("no cert need to unfreeze")
	}

	// the full cert
	certFullHashes := &bytes.Buffer{}
	certsStr := string(params[paramNameCerts])
	certHashesStr := string(params[paramNameCertHashes])

	if utils.IsAllBlank(certsStr, certHashesStr) {
		err = fmt.Errorf("%s, unfreeze cert require param [%s or %s] not found",
			common.ErrParams.Error(), paramNameCerts, paramNameCertHashes)
		r.log.Error(err)
		return nil, err
	}

	config, _ := chainconfigmgr.GetChainConfig(txSimContext)
	certs := strings.Split(certsStr, ",")
	for _, cert := range certs {
		if msg := r.checkCert(cert, config.TrustRoots); msg != nil {
			return nil, msg
		}
		if len(cert) == 0 {
			continue
		}
		certHash, err1 := utils.GetCertificateIdHex([]byte(cert), hashType)
		if err1 != nil {
			r.log.Errorf("GetCertificateIdHex failed, err: ", err1.Error())
			continue
		}
		freezeKeyArray, changed = r.recoverFrozenCert(txSimContext, certHash, freezeKeyArray, certFullHashes, changed)
	}

	certHashes := strings.Split(certHashesStr, ",")
	for _, certHash := range certHashes {
		if len(certHashes) == 0 {
			continue
		}
		freezeKeyArray, changed = r.recoverFrozenCert(txSimContext, certHash, freezeKeyArray, certFullHashes, changed)
	}

	if !changed {
		r.log.Error(common.ErrParams)
		return nil, common.ErrParams
	}

	marshal, err := json.Marshal(freezeKeyArray)
	if err != nil {
		r.log.Errorf("freezeKeyArray err: ", err.Error())
		return nil, err
	}
	err = txSimContext.Put(syscontract.SystemContract_CERT_MANAGE.String(), []byte(protocol.CertFreezeKey), marshal)
	if err != nil {
		r.log.Errorf("txSimContext put CERT_FREEZE_KEY err: ", err.Error())
		return nil, err
	}

	certHasheStr := strings.TrimRight(certFullHashes.String(), ",")
	r.log.Infof("certManage unfreeze success certHashes[%s]", certHasheStr)
	return []byte("Success"), nil
}

// Revoke certs revocation
func (r *CertManageRuntime) Revoke(txSimContext protocol.TxSimContext, params map[string][]byte) (
	[]byte, error) {

	// verify params
	changed := false

	crlStr, ok := params[paramNameCertCrl]
	if !ok {
		err := fmt.Errorf("certManage cert revocation params err,cert_cerl is empty")
		r.log.Error(err.Error())
		return nil, err
	}
	ac, err := txSimContext.GetAccessControl()
	if err != nil {
		r.log.Errorf("certManage txSimContext.GetOrganization failed, err: ", err.Error())
		return nil, err
	}
	_, err = ac.VerifyRelatedMaterial(pbac.VerifyType_CRL, crlStr)
	if err != nil {
		r.log.Errorf("certManage validate crl failed err: ", err.Error())
		return nil, err
	}

	crlPEM, rest := pem.Decode(crlStr)

	var crls []*pkix.CertificateList

	for crlPEM != nil {
		var crl *pkix.CertificateList
		crl, err = x509.ParseCRL(crlPEM.Bytes)
		if err != nil {
			r.log.Errorf("certManage parse crl failed err: ", err.Error())
			return nil, err
		}
		crlPEM, rest = pem.Decode(rest)
		crls = append(crls, crl)
	}

	crlBytes, err := txSimContext.Get(syscontract.SystemContract_CERT_MANAGE.String(), []byte(protocol.CertRevokeKey))
	if err != nil {
		r.log.Errorf("get certManage crlList fail err: ", err.Error())
		return nil, fmt.Errorf("get certManage crlList failed, err: %s", err)
	}

	crlKeyList := make([]string, 0)
	if len(crlBytes) > 0 {
		err = json.Unmarshal(crlBytes, &crlKeyList)
		if err != nil {
			r.log.Errorf("certManage unmarshal crl list err: ", err.Error())
			return nil, errors.New("unmarshal crl list err")
		}
	}

	var crlResult bytes.Buffer
	for _, crtList := range crls {
		aki, err1 := getAKI(crtList)
		if err1 != nil {
			r.log.Errorf("certManage getAKI err: ", err1.Error())
			continue
		}

		key := fmt.Sprintf("%s%s", protocol.CertRevokeKeyPrefix, hex.EncodeToString(aki))
		crtListBytes, err1 := asn1.Marshal(*crtList)
		if err1 != nil {
			r.log.Errorf("certManage marshal crt list err: ", err1.Error())
			continue
		}

		existed := false
		crtListBytes1, err1 := txSimContext.Get(syscontract.SystemContract_CERT_MANAGE.String(), []byte(key))
		if err1 != nil {
			r.log.Warnf("certManage txSimContext crtList err: ", err1.Error())
			continue
		}

		if len(crtListBytes1) > 0 {
			existed = true
		}

		// to pem bytes
		toMemory := pem.EncodeToMemory(&pem.Block{
			Type:    "crl",
			Headers: nil,
			Bytes:   crtListBytes,
		})

		err = txSimContext.Put(syscontract.SystemContract_CERT_MANAGE.String(), []byte(key), toMemory)
		if err != nil {
			r.log.Errorf("certManage save crl certs err: ", err.Error())
			return nil, err
		}

		if !existed {
			// add key to array
			crlKeyList = append(crlKeyList, key)
		}

		crlResult.WriteString(key + ",")
		changed = true
	}

	if !changed {
		r.log.Error(common.ErrParams)
		return nil, common.ErrParams
	}

	crlBytesResult, err := json.Marshal(crlKeyList)
	if err != nil {
		r.log.Errorf("certManage marshal crlKeyList err: ", err.Error())
		return nil, err
	}
	err = txSimContext.Put(syscontract.SystemContract_CERT_MANAGE.String(), []byte(protocol.CertRevokeKey), crlBytesResult)
	if err != nil {
		r.log.Errorf("certManage txSimContext put CertRevokeKey err: ", err.Error())
		return nil, err
	}
	crlResultStr := strings.TrimRight(crlResult.String(), ",")
	r.log.Infof("certManage revocation success crlResult[%s]", crlResultStr)
	return []byte(crlResultStr), nil
}

func getAKI(crl *pkix.CertificateList) (aki []byte, err error) {
	aki, _, err = bcx509.GetAKIFromExtensions(crl.TBSCertList.Extensions)
	if err != nil {
		return nil, fmt.Errorf("fail to get AKI of CRL [%s]: %v", crl.TBSCertList.Issuer.String(), err)
	}
	return aki, nil
}

func (r *CertManageRuntime) getFreezeKeyArray(txSimContext protocol.TxSimContext) (string, []string, error) {
	ac, err := txSimContext.GetAccessControl()
	if err != nil {
		r.log.Errorf("txSimContext.GetAccessControl failed, err: ", err.Error())
		return "", nil, err
	}
	hashType := ac.GetHashAlg()

	// the freeze key array
	freezeKeyArray := make([]string, 0)
	freezeKeyArrayBytes, err := txSimContext.Get(syscontract.SystemContract_CERT_MANAGE.String(),
		[]byte(protocol.CertFreezeKey))
	if err != nil {
		r.log.Errorf("txSimContext get CERT_FREEZE_KEY err: ", err.Error())
		return "", nil, err
	}

	if len(freezeKeyArrayBytes) > 0 {
		err := json.Unmarshal(freezeKeyArrayBytes, &freezeKeyArray)
		if err != nil {
			r.log.Errorf("unmarshal freeze key array err: ", err.Error())
			return "", nil, err
		}
	}
	return hashType, freezeKeyArray, nil
}

func (r *CertManageRuntime) recoverFrozenCert(txSimContext protocol.TxSimContext, certHash string,
	freezeKeyArray []string, certFullHashes *bytes.Buffer, changed bool) ([]string, bool) {
	certHashKey := protocol.CertFreezeKeyPrefix + certHash
	certHashBytes, err := txSimContext.Get(syscontract.SystemContract_CERT_MANAGE.String(), []byte(certHashKey))
	if err != nil {
		r.log.Warnf("txSimContext get certHashKey err certHashKey[%s] err: ", certHashKey, err.Error())
		return nil, changed
	}

	if len(certHashBytes) == 0 {
		// the certHashKey is not exist
		r.log.Debugf("the certHashKey is not exist certHashKey[%s]", certHashKey)
		return nil, changed
	}

	err = txSimContext.Del(syscontract.SystemContract_CERT_MANAGE.String(), []byte(certHashKey))
	if err != nil {
		r.log.Warnf("certManage unfreeze txSimContext.Del failed, certHash[%s] err:%s", certHash, err.Error())
		return nil, changed
	}

	for i := 0; i < len(freezeKeyArray); i++ {
		if strings.EqualFold(freezeKeyArray[i], certHashKey) {
			freezeKeyArray = append(freezeKeyArray[:i], freezeKeyArray[i+1:]...)
			certFullHashes.WriteString(certHash)
			certFullHashes.WriteString(",")
			changed = true
			break
		}
	}
	return freezeKeyArray, changed
}

func (r *CertManageRuntime) checkCert(cert string, trustRoots []*configPb.TrustRootConfig) error {
	c, err := utils.ParseCert([]byte(cert))
	if err != nil {
		return err
	}
	if c.IsCA {
		return errors.New("can not freeze/unfreeze root certificate")
	}

	// 判断是否是该ca签发的证书
	caPool := bcx509.NewCertPool()
	for _, root := range trustRoots {
		for _, s := range root.Root {
			pemBlock, rest := pem.Decode([]byte(s))
			for pemBlock != nil {
				cert, _ := bcx509.ParseCertificate(pemBlock.Bytes)
				caPool.AddCert(cert)
				pemBlock, rest = pem.Decode(rest)
			}
		}
	}
	certChain, err := c.Verify(bcx509.VerifyOptions{
		Intermediates:             caPool,
		Roots:                     caPool,
		CurrentTime:               time.Time{},
		KeyUsages:                 []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
		MaxConstraintComparisions: 0,
	})
	if err != nil {
		r.log.Warn(err)
		return err
	}
	if len(certChain) > 0 && len(certChain[0]) > 0 {
		return nil
	}
	return errors.New("the cert is not in trust root")
}
