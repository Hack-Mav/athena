package ota

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test binary signing and verification
func TestSigner_SignAndVerifyBinary(t *testing.T) {
	// Generate test key pair
	privateKeyPEM, publicKeyPEM, err := GenerateKeyPair(2048)
	require.NoError(t, err)
	require.NotNil(t, privateKeyPEM)
	require.NotNil(t, publicKeyPEM)

	// Create signer
	signer, err := NewSigner(privateKeyPEM, publicKeyPEM)
	require.NoError(t, err)
	require.NotNil(t, signer)

	// Test binary data
	binaryData := []byte("test firmware binary data for signing")

	// Sign the binary
	signature, err := signer.SignBinary(binaryData)
	require.NoError(t, err)
	assert.NotEmpty(t, signature)

	// Verify the signature
	err = signer.VerifySignature(binaryData, signature)
	assert.NoError(t, err)
}

// Test signature verification with tampered data
func TestSigner_VerifySignature_TamperedData(t *testing.T) {
	privateKeyPEM, publicKeyPEM, err := GenerateKeyPair(2048)
	require.NoError(t, err)

	signer, err := NewSigner(privateKeyPEM, publicKeyPEM)
	require.NoError(t, err)

	originalData := []byte("original firmware binary data")
	tamperedData := []byte("tampered firmware binary data")

	// Sign original data
	signature, err := signer.SignBinary(originalData)
	require.NoError(t, err)

	// Try to verify with tampered data
	err = signer.VerifySignature(tamperedData, signature)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "signature verification failed")
}

// Test signature verification with invalid signature
func TestSigner_VerifySignature_InvalidSignature(t *testing.T) {
	privateKeyPEM, publicKeyPEM, err := GenerateKeyPair(2048)
	require.NoError(t, err)

	signer, err := NewSigner(privateKeyPEM, publicKeyPEM)
	require.NoError(t, err)

	binaryData := []byte("test firmware binary data")
	invalidSignature := "invalid-signature-data"

	err = signer.VerifySignature(binaryData, invalidSignature)
	assert.Error(t, err)
}

// Test signature verification with wrong key
func TestSigner_VerifySignature_WrongKey(t *testing.T) {
	// Generate first key pair
	privateKeyPEM1, _, err := GenerateKeyPair(2048)
	require.NoError(t, err)

	// Generate second key pair
	_, publicKeyPEM2, err := GenerateKeyPair(2048)
	require.NoError(t, err)

	// Create signer with first private key
	signer1, err := NewSigner(privateKeyPEM1, publicKeyPEM2)
	require.NoError(t, err)

	binaryData := []byte("test firmware binary data")

	// Sign with first key
	signature, err := signer1.SignBinary(binaryData)
	require.NoError(t, err)

	// Try to verify with second public key (should fail)
	err = signer1.VerifySignature(binaryData, signature)
	assert.Error(t, err)
}

// Test hash computation consistency
func TestComputeHash_Consistency(t *testing.T) {
	binaryData := []byte("test firmware binary data")

	hash1 := ComputeHash(binaryData)
	hash2 := ComputeHash(binaryData)

	assert.Equal(t, hash1, hash2)
	assert.NotEmpty(t, hash1)
	assert.Len(t, hash1, 64) // SHA-256 produces 64 hex characters
}

// Test hash computation uniqueness
func TestComputeHash_Uniqueness(t *testing.T) {
	data1 := []byte("firmware version 1.0.0")
	data2 := []byte("firmware version 2.0.0")

	hash1 := ComputeHash(data1)
	hash2 := ComputeHash(data2)

	assert.NotEqual(t, hash1, hash2)
}

// Test key pair generation
func TestGenerateKeyPair(t *testing.T) {
	tests := []struct {
		name string
		bits int
	}{
		{"2048 bits", 2048},
		{"4096 bits", 4096},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			privateKeyPEM, publicKeyPEM, err := GenerateKeyPair(tt.bits)
			require.NoError(t, err)
			assert.NotNil(t, privateKeyPEM)
			assert.NotNil(t, publicKeyPEM)
			assert.Contains(t, string(privateKeyPEM), "RSA PRIVATE KEY")
			assert.Contains(t, string(publicKeyPEM), "PUBLIC KEY")
		})
	}
}

// Test signer creation with invalid keys
func TestNewSigner_InvalidKeys(t *testing.T) {
	tests := []struct {
		name          string
		privateKeyPEM []byte
		publicKeyPEM  []byte
	}{
		{
			name:          "invalid private key",
			privateKeyPEM: []byte("invalid private key data"),
			publicKeyPEM:  []byte("-----BEGIN PUBLIC KEY-----\ntest\n-----END PUBLIC KEY-----"),
		},
		{
			name:          "invalid public key",
			privateKeyPEM: []byte("-----BEGIN RSA PRIVATE KEY-----\ntest\n-----END RSA PRIVATE KEY-----"),
			publicKeyPEM:  []byte("invalid public key data"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signer, err := NewSigner(tt.privateKeyPEM, tt.publicKeyPEM)
			assert.Error(t, err)
			assert.Nil(t, signer)
		})
	}
}

// Test signing without private key
func TestSigner_SignBinary_NoPrivateKey(t *testing.T) {
	signer := &Signer{
		privateKey: nil,
		publicKey:  nil,
	}

	binaryData := []byte("test data")
	signature, err := signer.SignBinary(binaryData)

	assert.Error(t, err)
	assert.Empty(t, signature)
	assert.Contains(t, err.Error(), "private key not configured")
}

// Test verification without public key
func TestSigner_VerifySignature_NoPublicKey(t *testing.T) {
	signer := &Signer{
		privateKey: nil,
		publicKey:  nil,
	}

	binaryData := []byte("test data")
	signature := "test-signature"

	err := signer.VerifySignature(binaryData, signature)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "public key not configured")
}

// Test signing large binary data
func TestSigner_SignBinary_LargeData(t *testing.T) {
	privateKeyPEM, publicKeyPEM, err := GenerateKeyPair(2048)
	require.NoError(t, err)

	signer, err := NewSigner(privateKeyPEM, publicKeyPEM)
	require.NoError(t, err)

	// Create 1MB of test data
	largeData := make([]byte, 1024*1024)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	signature, err := signer.SignBinary(largeData)
	require.NoError(t, err)
	assert.NotEmpty(t, signature)

	err = signer.VerifySignature(largeData, signature)
	assert.NoError(t, err)
}

// Test empty binary data
func TestSigner_SignBinary_EmptyData(t *testing.T) {
	privateKeyPEM, publicKeyPEM, err := GenerateKeyPair(2048)
	require.NoError(t, err)

	signer, err := NewSigner(privateKeyPEM, publicKeyPEM)
	require.NoError(t, err)

	emptyData := []byte{}

	signature, err := signer.SignBinary(emptyData)
	require.NoError(t, err)
	assert.NotEmpty(t, signature)

	err = signer.VerifySignature(emptyData, signature)
	assert.NoError(t, err)
}
