package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Profile represents a CLI profile configuration
type Profile struct {
	Name            string            `yaml:"name"`
	TemplateID      string            `yaml:"template_id"`
	TemplateVersion string            `yaml:"template_version,omitempty"`
	Board           string            `yaml:"board,omitempty"`
	Port            string            `yaml:"port,omitempty"`
	Parameters      map[string]string `yaml:"parameters,omitempty"`
	Metadata        map[string]string `yaml:"metadata,omitempty"`
}

// ProfileConfig represents the CLI profile configuration file
type ProfileConfig struct {
	CurrentProfile string             `yaml:"current_profile"`
	Profiles       map[string]Profile `yaml:"profiles"`
}

// ProfileManager manages CLI profiles
type ProfileManager struct {
	configPath string
	config     *ProfileConfig
}

// NewProfileManager creates a new profile manager
func NewProfileManager() (*ProfileManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	athenaDir := filepath.Join(homeDir, ".athena")
	if err := os.MkdirAll(athenaDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create .athena directory: %w", err)
	}

	configPath := filepath.Join(athenaDir, "cli.yaml")
	pm := &ProfileManager{
		configPath: configPath,
	}

	if err := pm.load(); err != nil {
		// Initialize with empty config if file doesn't exist
		if os.IsNotExist(err) {
			pm.config = &ProfileConfig{
				CurrentProfile: "default",
				Profiles: map[string]Profile{
					"default": {
						Name: "default",
					},
				},
			}
			if err := pm.save(); err != nil {
				return nil, fmt.Errorf("failed to initialize profile config: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to load profile config: %w", err)
		}
	}

	return pm, nil
}

// load loads the profile configuration from disk
func (pm *ProfileManager) load() error {
	data, err := os.ReadFile(pm.configPath)
	if err != nil {
		return err
	}

	var config ProfileConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse profile config: %w", err)
	}

	pm.config = &config
	return nil
}

// save saves the profile configuration to disk
func (pm *ProfileManager) save() error {
	data, err := yaml.Marshal(pm.config)
	if err != nil {
		return fmt.Errorf("failed to marshal profile config: %w", err)
	}

	if err := os.WriteFile(pm.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write profile config: %w", err)
	}

	return nil
}

// GetCurrentProfile returns the current profile
func (pm *ProfileManager) GetCurrentProfile() (*Profile, error) {
	if pm.config.CurrentProfile == "" {
		return nil, fmt.Errorf("no current profile set")
	}

	profile, exists := pm.config.Profiles[pm.config.CurrentProfile]
	if !exists {
		return nil, fmt.Errorf("current profile '%s' does not exist", pm.config.CurrentProfile)
	}

	return &profile, nil
}

// SetCurrentProfile sets the current profile
func (pm *ProfileManager) SetCurrentProfile(name string) error {
	if _, exists := pm.config.Profiles[name]; !exists {
		return fmt.Errorf("profile '%s' does not exist", name)
	}

	pm.config.CurrentProfile = name
	return pm.save()
}

// ListProfiles returns all profiles
func (pm *ProfileManager) ListProfiles() map[string]Profile {
	return pm.config.Profiles
}

// GetProfile returns a specific profile by name
func (pm *ProfileManager) GetProfile(name string) (*Profile, error) {
	profile, exists := pm.config.Profiles[name]
	if !exists {
		return nil, fmt.Errorf("profile '%s' does not exist", name)
	}

	return &profile, nil
}

// CreateProfile creates a new profile
func (pm *ProfileManager) CreateProfile(name string, profile Profile) error {
	if _, exists := pm.config.Profiles[name]; exists {
		return fmt.Errorf("profile '%s' already exists", name)
	}

	profile.Name = name
	pm.config.Profiles[name] = profile

	// If this is the first profile, make it current
	if pm.config.CurrentProfile == "" {
		pm.config.CurrentProfile = name
	}

	return pm.save()
}

// UpdateProfile updates an existing profile
func (pm *ProfileManager) UpdateProfile(name string, profile Profile) error {
	if _, exists := pm.config.Profiles[name]; !exists {
		return fmt.Errorf("profile '%s' does not exist", name)
	}

	profile.Name = name
	pm.config.Profiles[name] = profile
	return pm.save()
}

// DeleteProfile deletes a profile
func (pm *ProfileManager) DeleteProfile(name string) error {
	if _, exists := pm.config.Profiles[name]; !exists {
		return fmt.Errorf("profile '%s' does not exist", name)
	}

	// Don't allow deleting the current profile if it's the only one
	if pm.config.CurrentProfile == name && len(pm.config.Profiles) == 1 {
		return fmt.Errorf("cannot delete the only profile")
	}

	delete(pm.config.Profiles, name)

	// If we deleted the current profile, switch to another one
	if pm.config.CurrentProfile == name {
		// Find any other profile to make current
		for profileName := range pm.config.Profiles {
			pm.config.CurrentProfile = profileName
			break
		}
	}

	return pm.save()
}

// UpdateCurrentProfile updates the current profile with new values
func (pm *ProfileManager) UpdateCurrentProfile(updates map[string]interface{}) error {
	profile, err := pm.GetCurrentProfile()
	if err != nil {
		return err
	}

	// Apply updates
	if templateID, ok := updates["template_id"].(string); ok {
		profile.TemplateID = templateID
	}
	if templateVersion, ok := updates["template_version"].(string); ok {
		profile.TemplateVersion = templateVersion
	}
	if board, ok := updates["board"].(string); ok {
		profile.Board = board
	}
	if port, ok := updates["port"].(string); ok {
		profile.Port = port
	}
	if parameters, ok := updates["parameters"].(map[string]string); ok {
		if profile.Parameters == nil {
			profile.Parameters = make(map[string]string)
		}
		for k, v := range parameters {
			profile.Parameters[k] = v
		}
	}

	return pm.UpdateProfile(profile.Name, *profile)
}
