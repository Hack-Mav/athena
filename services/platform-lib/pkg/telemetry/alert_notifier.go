package telemetry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/athena/platform-lib/pkg/logger"
)

// NotificationChannel represents a notification delivery channel
type NotificationChannel string

const (
	ChannelWebhook NotificationChannel = "webhook"
	ChannelEmail   NotificationChannel = "email"
	ChannelLog     NotificationChannel = "log"
)

// NotificationConfig represents configuration for a notification channel
type NotificationConfig struct {
	Channel  NotificationChannel    `json:"channel"`
	Enabled  bool                   `json:"enabled"`
	Settings map[string]interface{} `json:"settings"`
}

// AlertNotifier handles alert notifications through multiple channels
type AlertNotifier struct {
	logger   *logger.Logger
	channels []NotificationConfig
	mu       sync.RWMutex
	client   *http.Client
}

// NewAlertNotifier creates a new alert notifier
func NewAlertNotifier(logger *logger.Logger, channels []NotificationConfig) *AlertNotifier {
	return &AlertNotifier{
		logger:   logger,
		channels: channels,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// SendAlert sends an alert through all configured channels
func (an *AlertNotifier) SendAlert(alert *Alert) {
	an.mu.RLock()
	channels := an.channels
	an.mu.RUnlock()

	for _, channel := range channels {
		if !channel.Enabled {
			continue
		}

		switch channel.Channel {
		case ChannelWebhook:
			go an.sendWebhook(alert, channel.Settings)
		case ChannelEmail:
			go an.sendEmail(alert, channel.Settings)
		case ChannelLog:
			an.sendLog(alert)
		default:
			an.logger.Warn(fmt.Sprintf("Unknown notification channel: %s", channel.Channel))
		}
	}
}

// sendWebhook sends alert via webhook
func (an *AlertNotifier) sendWebhook(alert *Alert, settings map[string]interface{}) {
	webhookURL, ok := settings["url"].(string)
	if !ok || webhookURL == "" {
		an.logger.Error("Webhook URL not configured")
		return
	}

	payload := map[string]interface{}{
		"alert_id":        alert.AlertID,
		"device_id":       alert.DeviceID,
		"metric_name":     alert.MetricName,
		"severity":        alert.Severity,
		"message":         alert.Message,
		"current_value":   alert.CurrentValue,
		"threshold_value": alert.ThresholdValue,
		"triggered_at":    alert.TriggeredAt.Format(time.RFC3339),
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		an.logger.Error(fmt.Sprintf("Failed to marshal webhook payload: %v", err))
		return
	}

	req, err := http.NewRequest("POST", webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		an.logger.Error(fmt.Sprintf("Failed to create webhook request: %v", err))
		return
	}

	req.Header.Set("Content-Type", "application/json")

	// Add custom headers if configured
	if headers, ok := settings["headers"].(map[string]interface{}); ok {
		for key, value := range headers {
			if strValue, ok := value.(string); ok {
				req.Header.Set(key, strValue)
			}
		}
	}

	resp, err := an.client.Do(req)
	if err != nil {
		an.logger.Error(fmt.Sprintf("Failed to send webhook: %v", err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		an.logger.Info(fmt.Sprintf("Webhook notification sent for alert %s", alert.AlertID))
	} else {
		an.logger.Error(fmt.Sprintf("Webhook returned status %d for alert %s", resp.StatusCode, alert.AlertID))
	}
}

// sendEmail sends alert via email (placeholder implementation)
func (an *AlertNotifier) sendEmail(alert *Alert, settings map[string]interface{}) {
	// In a real implementation, this would integrate with an email service
	// like SendGrid, AWS SES, or SMTP

	to, ok := settings["to"].(string)
	if !ok || to == "" {
		an.logger.Error("Email recipient not configured")
		return
	}

	an.logger.Info(fmt.Sprintf("Email notification would be sent to %s for alert %s", to, alert.AlertID))
	an.logger.Info(fmt.Sprintf("Subject: [%s] Alert for device %s", alert.Severity, alert.DeviceID))
	an.logger.Info(fmt.Sprintf("Body: %s", alert.Message))
}

// sendLog logs the alert
func (an *AlertNotifier) sendLog(alert *Alert) {
	logMessage := fmt.Sprintf("[ALERT] Device: %s, Metric: %s, Severity: %s, Message: %s",
		alert.DeviceID, alert.MetricName, alert.Severity, alert.Message)

	switch alert.Severity {
	case "critical":
		an.logger.Error(logMessage)
	case "warning":
		an.logger.Warn(logMessage)
	default:
		an.logger.Info(logMessage)
	}
}

// AddChannel adds a notification channel
func (an *AlertNotifier) AddChannel(config NotificationConfig) {
	an.mu.Lock()
	defer an.mu.Unlock()

	an.channels = append(an.channels, config)
	an.logger.Info(fmt.Sprintf("Added notification channel: %s", config.Channel))
}

// RemoveChannel removes a notification channel
func (an *AlertNotifier) RemoveChannel(channel NotificationChannel) {
	an.mu.Lock()
	defer an.mu.Unlock()

	for i, config := range an.channels {
		if config.Channel == channel {
			an.channels = append(an.channels[:i], an.channels[i+1:]...)
			an.logger.Info(fmt.Sprintf("Removed notification channel: %s", channel))
			break
		}
	}
}

// UpdateChannel updates a notification channel configuration
func (an *AlertNotifier) UpdateChannel(channel NotificationChannel, settings map[string]interface{}) {
	an.mu.Lock()
	defer an.mu.Unlock()

	for i, config := range an.channels {
		if config.Channel == channel {
			an.channels[i].Settings = settings
			an.logger.Info(fmt.Sprintf("Updated notification channel: %s", channel))
			break
		}
	}
}

// GetChannels returns all configured channels
func (an *AlertNotifier) GetChannels() []NotificationConfig {
	an.mu.RLock()
	defer an.mu.RUnlock()

	channels := make([]NotificationConfig, len(an.channels))
	copy(channels, an.channels)
	return channels
}
