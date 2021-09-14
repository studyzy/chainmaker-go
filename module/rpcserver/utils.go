/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package rpcserver

import (
	pbac "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	cmx509 "chainmaker.org/chainmaker/common/v2/crypto/x509"
	"chainmaker.org/chainmaker/protocol/v2"
)

func createVerifyPeerCertificateFunc(
	accessControls []protocol.AccessControlProvider,
) func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	return func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
		revoked, err := isRevoked(accessControls, rawCerts, verifiedChains)
		if err != nil {
			return err
		}

		if revoked {
			return fmt.Errorf("certificate revoked")
		}

		return nil
	}
}

func createGMVerifyPeerCertificateFunc(
	accessControls []protocol.AccessControlProvider,
) func(rawCerts [][]byte, verifiedChains [][]*cmx509.Certificate) error {
	return func(rawCerts [][]byte, verifiedChains [][]*cmx509.Certificate) error {
		revoked, err := isGMRevoked(accessControls, rawCerts, verifiedChains)
		if err != nil {
			return err
		}

		if revoked {
			return fmt.Errorf("certificate revoked")
		}

		return nil
	}
}

func isRevoked(accessControls []protocol.AccessControlProvider, rawCerts [][]byte,
	verifiedChains [][]*x509.Certificate) (bool, error) {

	members := make([]*pbac.Member, 0)
	for idx := range rawCerts {
		m := &pbac.Member{
			OrgId:      "",
			MemberType: pbac.MemberType_CERT,
			MemberInfo: rawCerts[idx],
		}
		members = append(members, m)
	}

	for i := range verifiedChains {
		for j := range verifiedChains[i] {
			cert, err := cmx509.X509CertToChainMakerCert(verifiedChains[i][j])
			if err != nil {
				return false, err
			}

			certBytes := pem.EncodeToMemory(&pem.Block{
				Type:    "CERTIFICATE",
				Headers: nil,
				Bytes:   cert.Raw,
			})

			m := &pbac.Member{
				OrgId:      "",
				MemberType: pbac.MemberType_CERT,
				MemberInfo: certBytes,
			}
			members = append(members, m)
		}
	}

	return checkMemberStatusIsRevoked(accessControls, members)
}

func isGMRevoked(accessControls []protocol.AccessControlProvider, rawCerts [][]byte,
	verifiedChains [][]*cmx509.Certificate) (bool, error) {

	members := make([]*pbac.Member, 0)
	for idx := range rawCerts {
		m := &pbac.Member{
			OrgId:      "",
			MemberType: pbac.MemberType_CERT,
			MemberInfo: rawCerts[idx],
		}
		members = append(members, m)
	}

	for i := range verifiedChains {
		for j := range verifiedChains[i] {
			//cert, err := cmx509.SM2CertToChainMakerCert(verifiedChains[i][j])
			//if err != nil {
			//	return false, err
			//}
			//
			//certs = append(certs, cert)

			//certs = append(certs, verifiedChains[i][j])

			certBytes := pem.EncodeToMemory(&pem.Block{
				Type:    "CERTIFICATE",
				Headers: nil,
				Bytes:   verifiedChains[i][j].Raw,
			})

			m := &pbac.Member{
				OrgId:      "",
				MemberType: pbac.MemberType_CERT,
				MemberInfo: certBytes,
			}
			members = append(members, m)
		}
	}

	return checkMemberStatusIsRevoked(accessControls, members)
}

// ValidateMemberStatus check the status of members.
func checkMemberStatusIsRevoked(accessControls []protocol.AccessControlProvider,
	members []*pbac.Member) (bool, error) {

	var err error

	for _, ac := range accessControls {
		if ac == nil {
			return false, fmt.Errorf("ac is nil")
		}

		for _, member := range members {
			var s pbac.MemberStatus
			s, err = ac.GetMemberStatus(member)
			if err != nil {
				return false, err
			}

			if s == pbac.MemberStatus_INVALID || s == pbac.MemberStatus_FROZEN || s == pbac.MemberStatus_REVOKED {
				return true, nil
			}
		}
	}

	return false, nil
}
