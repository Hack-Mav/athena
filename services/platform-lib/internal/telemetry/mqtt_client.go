package telemetry

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/athena/platform-lib/pkg/logger"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// MQTTClient handles MQTT connections and message routing
type MQTTClient struct {
	client     mqtt.Client
	logger     *logger.Logger
	repository Repository
	handlers   map[string]MessageHandler
	mu         sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
}

// MessageHandler is a function that processes MQTT messages
type MessageHandler func(topic string, payload []byte) error

// MQTTConfig holds MQTT connection configuration
type MQTTConfig struct {
	BrokerURL      string
	ClientID       string
	Username       string
	Password       string
	QoS            byte
	CleanSession   bool
	ConnectTimeout time.Duration
	KeepAlive      time.Duration
}

// NewMQTTClient creates a new MQTT client for telemetry ingestion
func NewMQTTClient(config *MQTTConfig, repository Repository, log *logger.Logger) (*MQTTClient, error) {
	ctx, cancel := context.WithCancel(context.Background())

	client := &MQTTClient{
		logger:     log,
		repository: repository,
		handlers:   make(map[string]MessageHandler),
		ctx:        ctx,
		cancel:     cancel,
	}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(config.BrokerURL)
	opts.SetClientID(config.ClientID)
	opts.SetUsername(config.Username)
	opts.SetPassword(config.Password)
	opts.SetCleanSession(config.CleanSession)
	opts.SetConnectTimeout(config.ConnectTimeout)
	opts.SetKeepAlive(config.KeepAlive)
	opts.SetAutoReconnect(true)
	opts.SetConnectionLostHandler(client.onConnectionLost)
	opts.SetOnConnectHandler(client.onConnect)

	client.client = mqtt.NewClient(opts)

	return client, nil
}

// Connect establishes connection to the MQTT broker
func (c *MQTTClient) Connect() error {
	token := c.client.Connect()
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to connect to MQTT broker: %w", token.Error())
	}

	c.logger.Info("Connected to MQTT broker")
	return nil
}

// Disconnect closes the MQTT connection
func (c *MQTTClient) Disconnect() {
	c.cancel()
	c.client.Disconnect(250)
	c.logger.Info("Disconnected from MQTT broker")
}

// Subscribe subscribes to a topic with a handler
func (c *MQTTClient) Subscribe(topic string, qos byte, handler MessageHandler) error {
	c.mu.Lock()
	c.handlers[topic] = handler
	c.mu.Unlock()

	token := c.client.Subscribe(topic, qos, func(client mqtt.Client, msg mqtt.Message) {
		c.handleMessage(msg.Topic(), msg.Payload())
	})

	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to subscribe to topic %s: %w", topic, token.Error())
	}

	c.logger.Info(fmt.Sprintf("Subscribed to topic: %s", topic))
	return nil
}

// Unsubscribe unsubscribes from a topic
func (c *MQTTClient) Unsubscribe(topic string) error {
	c.mu.Lock()
	delete(c.handlers, topic)
	c.mu.Unlock()

	token := c.client.Unsubscribe(topic)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to unsubscribe from topic %s: %w", topic, token.Error())
	}

	c.logger.Info(fmt.Sprintf("Unsubscribed from topic: %s", topic))
	return nil
}

// handleMessage processes incoming MQTT messages
func (c *MQTTClient) handleMessage(topic string, payload []byte) {
	c.mu.RLock()
	handler, exists := c.handlers[topic]
	c.mu.RUnlock()

	if !exists {
		c.logger.Warn(fmt.Sprintf("No handler for topic: %s", topic))
		return
	}

	if err := handler(topic, payload); err != nil {
		c.logger.Error(fmt.Sprintf("Error handling message from topic %s: %v", topic, err))
	}
}

// onConnectionLost is called when connection to broker is lost
func (c *MQTTClient) onConnectionLost(client mqtt.Client, err error) {
	c.logger.Warn(fmt.Sprintf("MQTT connection lost: %v", err))
}

// onConnect is called when connection to broker is established
func (c *MQTTClient) onConnect(client mqtt.Client) {
	c.logger.Info("MQTT connection established")
}

// SubscribeToDeviceTelemetry subscribes to telemetry topics for all devices
func (c *MQTTClient) SubscribeToDeviceTelemetry() error {
	// Subscribe to wildcard topic for all device telemetry
	// Topic pattern: telemetry/{device_id}/data
	topic := "telemetry/+/data"

	handler := func(topic string, payload []byte) error {
		var data TelemetryData
		if err := json.Unmarshal(payload, &data); err != nil {
			return fmt.Errorf("failed to unmarshal telemetry data: %w", err)
		}

		// Set timestamp if not provided
		if data.Timestamp.IsZero() {
			data.Timestamp = time.Now()
		}

		// Store telemetry data
		ctx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
		defer cancel()

		if err := c.repository.StoreTelemetry(ctx, &data); err != nil {
			return fmt.Errorf("failed to store telemetry data: %w", err)
		}

		c.logger.Debug(fmt.Sprintf("Stored telemetry for device %s", data.DeviceID))
		return nil
	}

	return c.Subscribe(topic, 1, handler)
}

// SubscribeToDeviceHeartbeats subscribes to device heartbeat messages
func (c *MQTTClient) SubscribeToDeviceHeartbeats() error {
	// Topic pattern: telemetry/{device_id}/heartbeat
	topic := "telemetry/+/heartbeat"

	handler := func(topic string, payload []byte) error {
		var heartbeat struct {
			DeviceID  string    `json:"device_id"`
			Timestamp time.Time `json:"timestamp"`
			Status    string    `json:"status"`
		}

		if err := json.Unmarshal(payload, &heartbeat); err != nil {
			return fmt.Errorf("failed to unmarshal heartbeat: %w", err)
		}

		c.logger.Debug(fmt.Sprintf("Received heartbeat from device %s", heartbeat.DeviceID))
		return nil
	}

	return c.Subscribe(topic, 1, handler)
}

// Publish publishes a message to a topic
func (c *MQTTClient) Publish(topic string, qos byte, retained bool, payload interface{}) error {
	var data []byte
	var err error

	switch v := payload.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		data, err = json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal payload: %w", err)
		}
	}

	token := c.client.Publish(topic, qos, retained, data)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to publish to topic %s: %w", topic, token.Error())
	}

	return nil
}

// IsConnected returns whether the client is connected to the broker
func (c *MQTTClient) IsConnected() bool {
	return c.client.IsConnected()
}
