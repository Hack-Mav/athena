package nlp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// LLMClientInterface defines the interface for LLM clients
type LLMClientInterface interface {
	Complete(ctx context.Context, prompt string) (*LLMResponse, error)
	CompleteWithRequest(ctx context.Context, request *LLMRequest) (*LLMResponse, error)
}

// LLMClient handles communication with LLM providers
type LLMClient struct {
	endpoint    string
	apiKey      string
	model       string
	temperature float64
	maxTokens   int
	httpClient  *http.Client
}

// LLMConfig represents LLM client configuration
type LLMConfig struct {
	Endpoint    string  `json:"endpoint"`
	APIKey      string  `json:"api_key"`
	Model       string  `json:"model"`
	Temperature float64 `json:"temperature"`
	MaxTokens   int     `json:"max_tokens"`
	Timeout     int     `json:"timeout"` // seconds
}

// NewLLMClient creates a new LLM client
func NewLLMClient(config *LLMConfig) *LLMClient {
	timeout := 30 * time.Second
	if config.Timeout > 0 {
		timeout = time.Duration(config.Timeout) * time.Second
	}

	return &LLMClient{
		endpoint:    config.Endpoint,
		apiKey:      config.APIKey,
		model:       config.Model,
		temperature: config.Temperature,
		maxTokens:   config.MaxTokens,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// Complete sends a completion request to the LLM provider
func (c *LLMClient) Complete(ctx context.Context, prompt string) (*LLMResponse, error) {
	request := &LLMRequest{
		Prompt:      prompt,
		Model:       c.model,
		Temperature: c.temperature,
		MaxTokens:   c.maxTokens,
	}

	return c.CompleteWithRequest(ctx, request)
}

// CompleteWithRequest sends a custom completion request
func (c *LLMClient) CompleteWithRequest(ctx context.Context, request *LLMRequest) (*LLMResponse, error) {
	// Build request payload based on provider
	payload := c.buildRequestPayload(request)

	// Marshal to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	// Send request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("LLM request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	return c.parseResponse(body)
}

// buildRequestPayload builds the request payload for the LLM provider
func (c *LLMClient) buildRequestPayload(request *LLMRequest) map[string]interface{} {
	// OpenAI-compatible format
	return map[string]interface{}{
		"model": request.Model,
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": request.Prompt,
			},
		},
		"temperature": request.Temperature,
		"max_tokens":  request.MaxTokens,
	}
}

// parseResponse parses the LLM provider response
func (c *LLMClient) parseResponse(body []byte) (*LLMResponse, error) {
	var response struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			TotalTokens int `json:"total_tokens"`
		} `json:"usage"`
		Model string `json:"model"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	return &LLMResponse{
		Content:      response.Choices[0].Message.Content,
		Model:        response.Model,
		TokensUsed:   response.Usage.TotalTokens,
		FinishReason: response.Choices[0].FinishReason,
	}, nil
}

// ExtractJSON attempts to extract JSON from LLM response
func ExtractJSON(content string) (string, error) {
	// Find JSON content between ```json and ``` markers
	start := -1
	end := -1

	// Look for ```json marker
	jsonMarker := "```json"
	if idx := findString(content, jsonMarker); idx != -1 {
		start = idx + len(jsonMarker)
	} else if idx := findString(content, "```"); idx != -1 {
		// Try plain ``` marker
		start = idx + 3
	}

	// Look for closing ``` marker
	if start != -1 {
		if idx := findStringFrom(content, "```", start); idx != -1 {
			end = idx
		}
	}

	// Extract JSON content
	if start != -1 && end != -1 {
		jsonContent := content[start:end]
		// Trim whitespace
		jsonContent = trimSpace(jsonContent)
		return jsonContent, nil
	}

	// If no markers found, try to find JSON object directly
	if idx := findString(content, "{"); idx != -1 {
		// Find matching closing brace
		braceCount := 0
		for i := idx; i < len(content); i++ {
			if content[i] == '{' {
				braceCount++
			} else if content[i] == '}' {
				braceCount--
				if braceCount == 0 {
					return content[idx : i+1], nil
				}
			}
		}
	}

	return "", fmt.Errorf("no JSON content found in response")
}

// Helper functions

func findString(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func findStringFrom(s, substr string, start int) int {
	for i := start; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func trimSpace(s string) string {
	start := 0
	end := len(s)

	// Trim leading whitespace
	for start < end && isWhitespace(s[start]) {
		start++
	}

	// Trim trailing whitespace
	for end > start && isWhitespace(s[end-1]) {
		end--
	}

	return s[start:end]
}

func isWhitespace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}
