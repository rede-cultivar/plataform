package federation

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
)

const rsaKeyBits = 2048

// KeyPair reúne as chaves PEM de um Actor.
type KeyPair struct {
	PublicPEM  string
	PrivatePEM string
}

// GenerateKeyPair cria um par de chaves RSA para assinatura HTTP.
func GenerateKeyPair() (*KeyPair, error) {
	key, err := rsa.GenerateKey(rand.Reader, rsaKeyBits)
	if err != nil {
		return nil, fmt.Errorf("generate actor key: %w", err)
	}

	publicDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("encode actor public key: %w", err)
	}

	return &KeyPair{
		PublicPEM:  string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicDER})),
		PrivatePEM: string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})),
	}, nil
}

// ParsePrivateKey interpreta uma chave privada RSA em PEM.
func ParsePrivateKey(value string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(value))
	if block == nil {
		return nil, fmt.Errorf("invalid actor private key PEM")
	}

	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}

	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse actor private key: %w", err)
	}

	key, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("actor private key is not RSA")
	}
	return key, nil
}

// ParsePublicKey interpreta uma chave pública RSA em PEM.
func ParsePublicKey(value string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(value))
	if block == nil {
		return nil, fmt.Errorf("invalid actor public key PEM")
	}

	if parsed, err := x509.ParsePKIXPublicKey(block.Bytes); err == nil {
		key, ok := parsed.(*rsa.PublicKey)
		if !ok {
			return nil, fmt.Errorf("actor public key is not RSA")
		}
		return key, nil
	}

	key, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse actor public key: %w", err)
	}
	return key, nil
}
