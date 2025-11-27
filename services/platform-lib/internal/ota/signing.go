package ota

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
)

// Signer handles firmware binary signing and verification
type Signer struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
}

// NewSigner creates a new Signer with the provided RSA keys
func NewSigner(privateKeyPEM, publicKeyPEM []byte) (*Signer, error) {
	// Parse private key
	privateKey, err := parsePrivateKey(privateKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	// Parse public key
	publicKey, err := parsePublicKey(publicKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	return &Signer{
		privateKey: privateKey,
		publicKey:  publicKey,
	}, nil
}

// SignBinary signs the firmware binary and returns the signature
func (s *Signer) SignBinary(binaryData []byte) (string, error) {
	if s.privateKey == nil {
		return "", fmt.Errorf("private key not configured")
	}

	// Compute SHA-256 hash of the binary
	hash := sha256.Sum256(binaryData)

	// Sign the hash using RSA-PSS
	signature, err := rsa.SignPSS(rand.Reader, s.privateKey, crypto.SHA256, hash[:], nil)
	if err != nil {
		return "", fmt.Errorf("failed to sign binary: %w", err)
	}

	// Encode signature as base64
	return base64.StdEncoding.EncodeToString(signature), nil
}

// VerifySignature verifies the signature of a firmware binary
func (s *Signer) VerifySignature(binaryData []byte, signatureBase64 string) error {
	if s.publicKey == nil {
		return fmt.Errorf("public key not configured")
	}

	// Decode signature from base64
	signature, err := base64.StdEncoding.DecodeString(signatureBase64)
	if err != nil {
		return fmt.Errorf("failed to decode signature: %w", err)
	}

	// Compute SHA-256 hash of the binary
	hash := sha256.Sum256(binaryData)

	// Verify the signature using RSA-PSS
	err = rsa.VerifyPSS(s.publicKey, crypto.SHA256, hash[:], signature, nil)
	if err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}

	return nil
}

// ComputeHash computes the SHA-256 hash of the binary data
func ComputeHash(binaryData []byte) string {
	hash := sha256.Sum256(binaryData)
	return fmt.Sprintf("%x", hash)
}

// GenerateKeyPair generates a new RSA key pair for signing
func GenerateKeyPair(bits int) (privateKeyPEM, publicKeyPEM []byte, err error) {
	// Generate private key
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	// Encode private key to PEM
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	})

	// Encode public key to PEM
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal public key: %w", err)
	}

	publicKeyPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	})

	return privateKeyPEM, publicKeyPEM, nil
}

// parsePrivateKey parses a PEM-encoded RSA private key
func parsePrivateKey(privateKeyPEM []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(privateKeyPEM)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		// Try PKCS8 format
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}
		
		rsaKey, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("key is not an RSA private key")
		}
		
		return rsaKey, nil
	}

	return privateKey, nil
}

// parsePublicKey parses a PEM-encoded RSA public key
func parsePublicKey(publicKeyPEM []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(publicKeyPEM)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	rsaKey, ok := publicKey.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("key is not an RSA public key")
	}

	return rsaKey, nil
}
