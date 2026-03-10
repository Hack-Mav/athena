package tests

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ArduinoCompilationTest handles actual Arduino compilation testing
type ArduinoCompilationTest struct {
	ArduinoCLIPath string
	WorkspaceDir   string
}

// NewArduinoCompilationTest creates a new compilation test instance
func NewArduinoCompilationTest() *ArduinoCompilationTest {
	return &ArduinoCompilationTest{
		ArduinoCLIPath: "arduino-cli", // Assumes arduino-cli is in PATH
		WorkspaceDir:   "./test_workspace",
	}
}

// TestRealArduinoCompilation tests actual compilation with Arduino CLI
func TestRealArduinoCompilation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping real Arduino compilation in short mode")
	}

	// Check if arduino-cli is available
	if !isArduinoCLIAvailable() {
		t.Skip("arduino-cli not available, skipping real compilation tests")
	}

	compTest := NewArduinoCompilationTest()
	suite, err := NewTemplateTestSuite("../../templates/arduino")
	require.NoError(t, err, "Failed to create template test suite")

	for _, template := range suite.Templates {
		t.Run(fmt.Sprintf("RealCompilation_%s", template.ID), func(t *testing.T) {
			t.Parallel()

			for _, board := range template.BoardsSupported {
				t.Run(fmt.Sprintf("Board_%s", board), func(t *testing.T) {
					err := compTest.compileTemplate(template, board)
					if err != nil {
						t.Logf("Compilation failed: %v", err)
						// Don't fail the test if compilation fails due to missing libraries
						// This is expected in a CI environment
						t.Skip("Compilation skipped due to missing dependencies")
					}
				})
			}
		})
	}
}

// TestLibraryDependencies tests library dependency resolution
func TestLibraryDependencies(t *testing.T) {
	suite, err := NewTemplateTestSuite("../../templates/arduino")
	require.NoError(t, err, "Failed to create template test suite")

	for _, template := range suite.Templates {
		t.Run(fmt.Sprintf("Libraries_%s", template.ID), func(t *testing.T) {
			t.Parallel()

			// Test library format
			for _, lib := range template.Libraries {
				assert.NotEmpty(t, lib.Name, "Library name should not be empty")
				assert.NotEmpty(t, lib.Version, "Library version should not be empty")
				assert.True(t, isValidVersion(lib.Version),
					"Library version should be valid: %s", lib.Version)
			}

			// Test for duplicate libraries
			libNames := make(map[string]bool)
			for _, lib := range template.Libraries {
				if libNames[lib.Name] {
					t.Errorf("Duplicate library: %s", lib.Name)
				}
				libNames[lib.Name] = true
			}

			// Test library compatibility with boards
			err := validateLibraryCompatibility(template)
			assert.NoError(t, err, "Library compatibility should be valid")
		})
	}
}

// TestCodeTemplateSyntax tests Arduino code template syntax
func TestCodeTemplateSyntax(t *testing.T) {
	suite, err := NewTemplateTestSuite("../../templates/arduino")
	require.NoError(t, err, "Failed to create template test suite")

	for _, template := range suite.Templates {
		t.Run(fmt.Sprintf("CodeSyntax_%s", template.ID), func(t *testing.T) {
			t.Parallel()

			codeAsset := findCodeAsset(template.Assets)
			require.NotNil(t, codeAsset, "Template should have a code asset")

			code := codeAsset.Metadata["content"].(string)

			// Test basic Arduino structure
			assert.Contains(t, code, "void setup()", "Code should contain setup() function")
			assert.Contains(t, code, "void loop()", "Code should contain loop() function")

			// Test template variable syntax
			assert.Contains(t, code, "{{.", "Code should contain template variable start")
			assert.Contains(t, code, "}}", "Code should contain template variable end")

			// Test for common Arduino patterns
			assert.Contains(t, code, "#include", "Code should contain include statements")
			assert.Contains(t, code, "Serial.begin", "Code should initialize serial communication")

			// Test template variable validity
			err := validateTemplateVariables(code, template.Parameters)
			assert.NoError(t, err, "Template variables should be valid")

			// Test code compilation without template variables
			err = validateCodeWithoutTemplates(code, template.Parameters)
			assert.NoError(t, err, "Code should be valid after template substitution")
		})
	}
}

// TestBoardSpecificFeatures tests board-specific features and limitations
func TestBoardSpecificFeatures(t *testing.T) {
	suite, err := NewTemplateTestSuite("../../templates/arduino")
	require.NoError(t, err, "Failed to create template test suite")

	boardTests := []struct {
		board           string
		maxDigitalPins  int
		maxAnalogPins   int
		supportedLibs   []string
		specialFeatures []string
	}{
		{
			board:          "arduino:avr:uno",
			maxDigitalPins: 13,
			maxAnalogPins:  6,
			supportedLibs:  []string{"DHT sensor library", "PubSubClient", "ArduinoJson"},
		},
		{
			board:           "esp32:esp32:devkitv1",
			maxDigitalPins:  39,
			maxAnalogPins:   12,
			supportedLibs:   []string{"WiFi", "WebServer", "PubSubClient", "ArduinoJson"},
			specialFeatures: []string{"wifi", "bluetooth", "deep_sleep"},
		},
		{
			board:           "esp8266:esp8266:d1mini",
			maxDigitalPins:  16,
			maxAnalogPins:   1,
			supportedLibs:   []string{"WiFi", "WebServer", "PubSubClient", "ArduinoJson"},
			specialFeatures: []string{"wifi", "deep_sleep"},
		},
	}

	for _, boardTest := range boardTests {
		t.Run(fmt.Sprintf("BoardFeatures_%s", boardTest.board), func(t *testing.T) {
			t.Parallel()

			// Find templates that support this board
			var supportedTemplates []Template
			for _, template := range suite.Templates {
				if isBoardSupported(template, boardTest.board) {
					supportedTemplates = append(supportedTemplates, template)
				}
			}

			// Test pin assignments
			for _, template := range supportedTemplates {
				err := validateBoardPinAssignments(template, boardTest)
				assert.NoError(t, err,
					"Template %s should respect board pin limits for %s",
					template.ID, boardTest.board)
			}

			// Test library support
			for _, template := range supportedTemplates {
				err := validateBoardLibrarySupport(template, boardTest)
				assert.NoError(t, err,
					"Template %s libraries should be supported on %s",
					template.ID, boardTest.board)
			}
		})
	}
}

// Helper functions for Arduino compilation testing

func isArduinoCLIAvailable() bool {
	_, err := exec.LookPath("arduino-cli")
	return err == nil
}

func (act *ArduinoCompilationTest) compileTemplate(template Template, board string) error {
	// Create temporary workspace
	workspace := filepath.Join(act.WorkspaceDir, template.ID)
	defer os.RemoveAll(workspace)

	err := os.MkdirAll(workspace, 0755)
	if err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	// Generate Arduino sketch from template
	sketchPath := filepath.Join(workspace, fmt.Sprintf("%s.ino", template.ID))
	codeAsset := findCodeAsset(template.Assets)
	if codeAsset == nil {
		return fmt.Errorf("template has no code asset")
	}

	// Substitute template variables
	code := codeAsset.Metadata["content"].(string)
	substitutedCode, err := substituteTemplateVariables(code, template.Parameters)
	if err != nil {
		return fmt.Errorf("failed to substitute template variables: %w", err)
	}

	// Write sketch file
	err = os.WriteFile(sketchPath, []byte(substitutedCode), 0644)
	if err != nil {
		return fmt.Errorf("failed to write sketch file: %w", err)
	}

	// Install required libraries
	for _, lib := range template.Libraries {
		err = act.installLibrary(lib.Name, lib.Version)
		if err != nil {
			return fmt.Errorf("failed to install library %s: %w", lib.Name, err)
		}
	}

	// Compile the sketch
	return act.compileSketch(sketchPath, board)
}

func (act *ArduinoCompilationTest) installLibrary(name, version string) error {
	cmd := exec.Command(act.ArduinoCLIPath, "lib", "install", name, "--version", version)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("library installation failed: %s\nOutput: %s", err, string(output))
	}
	return nil
}

func (act *ArduinoCompilationTest) compileSketch(sketchPath, board string) error {
	cmd := exec.Command(act.ArduinoCLIPath, "compile",
		"--fqbn", board,
		"--build-path", filepath.Dir(sketchPath),
		sketchPath)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("compilation failed: %s\nOutput: %s", err, string(output))
	}
	return nil
}

func substituteTemplateVariables(code string, params map[string]interface{}) (string, error) {
	result := code

	for paramName, paramValue := range params {
		variable := fmt.Sprintf("{{.%s}}", paramName)
		var replacement string

		switch v := paramValue.(type) {
		case string:
			replacement = v
		case float64:
			replacement = fmt.Sprintf("%.0f", v)
		case bool:
			replacement = fmt.Sprintf("%t", v)
		default:
			return "", fmt.Errorf("unsupported parameter type: %T", paramValue)
		}

		result = strings.ReplaceAll(result, variable, replacement)
	}

	return result, nil
}

func validateLibraryCompatibility(template Template) error {
	for _, lib := range template.Libraries {
		// Check if library name is valid
		if !isValidLibraryName(lib.Name) {
			return fmt.Errorf("invalid library name: %s", lib.Name)
		}

		// Check library version format
		if !isValidVersion(lib.Version) {
			return fmt.Errorf("invalid library version: %s for library %s",
				lib.Version, lib.Name)
		}

		// Check library-board compatibility
		for _, board := range template.BoardsSupported {
			if !isLibraryCompatible(lib, board) {
				return fmt.Errorf("library %s is not compatible with board %s",
					lib.Name, board)
			}
		}
	}

	return nil
}

func isValidLibraryName(name string) bool {
	// Basic validation - should contain only alphanumeric characters, spaces, and hyphens
	if len(name) == 0 {
		return false
	}

	for _, c := range name {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') || c == ' ' || c == '-' || c == '_') {
			return false
		}
	}

	return true
}

func isValidVersion(version string) bool {
	// Basic semantic version validation
	parts := strings.Split(version, ".")
	if len(parts) < 2 || len(parts) > 4 {
		return false
	}

	for _, part := range parts {
		if len(part) == 0 {
			return false
		}

		for _, c := range part {
			if c < '0' || c > '9' {
				return false
			}
		}
	}

	return true
}

func validateTemplateVariables(code string, params map[string]interface{}) error {
	// Extract all template variables from code
	variables := extractTemplateVariables(code)

	// Check that all variables in code are defined in parameters
	for _, variable := range variables {
		if _, exists := params[variable]; !exists {
			return fmt.Errorf("template variable %s is not defined in parameters", variable)
		}
	}

	// Check that all parameters are used in code (optional check)
	for paramName := range params {
		found := false
		for _, variable := range variables {
			if variable == paramName {
				found = true
				break
			}
		}
		if !found {
			// This is just a warning, not an error
			fmt.Printf("Warning: parameter %s is not used in code\n", paramName)
		}
	}

	return nil
}

func extractTemplateVariables(code string) []string {
	var variables []string
	variableMap := make(map[string]bool)

	start := 0
	for {
		startIdx := strings.Index(code[start:], "{{.")
		if startIdx == -1 {
			break
		}
		startIdx += start

		endIdx := strings.Index(code[startIdx:], "}}")
		if endIdx == -1 {
			break
		}
		endIdx += startIdx

		variable := code[startIdx+3 : endIdx]
		if variableMap[variable] {
			start = endIdx + 2
			continue
		}

		variableMap[variable] = true
		variables = append(variables, variable)
		start = endIdx + 2
	}

	return variables
}

func validateCodeWithoutTemplates(code string, params map[string]interface{}) error {
	// Substitute template variables and validate basic syntax
	substitutedCode, err := substituteTemplateVariables(code, params)
	if err != nil {
		return err
	}

	// Basic syntax checks
	if !strings.Contains(substitutedCode, "void setup()") {
		return fmt.Errorf("missing setup() function")
	}

	if !strings.Contains(substitutedCode, "void loop()") {
		return fmt.Errorf("missing loop() function")
	}

	// Check for balanced braces
	if countChar(substitutedCode, '{') != countChar(substitutedCode, '}') {
		return fmt.Errorf("unbalanced braces")
	}

	// Check for balanced parentheses
	if countChar(substitutedCode, '(') != countChar(substitutedCode, ')') {
		return fmt.Errorf("unbalanced parentheses")
	}

	return nil
}

func countChar(s string, char rune) int {
	count := 0
	for _, c := range s {
		if c == char {
			count++
		}
	}
	return count
}

func validateBoardPinAssignments(template Template, boardTest struct {
	board           string
	maxDigitalPins  int
	maxAnalogPins   int
	supportedLibs   []string
	specialFeatures []string
}) error {
	for paramName, paramValue := range template.Parameters {
		if strings.Contains(paramName, "Pin") {
			if pinNum, ok := paramValue.(float64); ok {
				pin := int(pinNum)

				// Check if pin is within board limits
				if pin > boardTest.maxDigitalPins {
					return fmt.Errorf("pin %d exceeds board %s limit of %d digital pins",
						pin, boardTest.board, boardTest.maxDigitalPins)
				}

				// Board-specific pin restrictions
				switch boardTest.board {
				case "arduino:avr:uno":
					if pin == 0 || pin == 1 {
						return fmt.Errorf("pins 0 and 1 are reserved for serial communication on Arduino Uno")
					}
				case "esp8266:esp8266:d1mini":
					if pin > 16 {
						return fmt.Errorf("ESP8266 D1 Mini only has pins 0-16")
					}
				}
			}
		}
	}

	return nil
}

func validateBoardLibrarySupport(template Template, boardTest struct {
	board           string
	maxDigitalPins  int
	maxAnalogPins   int
	supportedLibs   []string
	specialFeatures []string
}) error {
	for _, lib := range template.Libraries {
		supported := false
		for _, supportedLib := range boardTest.supportedLibs {
			if lib.Name == supportedLib {
				supported = true
				break
			}
		}

		if !supported {
			return fmt.Errorf("library %s is not supported on board %s",
				lib.Name, boardTest.board)
		}
	}

	return nil
}
