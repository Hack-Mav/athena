package provisioning

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// BoardManager manages Arduino boards and their capabilities
type BoardManager struct {
	cli           *ArduinoCLI
	boardsCache   map[string]Board
	cacheMutex    sync.RWMutex
	cacheExpiry   time.Time
	cacheDuration time.Duration
}

// NewBoardManager creates a new board manager
func NewBoardManager(cli *ArduinoCLI) *BoardManager {
	return &BoardManager{
		cli:           cli,
		boardsCache:   make(map[string]Board),
		cacheDuration: 30 * time.Minute,
	}
}

// BoardCompatibility represents compatibility information
type BoardCompatibility struct {
	Compatible bool     `json:"compatible"`
	Reasons    []string `json:"reasons,omitempty"`
	Warnings   []string `json:"warnings,omitempty"`
}

// PinAssignment represents a pin assignment
type PinAssignment struct {
	Pin       int    `json:"pin"`
	Function  string `json:"function"`
	Component string `json:"component"`
}

// PinConflictCheck represents pin conflict validation
type PinConflictCheck struct {
	HasConflicts bool            `json:"has_conflicts"`
	Conflicts    []PinConflict   `json:"conflicts,omitempty"`
	Assignments  []PinAssignment `json:"assignments"`
}

// PinConflict represents a pin assignment conflict
type PinConflict struct {
	Pin        int      `json:"pin"`
	Components []string `json:"components"`
	Reason     string   `json:"reason"`
}

// GetBoard retrieves board information by FQBN
func (bm *BoardManager) GetBoard(ctx context.Context, fqbn string) (*Board, error) {
	bm.cacheMutex.RLock()
	if board, exists := bm.boardsCache[fqbn]; exists && time.Now().Before(bm.cacheExpiry) {
		bm.cacheMutex.RUnlock()
		return &board, nil
	}
	bm.cacheMutex.RUnlock()

	// Refresh cache if expired or board not found
	if err := bm.refreshBoardsCache(ctx); err != nil {
		return nil, fmt.Errorf("failed to refresh boards cache: %w", err)
	}

	bm.cacheMutex.RLock()
	defer bm.cacheMutex.RUnlock()

	if board, exists := bm.boardsCache[fqbn]; exists {
		return &board, nil
	}

	return nil, fmt.Errorf("board with FQBN %s not found", fqbn)
}

// ListBoards returns all available boards
func (bm *BoardManager) ListBoards(ctx context.Context) ([]Board, error) {
	bm.cacheMutex.RLock()
	if time.Now().Before(bm.cacheExpiry) && len(bm.boardsCache) > 0 {
		boards := make([]Board, 0, len(bm.boardsCache))
		for _, board := range bm.boardsCache {
			boards = append(boards, board)
		}
		bm.cacheMutex.RUnlock()
		return boards, nil
	}
	bm.cacheMutex.RUnlock()

	if err := bm.refreshBoardsCache(ctx); err != nil {
		return nil, fmt.Errorf("failed to refresh boards cache: %w", err)
	}

	bm.cacheMutex.RLock()
	defer bm.cacheMutex.RUnlock()

	boards := make([]Board, 0, len(bm.boardsCache))
	for _, board := range bm.boardsCache {
		boards = append(boards, board)
	}

	return boards, nil
}

// DetectConnectedBoards detects boards connected via USB
func (bm *BoardManager) DetectConnectedBoards(ctx context.Context) ([]Port, error) {
	return bm.cli.DetectBoards(ctx)
}

// ValidateBoardCompatibility checks if a board is compatible with requirements
func (bm *BoardManager) ValidateBoardCompatibility(ctx context.Context, fqbn string, requirements BoardRequirements) (*BoardCompatibility, error) {
	board, err := bm.GetBoard(ctx, fqbn)
	if err != nil {
		return &BoardCompatibility{
			Compatible: false,
			Reasons:    []string{fmt.Sprintf("Board not found: %s", err.Error())},
		}, nil
	}

	compatibility := &BoardCompatibility{
		Compatible: true,
		Reasons:    []string{},
		Warnings:   []string{},
	}

	// Check voltage requirements
	if requirements.Voltage > 0 && board.Capabilities.Voltage != requirements.Voltage {
		if abs(board.Capabilities.Voltage-requirements.Voltage) > 0.5 {
			compatibility.Compatible = false
			compatibility.Reasons = append(compatibility.Reasons,
				fmt.Sprintf("Voltage mismatch: board provides %.1fV, required %.1fV",
					board.Capabilities.Voltage, requirements.Voltage))
		} else {
			compatibility.Warnings = append(compatibility.Warnings,
				fmt.Sprintf("Minor voltage difference: board provides %.1fV, required %.1fV",
					board.Capabilities.Voltage, requirements.Voltage))
		}
	}

	// Check current requirements
	if requirements.MaxCurrent > board.Capabilities.MaxCurrent {
		compatibility.Compatible = false
		compatibility.Reasons = append(compatibility.Reasons,
			fmt.Sprintf("Insufficient current capacity: board provides %dmA, required %dmA",
				board.Capabilities.MaxCurrent, requirements.MaxCurrent))
	}

	// Check digital pins
	if requirements.DigitalPins > len(board.Capabilities.DigitalPins) {
		compatibility.Compatible = false
		compatibility.Reasons = append(compatibility.Reasons,
			fmt.Sprintf("Insufficient digital pins: board has %d, required %d",
				len(board.Capabilities.DigitalPins), requirements.DigitalPins))
	}

	// Check analog pins
	if requirements.AnalogPins > len(board.Capabilities.AnalogPins) {
		compatibility.Compatible = false
		compatibility.Reasons = append(compatibility.Reasons,
			fmt.Sprintf("Insufficient analog pins: board has %d, required %d",
				len(board.Capabilities.AnalogPins), requirements.AnalogPins))
	}

	// Check PWM pins
	if requirements.PWMPins > len(board.Capabilities.PWMPins) {
		compatibility.Compatible = false
		compatibility.Reasons = append(compatibility.Reasons,
			fmt.Sprintf("Insufficient PWM pins: board has %d, required %d",
				len(board.Capabilities.PWMPins), requirements.PWMPins))
	}

	// Check communication interfaces
	if requirements.I2C && len(board.Capabilities.I2CPins) < 2 {
		compatibility.Compatible = false
		compatibility.Reasons = append(compatibility.Reasons, "I2C interface not available")
	}

	if requirements.SPI && len(board.Capabilities.SPIPins) < 4 {
		compatibility.Compatible = false
		compatibility.Reasons = append(compatibility.Reasons, "SPI interface not available")
	}

	if requirements.SerialPorts > board.Capabilities.SerialPorts {
		compatibility.Compatible = false
		compatibility.Reasons = append(compatibility.Reasons,
			fmt.Sprintf("Insufficient serial ports: board has %d, required %d",
				board.Capabilities.SerialPorts, requirements.SerialPorts))
	}

	// Check memory requirements
	if requirements.MinFlashSize > board.Capabilities.FlashSize {
		compatibility.Compatible = false
		compatibility.Reasons = append(compatibility.Reasons,
			fmt.Sprintf("Insufficient flash memory: board has %dKB, required %dKB",
				board.Capabilities.FlashSize, requirements.MinFlashSize))
	}

	if requirements.MinRAMSize > board.Capabilities.RAMSize {
		compatibility.Compatible = false
		compatibility.Reasons = append(compatibility.Reasons,
			fmt.Sprintf("Insufficient RAM: board has %dKB, required %dKB",
				board.Capabilities.RAMSize, requirements.MinRAMSize))
	}

	// Check special features
	for feature, required := range requirements.Features {
		if required {
			if value, exists := board.Capabilities.Properties[feature]; !exists || value != "true" {
				compatibility.Warnings = append(compatibility.Warnings,
					fmt.Sprintf("Feature '%s' may not be available", feature))
			}
		}
	}

	return compatibility, nil
}

// ValidatePinAssignments checks for pin conflicts in assignments
func (bm *BoardManager) ValidatePinAssignments(ctx context.Context, fqbn string, assignments []PinAssignment) (*PinConflictCheck, error) {
	board, err := bm.GetBoard(ctx, fqbn)
	if err != nil {
		return nil, fmt.Errorf("failed to get board: %w", err)
	}

	check := &PinConflictCheck{
		HasConflicts: false,
		Conflicts:    []PinConflict{},
		Assignments:  assignments,
	}

	// Track pin usage
	pinUsage := make(map[int][]string)
	for _, assignment := range assignments {
		pinUsage[assignment.Pin] = append(pinUsage[assignment.Pin], assignment.Component)
	}

	// Check for conflicts
	for pin, components := range pinUsage {
		if len(components) > 1 {
			// Multiple components on same pin
			check.HasConflicts = true
			check.Conflicts = append(check.Conflicts, PinConflict{
				Pin:        pin,
				Components: components,
				Reason:     "Multiple components assigned to same pin",
			})
		}

		// Validate pin exists on board
		if !bm.isPinAvailable(board, pin) {
			check.HasConflicts = true
			check.Conflicts = append(check.Conflicts, PinConflict{
				Pin:        pin,
				Components: components,
				Reason:     "Pin not available on this board",
			})
		}
	}

	// Check for special pin conflicts (I2C, SPI)
	i2cPins := make(map[int]bool)
	for _, pin := range board.Capabilities.I2CPins {
		i2cPins[pin] = true
	}

	spiPins := make(map[int]bool)
	for _, pin := range board.Capabilities.SPIPins {
		spiPins[pin] = true
	}

	for _, assignment := range assignments {
		// Check if I2C pins are being used for other purposes
		if i2cPins[assignment.Pin] && assignment.Function != "i2c" {
			check.Conflicts = append(check.Conflicts, PinConflict{
				Pin:        assignment.Pin,
				Components: []string{assignment.Component},
				Reason:     "Pin is reserved for I2C communication",
			})
		}

		// Check if SPI pins are being used for other purposes
		if spiPins[assignment.Pin] && assignment.Function != "spi" {
			check.Conflicts = append(check.Conflicts, PinConflict{
				Pin:        assignment.Pin,
				Components: []string{assignment.Component},
				Reason:     "Pin is reserved for SPI communication",
			})
		}
	}

	return check, nil
}

// BoardRequirements represents board requirements for a template
type BoardRequirements struct {
	Voltage      float32         `json:"voltage"`
	MaxCurrent   int             `json:"max_current_ma"`
	DigitalPins  int             `json:"digital_pins"`
	AnalogPins   int             `json:"analog_pins"`
	PWMPins      int             `json:"pwm_pins"`
	I2C          bool            `json:"i2c"`
	SPI          bool            `json:"spi"`
	SerialPorts  int             `json:"serial_ports"`
	MinFlashSize int             `json:"min_flash_size_kb"`
	MinRAMSize   int             `json:"min_ram_size_kb"`
	Features     map[string]bool `json:"features"`
}

// refreshBoardsCache refreshes the internal boards cache
func (bm *BoardManager) refreshBoardsCache(ctx context.Context) error {
	boards, err := bm.cli.ListBoards(ctx)
	if err != nil {
		return err
	}

	bm.cacheMutex.Lock()
	defer bm.cacheMutex.Unlock()

	bm.boardsCache = make(map[string]Board)
	for _, board := range boards {
		bm.boardsCache[board.FQBN] = board
	}
	bm.cacheExpiry = time.Now().Add(bm.cacheDuration)

	return nil
}

// isPinAvailable checks if a pin is available on the board
func (bm *BoardManager) isPinAvailable(board *Board, pin int) bool {
	// Check digital pins
	for _, p := range board.Capabilities.DigitalPins {
		if p == pin {
			return true
		}
	}

	// Check analog pins
	for _, p := range board.Capabilities.AnalogPins {
		if p == pin {
			return true
		}
	}

	return false
}

// abs returns the absolute value of a float32
func abs(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}
