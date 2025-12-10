package health

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"sync"
	"time"
)

// Status represents health check status
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusUnhealthy Status = "unhealthy"
	StatusDegraded  Status = "degraded"
)

// CheckResult represents the result of a health check
type CheckResult struct {
	Name      string            `json:"name"`
	Status    Status            `json:"status"`
	Message   string            `json:"message,omitempty"`
	Duration  time.Duration     `json:"duration"`
	Timestamp time.Time         `json:"timestamp"`
	Details   map[string]string `json:"details,omitempty"`
}

// HealthResponse represents the overall health response
type HealthResponse struct {
	Status    Status                  `json:"status"`
	Timestamp time.Time               `json:"timestamp"`
	Uptime    time.Duration           `json:"uptime"`
	Version   string                  `json:"version"`
	Checks    map[string]*CheckResult `json:"checks"`
	System    *SystemInfo             `json:"system"`
}

// SystemInfo represents system information
type SystemInfo struct {
	GoVersion    string    `json:"go_version"`
	NumGoroutine int       `json:"num_goroutine"`
	MemStats     *MemStats `json:"memory_stats"`
}

// MemStats represents memory statistics
type MemStats struct {
	Alloc      uint64 `json:"alloc"`
	TotalAlloc uint64 `json:"total_alloc"`
	Sys        uint64 `json:"sys"`
	NumGC      uint32 `json:"num_gc"`
}

// Checker interface for health checks
type Checker interface {
	Check(ctx context.Context) *CheckResult
	Name() string
}

// HealthChecker manages health checks
type HealthChecker struct {
	checks    map[string]Checker
	startTime time.Time
	version   string
	mutex     sync.RWMutex
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(version string) *HealthChecker {
	return &HealthChecker{
		checks:    make(map[string]Checker),
		startTime: time.Now(),
		version:   version,
	}
}

// AddCheck adds a health check
func (hc *HealthChecker) AddCheck(checker Checker) {
	hc.mutex.Lock()
	defer hc.mutex.Unlock()
	hc.checks[checker.Name()] = checker
}

// RemoveCheck removes a health check
func (hc *HealthChecker) RemoveCheck(name string) {
	hc.mutex.Lock()
	defer hc.mutex.Unlock()
	delete(hc.checks, name)
}

// CheckHealth performs all health checks
func (hc *HealthChecker) CheckHealth(ctx context.Context) *HealthResponse {
	hc.mutex.RLock()
	checks := make(map[string]Checker, len(hc.checks))
	for name, checker := range hc.checks {
		checks[name] = checker
	}
	hc.mutex.RUnlock()

	results := make(map[string]*CheckResult)
	overallStatus := StatusHealthy

	for name, checker := range checks {
		result := checker.Check(ctx)
		results[name] = result

		// Update overall status
		if result.Status == StatusUnhealthy {
			overallStatus = StatusUnhealthy
		} else if result.Status == StatusDegraded && overallStatus == StatusHealthy {
			overallStatus = StatusDegraded
		}
	}

	return &HealthResponse{
		Status:    overallStatus,
		Timestamp: time.Now(),
		Uptime:    time.Since(hc.startTime),
		Version:   hc.version,
		Checks:    results,
		System:    getSystemInfo(),
	}
}

// HealthHandlerFunc returns a health check handler function
func (hc *HealthChecker) HealthHandlerFunc() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		response := hc.CheckHealth(ctx)

		statusCode := http.StatusOK
		if response.Status == StatusUnhealthy {
			statusCode = http.StatusServiceUnavailable
		} else if response.Status == StatusDegraded {
			statusCode = http.StatusOK // Still serve but indicate degraded state
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		// Simple JSON response - in production, use proper JSON encoding
		fmt.Fprintf(w, `{"status":"%s","timestamp":"%s","uptime":"%s","version":"%s"}`,
			response.Status, response.Timestamp.Format(time.RFC3339),
			response.Uptime.String(), response.Version)
	}
}

// ReadinessHandlerFunc returns a readiness check handler function
func (hc *HealthChecker) ReadinessHandlerFunc() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		response := hc.CheckHealth(ctx)

		// For readiness, we only care if the service is unhealthy
		if response.Status == StatusUnhealthy {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, `{"status":"unready","message":"Service is not ready"}`)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ready","message":"Service is ready"}`)
	}
}

// LivenessHandlerFunc returns a liveness check handler function
func (hc *HealthChecker) LivenessHandlerFunc() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Basic liveness check - if we can respond, we're alive
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"alive","message":"Service is alive","uptime":"%s"}`,
			time.Since(hc.startTime).String())
	}
}

// getSystemInfo returns system information
func getSystemInfo() *SystemInfo {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return &SystemInfo{
		GoVersion:    runtime.Version(),
		NumGoroutine: runtime.NumGoroutine(),
		MemStats: &MemStats{
			Alloc:      m.Alloc,
			TotalAlloc: m.TotalAlloc,
			Sys:        m.Sys,
			NumGC:      m.NumGC,
		},
	}
}

// Built-in health checkers

// DatabaseChecker checks database connectivity
type DatabaseChecker struct {
	name      string
	checkFunc func(ctx context.Context) error
}

// NewDatabaseChecker creates a new database health checker
func NewDatabaseChecker(name string, checkFunc func(ctx context.Context) error) *DatabaseChecker {
	return &DatabaseChecker{
		name:      name,
		checkFunc: checkFunc,
	}
}

func (dc *DatabaseChecker) Name() string {
	return dc.name
}

func (dc *DatabaseChecker) Check(ctx context.Context) *CheckResult {
	start := time.Now()

	err := dc.checkFunc(ctx)
	duration := time.Since(start)

	if err != nil {
		return &CheckResult{
			Name:      dc.name,
			Status:    StatusUnhealthy,
			Message:   fmt.Sprintf("Database connection failed: %v", err),
			Duration:  duration,
			Timestamp: time.Now(),
		}
	}

	return &CheckResult{
		Name:      dc.name,
		Status:    StatusHealthy,
		Message:   "Database connection successful",
		Duration:  duration,
		Timestamp: time.Now(),
	}
}

// RedisChecker checks Redis connectivity
type RedisChecker struct {
	name      string
	checkFunc func(ctx context.Context) error
}

// NewRedisChecker creates a new Redis health checker
func NewRedisChecker(name string, checkFunc func(ctx context.Context) error) *RedisChecker {
	return &RedisChecker{
		name:      name,
		checkFunc: checkFunc,
	}
}

func (rc *RedisChecker) Name() string {
	return rc.name
}

func (rc *RedisChecker) Check(ctx context.Context) *CheckResult {
	start := time.Now()

	err := rc.checkFunc(ctx)
	duration := time.Since(start)

	if err != nil {
		return &CheckResult{
			Name:      rc.name,
			Status:    StatusUnhealthy,
			Message:   fmt.Sprintf("Redis connection failed: %v", err),
			Duration:  duration,
			Timestamp: time.Now(),
		}
	}

	return &CheckResult{
		Name:      rc.name,
		Status:    StatusHealthy,
		Message:   "Redis connection successful",
		Duration:  duration,
		Timestamp: time.Now(),
	}
}

// HTTPChecker checks HTTP endpoint connectivity
type HTTPChecker struct {
	name    string
	url     string
	timeout time.Duration
}

// NewHTTPChecker creates a new HTTP health checker
func NewHTTPChecker(name, url string, timeout time.Duration) *HTTPChecker {
	return &HTTPChecker{
		name:    name,
		url:     url,
		timeout: timeout,
	}
}

func (hc *HTTPChecker) Name() string {
	return hc.name
}

func (hc *HTTPChecker) Check(ctx context.Context) *CheckResult {
	start := time.Now()

	client := &http.Client{
		Timeout: hc.timeout,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", hc.url, nil)
	if err != nil {
		return &CheckResult{
			Name:      hc.name,
			Status:    StatusUnhealthy,
			Message:   fmt.Sprintf("Failed to create request: %v", err),
			Duration:  time.Since(start),
			Timestamp: time.Now(),
		}
	}

	resp, err := client.Do(req)
	duration := time.Since(start)

	if err != nil {
		return &CheckResult{
			Name:      hc.name,
			Status:    StatusUnhealthy,
			Message:   fmt.Sprintf("HTTP request failed: %v", err),
			Duration:  duration,
			Timestamp: time.Now(),
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return &CheckResult{
			Name:      hc.name,
			Status:    StatusDegraded,
			Message:   fmt.Sprintf("HTTP endpoint returned status: %d", resp.StatusCode),
			Duration:  duration,
			Timestamp: time.Now(),
		}
	}

	return &CheckResult{
		Name:      hc.name,
		Status:    StatusHealthy,
		Message:   "HTTP endpoint is responding",
		Duration:  duration,
		Timestamp: time.Now(),
		Details: map[string]string{
			"status_code": fmt.Sprintf("%d", resp.StatusCode),
			"url":         hc.url,
		},
	}
}

// CustomChecker allows for custom health check logic
type CustomChecker struct {
	name      string
	checkFunc func(ctx context.Context) *CheckResult
}

// NewCustomChecker creates a new custom health checker
func NewCustomChecker(name string, checkFunc func(ctx context.Context) *CheckResult) *CustomChecker {
	return &CustomChecker{
		name:      name,
		checkFunc: checkFunc,
	}
}

func (cc *CustomChecker) Name() string {
	return cc.name
}

func (cc *CustomChecker) Check(ctx context.Context) *CheckResult {
	return cc.checkFunc(ctx)
}
