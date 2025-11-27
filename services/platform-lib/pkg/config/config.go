package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config holds the application configuration
type Config struct {
	ServiceName string `mapstructure:"service_name"`
	Environment string `mapstructure:"environment"`
	LogLevel    string `mapstructure:"log_level"`
	HTTPPort    string `mapstructure:"http_port"`
	GRPCPort    string `mapstructure:"grpc_port"`

	// Database configuration
	DatastoreProject string `mapstructure:"datastore_project"`
	DatastoreHost    string `mapstructure:"datastore_host"`

	// Redis configuration
	RedisAddr     string `mapstructure:"redis_addr"`
	RedisPassword string `mapstructure:"redis_password"`
	RedisDB       int    `mapstructure:"redis_db"`

	// MQTT configuration
	MQTT MQTTConfig `mapstructure:"mqtt"`

	// MinIO configuration
	MinIOEndpoint  string `mapstructure:"minio_endpoint"`
	MinIOAccessKey string `mapstructure:"minio_access_key"`
	MinIOSecretKey string `mapstructure:"minio_secret_key"`
	MinIOBucket    string `mapstructure:"minio_bucket"`

	// Service discovery
	Services map[string]string `mapstructure:"services"`

	// Security
	JWTSecret            string `mapstructure:"jwt_secret"`
	SecretsEncryptionKey string `mapstructure:"secrets_encryption_key"`

	// External services
	LLMProvider string `mapstructure:"llm_provider"`
	LLMAPIKey   string `mapstructure:"llm_api_key"`
	LLMEndpoint string `mapstructure:"llm_endpoint"`

	// Arduino CLI configuration
	ArduinoCLIPath string `mapstructure:"arduino_cli_path"`
}

// MQTTConfig holds MQTT-specific configuration
type MQTTConfig struct {
	Enabled   bool   `mapstructure:"enabled"`
	BrokerURL string `mapstructure:"broker_url"`
	Username  string `mapstructure:"username"`
	Password  string `mapstructure:"password"`
}

// Load loads configuration for the specified service
func Load(serviceName string) (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath("../configs")
	viper.AddConfigPath("../../configs")
	viper.AddConfigPath("/etc/athena")

	// Set environment variable prefix
	viper.SetEnvPrefix("ATHENA")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Set defaults
	setDefaults(serviceName)

	if err := viper.ReadInConfig(); err != nil {
		// Config file not found, use environment variables and defaults
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Override service name
	config.ServiceName = serviceName

	return &config, nil
}

// Default returns a default configuration for the specified service
func Default(serviceName string) *Config {
	return &Config{
		ServiceName:      serviceName,
		Environment:      "development",
		LogLevel:         "info",
		HTTPPort:         getDefaultHTTPPort(serviceName),
		GRPCPort:         getDefaultGRPCPort(serviceName),
		DatastoreProject: "athena-dev",
		DatastoreHost:    "localhost:8081",
		RedisAddr:        "localhost:6379",
		RedisPassword:    "",
		RedisDB:          0,
		MQTT: MQTTConfig{
			Enabled:   true,
			BrokerURL: "tcp://localhost:1883",
			Username:  "",
			Password:  "",
		},
		MinIOEndpoint:    "localhost:9000",
		MinIOAccessKey:   "athena",
		MinIOSecretKey:   "dev_password",
		MinIOBucket:      "athena-dev",
		Services: map[string]string{
			"template-service":     "http://localhost:8001",
			"nlp-service":          "http://localhost:8002",
			"provisioning-service": "http://localhost:8003",
			"device-service":       "http://localhost:8004",
			"telemetry-service":    "http://localhost:8005",
			"ota-service":          "http://localhost:8006",
			"secrets-service":      "http://localhost:8007",
			"api-gateway":          "http://localhost:8000",
		},
		JWTSecret:            "dev-secret-key",
		SecretsEncryptionKey: "dev-encryption-key-change-in-production",
		LLMProvider:    "openai",
		LLMAPIKey:      "",
		LLMEndpoint:    "https://api.openai.com/v1",
		ArduinoCLIPath: "arduino-cli",
	}
}

func setDefaults(serviceName string) {
	viper.SetDefault("service_name", serviceName)
	viper.SetDefault("environment", "development")
	viper.SetDefault("log_level", "info")
	viper.SetDefault("http_port", getDefaultHTTPPort(serviceName))
	viper.SetDefault("grpc_port", getDefaultGRPCPort(serviceName))
	viper.SetDefault("datastore_project", "athena-dev")
	viper.SetDefault("datastore_host", "localhost:8081")
	viper.SetDefault("redis_addr", "localhost:6379")
	viper.SetDefault("redis_password", "")
	viper.SetDefault("redis_db", 0)
	viper.SetDefault("mqtt.enabled", true)
	viper.SetDefault("mqtt.broker_url", "tcp://localhost:1883")
	viper.SetDefault("mqtt.username", "")
	viper.SetDefault("mqtt.password", "")
	viper.SetDefault("minio_endpoint", "localhost:9000")
	viper.SetDefault("minio_access_key", "athena")
	viper.SetDefault("minio_secret_key", "dev_password")
	viper.SetDefault("minio_bucket", "athena-dev")
	viper.SetDefault("jwt_secret", "dev-secret-key")
	viper.SetDefault("secrets_encryption_key", "dev-encryption-key-change-in-production")
	viper.SetDefault("llm_provider", "openai")
	viper.SetDefault("llm_endpoint", "https://api.openai.com/v1")
	viper.SetDefault("arduino_cli_path", "arduino-cli")
}

func getDefaultHTTPPort(serviceName string) string {
	ports := map[string]string{
		"api-gateway":          ":8000",
		"template-service":     ":8001",
		"nlp-service":          ":8002",
		"provisioning-service": ":8003",
		"device-service":       ":8004",
		"telemetry-service":    ":8005",
		"ota-service":          ":8006",
		"secrets-service":      ":8007",
		"athena-cli":           ":8008",
	}
	if port, exists := ports[serviceName]; exists {
		return port
	}
	return ":8080"
}

func getDefaultGRPCPort(serviceName string) string {
	ports := map[string]string{
		"template-service":     ":9001",
		"nlp-service":          ":9002",
		"provisioning-service": ":9003",
		"device-service":       ":9004",
		"telemetry-service":    ":9005",
		"ota-service":          ":9006",
		"secrets-service":      ":9007",
	}
	if port, exists := ports[serviceName]; exists {
		return port
	}
	return ":9090"
}