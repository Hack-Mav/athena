package provisioning

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

// Compiler handles Arduino firmware compilation
type Compiler struct {
	cli           *ArduinoCLI
	workspaceDir  string
	cacheDir      string
	enableCache   bool
}

// NewCompiler creates a new compiler instance
func NewCompiler(cli *ArduinoCLI, workspaceDir, cacheDir string) *Compiler {
	return &Compiler{
		cli:          cli,
		workspaceDir: workspaceDir,
		cacheDir:     cacheDir,
		enableCache:  true,
	}
}

// CompilationRequest represents a compilation request
type CompilationRequest struct {
	TemplateID   string                 `json:"template_id"`
	TemplateCode string                 `json:"template_code"`
	Parameters   map[string]interface{} `json:"parameters"`
	Board        string                 `json:"board"` // FQBN
	Libraries    []LibraryDependency    `json:"libraries"`
	Secrets      map[string]string      `json:"secrets,omitempty"`
}

// CompilationResult represents the result of compilation
type CompilationResult struct {
	Success      bool                   `json:"success"`
	BinaryPath   string                 `json:"binary_path,omitempty"`
	BinaryHash   string                 `json:"binary_hash,omitempty"`
	Size         CompilationSize        `json:"size,omitempty"`
	Duration     time.Duration          `json:"duration"`
	CacheHit     bool                   `json:"cache_hit"`
	Metadata     CompilationMetadata    `json:"metadata"`
	Errors       []CompilationError     `json:"errors,omitempty"`
	Warnings     []CompilationWarning   `json:"warnings,omitempty"`
}

// CompilationSize represents binary size information
type CompilationSize struct {
	ProgramSize int `json:"program_size"`
	DataSize    int `json:"data_size"`
	MaxProgram  int `json:"max_program"`
	MaxData     int `json:"max_data"`
}

// CompilationMetadata contains compilation metadata
type CompilationMetadata struct {
	Board         string                 `json:"board"`
	TemplateID    string                 `json:"template_id"`
	Parameters    map[string]interface{} `json:"parameters"`
	Libraries     []LibraryDependency    `json:"libraries"`
	CompiledAt    time.Time              `json:"compiled_at"`
	ArduinoCLI    string                 `json:"arduino_cli_version"`
	CompilerFlags []string               `json:"compiler_flags,omitempty"`
}

// CompilationError represents a compilation error
type CompilationError struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Column  int    `json:"column"`
	Message string `json:"message"`
	Type    string `json:"type"` // "error", "fatal"
}

// CompilationWarning represents a compilation warning
type CompilationWarning struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Column  int    `json:"column"`
	Message string `json:"message"`
}

// CompileTemplate compiles an Arduino template with parameters
func (c *Compiler) CompileTemplate(ctx context.Context, request *CompilationRequest) (*CompilationResult, error) {
	startTime := time.Now()
	
	result := &CompilationResult{
		Success:  false,
		Duration: 0,
		CacheHit: false,
		Metadata: CompilationMetadata{
			Board:      request.Board,
			TemplateID: request.TemplateID,
			Parameters: request.Parameters,
			Libraries:  request.Libraries,
			CompiledAt: startTime,
		},
		Errors:   []CompilationError{},
		Warnings: []CompilationWarning{},
	}

	// Generate cache key
	cacheKey := c.generateCacheKey(request)
	
	// Check cache if enabled
	if c.enableCache {
		if cachedResult, found := c.checkCache(cacheKey); found {
			cachedResult.Duration = time.Since(startTime)
			cachedResult.CacheHit = true
			return cachedResult, nil
		}
	}

	// Create temporary project directory
	projectDir, err := c.createProjectDirectory(request)
	if err != nil {
		return result, fmt.Errorf("failed to create project directory: %w", err)
	}
	defer os.RemoveAll(projectDir)

	// Render template with parameters
	renderedCode, err := c.renderTemplate(request.TemplateCode, request.Parameters, request.Secrets)
	if err != nil {
		result.Errors = append(result.Errors, CompilationError{
			File:    "template",
			Message: "Template rendering failed: " + err.Error(),
			Type:    "fatal",
		})
		result.Duration = time.Since(startTime)
		return result, nil
	}

	// Write Arduino sketch
	sketchPath := filepath.Join(projectDir, request.TemplateID+".ino")
	if err := os.WriteFile(sketchPath, []byte(renderedCode), 0644); err != nil {
		return result, fmt.Errorf("failed to write sketch file: %w", err)
	}

	// Get Arduino CLI version for metadata
	if version, err := c.cli.Version(ctx); err == nil {
		result.Metadata.ArduinoCLI = version
	}

	// Compile the sketch
	binaryPath, compileOutput, err := c.compileSketch(ctx, projectDir, request.Board)
	if err != nil {
		// Parse compilation errors and warnings
		c.parseCompilationOutput(compileOutput, result)
		result.Duration = time.Since(startTime)
		return result, nil
	}

	// Calculate binary hash and size
	hash, size, err := c.analyzeBinary(binaryPath)
	if err != nil {
		return result, fmt.Errorf("failed to analyze binary: %w", err)
	}

	result.Success = true
	result.BinaryPath = binaryPath
	result.BinaryHash = hash
	result.Size = size
	result.Duration = time.Since(startTime)

	// Parse any warnings from successful compilation
	c.parseCompilationOutput(compileOutput, result)

	// Cache the result if enabled
	if c.enableCache {
		c.cacheResult(cacheKey, result)
	}

	return result, nil
}

// createProjectDirectory creates a temporary project directory
func (c *Compiler) createProjectDirectory(request *CompilationRequest) (string, error) {
	projectDir := filepath.Join(c.workspaceDir, "compile_"+request.TemplateID+"_"+fmt.Sprintf("%d", time.Now().UnixNano()))
	
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create project directory: %w", err)
	}

	return projectDir, nil
}

// renderTemplate renders the Arduino template with parameters and secrets
func (c *Compiler) renderTemplate(templateCode string, parameters map[string]interface{}, secrets map[string]string) (string, error) {
	// Create template with custom functions
	tmpl, err := template.New("arduino").Funcs(template.FuncMap{
		"secret": func(key string) string {
			if value, exists := secrets[key]; exists {
				return value
			}
			return ""
		},
		"param": func(key string) interface{} {
			if value, exists := parameters[key]; exists {
				return value
			}
			return ""
		},
		"quote": func(s interface{}) string {
			return fmt.Sprintf("\"%v\"", s)
		},
		"int": func(v interface{}) int {
			switch val := v.(type) {
			case int:
				return val
			case float64:
				return int(val)
			case string:
				// Simple string to int conversion
				if val == "true" {
					return 1
				}
				return 0
			default:
				return 0
			}
		},
		"bool": func(v interface{}) bool {
			switch val := v.(type) {
			case bool:
				return val
			case string:
				return val == "true"
			case int:
				return val != 0
			case float64:
				return val != 0
			default:
				return false
			}
		},
	}).Parse(templateCode)
	
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	// Combine parameters and secrets for template context
	context := make(map[string]interface{})
	for k, v := range parameters {
		context[k] = v
	}
	// Don't expose secrets directly in context for security

	var rendered strings.Builder
	if err := tmpl.Execute(&rendered, context); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return rendered.String(), nil
}

// compileSketch compiles the Arduino sketch
func (c *Compiler) compileSketch(ctx context.Context, projectDir, board string) (string, string, error) {
	// Build output directory
	buildDir := filepath.Join(projectDir, "build")
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		return "", "", fmt.Errorf("failed to create build directory: %w", err)
	}

	// Compile command
	args := []string{
		"compile",
		"--fqbn", board,
		"--build-path", buildDir,
		"--output-dir", buildDir,
		projectDir,
	}

	output, err := c.cli.ExecuteCommand(ctx, args...)
	outputStr := string(output)
	
	if err != nil {
		return "", outputStr, fmt.Errorf("compilation failed: %w", err)
	}

	// Find the compiled binary
	binaryPath, err := c.findCompiledBinary(buildDir, board)
	if err != nil {
		return "", outputStr, fmt.Errorf("failed to find compiled binary: %w", err)
	}

	return binaryPath, outputStr, nil
}

// findCompiledBinary finds the compiled binary file
func (c *Compiler) findCompiledBinary(buildDir, board string) (string, error) {
	// Common binary extensions for different platforms
	extensions := []string{".hex", ".bin", ".elf"}
	
	for _, ext := range extensions {
		pattern := filepath.Join(buildDir, "*"+ext)
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}
		
		if len(matches) > 0 {
			return matches[0], nil
		}
	}

	return "", fmt.Errorf("no compiled binary found in %s", buildDir)
}

// analyzeBinary calculates hash and size information for the binary
func (c *Compiler) analyzeBinary(binaryPath string) (string, CompilationSize, error) {
	file, err := os.Open(binaryPath)
	if err != nil {
		return "", CompilationSize{}, fmt.Errorf("failed to open binary: %w", err)
	}
	defer file.Close()

	// Calculate SHA256 hash
	hasher := sha256.New()
	size, err := io.Copy(hasher, file)
	if err != nil {
		return "", CompilationSize{}, fmt.Errorf("failed to calculate hash: %w", err)
	}

	hash := hex.EncodeToString(hasher.Sum(nil))

	// For now, return basic size info
	// In a full implementation, this would parse ELF files or use arduino-cli size command
	compilationSize := CompilationSize{
		ProgramSize: int(size),
		DataSize:    0,
		MaxProgram:  32768, // Default for Arduino Uno
		MaxData:     2048,  // Default for Arduino Uno
	}

	return hash, compilationSize, nil
}

// parseCompilationOutput parses compilation output for errors and warnings
func (c *Compiler) parseCompilationOutput(output string, result *CompilationResult) {
	lines := strings.Split(output, "\n")
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Simple error/warning parsing
		// In practice, this would be more sophisticated
		if strings.Contains(line, "error:") {
			result.Errors = append(result.Errors, CompilationError{
				Message: line,
				Type:    "error",
			})
		} else if strings.Contains(line, "warning:") {
			result.Warnings = append(result.Warnings, CompilationWarning{
				Message: line,
			})
		}
	}
}

// generateCacheKey generates a cache key for the compilation request
func (c *Compiler) generateCacheKey(request *CompilationRequest) string {
	hasher := sha256.New()
	
	// Include template code, parameters, board, and libraries in hash
	hasher.Write([]byte(request.TemplateCode))
	hasher.Write([]byte(request.Board))
	
	// Hash parameters (excluding secrets for security)
	for k, v := range request.Parameters {
		hasher.Write([]byte(fmt.Sprintf("%s:%v", k, v)))
	}
	
	// Hash libraries
	for _, lib := range request.Libraries {
		hasher.Write([]byte(fmt.Sprintf("%s:%s", lib.Name, lib.Version)))
	}

	return hex.EncodeToString(hasher.Sum(nil))
}

// checkCache checks if a compilation result is cached
func (c *Compiler) checkCache(cacheKey string) (*CompilationResult, bool) {
	// Simple file-based cache implementation
	// In practice, this would use Redis or another cache store
	cachePath := filepath.Join(c.cacheDir, cacheKey+".json")
	
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		return nil, false
	}

	// For now, return cache miss
	// Full implementation would deserialize cached result
	return nil, false
}

// cacheResult caches a compilation result
func (c *Compiler) cacheResult(cacheKey string, result *CompilationResult) {
	// Simple file-based cache implementation
	// In practice, this would serialize to Redis or another cache store
	if err := os.MkdirAll(c.cacheDir, 0755); err != nil {
		return
	}

	// For now, just create an empty cache file
	cachePath := filepath.Join(c.cacheDir, cacheKey+".json")
	os.WriteFile(cachePath, []byte("cached"), 0644)
}