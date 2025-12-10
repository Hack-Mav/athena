package provisioning

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// LibraryManager manages Arduino library dependencies
type LibraryManager struct {
	cli             *ArduinoCLI
	installedCache  map[string]Library
	availableCache  map[string][]Library
	cacheMutex      sync.RWMutex
	installedExpiry time.Time
	availableExpiry time.Time
	cacheDuration   time.Duration
}

// NewLibraryManager creates a new library manager
func NewLibraryManager(cli *ArduinoCLI) *LibraryManager {
	return &LibraryManager{
		cli:            cli,
		installedCache: make(map[string]Library),
		availableCache: make(map[string][]Library),
		cacheDuration:  15 * time.Minute,
	}
}

// DependencyResolution represents the result of dependency resolution
type DependencyResolution struct {
	ToInstall    []LibraryInstallation `json:"to_install"`
	AlreadyMet   []Library             `json:"already_met"`
	Conflicts    []DependencyConflict  `json:"conflicts"`
	HasConflicts bool                  `json:"has_conflicts"`
}

// LibraryInstallation represents a library to be installed
type LibraryInstallation struct {
	Library Library `json:"library"`
	Reason  string  `json:"reason"`
}

// DependencyConflict represents a dependency conflict
type DependencyConflict struct {
	LibraryName      string   `json:"library_name"`
	RequiredVersions []string `json:"required_versions"`
	Reason           string   `json:"reason"`
}

// InstallationResult represents the result of library installation
type InstallationResult struct {
	Installed []Library                  `json:"installed"`
	Failed    []LibraryInstallationError `json:"failed"`
	Skipped   []Library                  `json:"skipped"`
}

// LibraryInstallationError represents a library installation error
type LibraryInstallationError struct {
	Library Library `json:"library"`
	Error   string  `json:"error"`
}

// ResolveDependencies resolves library dependencies for a list of required libraries
func (lm *LibraryManager) ResolveDependencies(ctx context.Context, required []LibraryDependency) (*DependencyResolution, error) {
	resolution := &DependencyResolution{
		ToInstall:    []LibraryInstallation{},
		AlreadyMet:   []Library{},
		Conflicts:    []DependencyConflict{},
		HasConflicts: false,
	}

	// Get currently installed libraries
	installed, err := lm.GetInstalledLibraries(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get installed libraries: %w", err)
	}

	installedMap := make(map[string]Library)
	for _, lib := range installed {
		installedMap[lib.Name] = lib
	}

	// Track version requirements for conflict detection
	versionRequirements := make(map[string][]string)

	// Process each required library
	for _, req := range required {
		versionRequirements[req.Name] = append(versionRequirements[req.Name], req.Version)

		if installed, exists := installedMap[req.Name]; exists {
			// Check if installed version satisfies requirement
			if lm.versionSatisfies(installed.Version, req.Version) {
				resolution.AlreadyMet = append(resolution.AlreadyMet, installed)
				continue
			}
		}

		// Find available versions
		available, err := lm.SearchLibrary(ctx, req.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to search for library %s: %w", req.Name, err)
		}

		// Find best matching version
		bestMatch := lm.findBestVersion(available, req.Version)
		if bestMatch == nil {
			resolution.HasConflicts = true
			resolution.Conflicts = append(resolution.Conflicts, DependencyConflict{
				LibraryName:      req.Name,
				RequiredVersions: []string{req.Version},
				Reason:           "No compatible version found",
			})
			continue
		}

		resolution.ToInstall = append(resolution.ToInstall, LibraryInstallation{
			Library: *bestMatch,
			Reason:  fmt.Sprintf("Required by template (version %s)", req.Version),
		})

		// Recursively resolve dependencies
		if err := lm.resolveDependenciesRecursive(ctx, bestMatch.Dependencies, resolution, installedMap, versionRequirements, 0); err != nil {
			return nil, fmt.Errorf("failed to resolve recursive dependencies: %w", err)
		}
	}

	// Check for version conflicts
	for libName, versions := range versionRequirements {
		if len(versions) > 1 {
			uniqueVersions := lm.uniqueStrings(versions)
			if len(uniqueVersions) > 1 {
				resolution.HasConflicts = true
				resolution.Conflicts = append(resolution.Conflicts, DependencyConflict{
					LibraryName:      libName,
					RequiredVersions: uniqueVersions,
					Reason:           "Multiple incompatible versions required",
				})
			}
		}
	}

	return resolution, nil
}

// InstallLibraries installs a list of libraries
func (lm *LibraryManager) InstallLibraries(ctx context.Context, libraries []Library) (*InstallationResult, error) {
	result := &InstallationResult{
		Installed: []Library{},
		Failed:    []LibraryInstallationError{},
		Skipped:   []Library{},
	}

	// Get currently installed libraries to avoid reinstalling
	installed, err := lm.GetInstalledLibraries(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get installed libraries: %w", err)
	}

	installedMap := make(map[string]Library)
	for _, lib := range installed {
		installedMap[lib.Name] = lib
	}

	for _, lib := range libraries {
		// Check if already installed with same or compatible version
		if installedLib, exists := installedMap[lib.Name]; exists {
			if lm.versionSatisfies(installedLib.Version, lib.Version) {
				result.Skipped = append(result.Skipped, installedLib)
				continue
			}
		}

		// Install library
		installSpec := lib.Name
		if lib.Version != "" && lib.Version != "latest" {
			installSpec = fmt.Sprintf("%s@%s", lib.Name, lib.Version)
		}

		if err := lm.cli.InstallLibrary(ctx, installSpec); err != nil {
			result.Failed = append(result.Failed, LibraryInstallationError{
				Library: lib,
				Error:   err.Error(),
			})
		} else {
			result.Installed = append(result.Installed, lib)
		}
	}

	// Invalidate cache after installation
	lm.invalidateInstalledCache()

	return result, nil
}

// GetInstalledLibraries returns currently installed libraries
func (lm *LibraryManager) GetInstalledLibraries(ctx context.Context) ([]Library, error) {
	lm.cacheMutex.RLock()
	if time.Now().Before(lm.installedExpiry) && len(lm.installedCache) > 0 {
		libraries := make([]Library, 0, len(lm.installedCache))
		for _, lib := range lm.installedCache {
			libraries = append(libraries, lib)
		}
		lm.cacheMutex.RUnlock()
		return libraries, nil
	}
	lm.cacheMutex.RUnlock()

	libraries, err := lm.cli.ListInstalledLibraries(ctx)
	if err != nil {
		return nil, err
	}

	lm.cacheMutex.Lock()
	lm.installedCache = make(map[string]Library)
	for _, lib := range libraries {
		lm.installedCache[lib.Name] = lib
	}
	lm.installedExpiry = time.Now().Add(lm.cacheDuration)
	lm.cacheMutex.Unlock()

	return libraries, nil
}

// SearchLibrary searches for available libraries
func (lm *LibraryManager) SearchLibrary(ctx context.Context, query string) ([]Library, error) {
	lm.cacheMutex.RLock()
	if cached, exists := lm.availableCache[query]; exists && time.Now().Before(lm.availableExpiry) {
		lm.cacheMutex.RUnlock()
		return cached, nil
	}
	lm.cacheMutex.RUnlock()

	libraries, err := lm.cli.SearchLibrary(ctx, query)
	if err != nil {
		return nil, err
	}

	lm.cacheMutex.Lock()
	lm.availableCache[query] = libraries
	lm.availableExpiry = time.Now().Add(lm.cacheDuration)
	lm.cacheMutex.Unlock()

	return libraries, nil
}

// ValidateLibraryCompatibility checks if libraries are compatible with board
func (lm *LibraryManager) ValidateLibraryCompatibility(libraries []Library, board Board) []LibraryCompatibilityIssue {
	var issues []LibraryCompatibilityIssue

	for _, lib := range libraries {
		// Check architecture compatibility
		if len(lib.Architectures) > 0 {
			compatible := false
			boardArch := lm.getBoardArchitecture(board.FQBN)

			for _, arch := range lib.Architectures {
				if arch == "*" || arch == boardArch {
					compatible = true
					break
				}
			}

			if !compatible {
				issues = append(issues, LibraryCompatibilityIssue{
					Library:     lib,
					Issue:       "Architecture incompatibility",
					Severity:    "warning",
					Description: fmt.Sprintf("Library supports %v, board uses %s", lib.Architectures, boardArch),
				})
			}
		}

		// Check for known incompatibilities based on board properties
		if board.Capabilities.Properties["mcu"] == "atmega328p" && lib.Name == "ESP32WiFi" {
			issues = append(issues, LibraryCompatibilityIssue{
				Library:     lib,
				Issue:       "Hardware incompatibility",
				Severity:    "error",
				Description: "ESP32 library cannot be used with ATmega328P boards",
			})
		}
	}

	return issues
}

// LibraryCompatibilityIssue represents a library compatibility issue
type LibraryCompatibilityIssue struct {
	Library     Library `json:"library"`
	Issue       string  `json:"issue"`
	Severity    string  `json:"severity"` // "error", "warning", "info"
	Description string  `json:"description"`
}

// resolveDependenciesRecursive recursively resolves dependencies
func (lm *LibraryManager) resolveDependenciesRecursive(ctx context.Context, deps []LibraryDependency, resolution *DependencyResolution, installedMap map[string]Library, versionRequirements map[string][]string, depth int) error {
	// Prevent infinite recursion
	if depth > 10 {
		return fmt.Errorf("dependency resolution depth exceeded")
	}

	for _, dep := range deps {
		versionRequirements[dep.Name] = append(versionRequirements[dep.Name], dep.Version)

		if installed, exists := installedMap[dep.Name]; exists {
			if lm.versionSatisfies(installed.Version, dep.Version) {
				continue
			}
		}

		// Check if already in install list
		alreadyQueued := false
		for _, install := range resolution.ToInstall {
			if install.Library.Name == dep.Name {
				alreadyQueued = true
				break
			}
		}
		if alreadyQueued {
			continue
		}

		available, err := lm.SearchLibrary(ctx, dep.Name)
		if err != nil {
			return fmt.Errorf("failed to search for dependency %s: %w", dep.Name, err)
		}

		bestMatch := lm.findBestVersion(available, dep.Version)
		if bestMatch == nil {
			resolution.HasConflicts = true
			resolution.Conflicts = append(resolution.Conflicts, DependencyConflict{
				LibraryName:      dep.Name,
				RequiredVersions: []string{dep.Version},
				Reason:           "Dependency not found",
			})
			continue
		}

		resolution.ToInstall = append(resolution.ToInstall, LibraryInstallation{
			Library: *bestMatch,
			Reason:  fmt.Sprintf("Dependency of other libraries (version %s)", dep.Version),
		})

		// Recursively resolve this library's dependencies
		if err := lm.resolveDependenciesRecursive(ctx, bestMatch.Dependencies, resolution, installedMap, versionRequirements, depth+1); err != nil {
			return err
		}
	}

	return nil
}

// findBestVersion finds the best matching version from available libraries
func (lm *LibraryManager) findBestVersion(available []Library, requiredVersion string) *Library {
	if len(available) == 0 {
		return nil
	}

	// If no specific version required, return latest
	if requiredVersion == "" || requiredVersion == "latest" {
		// Sort by version and return the latest
		sort.Slice(available, func(i, j int) bool {
			return lm.compareVersions(available[i].Version, available[j].Version) > 0
		})
		return &available[0]
	}

	// Find exact match first
	for _, lib := range available {
		if lib.Version == requiredVersion {
			return &lib
		}
	}

	// Find compatible version (semantic versioning)
	for _, lib := range available {
		if lm.versionSatisfies(lib.Version, requiredVersion) {
			return &lib
		}
	}

	return nil
}

// versionSatisfies checks if an installed version satisfies a requirement
func (lm *LibraryManager) versionSatisfies(installed, required string) bool {
	if required == "" || required == "latest" {
		return true
	}

	// Simple version comparison - in practice, this would use semantic versioning
	return installed == required || lm.compareVersions(installed, required) >= 0
}

// compareVersions compares two version strings
// Returns: 1 if v1 > v2, -1 if v1 < v2, 0 if equal
func (lm *LibraryManager) compareVersions(v1, v2 string) int {
	// Simple string comparison - in practice, this would parse semantic versions
	if v1 == v2 {
		return 0
	}
	if v1 > v2 {
		return 1
	}
	return -1
}

// getBoardArchitecture extracts architecture from FQBN
func (lm *LibraryManager) getBoardArchitecture(fqbn string) string {
	parts := strings.Split(fqbn, ":")
	if len(parts) >= 2 {
		return parts[1] // Return platform as architecture
	}
	return "unknown"
}

// uniqueStrings returns unique strings from a slice
func (lm *LibraryManager) uniqueStrings(strings []string) []string {
	keys := make(map[string]bool)
	var unique []string

	for _, str := range strings {
		if !keys[str] {
			keys[str] = true
			unique = append(unique, str)
		}
	}

	return unique
}

// invalidateInstalledCache invalidates the installed libraries cache
func (lm *LibraryManager) invalidateInstalledCache() {
	lm.cacheMutex.Lock()
	defer lm.cacheMutex.Unlock()
	lm.installedExpiry = time.Time{}
}
