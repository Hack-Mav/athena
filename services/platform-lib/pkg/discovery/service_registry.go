package discovery

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/athena/platform-lib/pkg/logger"
)

// ServiceInstance represents a service instance
type ServiceInstance struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Address  string    `json:"address"`
	Port     int       `json:"port"`
	Status   string    `json:"status"` // healthy, unhealthy, unknown
	LastSeen time.Time `json:"last_seen"`
	Metadata map[string]string `json:"metadata"`
}

// ServiceRegistry manages service discovery
type ServiceRegistry struct {
	services map[string][]*ServiceInstance
	mu       sync.RWMutex
	logger   *logger.Logger
	config   *RegistryConfig
}

// RegistryConfig holds registry configuration
type RegistryConfig struct {
	HealthCheckInterval time.Duration `json:"health_check_interval"`
	HealthCheckTimeout  time.Duration `json:"health_check_timeout"`
	MaxRetries          int           `json:"max_retries"`
}

// LoadBalancer represents load balancing strategy
type LoadBalancer interface {
	SelectInstance(instances []*ServiceInstance) *ServiceInstance
}

// RoundRobinLoadBalancer implements round-robin load balancing
type RoundRobinLoadBalancer struct {
	counter map[string]int
	mu      sync.Mutex
}

// NewServiceRegistry creates a new service registry
func NewServiceRegistry(logger *logger.Logger, config *RegistryConfig) *ServiceRegistry {
	if config == nil {
		config = &RegistryConfig{
			HealthCheckInterval: 30 * time.Second,
			HealthCheckTimeout:  5 * time.Second,
			MaxRetries:          3,
		}
	}

	registry := &ServiceRegistry{
		services: make(map[string][]*ServiceInstance),
		logger:   logger,
		config:   config,
	}

	// Start health check routine
	go registry.healthCheckLoop()

	return registry
}

// RegisterService registers a new service instance
func (sr *ServiceRegistry) RegisterService(instance *ServiceInstance) error {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	instance.LastSeen = time.Now()
	instance.Status = "healthy"

	if _, exists := sr.services[instance.Name]; !exists {
		sr.services[instance.Name] = []*ServiceInstance{instance}
	} else {
		sr.services[instance.Name] = append(sr.services[instance.Name], instance)
	}

	sr.logger.Infof("Registered service instance: %s (%s:%d)", instance.ID, instance.Address, instance.Port)
	return nil
}

// DeregisterService removes a service instance
func (sr *ServiceRegistry) DeregisterService(serviceName, instanceID string) error {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	instances, exists := sr.services[serviceName]
	if !exists {
		return fmt.Errorf("service %s not found", serviceName)
	}

	for i, instance := range instances {
		if instance.ID == instanceID {
			sr.services[serviceName] = append(instances[:i], instances[i+1:]...)
			sr.logger.Infof("Deregistered service instance: %s", instanceID)
			return nil
		}
	}

	return fmt.Errorf("instance %s not found in service %s", instanceID, serviceName)
}

// GetHealthyInstances returns healthy instances for a service
func (sr *ServiceRegistry) GetHealthyInstances(serviceName string) []*ServiceInstance {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	instances, exists := sr.services[serviceName]
	if !exists {
		return nil
	}

	var healthy []*ServiceInstance
	for _, instance := range instances {
		if instance.Status == "healthy" {
			healthy = append(healthy, instance)
		}
	}

	return healthy
}

// GetServiceURL returns a load-balanced URL for the service
func (sr *ServiceRegistry) GetServiceURL(serviceName string, loadBalancer LoadBalancer) (string, error) {
	instances := sr.GetHealthyInstances(serviceName)
	if len(instances) == 0 {
		return "", fmt.Errorf("no healthy instances found for service %s", serviceName)
	}

	selectedInstance := loadBalancer.SelectInstance(instances)
	if selectedInstance == nil {
		return "", fmt.Errorf("load balancer failed to select instance for service %s", serviceName)
	}

	return fmt.Sprintf("http://%s:%d", selectedInstance.Address, selectedInstance.Port), nil
}

// healthCheckLoop runs periodic health checks
func (sr *ServiceRegistry) healthCheckLoop() {
	ticker := time.NewTicker(sr.config.HealthCheckInterval)
	defer ticker.Stop()

	for range ticker.C {
		sr.performHealthChecks()
	}
}

// performHealthChecks checks health of all registered instances
func (sr *ServiceRegistry) performHealthChecks() {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	for serviceName, instances := range sr.services {
		for _, instance := range instances {
			go sr.checkInstanceHealth(serviceName, instance)
		}
	}
}

// checkInstanceHealth checks health of a single instance
func (sr *ServiceRegistry) checkInstanceHealth(serviceName string, instance *ServiceInstance) {
	ctx, cancel := context.WithTimeout(context.Background(), sr.config.HealthCheckTimeout)
	defer cancel()

	url := fmt.Sprintf("http://%s:%d/health", instance.Address, instance.Port)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		sr.markInstanceUnhealthy(serviceName, instance.ID)
		return
	}

	client := &http.Client{Timeout: sr.config.HealthCheckTimeout}
	resp, err := client.Do(req)
	if err != nil {
		sr.markInstanceUnhealthy(serviceName, instance.ID)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		sr.markInstanceHealthy(serviceName, instance.ID)
	} else {
		sr.markInstanceUnhealthy(serviceName, instance.ID)
	}
}

// markInstanceHealthy marks an instance as healthy
func (sr *ServiceRegistry) markInstanceHealthy(serviceName, instanceID string) {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	instances, exists := sr.services[serviceName]
	if !exists {
		return
	}

	for _, instance := range instances {
		if instance.ID == instanceID {
			if instance.Status != "healthy" {
				instance.Status = "healthy"
				instance.LastSeen = time.Now()
				sr.logger.Infof("Service instance %s marked as healthy", instanceID)
			}
			return
		}
	}
}

// markInstanceUnhealthy marks an instance as unhealthy
func (sr *ServiceRegistry) markInstanceUnhealthy(serviceName, instanceID string) {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	instances, exists := sr.services[serviceName]
	if !exists {
		return
	}

	for _, instance := range instances {
		if instance.ID == instanceID {
			if instance.Status != "unhealthy" {
				instance.Status = "unhealthy"
				sr.logger.Warnf("Service instance %s marked as unhealthy", instanceID)
			}
			return
		}
	}
}

// GetServiceStats returns statistics for a service
func (sr *ServiceRegistry) GetServiceStats(serviceName string) map[string]int {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	stats := map[string]int{
		"total":     0,
		"healthy":   0,
		"unhealthy": 0,
	}

	instances, exists := sr.services[serviceName]
	if !exists {
		return stats
	}

	for _, instance := range instances {
		stats["total"]++
		switch instance.Status {
		case "healthy":
			stats["healthy"]++
		case "unhealthy":
			stats["unhealthy"]++
		}
	}

	return stats
}

// NewRoundRobinLoadBalancer creates a new round-robin load balancer
func NewRoundRobinLoadBalancer() *RoundRobinLoadBalancer {
	return &RoundRobinLoadBalancer{
		counter: make(map[string]int),
	}
}

// SelectInstance selects an instance using round-robin strategy
func (lb *RoundRobinLoadBalancer) SelectInstance(instances []*ServiceInstance) *ServiceInstance {
	if len(instances) == 0 {
		return nil
	}

	lb.mu.Lock()
	defer lb.mu.Unlock()

	// Use service name as key (all instances belong to same service)
	serviceName := instances[0].Name
	current := lb.counter[serviceName]
	selected := instances[current%len(instances)]
	lb.counter[serviceName] = current + 1

	return selected
}
