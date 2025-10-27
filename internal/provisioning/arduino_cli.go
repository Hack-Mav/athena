package provisioning

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// ArduinoCLI represents the Arduino CLI wrapper
type ArduinoCLI struct {
	cliPath string
	timeout time.Duration
}

// NewArduinoCLI creates a new Arduino CLI wrapper
func NewArduinoCLI(cliPath string) *ArduinoCLI {
	if cliPath == "" {
		cliPath = "arduino-cli"
	}
	return &ArduinoCLI{
		cliPath: cliPath,
		timeout: 5 * time.Minute,
	}
}

// Board represents an Arduino board
type Board struct {
	FQBN         string            `json:"fqbn"`
	Name         string            `json:"name"`
	Platform     string            `json:"platform"`
	Capabilities BoardCapabilities `json:"capabilities"`
}

// BoardCapabilities represents the capabilities of a board
type BoardCapabilities struct {
	DigitalPins  []int             `json:"digital_pins"`
	AnalogPins   []int             `json:"analog_pins"`
	PWMPins      []int             `json:"pwm_pins"`
	I2CPins      []int             `json:"i2c_pins"`
	SPIPins      []int             `json:"spi_pins"`
	SerialPorts  int               `json:"serial_ports"`
	Voltage      float32           `json:"voltage"`
	MaxCurrent   int               `json:"max_current_ma"`
	FlashSize    int               `json:"flash_size_kb"`
	RAMSize      int               `json:"ram_size_kb"`
	Properties   map[string]string `json:"properties"`
}

// Library represents an Arduino library
type Library struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Author      string `json:"author"`
	Maintainer  string `json:"maintainer"`
	Sentence    string `json:"sentence"`
	Paragraph   string `json:"paragraph"`
	Website     string `json:"website"`
	Category    string `json:"category"`
	Architectures []string `json:"architectures"`
	Types       []string `json:"types"`
	Repository  string `json:"repository"`
	Dependencies []LibraryDependency `json:"dependencies"`
}

// LibraryDependency represents a library dependency
type LibraryDependency struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Port represents a serial port
type Port struct {
	Address       string            `json:"address"`
	Label         string            `json:"label"`
	Protocol      string            `json:"protocol"`
	ProtocolLabel string            `json:"protocol_label"`
	Properties    map[string]string `json:"properties"`
	HardwareID    string            `json:"hardware_id"`
}

// ExecuteCommand executes an Arduino CLI command with timeout
func (a *ArduinoCLI) ExecuteCommand(ctx context.Context, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, a.cliPath, args...)
	output, err := cmd.CombinedOutput()
	
	if err != nil {
		return output, fmt.Errorf("arduino-cli command failed: %w, output: %s", err, string(output))
	}
	
	return output, nil
}

// Version returns the Arduino CLI version
func (a *ArduinoCLI) Version(ctx context.Context) (string, error) {
	output, err := a.ExecuteCommand(ctx, "version", "--format", "json")
	if err != nil {
		return "", err
	}

	var versionInfo struct {
		Version string `json:"version"`
	}
	
	if err := json.Unmarshal(output, &versionInfo); err != nil {
		return "", fmt.Errorf("failed to parse version output: %w", err)
	}

	return versionInfo.Version, nil
}

// UpdateIndex updates the package index
func (a *ArduinoCLI) UpdateIndex(ctx context.Context) error {
	_, err := a.ExecuteCommand(ctx, "core", "update-index")
	return err
}

// InstallCore installs a core platform
func (a *ArduinoCLI) InstallCore(ctx context.Context, core string) error {
	_, err := a.ExecuteCommand(ctx, "core", "install", core)
	return err
}

// ListBoards returns available boards
func (a *ArduinoCLI) ListBoards(ctx context.Context) ([]Board, error) {
	output, err := a.ExecuteCommand(ctx, "board", "listall", "--format", "json")
	if err != nil {
		return nil, err
	}

	var boardsResponse struct {
		Boards []struct {
			Name     string `json:"name"`
			FQBN     string `json:"fqbn"`
			Platform struct {
				ID       string `json:"id"`
				Installed string `json:"installed"`
			} `json:"platform"`
		} `json:"boards"`
	}

	if err := json.Unmarshal(output, &boardsResponse); err != nil {
		return nil, fmt.Errorf("failed to parse boards output: %w", err)
	}

	boards := make([]Board, 0, len(boardsResponse.Boards))
	for _, b := range boardsResponse.Boards {
		board := Board{
			FQBN:     b.FQBN,
			Name:     b.Name,
			Platform: b.Platform.ID,
			Capabilities: a.getBoardCapabilities(b.FQBN),
		}
		boards = append(boards, board)
	}

	return boards, nil
}

// DetectBoards detects connected boards
func (a *ArduinoCLI) DetectBoards(ctx context.Context) ([]Port, error) {
	output, err := a.ExecuteCommand(ctx, "board", "list", "--format", "json")
	if err != nil {
		return nil, err
	}

	var portsResponse []Port
	if err := json.Unmarshal(output, &portsResponse); err != nil {
		return nil, fmt.Errorf("failed to parse ports output: %w", err)
	}

	return portsResponse, nil
}

// InstallLibrary installs a library
func (a *ArduinoCLI) InstallLibrary(ctx context.Context, library string) error {
	_, err := a.ExecuteCommand(ctx, "lib", "install", library)
	return err
}

// SearchLibrary searches for libraries
func (a *ArduinoCLI) SearchLibrary(ctx context.Context, query string) ([]Library, error) {
	output, err := a.ExecuteCommand(ctx, "lib", "search", query, "--format", "json")
	if err != nil {
		return nil, err
	}

	var searchResponse struct {
		Libraries []Library `json:"libraries"`
	}

	if err := json.Unmarshal(output, &searchResponse); err != nil {
		return nil, fmt.Errorf("failed to parse library search output: %w", err)
	}

	return searchResponse.Libraries, nil
}

// ListInstalledLibraries returns installed libraries
func (a *ArduinoCLI) ListInstalledLibraries(ctx context.Context) ([]Library, error) {
	output, err := a.ExecuteCommand(ctx, "lib", "list", "--format", "json")
	if err != nil {
		return nil, err
	}

	var libResponse struct {
		InstalledLibraries []Library `json:"installed_libraries"`
	}

	if err := json.Unmarshal(output, &libResponse); err != nil {
		return nil, fmt.Errorf("failed to parse installed libraries output: %w", err)
	}

	return libResponse.InstalledLibraries, nil
}

// getBoardCapabilities returns board capabilities based on FQBN
// This is a simplified implementation - in practice, this would query
// board specifications or maintain a capabilities database
func (a *ArduinoCLI) getBoardCapabilities(fqbn string) BoardCapabilities {
	// Default capabilities for common boards
	capabilities := BoardCapabilities{
		Properties: make(map[string]string),
	}

	// Parse FQBN to determine board type
	parts := strings.Split(fqbn, ":")
	if len(parts) >= 3 {
		boardType := parts[2]
		
		switch {
		case strings.Contains(boardType, "uno"):
			capabilities = BoardCapabilities{
				DigitalPins: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13},
				AnalogPins:  []int{14, 15, 16, 17, 18, 19}, // A0-A5
				PWMPins:     []int{3, 5, 6, 9, 10, 11},
				I2CPins:     []int{18, 19}, // A4, A5
				SPIPins:     []int{10, 11, 12, 13},
				SerialPorts: 1,
				Voltage:     5.0,
				MaxCurrent:  500,
				FlashSize:   32,
				RAMSize:     2,
				Properties:  map[string]string{"mcu": "atmega328p"},
			}
		case strings.Contains(boardType, "nano"):
			capabilities = BoardCapabilities{
				DigitalPins: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13},
				AnalogPins:  []int{14, 15, 16, 17, 18, 19, 20, 21}, // A0-A7
				PWMPins:     []int{3, 5, 6, 9, 10, 11},
				I2CPins:     []int{18, 19}, // A4, A5
				SPIPins:     []int{10, 11, 12, 13},
				SerialPorts: 1,
				Voltage:     5.0,
				MaxCurrent:  500,
				FlashSize:   32,
				RAMSize:     2,
				Properties:  map[string]string{"mcu": "atmega328p"},
			}
		case strings.Contains(boardType, "esp32"):
			capabilities = BoardCapabilities{
				DigitalPins: []int{0, 1, 2, 3, 4, 5, 12, 13, 14, 15, 16, 17, 18, 19, 21, 22, 23, 25, 26, 27, 32, 33},
				AnalogPins:  []int{32, 33, 34, 35, 36, 37, 38, 39},
				PWMPins:     []int{0, 1, 2, 3, 4, 5, 12, 13, 14, 15, 16, 17, 18, 19, 21, 22, 23, 25, 26, 27},
				I2CPins:     []int{21, 22},
				SPIPins:     []int{5, 18, 19, 23},
				SerialPorts: 3,
				Voltage:     3.3,
				MaxCurrent:  1000,
				FlashSize:   4096,
				RAMSize:     520,
				Properties:  map[string]string{"mcu": "esp32", "wifi": "true", "bluetooth": "true"},
			}
		default:
			// Generic Arduino-compatible board
			capabilities = BoardCapabilities{
				DigitalPins: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13},
				AnalogPins:  []int{14, 15, 16, 17, 18, 19},
				PWMPins:     []int{3, 5, 6, 9, 10, 11},
				I2CPins:     []int{18, 19},
				SPIPins:     []int{10, 11, 12, 13},
				SerialPorts: 1,
				Voltage:     5.0,
				MaxCurrent:  500,
				FlashSize:   32,
				RAMSize:     2,
				Properties:  map[string]string{"mcu": "unknown"},
			}
		}
	}

	return capabilities
}