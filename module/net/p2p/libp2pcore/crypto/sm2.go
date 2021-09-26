package crypto

import (
	"crypto/rand"
	"encoding/asn1"
	"math/big"

	"github.com/tjfoc/gmsm/sm2"
	tjx509 "github.com/tjfoc/gmsm/x509"

	pb "github.com/libp2p/go-libp2p-core/crypto/pb"

	sha256 "github.com/minio/sha256-simd"
)

var _ PrivKey = (*SM2PrivateKey)(nil)

// SM2PrivateKey is an implementation of an SM2 private key
type SM2PrivateKey struct {
	priv *sm2.PrivateKey
}

var _ PubKey = (*SM2PublicKey)(nil)

// SM2PublicKey is an implementation of an SM2 public key
type SM2PublicKey struct {
	pub *sm2.PublicKey
}

// NewSM2PublicKey create a SM2PublicKey with sm2.PublicKey.
func NewSM2PublicKey(pub *sm2.PublicKey) *SM2PublicKey {
	return &SM2PublicKey{pub: pub}
}

// ECDSASig holds the r and s values of an ECDSA signature
type SM2Sig struct {
	R, S *big.Int
}

// GenerateSM2KeyPair generates a new sm2 private and public key
func GenerateSM2KeyPair() (PrivKey, PubKey, error) {
	priv, err := sm2.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	return &SM2PrivateKey{priv}, &SM2PublicKey{&priv.PublicKey}, nil
}

// SM2KeyPairFromKey generates a new SM2 private and public key from an input private key
func SM2KeyPairFromKey(priv *sm2.PrivateKey) (PrivKey, PubKey, error) {
	if priv == nil {
		return nil, nil, ErrNilPrivateKey
	}
	return &SM2PrivateKey{priv}, &SM2PublicKey{&priv.PublicKey}, nil
}

// MarshalSM2PrivateKey returns x509 bytes from a private key
func MarshalSM2PrivateKey(ePriv SM2PrivateKey) ([]byte, error) {
	return tjx509.MarshalSm2UnecryptedPrivateKey(ePriv.priv)
}

// MarshalSM2PublicKey returns x509 bytes from a public key
func MarshalSM2PublicKey(ePub SM2PublicKey) ([]byte, error) {
	return tjx509.MarshalSm2PublicKey(ePub.pub)
}

// UnmarshalSM2PrivateKey returns a private key from x509 bytes
func UnmarshalSM2PrivateKey(data []byte) (PrivKey, error) {
	priv, err := tjx509.ParsePKCS8UnecryptedPrivateKey(data)
	if err != nil {
		return nil, err
	}
	return &SM2PrivateKey{priv}, nil
}

// UnmarshalSM2PublicKey returns the public key from x509 bytes
func UnmarshalSM2PublicKey(data []byte) (PubKey, error) {
	pub, err := tjx509.ParseSm2PublicKey(data)
	if err != nil {
		return nil, err
	}
	return &SM2PublicKey{pub}, nil
}

// Bytes returns the private key as protobuf bytes
func (ePriv *SM2PrivateKey) Bytes() ([]byte, error) {
	return MarshalPrivateKey(ePriv)
}

// Type returns the key type
func (ePriv *SM2PrivateKey) Type() pb.KeyType {
	return pb.KeyType_SM2
}

// Raw returns x509 bytes from a private key
func (ePriv *SM2PrivateKey) Raw() ([]byte, error) {
	return tjx509.MarshalSm2UnecryptedPrivateKey(ePriv.priv)
}

// Equals compares two private keys
func (ePriv *SM2PrivateKey) Equals(o Key) bool {
	return basicEquals(ePriv, o)
}

// Sign returns the signature of the input data
func (ePriv *SM2PrivateKey) Sign(data []byte) ([]byte, error) {
	hash := sha256.Sum256(data)
	return ePriv.priv.Sign(rand.Reader, hash[:], nil)
}

// GetPublic returns a public key
func (ePriv *SM2PrivateKey) GetPublic() PubKey {
	return &SM2PublicKey{&ePriv.priv.PublicKey}
}

// Bytes returns the public key as protobuf bytes
func (ePub *SM2PublicKey) Bytes() ([]byte, error) {
	return MarshalPublicKey(ePub)
}

// Type returns the key type
func (ePub *SM2PublicKey) Type() pb.KeyType {
	return pb.KeyType_SM2
}

// Raw returns x509 bytes from a public key
func (ePub *SM2PublicKey) Raw() ([]byte, error) {
	return tjx509.MarshalPKIXPublicKey(ePub.pub)
}

// Equals compares to public keys
func (ePub *SM2PublicKey) Equals(o Key) bool {
	return basicEquals(ePub, o)
}

// Verify compares data to a signature
func (ePub *SM2PublicKey) Verify(data, sigBytes []byte) (bool, error) {
	sig := new(SM2Sig)
	if _, err := asn1.Unmarshal(sigBytes, sig); err != nil {
		return false, err
	}
	if sig == nil {
		return false, ErrNilSig
	}

	hash := sha256.Sum256(data)

	return sm2.Verify(ePub.pub, hash[:], sig.R, sig.S), nil
}
