/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package protocol

import (
	"chainmaker.org/chainmaker-go/common/crypto/x509"
	pbac "chainmaker.org/chainmaker-go/pb/protogo/accesscontrol"
	"chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/pb/protogo/config"
	"crypto/x509/pkix"
)

const (
	ConfigNameOrgId = "org_id"
	ConfigNameRoot  = "root"

	CertFreezeKey       = "CERT_FREEZE"
	CertFreezeKeyPrefix = "freeze_"
	CertRevokeKey       = "CERT_CRL"
	CertRevokeKeyPrefix = "c_"

	// fine-grained resource id for different policies
	ResourceNameUnknown          = "UNKNOWN"
	ResourceNameReadData         = "READ"
	ResourceNameWriteData        = "WRITE"
	ResourceNameP2p              = "P2P"
	ResourceNameConsensusNode    = "CONSENSUS"
	ResourceNameAdmin            = "ADMIN"
	ResourceNameUpdateConfig     = "CONFIG"
	ResourceNameUpdateSelfConfig = "SELF_CONFIG"
	ResourceNameAllTest          = "ALL_TEST"

	ResourceNameTxQuery    = "query"
	ResourceNameTxTransact = "transaction"

	RoleAdmin         Role = "ADMIN"
	RoleClient        Role = "CLIENT"
	RoleConsensusNode Role = "CONSENSUS"
	RoleCommonNode    Role = "COMMON"

	RuleMajority  Rule = "MAJORITY"
	RuleAll       Rule = "ALL"
	RuleAny       Rule = "ANY"
	RuleSelf      Rule = "SELF"
	RuleForbidden Rule = "FORBIDDEN"
	RuleDelete    Rule = "DELETE"
)

// Role for members in an organization
type Role string

// Keywords of authentication rules
type Rule string

const ()

// Principal contains all information related to one time verification
type Principal interface {
	// GetResourceName returns resource name of the verification
	GetResourceName() string

	// GetEndorsement returns all endorsements (signatures) of the verification
	GetEndorsement() []*common.EndorsementEntry

	// GetMessage returns signing data of the verification
	GetMessage() []byte

	// GetTargetOrgId returns target organization id of the verification if the verification is for a specific organization
	GetTargetOrgId() string
}

// AccessControlProvider manages policies and principals.
type AccessControlProvider interface {
	MemberDeserializer

	// GetHashAlg return hash algorithm the access control provider uses
	GetHashAlg() string

	// ValidateResourcePolicy checks whether the given resource policy is valid
	ValidateResourcePolicy(resourcePolicy *config.ResourcePolicy) bool

	// LookUpResourceNameByTxType returns resource name corresponding to the tx type
	LookUpResourceNameByTxType(txType common.TxType) (string, error)

	// CreatePrincipal creates a principal for one time authentication
	CreatePrincipal(resourceName string, endorsements []*common.EndorsementEntry, message []byte) (Principal, error)

	// CreatePrincipalForTargetOrg creates a principal for "SELF" type policy,
	// which needs to convert SELF to a sepecific organization id in one authentication
	CreatePrincipalForTargetOrg(resourceName string, endorsements []*common.EndorsementEntry, message []byte, targetOrgId string) (Principal, error)

	// GetValidEndorsements filters all endorsement entries and returns all valid ones
	GetValidEndorsements(principal Principal) ([]*common.EndorsementEntry, error)

	// VerifyPrincipal verifies if the policy for the resource is met
	VerifyPrincipal(principal Principal) (bool, error)

	// ValidateCRL validates whether the CRL is issued by a trusted CA
	ValidateCRL(crl []byte) ([]*pkix.CertificateList, error)

	// IsCertRevoked verify whether cert chain is revoked by a trusted CA.
	IsCertRevoked(certChain []*x509.Certificate) bool

	// GetLocalOrgId returns local organization id
	GetLocalOrgId() string

	// GetLocalSigningMember returns local SigningMember
	GetLocalSigningMember() SigningMember

	// NewMemberFromCertPem creates a member from cert pem
	NewMemberFromCertPem(orgId, certPEM string) (Member, error)

	// NewMemberFromProto creates a member from SerializedMember
	NewMemberFromProto(serializedMember *pbac.SerializedMember) (Member, error)

	// NewSigningMemberFromCertFile creates a signing member from private key and cert files
	NewSigningMemberFromCertFile(orgId, prvKeyFile, password, certFile string) (SigningMember, error)

	// NewSigningMember creates a signing member from existing member
	NewSigningMember(member Member, privateKeyPem, password string) (SigningMember, error)
}

// MemberDeserializer interface for members
type MemberDeserializer interface {
	// DeserializeMember converts bytes to Member
	DeserializeMember(serializedMember []byte) (Member, error)
}

// Member is the identity of a node or user.
type Member interface {
	// GetMemberId returns the identity of this member
	GetMemberId() string

	// GetOrgId returns the organization id which this member belongs to
	GetOrgId() string

	// GetRole returns roles of this member
	GetRole() []Role

	// GetSKI returns SKI for certificate mode or Public key PEM for pk mode
	GetSKI() []byte

	// GetCertificate returns certificate object.
	// If in public key mode, return a certificate which contains public key object in PublicKey field.
	GetCertificate() (*x509.Certificate, error)

	// Verify verifies a signature over some message using this member
	Verify(hashType string, msg []byte, sig []byte) error

	// Serialize converts member to bytes
	Serialize(isFullCert bool) ([]byte, error)

	// GetSerializedMember returns SerializedMember
	GetSerializedMember(isFullCert bool) (*pbac.SerializedMember, error)
}

type SigningMember interface {
	// Extends Member interface
	Member

	// Sign signs the message with the given hash type and returns signature bytes
	Sign(hashType string, msg []byte) ([]byte, error)
}
