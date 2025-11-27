package telemetry

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/athena/platform-lib/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// StreamManager manages WebSocket connections for real-time telemetry streaming
type StreamManager struct {
	clients    map[string]map[*websocket.Conn]bool // deviceID -> connections
	mu         sync.RWMutex
	logger     *logger.Logger
	repository Repository
	upgrader   websocket.Upgrader
}

// NewStreamManager creates a new stream manager
func NewStreamManager(logger *logger.Logger, repository Repository) *StreamManager {
	return &StreamManager{
		clients:    make(map[string]map[*websocket.Conn]bool),
		logger:     logger,
		repository: repository,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				// In production, implement proper origin checking
				return true
			},
		},
	}
}

// HandleWebSocket handles WebSocket connections for device telemetry streaming
func (sm *StreamManager) HandleWebSocket(c *gin.Context) {
	deviceID := c.Param("deviceId")

	conn, err := sm.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		sm.logger.Error(fmt.Sprintf("Failed to upgrade WebSocket connection: %v", err))
		return
	}
	defer conn.Close()

	// Register client
	sm.registerClient(deviceID, conn)
	defer sm.unregisterClient(deviceID, conn)

	sm.logger.Info(fmt.Sprintf("WebSocket client connected for device %s", deviceID))

	// Send initial historical data
	if err := sm.sendHistoricalData(conn, deviceID); err != nil {
		sm.logger.Error(fmt.Sprintf("Failed to send historical data: %v", err))
	}

	// Keep connection alive and handle incoming messages
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				sm.logger.Error(fmt.Sprintf("WebSocket error: %v", err))
			}
			break
		}
	}

	sm.logger.Info(fmt.Sprintf("WebSocket client disconnected for device %s", deviceID))
}

// registerClient registers a WebSocket connection for a device
func (sm *StreamManager) registerClient(deviceID string, conn *websocket.Conn) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.clients[deviceID] == nil {
		sm.clients[deviceID] = make(map[*websocket.Conn]bool)
	}
	sm.clients[deviceID][conn] = true
}

// unregisterClient unregisters a WebSocket connection
func (sm *StreamManager) unregisterClient(deviceID string, conn *websocket.Conn) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if clients, exists := sm.clients[deviceID]; exists {
		delete(clients, conn)
		if len(clients) == 0 {
			delete(sm.clients, deviceID)
		}
	}
}

// BroadcastTelemetry broadcasts telemetry data to all connected clients for a device
func (sm *StreamManager) BroadcastTelemetry(deviceID string, data *TelemetryData) {
	sm.mu.RLock()
	clients, exists := sm.clients[deviceID]
	sm.mu.RUnlock()

	if !exists || len(clients) == 0 {
		return
	}

	message, err := json.Marshal(data)
	if err != nil {
		sm.logger.Error(fmt.Sprintf("Failed to marshal telemetry data: %v", err))
		return
	}

	sm.mu.RLock()
	defer sm.mu.RUnlock()

	for conn := range clients {
		if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
			sm.logger.Error(fmt.Sprintf("Failed to send message to WebSocket client: %v", err))
			conn.Close()
			delete(clients, conn)
		}
	}
}

// sendHistoricalData sends recent historical data to a newly connected client
func (sm *StreamManager) sendHistoricalData(conn *websocket.Conn, deviceID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get last hour of data
	timeRange := TimeRange{
		Start: time.Now().Add(-1 * time.Hour),
		End:   time.Now(),
	}

	metrics, err := sm.repository.GetDeviceMetrics(ctx, deviceID, timeRange)
	if err != nil {
		return fmt.Errorf("failed to get historical metrics: %w", err)
	}

	// Send historical data
	for _, metric := range metrics {
		message, err := json.Marshal(metric)
		if err != nil {
			continue
		}

		if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
			return fmt.Errorf("failed to send historical data: %w", err)
		}
	}

	return nil
}

// GetActiveConnections returns the number of active connections for a device
func (sm *StreamManager) GetActiveConnections(deviceID string) int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if clients, exists := sm.clients[deviceID]; exists {
		return len(clients)
	}
	return 0
}

// CloseAllConnections closes all WebSocket connections
func (sm *StreamManager) CloseAllConnections() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for deviceID, clients := range sm.clients {
		for conn := range clients {
			conn.Close()
		}
		delete(sm.clients, deviceID)
	}
}
