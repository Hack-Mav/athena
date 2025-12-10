package device

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/datastore"
)

// DatastoreRepository implements the Repository interface using Google Cloud Datastore
type DatastoreRepository struct {
	client *datastore.Client
}

// NewDatastoreRepository creates a new Datastore repository
func NewDatastoreRepository(client *datastore.Client) *DatastoreRepository {
	return &DatastoreRepository{
		client: client,
	}
}

// RegisterDevice registers a new device in Datastore
func (r *DatastoreRepository) RegisterDevice(ctx context.Context, device *Device) error {
	if device == nil {
		return fmt.Errorf("device cannot be nil")
	}

	// Check if device already exists
	exists, err := r.DeviceExists(ctx, device.DeviceID)
	if err != nil {
		return fmt.Errorf("failed to check device existence: %w", err)
	}
	if exists {
		return fmt.Errorf("device %s already exists", device.DeviceID)
	}

	// Convert to entity
	entity, err := device.ToEntity()
	if err != nil {
		return fmt.Errorf("failed to convert device to entity: %w", err)
	}

	// Set timestamps
	now := time.Now()
	entity.CreatedAt = now
	entity.UpdatedAt = now

	// Create Datastore key
	key := datastore.NameKey("Device", device.DeviceID, nil)

	// Store in Datastore
	_, err = r.client.Put(ctx, key, entity)
	if err != nil {
		return fmt.Errorf("failed to store device in Datastore: %w", err)
	}

	return nil
}

// GetDevice retrieves a device by ID from Datastore
func (r *DatastoreRepository) GetDevice(ctx context.Context, deviceID string) (*Device, error) {
	if deviceID == "" {
		return nil, fmt.Errorf("device ID cannot be empty")
	}

	// Create Datastore key
	key := datastore.NameKey("Device", deviceID, nil)

	// Retrieve from Datastore
	var entity DeviceEntity
	err := r.client.Get(ctx, key, &entity)
	if err != nil {
		if err == datastore.ErrNoSuchEntity {
			return nil, fmt.Errorf("device %s not found", deviceID)
		}
		return nil, fmt.Errorf("failed to retrieve device from Datastore: %w", err)
	}

	// Convert to device
	device, err := entity.FromEntity()
	if err != nil {
		return nil, fmt.Errorf("failed to convert entity to device: %w", err)
	}

	return device, nil
}

// UpdateDevice updates an existing device in Datastore
func (r *DatastoreRepository) UpdateDevice(ctx context.Context, device *Device) error {
	if device == nil {
		return fmt.Errorf("device cannot be nil")
	}

	// Check if device exists
	exists, err := r.DeviceExists(ctx, device.DeviceID)
	if err != nil {
		return fmt.Errorf("failed to check device existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("device %s not found", device.DeviceID)
	}

	// Convert to entity
	entity, err := device.ToEntity()
	if err != nil {
		return fmt.Errorf("failed to convert device to entity: %w", err)
	}

	// Update timestamp
	entity.UpdatedAt = time.Now()

	// Create Datastore key
	key := datastore.NameKey("Device", device.DeviceID, nil)

	// Update in Datastore
	_, err = r.client.Put(ctx, key, entity)
	if err != nil {
		return fmt.Errorf("failed to update device in Datastore: %w", err)
	}

	return nil
}

// DeleteDevice deletes a device by ID from Datastore
func (r *DatastoreRepository) DeleteDevice(ctx context.Context, deviceID string) error {
	if deviceID == "" {
		return fmt.Errorf("device ID cannot be empty")
	}

	// Check if device exists
	exists, err := r.DeviceExists(ctx, deviceID)
	if err != nil {
		return fmt.Errorf("failed to check device existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("device %s not found", deviceID)
	}

	// Create Datastore key
	key := datastore.NameKey("Device", deviceID, nil)

	// Delete from Datastore
	err = r.client.Delete(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to delete device from Datastore: %w", err)
	}

	return nil
}

// ListDevices returns devices matching the given filters from Datastore
func (r *DatastoreRepository) ListDevices(ctx context.Context, filters *DeviceFilters) ([]*Device, error) {
	query := datastore.NewQuery("Device")

	// Apply filters
	if filters != nil {
		if filters.Status != "" {
			query = query.Filter("status =", string(filters.Status))
		}
		if filters.BoardType != "" {
			query = query.Filter("board_type =", filters.BoardType)
		}
		if filters.TemplateID != "" {
			query = query.Filter("template_id =", filters.TemplateID)
		}
		if filters.OTAChannel != "" {
			query = query.Filter("ota_channel =", filters.OTAChannel)
		}
		if filters.LastSeenBefore != nil {
			query = query.Filter("last_seen <", *filters.LastSeenBefore)
		}
		if filters.LastSeenAfter != nil {
			query = query.Filter("last_seen >", *filters.LastSeenAfter)
		}
		if filters.Limit > 0 {
			query = query.Limit(filters.Limit)
		}
		if filters.Offset > 0 {
			query = query.Offset(filters.Offset)
		}
	}

	// Order by last seen (most recent first)
	query = query.Order("-last_seen")

	// Execute query
	var entities []DeviceEntity
	_, err := r.client.GetAll(ctx, query, &entities)
	if err != nil {
		return nil, fmt.Errorf("failed to query devices from Datastore: %w", err)
	}

	// Convert entities to devices
	var devices []*Device
	for _, entity := range entities {
		device, err := entity.FromEntity()
		if err != nil {
			return nil, fmt.Errorf("failed to convert entity to device: %w", err)
		}
		devices = append(devices, device)
	}

	return devices, nil
}

// GetDeviceCount returns the count of devices matching the filters from Datastore
func (r *DatastoreRepository) GetDeviceCount(ctx context.Context, filters *DeviceFilters) (int64, error) {
	query := datastore.NewQuery("Device")

	// Apply filters
	if filters != nil {
		if filters.Status != "" {
			query = query.Filter("status =", string(filters.Status))
		}
		if filters.BoardType != "" {
			query = query.Filter("board_type =", filters.BoardType)
		}
		if filters.TemplateID != "" {
			query = query.Filter("template_id =", filters.TemplateID)
		}
		if filters.OTAChannel != "" {
			query = query.Filter("ota_channel =", filters.OTAChannel)
		}
		if filters.LastSeenBefore != nil {
			query = query.Filter("last_seen <", *filters.LastSeenBefore)
		}
		if filters.LastSeenAfter != nil {
			query = query.Filter("last_seen >", *filters.LastSeenAfter)
		}
	}

	// Count only
	count, err := r.client.Count(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to count devices in Datastore: %w", err)
	}

	return int64(count), nil
}

// SearchDevices searches devices by query string in Datastore
func (r *DatastoreRepository) SearchDevices(ctx context.Context, query string, filters *DeviceFilters) ([]*Device, error) {
	// Note: Datastore doesn't support full-text search natively
	// For production, you would typically use Google Cloud Search API or Elasticsearch
	// Here we'll implement a simple approach by getting all devices and filtering in memory

	// Get all devices first
	allDevices, err := r.ListDevices(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get devices for search: %w", err)
	}

	// Filter by query
	var result []*Device
	queryLower := strings.ToLower(query)

	for _, device := range allDevices {
		if r.matchesQuery(device, queryLower) {
			result = append(result, device)
		}
	}

	return result, nil
}

// UpdateDeviceStatus updates the status and last seen time for a device
func (r *DatastoreRepository) UpdateDeviceStatus(ctx context.Context, deviceID string, status DeviceStatus, lastSeen time.Time) error {
	if deviceID == "" {
		return fmt.Errorf("device ID cannot be empty")
	}

	// Get existing device
	device, err := r.GetDevice(ctx, deviceID)
	if err != nil {
		return fmt.Errorf("failed to get device for status update: %w", err)
	}

	// Update status and last seen
	device.Status = status
	device.LastSeen = lastSeen
	device.UpdatedAt = time.Now()

	// Save updated device
	return r.UpdateDevice(ctx, device)
}

// GetDevicesByStatus returns all devices with the specified status
func (r *DatastoreRepository) GetDevicesByStatus(ctx context.Context, status DeviceStatus) ([]*Device, error) {
	filters := &DeviceFilters{
		Status: status,
	}
	return r.ListDevices(ctx, filters)
}

// GetOfflineDevices returns devices that haven't been seen within the timeout period
func (r *DatastoreRepository) GetOfflineDevices(ctx context.Context, timeout time.Duration) ([]*Device, error) {
	cutoffTime := time.Now().Add(-timeout)
	filters := &DeviceFilters{
		LastSeenBefore: &cutoffTime,
	}
	return r.ListDevices(ctx, filters)
}

// GetDeviceHealthStatus returns aggregated health status for all devices
func (r *DatastoreRepository) GetDeviceHealthStatus(ctx context.Context) (*DeviceHealthStatus, error) {
	// Get total count
	totalCount, err := r.GetDeviceCount(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get total device count: %w", err)
	}

	// Get online count
	onlineCount, err := r.GetDeviceCount(ctx, &DeviceFilters{Status: DeviceStatusOnline})
	if err != nil {
		return nil, fmt.Errorf("failed to get online device count: %w", err)
	}

	// Get offline count
	offlineCount, err := r.GetDeviceCount(ctx, &DeviceFilters{Status: DeviceStatusOffline})
	if err != nil {
		return nil, fmt.Errorf("failed to get offline device count: %w", err)
	}

	// Get error count
	errorCount, err := r.GetDeviceCount(ctx, &DeviceFilters{Status: DeviceStatusError})
	if err != nil {
		return nil, fmt.Errorf("failed to get error device count: %w", err)
	}

	return &DeviceHealthStatus{
		TotalDevices:   totalCount,
		OnlineDevices:  onlineCount,
		OfflineDevices: offlineCount,
		ErrorDevices:   errorCount,
	}, nil
}

// GetDevicesByTemplate returns all devices using the specified template
func (r *DatastoreRepository) GetDevicesByTemplate(ctx context.Context, templateID string) ([]*Device, error) {
	filters := &DeviceFilters{
		TemplateID: templateID,
	}
	return r.ListDevices(ctx, filters)
}

// GetDevicesByOTAChannel returns all devices on the specified OTA channel
func (r *DatastoreRepository) GetDevicesByOTAChannel(ctx context.Context, channel string) ([]*Device, error) {
	filters := &DeviceFilters{
		OTAChannel: channel,
	}
	return r.ListDevices(ctx, filters)
}

// DeviceExists checks if a device exists in Datastore
func (r *DatastoreRepository) DeviceExists(ctx context.Context, deviceID string) (bool, error) {
	if deviceID == "" {
		return false, fmt.Errorf("device ID cannot be empty")
	}

	// Create Datastore key
	key := datastore.NameKey("Device", deviceID, nil)

	// Check existence
	var entity DeviceEntity
	err := r.client.Get(ctx, key, &entity)
	if err != nil {
		if err == datastore.ErrNoSuchEntity {
			return false, nil
		}
		return false, fmt.Errorf("failed to check device existence in Datastore: %w", err)
	}

	return true, nil
}

// GetDevicesLastSeenBefore returns devices last seen before the specified time
func (r *DatastoreRepository) GetDevicesLastSeenBefore(ctx context.Context, before time.Time) ([]*Device, error) {
	filters := &DeviceFilters{
		LastSeenBefore: &before,
	}
	return r.ListDevices(ctx, filters)
}

// Helper methods

// matchesQuery checks if a device matches the search query
func (r *DatastoreRepository) matchesQuery(device *Device, query string) bool {
	if query == "" {
		return true
	}

	// Simple text search in device ID, board type, and template ID
	if strings.Contains(strings.ToLower(device.DeviceID), query) {
		return true
	}

	if strings.Contains(strings.ToLower(device.BoardType), query) {
		return true
	}

	if strings.Contains(strings.ToLower(device.TemplateID), query) {
		return true
	}

	return false
}
