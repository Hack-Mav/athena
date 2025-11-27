package provisioning

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// ArtifactManager manages build artifacts
type ArtifactManager struct {
	storageDir string
	maxAge     time.Duration
}

// NewArtifactManager creates a new artifact manager
func NewArtifactManager(storageDir string) *ArtifactManager {
	return &ArtifactManager{
		storageDir: storageDir,
		maxAge:     24 * time.Hour, // Keep artifacts for 24 hours by default
	}
}

// BuildArtifact represents a build artifact
type BuildArtifact struct {
	ID           string                 `json:"id"`
	TemplateID   string                 `json:"template_id"`
	Board        string                 `json:"board"`
	BinaryPath   string                 `json:"binary_path"`
	BinaryHash   string                 `json:"binary_hash"`
	Size         CompilationSize        `json:"size"`
	Metadata     ArtifactMetadata       `json:"metadata"`
	CreatedAt    time.Time              `json:"created_at"`
	ExpiresAt    time.Time              `json:"expires_at"`
}

// ArtifactMetadata contains artifact metadata
type ArtifactMetadata struct {
	Parameters    map[string]interface{} `json:"parameters"`
	Libraries     []LibraryDependency    `json:"libraries"`
	CompilerFlags []string               `json:"compiler_flags,omitempty"`
	BuildInfo     BuildInfo              `json:"build_info"`
}

// BuildInfo contains build environment information
type BuildInfo struct {
	ArduinoCLIVersion string    `json:"arduino_cli_version"`
	Platform          string    `json:"platform"`
	BuildTime         time.Time `json:"build_time"`
	BuildHost         string    `json:"build_host"`
}

// ArtifactQuery represents a query for artifacts
type ArtifactQuery struct {
	TemplateID string                 `json:"template_id,omitempty"`
	Board      string                 `json:"board,omitempty"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
	MaxAge     time.Duration          `json:"max_age,omitempty"`
}

// StoreArtifact stores a build artifact
func (am *ArtifactManager) StoreArtifact(ctx context.Context, result *CompilationResult) (*BuildArtifact, error) {
	// Generate artifact ID
	artifactID := am.generateArtifactID(result)
	
	// Create artifact directory
	artifactDir := filepath.Join(am.storageDir, artifactID)
	if err := os.MkdirAll(artifactDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create artifact directory: %w", err)
	}

	// Copy binary to artifact storage
	binaryName := filepath.Base(result.BinaryPath)
	storedBinaryPath := filepath.Join(artifactDir, binaryName)
	
	if err := am.copyFile(result.BinaryPath, storedBinaryPath); err != nil {
		return nil, fmt.Errorf("failed to copy binary: %w", err)
	}

	// Create artifact metadata
	artifact := &BuildArtifact{
		ID:         artifactID,
		TemplateID: result.Metadata.TemplateID,
		Board:      result.Metadata.Board,
		BinaryPath: storedBinaryPath,
		BinaryHash: result.BinaryHash,
		Size:       result.Size,
		Metadata: ArtifactMetadata{
			Parameters:    result.Metadata.Parameters,
			Libraries:     result.Metadata.Libraries,
			CompilerFlags: result.Metadata.CompilerFlags,
			BuildInfo: BuildInfo{
				ArduinoCLIVersion: result.Metadata.ArduinoCLI,
				Platform:          result.Metadata.Board,
				BuildTime:         result.Metadata.CompiledAt,
				BuildHost:         "localhost", // In practice, get actual hostname
			},
		},
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(am.maxAge),
	}

	// Store artifact metadata
	metadataPath := filepath.Join(artifactDir, "metadata.json")
	if err := am.saveArtifactMetadata(artifact, metadataPath); err != nil {
		return nil, fmt.Errorf("failed to save artifact metadata: %w", err)
	}

	return artifact, nil
}

// GetArtifact retrieves an artifact by ID
func (am *ArtifactManager) GetArtifact(ctx context.Context, artifactID string) (*BuildArtifact, error) {
	artifactDir := filepath.Join(am.storageDir, artifactID)
	metadataPath := filepath.Join(artifactDir, "metadata.json")

	// Check if artifact exists
	if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("artifact not found: %s", artifactID)
	}

	// Load artifact metadata
	artifact, err := am.loadArtifactMetadata(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load artifact metadata: %w", err)
	}

	// Check if artifact has expired
	if time.Now().After(artifact.ExpiresAt) {
		// Clean up expired artifact
		os.RemoveAll(artifactDir)
		return nil, fmt.Errorf("artifact expired: %s", artifactID)
	}

	// Verify binary still exists
	if _, err := os.Stat(artifact.BinaryPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("artifact binary missing: %s", artifactID)
	}

	return artifact, nil
}

// FindArtifacts finds artifacts matching the query
func (am *ArtifactManager) FindArtifacts(ctx context.Context, query ArtifactQuery) ([]*BuildArtifact, error) {
	var artifacts []*BuildArtifact

	// Walk through artifact storage directory
	err := filepath.Walk(am.storageDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Look for metadata.json files
		if info.Name() == "metadata.json" {
			artifact, err := am.loadArtifactMetadata(path)
			if err != nil {
				// Skip invalid artifacts
				return nil
			}

			// Check if artifact matches query
			if am.matchesQuery(artifact, query) {
				artifacts = append(artifacts, artifact)
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to search artifacts: %w", err)
	}

	return artifacts, nil
}

// DeleteArtifact deletes an artifact
func (am *ArtifactManager) DeleteArtifact(ctx context.Context, artifactID string) error {
	artifactDir := filepath.Join(am.storageDir, artifactID)
	
	if err := os.RemoveAll(artifactDir); err != nil {
		return fmt.Errorf("failed to delete artifact: %w", err)
	}

	return nil
}

// CleanupExpiredArtifacts removes expired artifacts
func (am *ArtifactManager) CleanupExpiredArtifacts(ctx context.Context) error {
	now := time.Now()
	var deletedCount int

	err := filepath.Walk(am.storageDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.Name() == "metadata.json" {
			artifact, err := am.loadArtifactMetadata(path)
			if err != nil {
				// Delete invalid artifacts
				os.RemoveAll(filepath.Dir(path))
				deletedCount++
				return nil
			}

			// Delete expired artifacts
			if now.After(artifact.ExpiresAt) {
				os.RemoveAll(filepath.Dir(path))
				deletedCount++
			}
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to cleanup artifacts: %w", err)
	}

	return nil
}

// GetArtifactBinary returns the binary data for an artifact
func (am *ArtifactManager) GetArtifactBinary(ctx context.Context, artifactID string) ([]byte, error) {
	artifact, err := am.GetArtifact(ctx, artifactID)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(artifact.BinaryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read artifact binary: %w", err)
	}

	return data, nil
}

// VerifyArtifact verifies the integrity of an artifact
func (am *ArtifactManager) VerifyArtifact(ctx context.Context, artifactID string) error {
	artifact, err := am.GetArtifact(ctx, artifactID)
	if err != nil {
		return err
	}

	// Calculate current hash of binary
	currentHash, err := am.calculateFileHash(artifact.BinaryPath)
	if err != nil {
		return fmt.Errorf("failed to calculate binary hash: %w", err)
	}

	// Compare with stored hash
	if currentHash != artifact.BinaryHash {
		return fmt.Errorf("artifact integrity check failed: hash mismatch")
	}

	return nil
}

// generateArtifactID generates a unique artifact ID
func (am *ArtifactManager) generateArtifactID(result *CompilationResult) string {
	hasher := sha256.New()
	
	// Include template ID, board, and binary hash
	hasher.Write([]byte(result.Metadata.TemplateID))
	hasher.Write([]byte(result.Metadata.Board))
	hasher.Write([]byte(result.BinaryHash))
	hasher.Write([]byte(result.Metadata.CompiledAt.Format(time.RFC3339)))

	return hex.EncodeToString(hasher.Sum(nil))[:16] // Use first 16 chars
}

// copyFile copies a file from src to dst
func (am *ArtifactManager) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

// saveArtifactMetadata saves artifact metadata to JSON file
func (am *ArtifactManager) saveArtifactMetadata(artifact *BuildArtifact, path string) error {
	data, err := json.MarshalIndent(artifact, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// loadArtifactMetadata loads artifact metadata from JSON file
func (am *ArtifactManager) loadArtifactMetadata(path string) (*BuildArtifact, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var artifact BuildArtifact
	if err := json.Unmarshal(data, &artifact); err != nil {
		return nil, err
	}

	return &artifact, nil
}

// matchesQuery checks if an artifact matches the query criteria
func (am *ArtifactManager) matchesQuery(artifact *BuildArtifact, query ArtifactQuery) bool {
	// Check template ID
	if query.TemplateID != "" && artifact.TemplateID != query.TemplateID {
		return false
	}

	// Check board
	if query.Board != "" && artifact.Board != query.Board {
		return false
	}

	// Check age
	maxAge := query.MaxAge
	if maxAge == 0 {
		maxAge = am.maxAge
	}
	if time.Since(artifact.CreatedAt) > maxAge {
		return false
	}

	// Check parameters (simplified matching)
	if query.Parameters != nil {
		for key, value := range query.Parameters {
			if artifactValue, exists := artifact.Metadata.Parameters[key]; !exists || artifactValue != value {
				return false
			}
		}
	}

	return true
}

// calculateFileHash calculates SHA256 hash of a file
func (am *ArtifactManager) calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}