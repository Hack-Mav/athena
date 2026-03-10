package grpc

import (
	"context"
	"fmt"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

// ClientConfig holds gRPC client configuration
type ClientConfig struct {
	Address           string        `json:"address"`
	Timeout           time.Duration `json:"timeout"`
	KeepAliveTime     time.Duration `json:"keep_alive_time"`
	KeepAliveTimeout  time.Duration `json:"keep_alive_timeout"`
	MaxRetries        int           `json:"max_retries"`
	RetryDelay        time.Duration `json:"retry_delay"`
	EnableCompression bool          `json:"enable_compression"`
	EnableTLS         bool          `json:"enable_tls"`
}

// ClientManager manages gRPC client connections
type ClientManager struct {
	connections map[string]*grpc.ClientConn
	configs     map[string]*ClientConfig
	mu          sync.RWMutex
}

// NewClientManager creates a new gRPC client manager
func NewClientManager() *ClientManager {
	return &ClientManager{
		connections: make(map[string]*grpc.ClientConn),
		configs:     make(map[string]*ClientConfig),
	}
}

// RegisterClient registers a new gRPC client
func (cm *ClientManager) RegisterClient(name string, config *ClientConfig) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if config == nil {
		config = &ClientConfig{
			Timeout:           30 * time.Second,
			KeepAliveTime:     30 * time.Second,
			KeepAliveTimeout:  5 * time.Second,
			MaxRetries:        3,
			RetryDelay:        100 * time.Millisecond,
			EnableCompression: true,
			EnableTLS:         false,
		}
	}

	// Create dial options
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                config.KeepAliveTime,
			Timeout:             config.KeepAliveTimeout,
			PermitWithoutStream: true,
		}),
	}

	if config.EnableCompression {
		opts = append(opts, grpc.WithDefaultCallOptions(grpc.UseCompressor("gzip")))
	}

	// Create connection
	conn, err := grpc.Dial(config.Address, opts...)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", name, err)
	}

	cm.connections[name] = conn
	cm.configs[name] = config

	return nil
}

// GetClient returns a gRPC client connection
func (cm *ClientManager) GetClient(name string) (*grpc.ClientConn, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	conn, exists := cm.connections[name]
	if !exists {
		return nil, fmt.Errorf("client %s not registered", name)
	}

	return conn, nil
}

// CloseClient closes a specific client connection
func (cm *ClientManager) CloseClient(name string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	conn, exists := cm.connections[name]
	if !exists {
		return fmt.Errorf("client %s not found", name)
	}

	err := conn.Close()
	delete(cm.connections, name)
	delete(cm.configs, name)

	return err
}

// CloseAll closes all client connections
func (cm *ClientManager) CloseAll() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	var errors []error
	for name, conn := range cm.connections {
		if err := conn.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close client %s: %w", name, err))
		}
	}

	cm.connections = make(map[string]*grpc.ClientConn)
	cm.configs = make(map[string]*ClientConfig)

	if len(errors) > 0 {
		return fmt.Errorf("errors closing clients: %v", errors)
	}

	return nil
}

// HealthCheck checks the health of all registered clients
func (cm *ClientManager) HealthCheck(ctx context.Context) map[string]error {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	results := make(map[string]error)

	for name, conn := range cm.connections {
		// Simple health check by getting connection state
		if conn.GetState() != connectivity.Ready && conn.GetState() != connectivity.Idle {
			results[name] = fmt.Errorf("connection state: %s", conn.GetState().String())
		} else {
			results[name] = nil
		}
	}

	return results
}

// RetryableCall executes a gRPC call with retry logic
func (cm *ClientManager) RetryableCall(ctx context.Context, name string, call func(*grpc.ClientConn) error) error {
	config, exists := cm.configs[name]
	if !exists {
		return fmt.Errorf("client %s not found", name)
	}

	conn, err := cm.GetClient(name)
	if err != nil {
		return err
	}

	var lastErr error
	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(config.RetryDelay):
			}
		}

		err = call(conn)
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryableError(err) {
			break
		}
	}

	return lastErr
}

// isRetryableError checks if an error is retryable
func isRetryableError(err error) bool {
	// Add logic to determine if error is retryable
	// For now, assume all errors are retryable except context errors
	if err == context.Canceled || err == context.DeadlineExceeded {
		return false
	}
	return true
}
