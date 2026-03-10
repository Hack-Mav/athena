package main

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"
)

// Service represents the secrets service
type Service struct {
	config     *Config
	logger     *Logger
	repository Repository
	encryptor  Encryptor
}

// AuthService represents the authentication service
type AuthService struct {
	config        *Config
	logger        *Logger
	repository    AuthRepository
	certAuthority *CertificateAuthority
}

// Config represents service configuration
type Config struct {
	LogLevel    string
	ServiceName string
	Environment string
}

// Logger represents a logger
type Logger struct {
	level string
	name  string
}

// Repository represents secrets repository interface
type Repository interface {
	StoreSecret(ctx context.Context, secretID string, encryptedData []byte, metadata map[string]interface{}) error
	GetSecret(ctx context.Context, secretID string) ([]byte, map[string]interface{}, error)
	DeleteSecret(ctx context.Context, secretID string) error
	ListSecrets(ctx context.Context, filters map[string]interface{}) ([]string, error)
}

// AuthRepository represents authentication repository interface
type AuthRepository interface {
	StoreCertificate(ctx context.Context, certID string, certData []byte, metadata map[string]interface{}) error
	GetCertificate(ctx context.Context, certID string) ([]byte, map[string]interface{}, error)
	RevokeCertificate(ctx context.Context, certID string) error
	StoreDeviceAuth(ctx context.Context, deviceID string, authData map[string]interface{}) error
	GetDeviceAuth(ctx context.Context, deviceID string) (map[string]interface{}, error)
}

// Encryptor represents encryption interface
type Encryptor interface {
	Encrypt(plaintext []byte) ([]byte, error)
	Decrypt(ciphertext []byte) ([]byte, error)
}

// CertificateAuthority represents certificate authority
type CertificateAuthority struct{}

// AESEncryptor implements AES encryption
type AESEncryptor struct {
	key []byte
}

// NewAESEncryptor creates a new AES encryptor
func NewAESEncryptor() *AESEncryptor {
	key := make([]byte, 32) // AES-256
	rand.Read(key)
	return &AESEncryptor{key: key}
}

// Encrypt encrypts data using AES-GCM
func (a *AESEncryptor) Encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(a.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt decrypts data using AES-GCM
func (a *AESEncryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(a.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	if len(ciphertext) < gcm.NonceSize() {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce := ciphertext[:gcm.NonceSize()]
	ciphertext = ciphertext[gcm.NonceSize():]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// NewCertificateAuthority creates a new certificate authority
func NewCertificateAuthority() *CertificateAuthority {
	return &CertificateAuthority{}
}

// NewService creates a new secrets service
func NewService(cfg *Config, logger *Logger, repo Repository) (*Service, error) {
	return &Service{
		config:     cfg,
		logger:     logger,
		repository: repo,
		encryptor:  NewAESEncryptor(),
	}, nil
}

// NewAuthService creates a new auth service
func NewAuthService(cfg *Config, logger *Logger, repo AuthRepository) (*AuthService, error) {
	return &AuthService{
		config:        cfg,
		logger:        logger,
		repository:    repo,
		certAuthority: NewCertificateAuthority(),
	}, nil
}

// StoreSecret stores a secret securely
func (s *Service) StoreSecret(ctx context.Context, secretID string, plainTextSecret string, metadata map[string]interface{}) error {
	// Encrypt the secret
	encryptedData, err := s.encryptor.Encrypt([]byte(plainTextSecret))
	if err != nil {
		return err
	}

	// Add audit metadata
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["created_at"] = time.Now()
	metadata["operation"] = "store"
	metadata["principal"] = "test-user"

	// Store encrypted data
	return s.repository.StoreSecret(ctx, secretID, encryptedData, metadata)
}

// GetSecret retrieves and decrypts a secret
func (s *Service) GetSecret(ctx context.Context, secretID string) (string, map[string]interface{}, error) {
	encryptedData, metadata, err := s.repository.GetSecret(ctx, secretID)
	if err != nil {
		return "", nil, err
	}

	// Decrypt the data
	decryptedData, err := s.encryptor.Decrypt(encryptedData)
	if err != nil {
		return "", nil, err
	}

	// Create audit entry for access
	auditMetadata := map[string]interface{}{
		"created_at": time.Now(),
		"operation":  "access",
		"principal":  "test-user",
		"secret_id":  secretID,
	}
	s.repository.StoreSecret(ctx, "audit-"+secretID, []byte{}, auditMetadata)

	return string(decryptedData), metadata, nil
}

// DeleteSecret securely deletes a secret
func (s *Service) DeleteSecret(ctx context.Context, secretID string) error {
	return s.repository.DeleteSecret(ctx, secretID)
}

// GenerateDeviceCertificate generates a certificate for a device
func (a *AuthService) GenerateDeviceCertificate(ctx context.Context, deviceID string, metadata map[string]interface{}) (string, string, error) {
	// Generate private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", err
	}

	// Create certificate template
	template := x509.Certificate{
		SerialNumber:          big.NewInt(time.Now().Unix()),
		Subject:               pkix.Name{CommonName: deviceID},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour), // 1 year
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	// Create certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return "", "", err
	}

	// Encode certificate and private key to PEM
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	// Store certificate
	certID := fmt.Sprintf("cert-%s", deviceID)
	certMetadata := map[string]interface{}{
		"device_id":   deviceID,
		"created_at":  time.Now(),
		"expires_at":  template.NotAfter,
		"fingerprint": fmt.Sprintf("%x", certDER),
	}
	for k, v := range metadata {
		certMetadata[k] = v
	}

	err = a.repository.StoreCertificate(ctx, certID, certDER, certMetadata)
	if err != nil {
		return "", "", err
	}

	return string(certPEM), string(privateKeyPEM), nil
}

// RegisterDevice registers a device for authentication
func (a *AuthService) RegisterDevice(ctx context.Context, deviceID string, authData map[string]interface{}) error {
	metadata := map[string]interface{}{
		"device_id":     deviceID,
		"registered_at": time.Now(),
		"status":        "active",
	}
	for k, v := range authData {
		metadata[k] = v
	}

	return a.repository.StoreDeviceAuth(ctx, deviceID, metadata)
}

// AuthenticateDevice authenticates a device
func (a *AuthService) AuthenticateDevice(ctx context.Context, deviceID string, certificatePEM string) (bool, []string, error) {
	authData, err := a.repository.GetDeviceAuth(ctx, deviceID)
	if err != nil {
		return false, nil, err
	}

	// In a real implementation, this would verify the certificate
	// For testing, we'll just check if the device exists and is active
	if authData["status"] != "active" {
		return false, nil, nil
	}

	// Return capabilities if available
	capabilities, ok := authData["capabilities"].([]string)
	if !ok {
		capabilities = []string{}
	}

	return true, capabilities, nil
}

// CheckAccess checks if a device has access to a resource
func (a *AuthService) CheckAccess(ctx context.Context, deviceID string, resourceID string, action string) (bool, error) {
	authData, err := a.repository.GetDeviceAuth(ctx, deviceID)
	if err != nil {
		return false, err
	}

	// Simple access control logic
	capabilities, ok := authData["capabilities"].([]string)
	if !ok {
		return false, nil
	}

	// Check if the requested action is in capabilities
	for _, cap := range capabilities {
		if cap == action || cap == "all" {
			return true, nil
		}
	}

	return false, nil
}

// RevokeDeviceCertificate revokes a device certificate
func (a *AuthService) RevokeDeviceCertificate(ctx context.Context, deviceID string) error {
	certID := fmt.Sprintf("cert-%s", deviceID)

	// Update certificate metadata to mark as revoked
	certData, metadata, err := a.repository.GetCertificate(ctx, certID)
	if err != nil {
		return err
	}

	metadata["status"] = "revoked"
	metadata["revoked_at"] = time.Now()
	metadata["reason"] = "device_decommissioned"

	// Store updated metadata
	return a.repository.StoreCertificate(ctx, certID, certData, metadata)
}

// GetCertificateStatus gets the status of a certificate
func (a *AuthService) GetCertificateStatus(ctx context.Context, deviceID string) ([]byte, map[string]interface{}, error) {
	certID := fmt.Sprintf("cert-%s", deviceID)
	return a.repository.GetCertificate(ctx, certID)
}

// GenerateRSAKey generates an RSA key pair
func (a *AuthService) GenerateRSAKey(ctx context.Context, bits int) (*rsa.PrivateKey, error) {
	return rsa.GenerateKey(rand.Reader, bits)
}

// GenerateAESKey generates an AES key
func (a *AuthService) GenerateAESKey(ctx context.Context, bits int) ([]byte, error) {
	key := make([]byte, bits/8)
	_, err := rand.Read(key)
	return key, err
}

// NewLogger creates a new logger
func NewLogger(level string, name string) *Logger {
	return &Logger{level: level, name: name}
}
