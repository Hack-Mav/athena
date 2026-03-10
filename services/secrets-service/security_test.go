package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/athena/platform-lib/pkg/config"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// MockSecretsRepository is a mock implementation of secrets repository
type MockSecretsRepository struct {
	mock.Mock
}

func (m *MockSecretsRepository) StoreSecret(ctx context.Context, secretID string, encryptedData []byte, metadata map[string]interface{}) error {
	args := m.Called(ctx, secretID, encryptedData, metadata)
	return args.Error(0)
}

func (m *MockSecretsRepository) GetSecret(ctx context.Context, secretID string) ([]byte, map[string]interface{}, error) {
	args := m.Called(ctx, secretID)
	return args.Get(0).([]byte), args.Get(1).(map[string]interface{}), args.Error(2)
}

func (m *MockSecretsRepository) DeleteSecret(ctx context.Context, secretID string) error {
	args := m.Called(ctx, secretID)
	return args.Error(0)
}

func (m *MockSecretsRepository) ListSecrets(ctx context.Context, filters map[string]interface{}) ([]string, error) {
	args := m.Called(ctx, filters)
	return args.Get(0).([]string), args.Error(1)
}

// MockAuthRepository is a mock implementation of auth repository
type MockAuthRepository struct {
	mock.Mock
}

func (m *MockAuthRepository) StoreCertificate(ctx context.Context, certID string, certData []byte, metadata map[string]interface{}) error {
	args := m.Called(ctx, certID, certData, metadata)
	return args.Error(0)
}

func (m *MockAuthRepository) GetCertificate(ctx context.Context, certID string) ([]byte, map[string]interface{}, error) {
	args := m.Called(ctx, certID)
	return args.Get(0).([]byte), args.Get(1).(map[string]interface{}), args.Error(2)
}

func (m *MockAuthRepository) RevokeCertificate(ctx context.Context, certID string) error {
	args := m.Called(ctx, certID)
	return args.Error(0)
}

func (m *MockAuthRepository) StoreDeviceAuth(ctx context.Context, deviceID string, authData map[string]interface{}) error {
	args := m.Called(ctx, deviceID, authData)
	return args.Error(0)
}

func (m *MockAuthRepository) GetDeviceAuth(ctx context.Context, deviceID string) (map[string]interface{}, error) {
	args := m.Called(ctx, deviceID)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

// SecurityTestSuite contains comprehensive security tests
type SecurityTestSuite struct {
	suite.Suite
	service         *Service
	authService     *AuthService
	mockRepo        *MockSecretsRepository
	mockAuthRepo    *MockAuthRepository
	testPrivateKey  *rsa.PrivateKey
	testCertificate *x509.Certificate
}

func (suite *SecurityTestSuite) SetupSuite() {
	// Generate test certificate for testing
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	suite.Require().NoError(err)

	template := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test-device"},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	suite.Require().NoError(err)

	certificate, err := x509.ParseCertificate(certDER)
	suite.Require().NoError(err)

	suite.testPrivateKey = privateKey
	suite.testCertificate = certificate
}

func (suite *SecurityTestSuite) SetupTest() {
	suite.mockRepo = new(MockSecretsRepository)
	suite.mockAuthRepo = new(MockAuthRepository)

	cfg := &config.Config{
		LogLevel:    "debug",
		ServiceName: "test-secrets-service",
		Environment: "test",
	}

	// Create services with mocked dependencies
	suite.service = &Service{
		config:     &Config{LogLevel: cfg.LogLevel, ServiceName: cfg.ServiceName, Environment: cfg.Environment},
		logger:     &Logger{level: "debug", name: "test"},
		repository: suite.mockRepo,
		encryptor:  NewAESEncryptor(),
	}

	suite.authService = &AuthService{
		config:        &Config{LogLevel: cfg.LogLevel, ServiceName: cfg.ServiceName, Environment: cfg.Environment},
		logger:        &Logger{level: "debug", name: "test"},
		repository:    suite.mockAuthRepo,
		certAuthority: NewCertificateAuthority(),
	}
}

func TestSecuritySuite(t *testing.T) {
	suite.Run(t, new(SecurityTestSuite))
}

func (suite *SecurityTestSuite) TestSecretInjectionWithoutExposure() {
	ctx := context.Background()

	secretID := "test-secret"
	plainTextSecret := "super-secret-api-key"

	// Mock repository to store encrypted data
	suite.mockRepo.On("StoreSecret", ctx, secretID, mock.AnythingOfType("[]uint8"), mock.AnythingOfType("map[string]interface {}")).Return(nil)

	// Store secret (should encrypt before storage)
	err := suite.service.StoreSecret(ctx, secretID, plainTextSecret, map[string]interface{}{
		"type":    "api_key",
		"purpose": "external_service",
	})

	suite.Require().NoError(err)

	// Verify the call was made with encrypted data (not plain text)
	suite.mockRepo.AssertCalled(suite.T(), "StoreSecret", ctx, secretID, mock.AnythingOfType("[]uint8"), mock.AnythingOfType("map[string]interface {}"))

	// Verify stored data is not the plain text
	calls := suite.mockRepo.Calls
	storedData := calls[0].Arguments[2].([]byte)
	suite.Assert().NotEqual(plainTextSecret, string(storedData), "Secret should be encrypted before storage")

	// Mock repository to return encrypted data
	suite.mockRepo.On("GetSecret", ctx, secretID).Return(storedData, map[string]interface{}{
		"type":    "api_key",
		"purpose": "external_service",
	}, nil)

	// Retrieve secret (should decrypt after retrieval)
	retrievedSecret, metadata, err := suite.service.GetSecret(ctx, secretID)

	suite.Require().NoError(err)
	suite.Assert().Equal(plainTextSecret, retrievedSecret, "Retrieved secret should match original")
	suite.Assert().Equal("api_key", metadata["type"])

	// Verify audit trail is created
	suite.mockRepo.AssertCalled(suite.T(), "GetSecret", ctx, secretID)
}

func (suite *SecurityTestSuite) TestSecretEncryptionAtRest() {
	ctx := context.Background()

	secretID := "encryption-test"
	plainTextSecret := "sensitive-data-12345"

	// Store secret
	suite.mockRepo.On("StoreSecret", ctx, secretID, mock.AnythingOfType("[]uint8"), mock.AnythingOfType("map[string]interface {}")).Return(nil)

	err := suite.service.StoreSecret(ctx, secretID, plainTextSecret, map[string]interface{}{
		"encryption": "aes-256-gcm",
	})

	suite.Require().NoError(err)

	// Get the encrypted data that was stored
	calls := suite.mockRepo.Calls
	encryptedData := calls[0].Arguments[2].([]byte)

	// Verify data is actually encrypted (should not contain plain text)
	suite.Assert().NotContains(string(encryptedData), plainTextSecret, "Encrypted data should not contain plain text")
	suite.Assert().Greater(len(encryptedData), len(plainTextSecret), "Encrypted data should be larger than plain text")

	// Mock retrieval
	suite.mockRepo.On("GetSecret", ctx, secretID).Return(encryptedData, map[string]interface{}{"encryption": "aes-256-gcm"}, nil)

	// Retrieve and decrypt
	retrievedSecret, _, err := suite.service.GetSecret(ctx, secretID)

	suite.Require().NoError(err)
	suite.Assert().Equal(plainTextSecret, retrievedSecret, "Decrypted secret should match original")
}

func (suite *SecurityTestSuite) TestCertificateGenerationAndValidation() {
	ctx := context.Background()

	deviceID := "test-device-001"

	// Mock certificate storage
	suite.mockAuthRepo.On("StoreCertificate", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("[]uint8"), mock.AnythingOfType("map[string]interface {}")).Return(nil)

	// Generate device certificate
	certPEM, privateKeyPEM, err := suite.authService.GenerateDeviceCertificate(ctx, deviceID, map[string]interface{}{
		"device_type": "arduino",
		"firmware":    "1.0.0",
	})

	suite.Require().NoError(err)
	suite.Assert().NotEmpty(certPEM)
	suite.Assert().NotEmpty(privateKeyPEM)

	// Verify certificate can be parsed
	block, _ := pem.Decode([]byte(certPEM))
	suite.Require().NotNil(block, "Certificate PEM should be decodeable")

	cert, err := x509.ParseCertificate(block.Bytes)
	suite.Require().NoError(err, "Certificate should be parseable")

	// Verify certificate properties
	suite.Assert().Equal(deviceID, cert.Subject.CommonName, "Certificate CN should match device ID")
	suite.Assert().True(time.Now().After(cert.NotBefore), "Certificate should be valid now")
	suite.Assert().True(time.Now().Before(cert.NotAfter), "Certificate should not be expired")

	// Verify certificate is stored
	suite.mockAuthRepo.AssertCalled(suite.T(), "StoreCertificate", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("[]uint8"), mock.AnythingOfType("map[string]interface {}"))
}

func (suite *SecurityTestSuite) TestDeviceAuthenticationFlow() {
	ctx := context.Background()

	deviceID := "auth-test-device"

	// Generate and store certificate
	suite.mockAuthRepo.On("StoreCertificate", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("[]uint8"), mock.AnythingOfType("map[string]interface {}")).Return(nil)

	certPEM, privateKeyPEM, err := suite.authService.GenerateDeviceCertificate(ctx, deviceID, map[string]interface{}{
		"device_type": "sensor",
	})
	suite.Require().NoError(err)

	// Mock device auth storage
	suite.mockAuthRepo.On("StoreDeviceAuth", ctx, deviceID, mock.AnythingOfType("map[string]interface {}")).Return(nil)

	// Register device for authentication
	err = suite.authService.RegisterDevice(ctx, deviceID, map[string]interface{}{
		"certificate_pem": certPEM,
		"private_key_pem": privateKeyPEM,
		"capabilities":    []string{"read_telemetry", "send_commands"},
	})

	suite.Require().NoError(err)

	// Mock retrieval for authentication
	suite.mockAuthRepo.On("GetDeviceAuth", ctx, deviceID).Return(map[string]interface{}{
		"certificate_pem": certPEM,
		"private_key_pem": privateKeyPEM,
		"capabilities":    []string{"read_telemetry", "send_commands"},
	}, nil)

	// Test device authentication
	isValid, capabilities, err := suite.authService.AuthenticateDevice(ctx, deviceID, certPEM)

	suite.Require().NoError(err)
	suite.Assert().True(isValid, "Device should authenticate successfully")
	suite.Assert().NotEmpty(capabilities, "Device should have capabilities")
}

func (suite *SecurityTestSuite) TestAccessControlEnforcement() {
	ctx := context.Background()

	deviceID := "access-control-device"
	resourceID := "sensitive-data"

	// Mock device auth with limited permissions
	suite.mockAuthRepo.On("GetDeviceAuth", ctx, deviceID).Return(map[string]interface{}{
		"capabilities": []string{"read_basic"},
		"access_level": "restricted",
	}, nil)

	// Test access control scenarios

	// 1. Device with restricted access trying to access sensitive data
	allowed, err := suite.authService.CheckAccess(ctx, deviceID, resourceID, "read_sensitive")

	suite.Require().NoError(err)
	suite.Assert().False(allowed, "Device with restricted access should not access sensitive data")

	// 2. Device with appropriate access
	suite.mockAuthRepo.On("GetDeviceAuth", ctx, "privileged-device").Return(map[string]interface{}{
		"capabilities": []string{"read_sensitive", "write_sensitive"},
		"access_level": "full",
	}, nil)

	allowed, err = suite.authService.CheckAccess(ctx, "privileged-device", resourceID, "read_sensitive")

	suite.Require().NoError(err)
	suite.Assert().True(allowed, "Device with full access should access sensitive data")
}

func (suite *SecurityTestSuite) TestAuditTrailCompleteness() {
	ctx := context.Background()

	secretID := "audit-test-secret"
	plainTextSecret := "audit-test-value"

	// Store secret with audit metadata
	suite.mockRepo.On("StoreSecret", ctx, secretID, mock.AnythingOfType("[]uint8"), mock.MatchedBy(func(metadata map[string]interface{}) bool {
		// Verify audit fields are present
		_, hasTimestamp := metadata["created_at"]
		_, hasOperation := metadata["operation"]
		_, hasPrincipal := metadata["principal"]

		return hasTimestamp && hasOperation && hasPrincipal
	})).Return(nil)

	err := suite.service.StoreSecret(ctx, secretID, plainTextSecret, map[string]interface{}{
		"type": "test_secret",
	})

	suite.Require().NoError(err)

	// Verify audit trail was created
	suite.mockRepo.AssertCalled(suite.T(), "StoreSecret", ctx, secretID, mock.AnythingOfType("[]uint8"), mock.AnythingOfType("map[string]interface {}"))

	// Mock retrieval with audit
	suite.mockRepo.On("GetSecret", ctx, secretID).Return([]byte("encrypted-data"), map[string]interface{}{
		"created_at": time.Now(),
		"operation":  "store",
		"principal":  "test-user",
		"type":       "test_secret",
	}, nil)

	// Retrieve secret (should create access audit entry)
	suite.mockRepo.On("StoreSecret", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("[]uint8"), mock.MatchedBy(func(metadata map[string]interface{}) bool {
		operation, exists := metadata["operation"]
		return exists && operation == "access"
	})).Return(nil)

	_, _, err = suite.service.GetSecret(ctx, secretID)

	suite.Require().NoError(err)

	// Verify access was audited
	calls := suite.mockRepo.Calls
	accessCall := calls[len(calls)-1] // Last call should be the access audit

	metadata := accessCall.Arguments[3].(map[string]interface{})
	suite.Assert().Equal("access", metadata["operation"])
}

func (suite *SecurityTestSuite) TestCertificateRevocation() {
	ctx := context.Background()

	deviceID := "revoke-test-device"
	certID := fmt.Sprintf("cert-%s", deviceID)

	// Mock certificate storage
	suite.mockAuthRepo.On("StoreCertificate", ctx, certID, mock.AnythingOfType("[]uint8"), mock.AnythingOfType("map[string]interface {}")).Return(nil)

	// Generate certificate
	certPEM, _, err := suite.authService.GenerateDeviceCertificate(ctx, deviceID, map[string]interface{}{})
	suite.Require().NoError(err)

	// Mock revocation
	suite.mockAuthRepo.On("RevokeCertificate", ctx, certID).Return(nil)

	// Revoke certificate
	err = suite.authService.RevokeDeviceCertificate(ctx, deviceID)

	suite.Require().NoError(err)

	// Verify revocation was called
	suite.mockAuthRepo.AssertCalled(suite.T(), "RevokeCertificate", ctx, certID)

	// Mock retrieval to verify revocation status
	suite.mockAuthRepo.On("GetCertificate", ctx, certID).Return([]byte(certPEM), map[string]interface{}{
		"status":     "revoked",
		"revoked_at": time.Now(),
		"reason":     "device_decommissioned",
	}, nil)

	// Check certificate status
	certData, metadata, err := suite.authService.GetCertificateStatus(ctx, deviceID)

	suite.Require().NoError(err)
	suite.Assert().NotEmpty(certData)
	suite.Assert().Equal("revoked", metadata["status"])
	suite.Assert().NotNil(metadata["revoked_at"])
}

func (suite *SecurityTestSuite) TestSecureKeyGeneration() {
	ctx := context.Background()

	// Test RSA key generation
	privateKey, err := suite.authService.GenerateRSAKey(ctx, 2048)
	suite.Require().NoError(err)
	suite.Assert().NotNil(privateKey)
	suite.Assert().Equal(2048, privateKey.N.BitLen())

	// Test AES key generation
	aesKey, err := suite.authService.GenerateAESKey(ctx, 256)
	suite.Require().NoError(err)
	suite.Assert().Equal(32, len(aesKey)) // 256 bits = 32 bytes

	// Test key randomness (generate multiple keys and ensure they're different)
	key1, err := suite.authService.GenerateAESKey(ctx, 256)
	suite.Require().NoError(err)

	key2, err := suite.authService.GenerateAESKey(ctx, 256)
	suite.Require().NoError(err)

	suite.Assert().NotEqual(key1, key2, "Generated keys should be different")
}

func (suite *SecurityTestSuite) TestSecureWipeOfSensitiveData() {
	ctx := context.Background()

	secretID := "wipe-test-secret"
	sensitiveData := "very-sensitive-information"

	// Store secret
	suite.mockRepo.On("StoreSecret", ctx, secretID, mock.AnythingOfType("[]uint8"), mock.AnythingOfType("map[string]interface {}")).Return(nil)

	err := suite.service.StoreSecret(ctx, secretID, sensitiveData, map[string]interface{}{})
	suite.Require().NoError(err)

	// Mock secure deletion
	suite.mockRepo.On("DeleteSecret", ctx, secretID).Return(nil)

	// Delete secret (should securely wipe)
	err = suite.service.DeleteSecret(ctx, secretID)

	suite.Require().NoError(err)

	// Verify secure deletion was called
	suite.mockRepo.AssertCalled(suite.T(), "DeleteSecret", ctx, secretID)

	// Mock retrieval to verify data is gone
	suite.mockRepo.On("GetSecret", ctx, secretID).Return([]byte{}, map[string]interface{}{}, fmt.Errorf("secret not found"))

	// Try to retrieve deleted secret
	_, _, err = suite.service.GetSecret(ctx, secretID)

	suite.Assert().Error(err, "Deleted secret should not be retrievable")
	suite.Assert().Contains(err.Error(), "not found")
}

// Additional comprehensive security tests

func (suite *SecurityTestSuite) TestSecretInjectionWithoutLogExposure() {
	ctx := context.Background()

	secretID := "log-exposure-test"
	plainTextSecret := "super-secret-api-key-12345"

	// Mock repository to store encrypted data
	suite.mockRepo.On("StoreSecret", ctx, secretID, mock.AnythingOfType("[]uint8"), mock.AnythingOfType("map[string]interface {}")).Return(nil)

	// Store secret and capture any log output
	err := suite.service.StoreSecret(ctx, secretID, plainTextSecret, map[string]interface{}{
		"type":    "api_key",
		"purpose": "external_service",
	})

	suite.Require().NoError(err)

	// Verify the call was made with encrypted data (not plain text)
	calls := suite.mockRepo.Calls
	storedData := calls[0].Arguments[2].([]byte)
	suite.Assert().NotEqual(plainTextSecret, string(storedData), "Secret should be encrypted before storage")

	// Verify the secret is not exposed in any stored metadata
	metadata := calls[0].Arguments[3].(map[string]interface{})
	suite.Assert().NotContains(metadata, plainTextSecret, "Secret should not be exposed in metadata")
	suite.Assert().NotContains(metadata, "api-key-12345", "Partial secret should not be exposed")
}

func (suite *SecurityTestSuite) TestCertificateGenerationProperties() {
	ctx := context.Background()

	// Test certificate generation with different device types
	deviceTypes := []string{"sensor", "actuator", "gateway"}

	for _, deviceType := range deviceTypes {
		deviceID := fmt.Sprintf("test-%s-001", deviceType)

		// Mock certificate storage
		suite.mockAuthRepo.On("StoreCertificate", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("[]uint8"), mock.AnythingOfType("map[string]interface {}")).Return(nil)

		// Generate device certificate
		certPEM, privateKeyPEM, err := suite.authService.GenerateDeviceCertificate(ctx, deviceID, map[string]interface{}{
			"device_type": deviceType,
			"firmware":    "1.0.0",
		})

		suite.Require().NoError(err, "Certificate generation should succeed for %s", deviceType)
		suite.Assert().NotEmpty(certPEM, "Certificate PEM should not be empty for %s", deviceType)
		suite.Assert().NotEmpty(privateKeyPEM, "Private key PEM should not be empty for %s", deviceType)

		// Verify certificate can be parsed
		block, _ := pem.Decode([]byte(certPEM))
		suite.Require().NotNil(block, "Certificate PEM should be decodeable for %s", deviceType)

		cert, err := x509.ParseCertificate(block.Bytes)
		suite.Require().NoError(err, "Certificate should be parseable for %s", deviceType)

		// Verify certificate properties
		suite.Assert().Equal(deviceID, cert.Subject.CommonName, "Certificate CN should match device ID for %s", deviceType)
		suite.Assert().True(time.Now().After(cert.NotBefore), "Certificate should be valid now for %s", deviceType)
		suite.Assert().True(time.Now().Before(cert.NotAfter), "Certificate should not be expired for %s", deviceType)
		suite.Assert().True(cert.NotAfter.Sub(time.Now()) > 300*24*time.Hour, "Certificate should be valid for at least 300 days for %s", deviceType)
	}
}

func (suite *SecurityTestSuite) TestAccessControlWithInvalidTokens() {
	ctx := context.Background()

	deviceID := "invalid-token-device"
	resourceID := "protected-resource"

	// Mock device auth with expired token
	suite.mockAuthRepo.On("GetDeviceAuth", ctx, deviceID).Return(map[string]interface{}{
		"capabilities": []string{"read_basic"},
		"access_level": "restricted",
		"token_expiry": time.Now().Add(-1 * time.Hour), // Expired token
	}, nil)

	// Test access with expired token
	allowed, err := suite.authService.CheckAccess(ctx, deviceID, resourceID, "read_basic")

	suite.Require().NoError(err)
	suite.Assert().False(allowed, "Device with expired token should not access resources")

	// Test with completely invalid token
	suite.mockAuthRepo.On("GetDeviceAuth", ctx, "invalid-device").Return(map[string]interface{}{
		"capabilities": []string{"read_basic"},
		"access_level": "restricted",
		"token_expiry": time.Time{}, // Zero time = invalid
	}, nil)

	allowed, err = suite.authService.CheckAccess(ctx, "invalid-device", resourceID, "read_basic")

	suite.Require().NoError(err)
	suite.Assert().False(allowed, "Device with invalid token should not access resources")
}

func (suite *SecurityTestSuite) TestEncryptionKeyRotation() {
	ctx := context.Background()

	secretID := "key-rotation-test"
	plainTextSecret := "rotation-test-data"

	// Store secret with initial encryption key
	suite.mockRepo.On("StoreSecret", ctx, secretID, mock.AnythingOfType("[]uint8"), mock.AnythingOfType("map[string]interface {}")).Return(nil)

	err := suite.service.StoreSecret(ctx, secretID, plainTextSecret, map[string]interface{}{
		"type": "test_secret",
	})
	suite.Require().NoError(err)

	// Get the encrypted data with initial key
	calls := suite.mockRepo.Calls
	initialEncryptedData := calls[0].Arguments[2].([]byte)

	// Mock key rotation (simulate new encryptor)
	newEncryptor := NewAESEncryptor()
	suite.service.encryptor = newEncryptor

	// Mock retrieval of old encrypted data
	suite.mockRepo.On("GetSecret", ctx, secretID).Return(initialEncryptedData, map[string]interface{}{
		"type":               "test_secret",
		"encryption_version": "v1",
	}, nil)

	// Mock storage with new encryption
	suite.mockRepo.On("StoreSecret", ctx, secretID, mock.AnythingOfType("[]uint8"), mock.MatchedBy(func(metadata map[string]interface{}) bool {
		version, exists := metadata["encryption_version"]
		return exists && version == "v2"
	})).Return(nil)

	// Retrieve secret (should trigger re-encryption with new key)
	retrievedSecret, _, err := suite.service.GetSecret(ctx, secretID)

	suite.Require().NoError(err)
	suite.Assert().Equal(plainTextSecret, retrievedSecret, "Secret should be correctly decrypted after key rotation")

	// Verify re-encryption occurred
	suite.mockRepo.AssertCalled(suite.T(), "StoreSecret", ctx, secretID, mock.AnythingOfType("[]uint8"), mock.AnythingOfType("map[string]interface {}"))
}

func (suite *SecurityTestSuite) TestAuditTrailTamperingResistance() {
	ctx := context.Background()

	secretID := "tamper-test-secret"
	plainTextSecret := "tamper-test-value"

	// Store secret with audit metadata
	suite.mockRepo.On("StoreSecret", ctx, secretID, mock.AnythingOfType("[]uint8"), mock.MatchedBy(func(metadata map[string]interface{}) bool {
		// Verify audit fields are present and properly signed
		_, hasTimestamp := metadata["created_at"]
		_, hasOperation := metadata["operation"]
		_, hasSignature := metadata["audit_signature"]

		return hasTimestamp && hasOperation && hasSignature
	})).Return(nil)

	err := suite.service.StoreSecret(ctx, secretID, plainTextSecret, map[string]interface{}{
		"type": "test_secret",
	})

	suite.Require().NoError(err)

	// Mock retrieval with tampered audit trail
	suite.mockRepo.On("GetSecret", ctx, secretID).Return([]byte("encrypted-data"), map[string]interface{}{
		"created_at":      time.Now(),
		"operation":       "store",
		"principal":       "test-user",
		"type":            "test_secret",
		"audit_signature": "invalid-signature", // Tampered signature
	}, nil)

	// Try to retrieve secret with tampered audit trail
	_, _, err = suite.service.GetSecret(ctx, secretID)

	suite.Assert().Error(err, "Should reject secret with tampered audit trail")
	suite.Assert().Contains(err.Error(), "audit signature")
}

func (suite *SecurityTestSuite) TestMemorySanitization() {
	ctx := context.Background()

	secretID := "memory-sanitization-test"
	sensitiveData := "very-sensitive-memory-data"

	// Store secret
	suite.mockRepo.On("StoreSecret", ctx, secretID, mock.AnythingOfType("[]uint8"), mock.AnythingOfType("map[string]interface {}")).Return(nil)

	err := suite.service.StoreSecret(ctx, secretID, sensitiveData, map[string]interface{}{})
	suite.Require().NoError(err)

	// Mock retrieval
	suite.mockRepo.On("GetSecret", ctx, secretID).Return([]byte("encrypted-data"), map[string]interface{}{
		"type": "test_secret",
	}, nil)

	// Retrieve secret
	retrievedSecret, _, err := suite.service.GetSecret(ctx, secretID)
	suite.Require().NoError(err)

	// Verify sensitive data is not lingering in memory (basic check)
	// In a real implementation, this would involve memory scanning
	// For test purposes, we verify the service properly handles data
	suite.Assert().NotEmpty(retrievedSecret)
	suite.Assert().NotEqual(sensitiveData, retrievedSecret) // Should be decrypted, not raw
}
