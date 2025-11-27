package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/athena/platform-lib/pkg/config"
	"github.com/athena/platform-lib/pkg/logger"
)

// ServiceClient wraps HTTP calls to ATHENA microservices
type ServiceClient struct {
	httpClient *http.Client
	cfg        *config.Config
	logger     *logger.Logger
}

// NewServiceClient creates a new service client
func NewServiceClient(cfg *config.Config, logger *logger.Logger) *ServiceClient {
	return &ServiceClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		cfg:    cfg,
		logger: logger,
	}
}

// doRequest performs an HTTP request with proper error handling
func (c *ServiceClient) doRequest(ctx context.Context, method, url string, body interface{}, target interface{}) error {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %d %s - %s", resp.StatusCode, resp.Status, string(respBody))
	}

	if target != nil {
		if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// Template Service methods

type TemplateListResponse struct {
	Templates []Template `json:"templates"`
}

type Template struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Category    string            `json:"category"`
	Tags        []string          `json:"tags"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	Metadata    map[string]string `json:"metadata"`
}

// ListTemplates calls template service to list all templates
func (c *ServiceClient) ListTemplates(ctx context.Context) ([]Template, error) {
	url := c.cfg.Services["template-service"] + "/api/v1/templates"
	var resp TemplateListResponse
	if err := c.doRequest(ctx, "GET", url, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Templates, nil
}

// GetTemplate retrieves a specific template by ID
func (c *ServiceClient) GetTemplate(ctx context.Context, id string) (*Template, error) {
	url := c.cfg.Services["template-service"] + "/api/v1/templates/" + id
	var tmpl Template
	if err := c.doRequest(ctx, "GET", url, nil, &tmpl); err != nil {
		return nil, err
	}
	return &tmpl, nil
}

// Provisioning Service methods

type CompileRequest struct {
	TemplateID string            `json:"template_id"`
	Board      string            `json:"board"`
	Parameters map[string]string `json:"parameters"`
}

type CompileResponse struct {
	ArtifactID string `json:"artifact_id"`
	Status     string `json:"status"`
	Message    string `json:"message"`
}

type FlashRequest struct {
	Port       string `json:"port"`
	Board      string `json:"board"`
	ArtifactID string `json:"artifact_id"`
}

type FlashResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// Compile calls provisioning service to compile a template
func (c *ServiceClient) Compile(ctx context.Context, req *CompileRequest) (*CompileResponse, error) {
	url := c.cfg.Services["provisioning-service"] + "/api/v1/provisioning/compile"
	var resp CompileResponse
	if err := c.doRequest(ctx, "POST", url, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Flash calls provisioning service to flash a device
func (c *ServiceClient) Flash(ctx context.Context, req *FlashRequest) (*FlashResponse, error) {
	url := c.cfg.Services["provisioning-service"] + "/api/v1/provisioning/flash"
	var resp FlashResponse
	if err := c.doRequest(ctx, "POST", url, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Device Service methods

type Device struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Board     string            `json:"board"`
	Status    string            `json:"status"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
	Metadata  map[string]string `json:"metadata"`
}

type DeviceListResponse struct {
	Devices []Device `json:"devices"`
}

// ListDevices calls device service to list all devices
func (c *ServiceClient) ListDevices(ctx context.Context) ([]Device, error) {
	url := c.cfg.Services["device-service"] + "/api/v1/devices"
	var resp DeviceListResponse
	if err := c.doRequest(ctx, "GET", url, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Devices, nil
}

// GetDevice retrieves a specific device by ID
func (c *ServiceClient) GetDevice(ctx context.Context, id string) (*Device, error) {
	url := c.cfg.Services["device-service"] + "/api/v1/devices/" + id
	var dev Device
	if err := c.doRequest(ctx, "GET", url, nil, &dev); err != nil {
		return nil, err
	}
	return &dev, nil
}

// Telemetry Service methods

type TelemetryMetrics struct {
	DeviceID  string                 `json:"device_id"`
	Timestamp time.Time              `json:"timestamp"`
	Metrics   map[string]interface{} `json:"metrics"`
}

// GetTelemetryMetrics calls telemetry service to get device metrics
func (c *ServiceClient) GetTelemetryMetrics(ctx context.Context, deviceID string) (*TelemetryMetrics, error) {
	url := c.cfg.Services["telemetry-service"] + "/api/v1/telemetry/metrics/" + deviceID
	var metrics TelemetryMetrics
	if err := c.doRequest(ctx, "GET", url, nil, &metrics); err != nil {
		return nil, err
	}
	return &metrics, nil
}

// OTA Service methods

type Release struct {
	ID          string    `json:"id"`
	Version     string    `json:"version"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	ArtifactID  string    `json:"artifact_id"`
}

type ReleaseListResponse struct {
	Releases []Release `json:"releases"`
}

// ListReleases calls OTA service to list available releases
func (c *ServiceClient) ListReleases(ctx context.Context) ([]Release, error) {
	url := c.cfg.Services["ota-service"] + "/api/v1/ota/releases"
	var resp ReleaseListResponse
	if err := c.doRequest(ctx, "GET", url, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Releases, nil
}
