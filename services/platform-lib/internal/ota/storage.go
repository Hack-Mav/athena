package ota

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// LocalStorageBackend implements StorageBackend using local filesystem
type LocalStorageBackend struct {
	basePath string
}

// NewLocalStorageBackend creates a new local storage backend
func NewLocalStorageBackend(basePath string) (*LocalStorageBackend, error) {
	// Create base directory if it doesn't exist
	err := os.MkdirAll(basePath, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	return &LocalStorageBackend{
		basePath: basePath,
	}, nil
}

// StoreBinary stores a binary file in the local filesystem
func (s *LocalStorageBackend) StoreBinary(ctx context.Context, releaseID string, data []byte) (string, error) {
	// Create release directory
	releaseDir := filepath.Join(s.basePath, releaseID)
	err := os.MkdirAll(releaseDir, 0755)
	if err != nil {
		return "", fmt.Errorf("failed to create release directory: %w", err)
	}

	// Write binary file
	binaryPath := filepath.Join(releaseDir, "firmware.bin")
	err = os.WriteFile(binaryPath, data, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write binary file: %w", err)
	}

	// Return relative path
	return filepath.Join(releaseID, "firmware.bin"), nil
}

// GetBinary retrieves a binary file from the local filesystem
func (s *LocalStorageBackend) GetBinary(ctx context.Context, path string) ([]byte, error) {
	fullPath := filepath.Join(s.basePath, path)
	
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read binary file: %w", err)
	}

	return data, nil
}

// GetBinaryURL returns a URL for accessing the binary (for local storage, returns file path)
func (s *LocalStorageBackend) GetBinaryURL(ctx context.Context, path string, expiry time.Duration) (string, error) {
	fullPath := filepath.Join(s.basePath, path)
	
	// Check if file exists
	_, err := os.Stat(fullPath)
	if err != nil {
		return "", fmt.Errorf("binary file not found: %w", err)
	}

	// For local storage, return file:// URL
	return "file://" + fullPath, nil
}

// DeleteBinary deletes a binary file from the local filesystem
func (s *LocalStorageBackend) DeleteBinary(ctx context.Context, path string) error {
	fullPath := filepath.Join(s.basePath, path)
	
	err := os.Remove(fullPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete binary file: %w", err)
	}

	// Try to remove the parent directory if empty
	parentDir := filepath.Dir(fullPath)
	_ = os.Remove(parentDir) // Ignore error if directory is not empty

	return nil
}
