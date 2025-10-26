package template

import (
	"fmt"
	"regexp"
	"strconv"
)

// VersionManager handles template versioning operations
type VersionManager struct{}

// NewVersionManager creates a new version manager
func NewVersionManager() *VersionManager {
	return &VersionManager{}
}

// ParseVersion parses a version string into major, minor, patch components
func (vm *VersionManager) ParseVersion(version string) (major, minor, patch int, err error) {
	// Support semantic versioning (e.g., "1.2.3") and simple versioning (e.g., "1.0")
	versionRegex := regexp.MustCompile(`^(\d+)\.(\d+)(?:\.(\d+))?$`)
	matches := versionRegex.FindStringSubmatch(version)
	
	if len(matches) < 3 {
		return 0, 0, 0, fmt.Errorf("invalid version format: %s", version)
	}
	
	major, err = strconv.Atoi(matches[1])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid major version: %s", matches[1])
	}
	
	minor, err = strconv.Atoi(matches[2])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid minor version: %s", matches[2])
	}
	
	// Patch version is optional
	if len(matches) > 3 && matches[3] != "" {
		patch, err = strconv.Atoi(matches[3])
		if err != nil {
			return 0, 0, 0, fmt.Errorf("invalid patch version: %s", matches[3])
		}
	}
	
	return major, minor, patch, nil
}

// CompareVersions compares two version strings
// Returns: -1 if v1 < v2, 0 if v1 == v2, 1 if v1 > v2
func (vm *VersionManager) CompareVersions(v1, v2 string) (int, error) {
	major1, minor1, patch1, err := vm.ParseVersion(v1)
	if err != nil {
		return 0, fmt.Errorf("failed to parse version %s: %w", v1, err)
	}
	
	major2, minor2, patch2, err := vm.ParseVersion(v2)
	if err != nil {
		return 0, fmt.Errorf("failed to parse version %s: %w", v2, err)
	}
	
	// Compare major version
	if major1 < major2 {
		return -1, nil
	} else if major1 > major2 {
		return 1, nil
	}
	
	// Compare minor version
	if minor1 < minor2 {
		return -1, nil
	} else if minor1 > minor2 {
		return 1, nil
	}
	
	// Compare patch version
	if patch1 < patch2 {
		return -1, nil
	} else if patch1 > patch2 {
		return 1, nil
	}
	
	return 0, nil
}

// GetLatestVersion returns the latest version from a list of versions
func (vm *VersionManager) GetLatestVersion(versions []string) (string, error) {
	if len(versions) == 0 {
		return "", fmt.Errorf("no versions provided")
	}
	
	latest := versions[0]
	for _, version := range versions[1:] {
		comparison, err := vm.CompareVersions(version, latest)
		if err != nil {
			return "", fmt.Errorf("failed to compare versions: %w", err)
		}
		if comparison > 0 {
			latest = version
		}
	}
	
	return latest, nil
}

// IsBackwardCompatible checks if a new version is backward compatible with an old version
// For now, we consider versions backward compatible if they have the same major version
func (vm *VersionManager) IsBackwardCompatible(oldVersion, newVersion string) (bool, error) {
	oldMajor, _, _, err := vm.ParseVersion(oldVersion)
	if err != nil {
		return false, fmt.Errorf("failed to parse old version %s: %w", oldVersion, err)
	}
	
	newMajor, _, _, err := vm.ParseVersion(newVersion)
	if err != nil {
		return false, fmt.Errorf("failed to parse new version %s: %w", newVersion, err)
	}
	
	// Same major version indicates backward compatibility
	return oldMajor == newMajor, nil
}

// GenerateNextVersion generates the next version based on the type of change
func (vm *VersionManager) GenerateNextVersion(currentVersion string, changeType VersionChangeType) (string, error) {
	major, minor, patch, err := vm.ParseVersion(currentVersion)
	if err != nil {
		return "", fmt.Errorf("failed to parse current version %s: %w", currentVersion, err)
	}
	
	switch changeType {
	case MajorChange:
		return fmt.Sprintf("%d.0.0", major+1), nil
	case MinorChange:
		return fmt.Sprintf("%d.%d.0", major, minor+1), nil
	case PatchChange:
		return fmt.Sprintf("%d.%d.%d", major, minor, patch+1), nil
	default:
		return "", fmt.Errorf("invalid change type: %v", changeType)
	}
}

// VersionChangeType represents the type of version change
type VersionChangeType int

const (
	// PatchChange represents a patch-level change (bug fixes)
	PatchChange VersionChangeType = iota
	// MinorChange represents a minor-level change (new features, backward compatible)
	MinorChange
	// MajorChange represents a major-level change (breaking changes)
	MajorChange
)

// VersionInfo contains detailed information about a template version
type VersionInfo struct {
	Version     string    `json:"version"`
	Major       int       `json:"major"`
	Minor       int       `json:"minor"`
	Patch       int       `json:"patch"`
	IsLatest    bool      `json:"is_latest"`
	ReleaseDate string    `json:"release_date"`
	Changes     []string  `json:"changes,omitempty"`
}

// GetVersionInfo returns detailed information about template versions
func (vm *VersionManager) GetVersionInfo(versions []string) ([]*VersionInfo, error) {
	if len(versions) == 0 {
		return []*VersionInfo{}, nil
	}
	
	latest, err := vm.GetLatestVersion(versions)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest version: %w", err)
	}
	
	var versionInfos []*VersionInfo
	for _, version := range versions {
		major, minor, patch, err := vm.ParseVersion(version)
		if err != nil {
			return nil, fmt.Errorf("failed to parse version %s: %w", version, err)
		}
		
		info := &VersionInfo{
			Version:  version,
			Major:    major,
			Minor:    minor,
			Patch:    patch,
			IsLatest: version == latest,
		}
		
		versionInfos = append(versionInfos, info)
	}
	
	return versionInfos, nil
}

// SortVersions sorts a list of versions in ascending order
func (vm *VersionManager) SortVersions(versions []string) ([]string, error) {
	if len(versions) <= 1 {
		return versions, nil
	}
	
	// Create a copy to avoid modifying the original slice
	sorted := make([]string, len(versions))
	copy(sorted, versions)
	
	// Simple bubble sort for versions
	for i := 0; i < len(sorted)-1; i++ {
		for j := 0; j < len(sorted)-i-1; j++ {
			comparison, err := vm.CompareVersions(sorted[j], sorted[j+1])
			if err != nil {
				return nil, fmt.Errorf("failed to compare versions: %w", err)
			}
			if comparison > 0 {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}
	
	return sorted, nil
}

// ValidateVersionSequence validates that a sequence of versions follows proper versioning rules
func (vm *VersionManager) ValidateVersionSequence(versions []string) error {
	if len(versions) <= 1 {
		return nil
	}
	
	sorted, err := vm.SortVersions(versions)
	if err != nil {
		return fmt.Errorf("failed to sort versions: %w", err)
	}
	
	for i := 1; i < len(sorted); i++ {
		prev := sorted[i-1]
		curr := sorted[i]
		
		// Check that each version is properly incremented
		prevMajor, prevMinor, _, err := vm.ParseVersion(prev)
		if err != nil {
			return fmt.Errorf("failed to parse version %s: %w", prev, err)
		}
		
		currMajor, currMinor, currPatch, err := vm.ParseVersion(curr)
		if err != nil {
			return fmt.Errorf("failed to parse version %s: %w", curr, err)
		}
		
		// Validate version increment rules
		if currMajor > prevMajor {
			// Major version increment should reset minor and patch to 0
			if currMinor != 0 || currPatch != 0 {
				return fmt.Errorf("major version increment should reset minor and patch to 0: %s -> %s", prev, curr)
			}
		} else if currMajor == prevMajor && currMinor > prevMinor {
			// Minor version increment should reset patch to 0
			if currPatch != 0 {
				return fmt.Errorf("minor version increment should reset patch to 0: %s -> %s", prev, curr)
			}
		}
	}
	
	return nil
}