package util

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
)

func GenerateRsaKey() (*rsa.PrivateKey, error) {
	return rsa.GenerateKey(rand.Reader, 2048)
}

func MustGenerateRsaKey() *rsa.PrivateKey {
	pk, err := GenerateRsaKey()
	AssertNilErr(err)
	return pk
}

func EncodeRsaPrivateKeyAsPem(key *rsa.PrivateKey) []byte {
	return pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(key),
		},
	)
}

func EncodeRsaPublicKeyAsPem(key *rsa.PublicKey) ([]byte, error) {
	data, err := x509.MarshalPKIXPublicKey(key)
	if err != nil {
		return nil, err
	}

	pem := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PUBLIC KEY",
			Bytes: data,
		},
	)
	return pem, nil
}

func DecodeRsaPrivateKeyPem(data []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New(`failed to parse PEM block`)
	}

	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

func MustDecodeRsaPrivateKeyPem(data []byte) *rsa.PrivateKey {
	key, err := DecodeRsaPrivateKeyPem(data)
	AssertNilErr(err)
	return key
}

func DecodeRsaPublicKeyPem(data []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New(`failed to parse PEM block`)
	}

	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	switch v := key.(type) {
	case *rsa.PublicKey:
		return v, nil
	default:
		return nil, errors.New(`not rsa public key`)
	}
}

func MustDecodeRsaPublicKeyPem(data []byte) *rsa.PublicKey {
	key, err := DecodeRsaPublicKeyPem(data)
	AssertNilErr(err)
	return key
}

var rsaPssOption = &rsa.PSSOptions{
	SaltLength: rsa.PSSSaltLengthAuto,
	Hash:       crypto.SHA256,
}

func RsaSignPssSha256(key *rsa.PrivateKey, data []byte) (sign []byte, err error) {
	h := rsaPssOption.Hash.New()
	h.Write(data)
	return rsa.SignPSS(rand.Reader, key, rsaPssOption.Hash, h.Sum(nil), rsaPssOption)
}

func RsaVerifyPssSha256(key *rsa.PublicKey, data, sign []byte) error {
	h := rsaPssOption.Hash.New()
	h.Write(data)
	return rsa.VerifyPSS(key, rsaPssOption.Hash, h.Sum(nil), sign, rsaPssOption)
}
