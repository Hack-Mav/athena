package grpc

import (
	"fmt"
	"net"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
)

// ServerConfig holds gRPC server configuration
type ServerConfig struct {
	Address               string        `json:"address"`
	Port                  string        `json:"port"`
	KeepAliveTime         time.Duration `json:"keep_alive_time"`
	KeepAliveTimeout      time.Duration `json:"keep_alive_timeout"`
	KeepAliveMinTime      time.Duration `json:"keep_alive_min_time"`
	MaxConnectionIdle     time.Duration `json:"max_connection_idle"`
	MaxConnectionAge      time.Duration `json:"max_connection_age"`
	MaxConnectionAgeGrace time.Duration `json:"max_connection_age_grace"`
	EnableReflection      bool          `json:"enable_reflection"`
	EnableHealthCheck     bool          `json:"enable_health_check"`
}

// ServerManager manages gRPC server instances
type ServerManager struct {
	servers map[string]*grpc.Server
	configs map[string]*ServerConfig
	health  map[string]*health.Server
	mu      sync.RWMutex
}

// NewServerManager creates a new gRPC server manager
func NewServerManager() *ServerManager {
	return &ServerManager{
		servers: make(map[string]*grpc.Server),
		configs: make(map[string]*ServerConfig),
		health:  make(map[string]*health.Server),
	}
}

// RegisterServer registers a new gRPC server
func (sm *ServerManager) RegisterServer(name string, config *ServerConfig) (*grpc.Server, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if config == nil {
		config = &ServerConfig{
			Address:           "0.0.0.0",
			Port:              "50051",
			KeepAliveTime:     30 * time.Second,
			KeepAliveTimeout:  5 * time.Second,
			KeepAliveMinTime:  5 * time.Second,
			MaxConnectionIdle: 5 * time.Minute,
			MaxConnectionAge:  30 * time.Minute,
			EnableReflection:  true,
			EnableHealthCheck: true,
		}
	}

	// Create server options
	opts := []grpc.ServerOption{
		grpc.Creds(insecure.NewCredentials()),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time:    config.KeepAliveTime,
			Timeout: config.KeepAliveTimeout,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             config.KeepAliveMinTime,
			PermitWithoutStream: false,
		}),
	}

	// Create server
	server := grpc.NewServer(opts...)

	// Enable reflection if requested
	if config.EnableReflection {
		reflection.Register(server)
	}

	// Enable health check if requested
	if config.EnableHealthCheck {
		healthServer := health.NewServer()
		grpc_health_v1.RegisterHealthServer(server, healthServer)
		sm.health[name] = healthServer
	}

	sm.servers[name] = server
	sm.configs[name] = config

	return server, nil
}

// StartServer starts a gRPC server
func (sm *ServerManager) StartServer(name string) error {
	sm.mu.RLock()
	server, exists := sm.servers[name]
	config, configExists := sm.configs[name]
	sm.mu.RUnlock()

	if !exists || !configExists {
		return fmt.Errorf("server %s not registered", name)
	}

	// Create listener
	addr := fmt.Sprintf("%s:%s", config.Address, config.Port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	// Start server in goroutine
	go func() {
		if err := server.Serve(lis); err != nil {
			fmt.Printf("Failed to start server %s: %v\n", name, err)
		}
	}()

	return nil
}

// GetServer returns a gRPC server instance
func (sm *ServerManager) GetServer(name string) (*grpc.Server, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	server, exists := sm.servers[name]
	if !exists {
		return nil, fmt.Errorf("server %s not found", name)
	}

	return server, nil
}

// StopServer stops a specific gRPC server
func (sm *ServerManager) StopServer(name string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	server, exists := sm.servers[name]
	if !exists {
		return fmt.Errorf("server %s not found", name)
	}

	// Graceful stop
	server.GracefulStop()
	delete(sm.servers, name)
	delete(sm.configs, name)
	delete(sm.health, name)

	return nil
}

// StopAll stops all gRPC servers
func (sm *ServerManager) StopAll() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for name, server := range sm.servers {
		server.GracefulStop()
		delete(sm.servers, name)
		delete(sm.configs, name)
		delete(sm.health, name)
	}

	return nil
}

// SetHealthStatus sets the health status for a server
func (sm *ServerManager) SetHealthStatus(name, service string, status grpc_health_v1.HealthCheckResponse_ServingStatus) {
	sm.mu.RLock()
	_, exists := sm.health[name]
	sm.mu.RUnlock()

	if exists {
		// Set health status - in a real implementation, you would use the health server
		sm.mu.Lock()
		if healthServer, ok := sm.health[name]; ok {
			healthServer.SetServingStatus(service, status)
		}
		sm.mu.Unlock()
	}
}

// GetHealthStatus gets the health status for a server
func (sm *ServerManager) GetHealthStatus(name, service string) (grpc_health_v1.HealthCheckResponse_ServingStatus, error) {
	sm.mu.RLock()
	_, exists := sm.health[name]
	sm.mu.RUnlock()

	if !exists {
		return grpc_health_v1.HealthCheckResponse_SERVICE_UNKNOWN, fmt.Errorf("health server %s not found", name)
	}

	// Note: In a real implementation, you would need to implement a way to get the current status
	// This is a simplified version
	return grpc_health_v1.HealthCheckResponse_SERVING, nil
}

// ServerStatus represents the status of a gRPC server
type ServerStatus struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	Port    string `json:"port"`
	Running bool   `json:"running"`
	Health  string `json:"health"`
}

// GetServerStatus returns the status of all servers
func (sm *ServerManager) GetServerStatus() []ServerStatus {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var status []ServerStatus
	for name, config := range sm.configs {
		serverStatus := ServerStatus{
			Name:    name,
			Address: config.Address,
			Port:    config.Port,
			Running: sm.servers[name] != nil,
		}

		// Get health status
		if _, exists := sm.health[name]; exists {
			// Simplified health check
			serverStatus.Health = "healthy"
		} else {
			serverStatus.Health = "unknown"
		}

		status = append(status, serverStatus)
	}

	return status
}
