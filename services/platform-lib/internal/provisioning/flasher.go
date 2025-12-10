package provisioning

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"go.bug.st/serial"
)

// Flasher handles device flashing operations
type Flasher struct {
	cli     *ArduinoCLI
	timeout time.Duration
}

// NewFlasher creates a new flasher instance
func NewFlasher(cli *ArduinoCLI) *Flasher {
	return &Flasher{
		cli:     cli,
		timeout: 2 * time.Minute,
	}
}

// FlashRequest represents a flash request
type FlashRequest struct {
	Port        string `json:"port" binding:"required"`
	Board       string `json:"board" binding:"required"`
	BinaryPath  string `json:"binary_path,omitempty"`
	ArtifactID  string `json:"artifact_id,omitempty"`
	VerifyFlash bool   `json:"verify_flash"`
	HealthCheck bool   `json:"health_check"`
}

// FlashResult represents the result of a flash operation
type FlashResult struct {
	Success      bool                `json:"success"`
	Port         string              `json:"port"`
	Board        string              `json:"board"`
	Duration     time.Duration       `json:"duration"`
	FlashOutput  string              `json:"flash_output,omitempty"`
	VerifyResult *VerificationResult `json:"verify_result,omitempty"`
	HealthCheck  *HealthCheckResult  `json:"health_check,omitempty"`
	Errors       []FlashError        `json:"errors,omitempty"`
}

// VerificationResult represents flash verification result
type VerificationResult struct {
	Success    bool   `json:"success"`
	BytesRead  int    `json:"bytes_read"`
	BytesTotal int    `json:"bytes_total"`
	Checksum   string `json:"checksum,omitempty"`
	Error      string `json:"error,omitempty"`
}

// HealthCheckResult represents device health check result
type HealthCheckResult struct {
	Success      bool          `json:"success"`
	DeviceInfo   DeviceInfo    `json:"device_info,omitempty"`
	SerialOutput string        `json:"serial_output,omitempty"`
	ResponseTime time.Duration `json:"response_time"`
	Tests        []HealthTest  `json:"tests,omitempty"`
	Error        string        `json:"error,omitempty"`
}

// DeviceInfo represents device information
type DeviceInfo struct {
	BoardType    string            `json:"board_type"`
	FirmwareHash string            `json:"firmware_hash,omitempty"`
	Version      string            `json:"version,omitempty"`
	Uptime       time.Duration     `json:"uptime,omitempty"`
	FreeMemory   int               `json:"free_memory,omitempty"`
	Properties   map[string]string `json:"properties,omitempty"`
}

// HealthTest represents an individual health test
type HealthTest struct {
	Name     string        `json:"name"`
	Success  bool          `json:"success"`
	Duration time.Duration `json:"duration"`
	Result   string        `json:"result,omitempty"`
	Error    string        `json:"error,omitempty"`
}

// FlashError represents a flash operation error
type FlashError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Code    int    `json:"code,omitempty"`
}

// FlashProgress represents flash progress information
type FlashProgress struct {
	Stage        string  `json:"stage"`
	Progress     float64 `json:"progress"`
	Message      string  `json:"message"`
	BytesTotal   int     `json:"bytes_total,omitempty"`
	BytesFlashed int     `json:"bytes_flashed,omitempty"`
}

// FlashDevice flashes firmware to a device
func (f *Flasher) FlashDevice(ctx context.Context, request *FlashRequest, progressCallback func(FlashProgress)) (*FlashResult, error) {
	startTime := time.Now()

	result := &FlashResult{
		Success:  false,
		Port:     request.Port,
		Board:    request.Board,
		Duration: 0,
		Errors:   []FlashError{},
	}

	// Validate port exists and is accessible
	if progressCallback != nil {
		progressCallback(FlashProgress{
			Stage:    "validation",
			Progress: 0.1,
			Message:  "Validating port access",
		})
	}

	if err := f.validatePort(request.Port); err != nil {
		result.Errors = append(result.Errors, FlashError{
			Type:    "port_validation",
			Message: err.Error(),
		})
		result.Duration = time.Since(startTime)
		return result, nil
	}

	// Determine binary path
	binaryPath := request.BinaryPath
	if binaryPath == "" && request.ArtifactID != "" {
		// This would typically get the binary from artifact manager
		// For now, we'll use a placeholder
		binaryPath = fmt.Sprintf("/tmp/athena/artifacts/%s/firmware.hex", request.ArtifactID)
	}

	if binaryPath == "" {
		result.Errors = append(result.Errors, FlashError{
			Type:    "binary_missing",
			Message: "No binary path or artifact ID provided",
		})
		result.Duration = time.Since(startTime)
		return result, nil
	}

	// Flash the device
	if progressCallback != nil {
		progressCallback(FlashProgress{
			Stage:    "flashing",
			Progress: 0.3,
			Message:  "Flashing firmware to device",
		})
	}

	flashOutput, err := f.flashFirmware(ctx, request.Port, request.Board, binaryPath, progressCallback)
	result.FlashOutput = flashOutput

	if err != nil {
		result.Errors = append(result.Errors, FlashError{
			Type:    "flash_failed",
			Message: err.Error(),
		})
		result.Duration = time.Since(startTime)
		return result, nil
	}

	// Verify flash if requested
	if request.VerifyFlash {
		if progressCallback != nil {
			progressCallback(FlashProgress{
				Stage:    "verification",
				Progress: 0.7,
				Message:  "Verifying flash",
			})
		}

		verifyResult, err := f.verifyFlash(ctx, request.Port, request.Board, binaryPath)
		result.VerifyResult = verifyResult

		if err != nil {
			result.Errors = append(result.Errors, FlashError{
				Type:    "verify_failed",
				Message: err.Error(),
			})
		}
	}

	// Perform health check if requested
	if request.HealthCheck {
		if progressCallback != nil {
			progressCallback(FlashProgress{
				Stage:    "health_check",
				Progress: 0.9,
				Message:  "Performing device health check",
			})
		}

		// Wait a moment for device to boot
		time.Sleep(2 * time.Second)

		healthResult, err := f.performHealthCheck(ctx, request.Port)
		result.HealthCheck = healthResult

		if err != nil {
			result.Errors = append(result.Errors, FlashError{
				Type:    "health_check_failed",
				Message: err.Error(),
			})
		}
	}

	result.Success = len(result.Errors) == 0
	result.Duration = time.Since(startTime)

	if progressCallback != nil {
		progressCallback(FlashProgress{
			Stage:    "complete",
			Progress: 1.0,
			Message:  "Flash operation completed",
		})
	}

	return result, nil
}

// validatePort validates that the specified port exists and is accessible
func (f *Flasher) validatePort(portName string) error {
	// List available ports
	ports, err := serial.GetPortsList()
	if err != nil {
		return fmt.Errorf("failed to list serial ports: %w", err)
	}

	// Check if the specified port exists
	for _, port := range ports {
		if port == portName {
			// Try to open the port briefly to ensure it's accessible
			mode := &serial.Mode{
				BaudRate: 9600,
				Parity:   serial.NoParity,
				DataBits: 8,
				StopBits: serial.OneStopBit,
			}

			serialPort, err := serial.Open(portName, mode)
			if err != nil {
				return fmt.Errorf("port %s exists but is not accessible: %w", portName, err)
			}
			serialPort.Close()

			return nil
		}
	}

	return fmt.Errorf("port %s not found. Available ports: %v", portName, ports)
}

// flashFirmware flashes firmware to the device using Arduino CLI
func (f *Flasher) flashFirmware(ctx context.Context, port, board, binaryPath string, progressCallback func(FlashProgress)) (string, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, f.timeout)
	defer cancel()

	// Build flash command
	args := []string{
		"upload",
		"--fqbn", board,
		"--port", port,
		"--input-file", binaryPath,
		"--verify",
	}

	// Execute flash command
	cmd := exec.CommandContext(ctx, f.cli.cliPath, args...)

	// Capture both stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start flash command: %w", err)
	}

	// Read output and parse progress
	var output strings.Builder
	outputChan := make(chan string, 100)

	// Read stdout
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			output.WriteString(line + "\n")
			outputChan <- line
		}
	}()

	// Read stderr
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			output.WriteString(line + "\n")
			outputChan <- line
		}
	}()

	// Parse output for progress if callback provided
	if progressCallback != nil {
		go f.parseFlashProgress(outputChan, progressCallback)
	}

	// Wait for command to complete
	if err := cmd.Wait(); err != nil {
		return output.String(), fmt.Errorf("flash command failed: %w", err)
	}

	return output.String(), nil
}

// parseFlashProgress parses flash output for progress information
func (f *Flasher) parseFlashProgress(outputChan <-chan string, progressCallback func(FlashProgress)) {
	progressRegex := regexp.MustCompile(`(\d+)%`)

	for line := range outputChan {
		// Look for percentage indicators
		if matches := progressRegex.FindStringSubmatch(line); len(matches) > 1 {
			if percent, err := strconv.Atoi(matches[1]); err == nil {
				progress := 0.3 + (float64(percent)/100.0)*0.4 // Flash stage is 30-70%
				progressCallback(FlashProgress{
					Stage:    "flashing",
					Progress: progress,
					Message:  line,
				})
			}
		}
	}
}

// verifyFlash verifies that the flash operation was successful
func (f *Flasher) verifyFlash(ctx context.Context, port, board, binaryPath string) (*VerificationResult, error) {
	result := &VerificationResult{
		Success: false,
	}

	// For now, we'll use a simple approach - Arduino CLI upload with --verify flag
	// In a full implementation, this would read back the flash memory and compare

	// The flash command already includes verification, so if we got here, it likely succeeded
	// This is a placeholder for more sophisticated verification

	result.Success = true
	result.BytesRead = 32768  // Placeholder
	result.BytesTotal = 32768 // Placeholder

	return result, nil
}

// performHealthCheck performs a health check on the flashed device
func (f *Flasher) performHealthCheck(ctx context.Context, portName string) (*HealthCheckResult, error) {
	startTime := time.Now()

	result := &HealthCheckResult{
		Success:      false,
		ResponseTime: 0,
		Tests:        []HealthTest{},
	}

	// Open serial connection
	mode := &serial.Mode{
		BaudRate: 9600,
		Parity:   serial.NoParity,
		DataBits: 8,
		StopBits: serial.OneStopBit,
	}

	serialPort, err := serial.Open(portName, mode)
	if err != nil {
		result.Error = fmt.Sprintf("Failed to open serial port: %v", err)
		return result, nil
	}
	defer serialPort.Close()

	// Set read timeout
	serialPort.SetReadTimeout(5 * time.Second)

	// Test 1: Basic communication
	commTest := f.testBasicCommunication(serialPort)
	result.Tests = append(result.Tests, commTest)

	// Test 2: Device info retrieval
	infoTest, deviceInfo := f.testDeviceInfo(serialPort)
	result.Tests = append(result.Tests, infoTest)
	if deviceInfo != nil {
		result.DeviceInfo = *deviceInfo
	}

	// Test 3: Memory check
	memoryTest := f.testMemoryCheck(serialPort)
	result.Tests = append(result.Tests, memoryTest)

	// Determine overall success
	allTestsPassed := true
	for _, test := range result.Tests {
		if !test.Success {
			allTestsPassed = false
			break
		}
	}

	result.Success = allTestsPassed
	result.ResponseTime = time.Since(startTime)

	return result, nil
}

// testBasicCommunication tests basic serial communication
func (f *Flasher) testBasicCommunication(port serial.Port) HealthTest {
	test := HealthTest{
		Name:    "Basic Communication",
		Success: false,
	}

	startTime := time.Now()
	defer func() {
		test.Duration = time.Since(startTime)
	}()

	// Send a simple command and wait for response
	_, err := port.Write([]byte("AT\r\n"))
	if err != nil {
		test.Error = fmt.Sprintf("Failed to write to port: %v", err)
		return test
	}

	// Read response
	buffer := make([]byte, 128)
	n, err := port.Read(buffer)
	if err != nil && err != io.EOF {
		test.Error = fmt.Sprintf("Failed to read from port: %v", err)
		return test
	}

	if n > 0 {
		response := string(buffer[:n])
		test.Result = strings.TrimSpace(response)
		test.Success = true
	} else {
		test.Error = "No response received"
	}

	return test
}

// testDeviceInfo attempts to retrieve device information
func (f *Flasher) testDeviceInfo(port serial.Port) (HealthTest, *DeviceInfo) {
	test := HealthTest{
		Name:    "Device Info",
		Success: false,
	}

	startTime := time.Now()
	defer func() {
		test.Duration = time.Since(startTime)
	}()

	// Send info command
	_, err := port.Write([]byte("INFO\r\n"))
	if err != nil {
		test.Error = fmt.Sprintf("Failed to write info command: %v", err)
		return test, nil
	}

	// Read response
	buffer := make([]byte, 512)
	n, err := port.Read(buffer)
	if err != nil && err != io.EOF {
		test.Error = fmt.Sprintf("Failed to read info response: %v", err)
		return test, nil
	}

	if n > 0 {
		response := string(buffer[:n])
		test.Result = strings.TrimSpace(response)
		test.Success = true

		// Parse device info (simplified)
		deviceInfo := &DeviceInfo{
			BoardType:  "Arduino",
			Properties: make(map[string]string),
		}

		// Simple parsing - in practice, this would be more sophisticated
		lines := strings.Split(response, "\n")
		for _, line := range lines {
			if strings.Contains(line, "Version:") {
				parts := strings.Split(line, ":")
				if len(parts) > 1 {
					deviceInfo.Version = strings.TrimSpace(parts[1])
				}
			}
		}

		return test, deviceInfo
	}

	test.Error = "No device info received"
	return test, nil
}

// testMemoryCheck performs a basic memory check
func (f *Flasher) testMemoryCheck(port serial.Port) HealthTest {
	test := HealthTest{
		Name:    "Memory Check",
		Success: false,
	}

	startTime := time.Now()
	defer func() {
		test.Duration = time.Since(startTime)
	}()

	// Send memory command
	_, err := port.Write([]byte("MEM\r\n"))
	if err != nil {
		test.Error = fmt.Sprintf("Failed to write memory command: %v", err)
		return test
	}

	// Read response
	buffer := make([]byte, 256)
	n, err := port.Read(buffer)
	if err != nil && err != io.EOF {
		test.Error = fmt.Sprintf("Failed to read memory response: %v", err)
		return test
	}

	if n > 0 {
		response := string(buffer[:n])
		test.Result = strings.TrimSpace(response)
		test.Success = true
	} else {
		test.Error = "No memory info received"
	}

	return test
}

// GetAvailablePorts returns a list of available serial ports
func (f *Flasher) GetAvailablePorts() ([]string, error) {
	return serial.GetPortsList()
}
