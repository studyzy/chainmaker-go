package native

import (
	"bytes"
	bccrypto "chainmaker.org/chainmaker-go/common/crypto"
	"chainmaker.org/chainmaker-go/common/crypto/asym"
	"chainmaker.org/chainmaker-go/common/crypto/asym/rsa"
	"chainmaker.org/chainmaker-go/common/crypto/tee"
	bcx509 "chainmaker.org/chainmaker-go/common/crypto/x509"
	"encoding/asn1"
	"encoding/pem"
	"fmt"
)

var(
	cryptoPubkeyOid = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 12, 10, 1, 1}
)

func splitProof(proof []byte) (bool, *tee.TEEProof, []byte, error) {
	challengeLen, err := tee.BinaryToUint32(proof[0:tee.KLV_LENGTH_SIZE])
	if err != nil {
		return false, nil, nil, fmt.Errorf("invalid input: %v", err)
	}
	challenge := proof[tee.KLV_LENGTH_SIZE : challengeLen+ tee.KLV_LENGTH_SIZE]

	reportLen, err := tee.BinaryToUint32(proof[challengeLen+tee.KLV_LENGTH_SIZE : challengeLen+tee.KLV_LENGTH_SIZE*2])
	if err != nil {
		return false, nil, nil, fmt.Errorf("invalid input: %v", err)
	}
	report := proof[challengeLen+tee.KLV_LENGTH_SIZE*2 : challengeLen+reportLen+tee.KLV_LENGTH_SIZE*2]

	certLen, err := tee.BinaryToUint32(proof[challengeLen+reportLen+tee.KLV_LENGTH_SIZE*2 : challengeLen+reportLen+tee.KLV_LENGTH_SIZE*3])
	if err != nil {
		return false, nil, nil, fmt.Errorf("invalid input: %v", err)
	}
	certDER := proof[challengeLen+reportLen+tee.KLV_LENGTH_SIZE*3 : challengeLen+reportLen+certLen+tee.KLV_LENGTH_SIZE*3]

	sigLen, err := tee.BinaryToUint32(proof[challengeLen+reportLen+certLen+tee.KLV_LENGTH_SIZE*3 : challengeLen+reportLen+certLen+tee.KLV_LENGTH_SIZE*4])
	if err != nil {
		return false, nil, nil, fmt.Errorf("invalid input: %v", err)
	}
	sig := proof[challengeLen+reportLen+certLen+tee.KLV_LENGTH_SIZE*4 : challengeLen+reportLen+certLen+sigLen+tee.KLV_LENGTH_SIZE*4]

	certificate, err := bcx509.ParseCertificate(certDER)
	if err != nil {
		return false, nil, nil, fmt.Errorf("fail to parse TEE certificate: %v", err)
	}

	verificationKey := certificate.PublicKey

	encryptionKeyPEM, err := bcx509.GetExtByOid(tee.OidKeyBag, certificate.Extensions)
	if err != nil {
		return false, nil, nil, fmt.Errorf("fail to get encryption key: %v", err)
	}

	//encryptionKeyBlock, _ := pem.Decode(encryptionKeyPEM)
	//if encryptionKeyBlock == nil {
	//	return false, nil, nil, fmt.Errorf("fail to decode encryption key")
	//}

	// encryptionKeyInterface, err := asym.PublicKeyFromPEM(encryptionKeyBlock.Bytes)
	encryptionKeyInterface, err := asym.PublicKeyFromPEM(encryptionKeyPEM)
	if err != nil {
		return false, nil, nil, fmt.Errorf("fail to parse TEE encryption key: %v", err)
	}

	var encryptionKey bccrypto.EncryptKey
	switch k := encryptionKeyInterface.(type) {
	case *rsa.PublicKey:
		encryptionKey = k
	default:
		return false, nil, nil, fmt.Errorf("unrecognized encryption key type")
	}

	verificationKeyPEM, err := verificationKey.String()
	if err != nil {
		return false, nil, nil, fmt.Errorf("fail to serialize verification key")
	}

	teeProof := &tee.TEEProof{
		VerificationKey:    verificationKey,
		VerificationKeyPEM: []byte(verificationKeyPEM),
		EncryptionKey:      encryptionKey,
		EncryptionKeyPEM:   encryptionKeyPEM,
		Certificate:        certificate,
		CertificateDER:     certDER,
		Report:             report,
		Challenge:          challenge,
		Signature:          sig,
	}

	msg := proof[0 : challengeLen+reportLen+certLen+tee.KLV_LENGTH_SIZE*3]

	return true, teeProof, msg, nil
}

func attestationVerify(msg []byte, proof *tee.TEEProof, certOpts bcx509.VerifyOptions, reportFromChain []byte, checkReport bool) (bool, error) {

	verificationKey := proof.VerificationKey
	sig := proof.Signature
	certificate := proof.Certificate
	report := proof.Report

	isValid, err := verificationKey.VerifyWithOpts(msg, sig, &bccrypto.SignOpts{
		Hash:         bccrypto.HASH_TYPE_SHA256,
		UID:          "",
		EncodingType: rsa.RSA_PSS,
	})
	if err != nil {
		return false, fmt.Errorf("invalid signature: %v", err)
	}
	if !isValid {
		return false, fmt.Errorf("invalid signature")
	}

	certChains, err := certificate.Verify(certOpts)
	if err != nil || certChains == nil {
		return false, fmt.Errorf("untrusted certificate: %v", err)
	}

	if checkReport {
		if !bytes.Equal(report, reportFromChain) {
			return false, fmt.Errorf("report does not match")
		}
	}

	return true, nil
}

func getPubkeyPairFromCert(pemData []byte) (verificationPubKey bccrypto.PublicKey, encryptPubKey bccrypto.PublicKey, retErr error) {

	// pem => der
	certBlock, _ := pem.Decode(pemData)
	if certBlock == nil {
		retErr = fmt.Errorf("decode pem failed, invalid certificate")
		return
	}

	// der => cert
	cert, err := bcx509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		err = fmt.Errorf("x509 parse cert failed, %s", err)
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
