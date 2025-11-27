package ota

import (
	"context"
	"fmt"
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

// CreateRelease creates a new firmware release in Datastore
func (r *DatastoreRepository) CreateRelease(ctx context.Context, release *FirmwareRelease) error {
	if release == nil {
		return fmt.Errorf("release cannot be nil")
	}

	// Check if release already exists
	exists, err := r.ReleaseExists(ctx, release.ReleaseID)
	if err != nil {
		return fmt.Errorf("failed to check release existence: %w", err)
	}
	if exists {
		return fmt.Errorf("release %s already exists", release.ReleaseID)
	}

	// Convert to entity
	entity, err := release.ToEntity()
	if err != nil {
		return fmt.Errorf("failed to convert release to entity: %w", err)
	}

	// Create Datastore key
	key := datastore.NameKey("FirmwareRelease", release.ReleaseID, nil)

	// Store in Datastore
	_, err = r.client.Put(ctx, key, entity)
	if err != nil {
		return fmt.Errorf("failed to store release in Datastore: %w", err)
	}

	return nil
}

// GetRelease retrieves a firmware release by ID from Datastore
func (r *DatastoreRepository) GetRelease(ctx context.Context, releaseID string) (*FirmwareRelease, error) {
	if releaseID == "" {
		return nil, fmt.Errorf("release ID cannot be empty")
	}

	// Create Datastore key
	key := datastore.NameKey("FirmwareRelease", releaseID, nil)

	// Retrieve from Datastore
	var entity FirmwareReleaseEntity
	err := r.client.Get(ctx, key, &entity)
	if err != nil {
		if err == datastore.ErrNoSuchEntity {
			return nil, fmt.Errorf("release %s not found", releaseID)
		}
		return nil, fmt.Errorf("failed to retrieve release from Datastore: %w", err)
	}

	// Convert to release
	release, err := entity.FromEntity()
	if err != nil {
		return nil, fmt.Errorf("failed to convert entity to release: %w", err)
	}

	return release, nil
}

// GetReleaseByVersion retrieves a firmware release by template ID, version, and channel
func (r *DatastoreRepository) GetReleaseByVersion(ctx context.Context, templateID, version string, channel ReleaseChannel) (*FirmwareRelease, error) {
	query := datastore.NewQuery("FirmwareRelease").
		Filter("template_id =", templateID).
		Filter("version =", version).
		Filter("channel =", string(channel)).
		Limit(1)

	var entities []FirmwareReleaseEntity
	_, err := r.client.GetAll(ctx, query, &entities)
	if err != nil {
		return nil, fmt.Errorf("failed to query release from Datastore: %w", err)
	}

	if len(entities) == 0 {
		return nil, fmt.Errorf("release not found for template %s, version %s, channel %s", templateID, version, channel)
	}

	release, err := entities[0].FromEntity()
	if err != nil {
		return nil, fmt.Errorf("failed to convert entity to release: %w", err)
	}

	return release, nil
}

// ListReleases lists all firmware releases for a template and channel
func (r *DatastoreRepository) ListReleases(ctx context.Context, templateID string, channel ReleaseChannel) ([]*FirmwareRelease, error) {
	query := datastore.NewQuery("FirmwareRelease")

	if templateID != "" {
		query = query.Filter("template_id =", templateID)
	}

	if channel != "" {
		query = query.Filter("channel =", string(channel))
	}

	// Order by created_at descending
	query = query.Order("-created_at")

	var entities []FirmwareReleaseEntity
	_, err := r.client.GetAll(ctx, query, &entities)
	if err != nil {
		return nil, fmt.Errorf("failed to query releases from Datastore: %w", err)
	}

	var releases []*FirmwareRelease
	for _, entity := range entities {
		release, err := entity.FromEntity()
		if err != nil {
			return nil, fmt.Errorf("failed to convert entity to release: %w", err)
		}
		releases = append(releases, release)
	}

	return releases, nil
}

// DeleteRelease deletes a firmware release from Datastore
func (r *DatastoreRepository) DeleteRelease(ctx context.Context, releaseID string) error {
	if releaseID == "" {
		return fmt.Errorf("release ID cannot be empty")
	}

	// Create Datastore key
	key := datastore.NameKey("FirmwareRelease", releaseID, nil)

	// Delete from Datastore
	err := r.client.Delete(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to delete release from Datastore: %w", err)
	}

	return nil
}

// ReleaseExists checks if a release exists in Datastore
func (r *DatastoreRepository) ReleaseExists(ctx context.Context, releaseID string) (bool, error) {
	if releaseID == "" {
		return false, fmt.Errorf("release ID cannot be empty")
	}

	key := datastore.NameKey("FirmwareRelease", releaseID, nil)

	var entity FirmwareReleaseEntity
	err := r.client.Get(ctx, key, &entity)
	if err != nil {
		if err == datastore.ErrNoSuchEntity {
			return false, nil
		}
		return false, fmt.Errorf("failed to check release existence in Datastore: %w", err)
	}

	return true, nil
}

// CreateDeployment creates a new OTA deployment in Datastore
func (r *DatastoreRepository) CreateDeployment(ctx context.Context, deployment *OTADeployment) error {
	if deployment == nil {
		return fmt.Errorf("deployment cannot be nil")
	}

	entity, err := deployment.ToEntity()
	if err != nil {
		return fmt.Errorf("failed to convert deployment to entity: %w", err)
	}

	key := datastore.NameKey("OTADeployment", deployment.DeploymentID, nil)

	_, err = r.client.Put(ctx, key, entity)
	if err != nil {
		return fmt.Errorf("failed to store deployment in Datastore: %w", err)
	}

	return nil
}

// GetDeployment retrieves an OTA deployment by ID from Datastore
func (r *DatastoreRepository) GetDeployment(ctx context.Context, deploymentID string) (*OTADeployment, error) {
	if deploymentID == "" {
		return nil, fmt.Errorf("deployment ID cannot be empty")
	}

	key := datastore.NameKey("OTADeployment", deploymentID, nil)

	var entity OTADeploymentEntity
	err := r.client.Get(ctx, key, &entity)
	if err != nil {
		if err == datastore.ErrNoSuchEntity {
			return nil, fmt.Errorf("deployment %s not found", deploymentID)
		}
		return nil, fmt.Errorf("failed to retrieve deployment from Datastore: %w", err)
	}

	deployment, err := entity.FromEntity()
	if err != nil {
		return nil, fmt.Errorf("failed to convert entity to deployment: %w", err)
	}

	return deployment, nil
}

// UpdateDeployment updates an existing OTA deployment in Datastore
func (r *DatastoreRepository) UpdateDeployment(ctx context.Context, deployment *OTADeployment) error {
	if deployment == nil {
		return fmt.Errorf("deployment cannot be nil")
	}

	deployment.UpdatedAt = time.Now()

	entity, err := deployment.ToEntity()
	if err != nil {
		return fmt.Errorf("failed to convert deployment to entity: %w", err)
	}

	key := datastore.NameKey("OTADeployment", deployment.DeploymentID, nil)

	_, err = r.client.Put(ctx, key, entity)
	if err != nil {
		return fmt.Errorf("failed to update deployment in Datastore: %w", err)
	}

	return nil
}

// ListDeployments lists all deployments for a release
func (r *DatastoreRepository) ListDeployments(ctx context.Context, releaseID string) ([]*OTADeployment, error) {
	query := datastore.NewQuery("OTADeployment")

	if releaseID != "" {
		query = query.Filter("release_id =", releaseID)
	}

	query = query.Order("-created_at")

	var entities []OTADeploymentEntity
	_, err := r.client.GetAll(ctx, query, &entities)
	if err != nil {
		return nil, fmt.Errorf("failed to query deployments from Datastore: %w", err)
	}

	var deployments []*OTADeployment
	for _, entity := range entities {
		deployment, err := entity.FromEntity()
		if err != nil {
			return nil, fmt.Errorf("failed to convert entity to deployment: %w", err)
		}
		deployments = append(deployments, deployment)
	}

	return deployments, nil
}

// GetActiveDeployments retrieves all active deployments
func (r *DatastoreRepository) GetActiveDeployments(ctx context.Context) ([]*OTADeployment, error) {
	query := datastore.NewQuery("OTADeployment").
		Filter("status =", string(DeploymentStatusActive)).
		Order("-created_at")

	var entities []OTADeploymentEntity
	_, err := r.client.GetAll(ctx, query, &entities)
	if err != nil {
		return nil, fmt.Errorf("failed to query active deployments from Datastore: %w", err)
	}

	var deployments []*OTADeployment
	for _, entity := range entities {
		deployment, err := entity.FromEntity()
		if err != nil {
			return nil, fmt.Errorf("failed to convert entity to deployment: %w", err)
		}
		deployments = append(deployments, deployment)
	}

	return deployments, nil
}

// CreateDeviceUpdate creates a new device update record in Datastore
func (r *DatastoreRepository) CreateDeviceUpdate(ctx context.Context, update *DeviceUpdate) error {
	if update == nil {
		return fmt.Errorf("update cannot be nil")
	}

	entity, err := update.ToEntity()
	if err != nil {
		return fmt.Errorf("failed to convert update to entity: %w", err)
	}

	// Use composite key: device_id#release_id
	keyName := fmt.Sprintf("%s#%s", update.DeviceID, update.ReleaseID)
	key := datastore.NameKey("DeviceUpdate", keyName, nil)

	_, err = r.client.Put(ctx, key, entity)
	if err != nil {
		return fmt.Errorf("failed to store device update in Datastore: %w", err)
	}

	return nil
}

// GetDeviceUpdate retrieves a device update by device ID and release ID
func (r *DatastoreRepository) GetDeviceUpdate(ctx context.Context, deviceID, releaseID string) (*DeviceUpdate, error) {
	if deviceID == "" || releaseID == "" {
		return nil, fmt.Errorf("device ID and release ID cannot be empty")
	}

	keyName := fmt.Sprintf("%s#%s", deviceID, releaseID)
	key := datastore.NameKey("DeviceUpdate", keyName, nil)

	var entity DeviceUpdateEntity
	err := r.client.Get(ctx, key, &entity)
	if err != nil {
		if err == datastore.ErrNoSuchEntity {
			return nil, fmt.Errorf("device update not found for device %s and release %s", deviceID, releaseID)
		}
		return nil, fmt.Errorf("failed to retrieve device update from Datastore: %w", err)
	}

	update, err := entity.FromEntity()
	if err != nil {
		return nil, fmt.Errorf("failed to convert entity to update: %w", err)
	}

	return update, nil
}

// UpdateDeviceUpdate updates an existing device update in Datastore
func (r *DatastoreRepository) UpdateDeviceUpdate(ctx context.Context, update *DeviceUpdate) error {
	if update == nil {
		return fmt.Errorf("update cannot be nil")
	}

	entity, err := update.ToEntity()
	if err != nil {
		return fmt.Errorf("failed to convert update to entity: %w", err)
	}

	keyName := fmt.Sprintf("%s#%s", update.DeviceID, update.ReleaseID)
	key := datastore.NameKey("DeviceUpdate", keyName, nil)

	_, err = r.client.Put(ctx, key, entity)
	if err != nil {
		return fmt.Errorf("failed to update device update in Datastore: %w", err)
	}

	return nil
}

// ListDeviceUpdates lists all device updates for a deployment
func (r *DatastoreRepository) ListDeviceUpdates(ctx context.Context, deploymentID string) ([]*DeviceUpdate, error) {
	query := datastore.NewQuery("DeviceUpdate").
		Filter("deployment_id =", deploymentID).
		Order("-started_at")

	var entities []DeviceUpdateEntity
	_, err := r.client.GetAll(ctx, query, &entities)
	if err != nil {
		return nil, fmt.Errorf("failed to query device updates from Datastore: %w", err)
	}

	var updates []*DeviceUpdate
	for _, entity := range entities {
		update, err := entity.FromEntity()
		if err != nil {
			return nil, fmt.Errorf("failed to convert entity to update: %w", err)
		}
		updates = append(updates, update)
	}

	return updates, nil
}

// GetDeviceUpdatesByStatus retrieves device updates by deployment and status
func (r *DatastoreRepository) GetDeviceUpdatesByStatus(ctx context.Context, deploymentID string, status UpdateStatus) ([]*DeviceUpdate, error) {
	query := datastore.NewQuery("DeviceUpdate").
		Filter("deployment_id =", deploymentID).
		Filter("status =", string(status)).
		Order("-started_at")

	var entities []DeviceUpdateEntity
	_, err := r.client.GetAll(ctx, query, &entities)
	if err != nil {
		return nil, fmt.Errorf("failed to query device updates from Datastore: %w", err)
	}

	var updates []*DeviceUpdate
	for _, entity := range entities {
		update, err := entity.FromEntity()
		if err != nil {
			return nil, fmt.Errorf("failed to convert entity to update: %w", err)
		}
		updates = append(updates, update)
	}

	return updates, nil
}

// GetLatestUpdateForDevice retrieves the latest update for a device
func (r *DatastoreRepository) GetLatestUpdateForDevice(ctx context.Context, deviceID string) (*DeviceUpdate, error) {
	query := datastore.NewQuery("DeviceUpdate").
		Filter("device_id =", deviceID).
		Order("-started_at").
		Limit(1)

	var entities []DeviceUpdateEntity
	_, err := r.client.GetAll(ctx, query, &entities)
	if err != nil {
		return nil, fmt.Errorf("failed to query device update from Datastore: %w", err)
	}

	if len(entities) == 0 {
		return nil, fmt.Errorf("no updates found for device %s", deviceID)
	}

	update, err := entities[0].FromEntity()
	if err != nil {
		return nil, fmt.Errorf("failed to convert entity to update: %w", err)
	}

	return update, nil
}

// GetDeploymentStats retrieves deployment statistics
func (r *DatastoreRepository) GetDeploymentStats(ctx context.Context, deploymentID string) (successCount, failureCount, pendingCount int, err error) {
	// Get all updates for the deployment
	updates, err := r.ListDeviceUpdates(ctx, deploymentID)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to get device updates: %w", err)
	}

	// Count by status
	for _, update := range updates {
		switch update.Status {
		case UpdateStatusCompleted:
			successCount++
		case UpdateStatusFailed:
			failureCount++
		case UpdateStatusPending, UpdateStatusDownloading, UpdateStatusInstalling:
			pendingCount++
		}
	}

	return successCount, failureCount, pendingCount, nil
}

// GetDevicesPendingUpdate retrieves devices pending update for a deployment
func (r *DatastoreRepository) GetDevicesPendingUpdate(ctx context.Context, deploymentID string, limit int) ([]*DeviceUpdate, error) {
	query := datastore.NewQuery("DeviceUpdate").
		Filter("deployment_id =", deploymentID).
		Filter("status =", string(UpdateStatusPending)).
		Order("started_at")

	if limit > 0 {
		query = query.Limit(limit)
	}

	var entities []DeviceUpdateEntity
	_, err := r.client.GetAll(ctx, query, &entities)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending device updates from Datastore: %w", err)
	}

	var updates []*DeviceUpdate
	for _, entity := range entities {
		update, err := entity.FromEntity()
		if err != nil {
			return nil, fmt.Errorf("failed to convert entity to update: %w", err)
		}
		updates = append(updates, update)
	}

	return updates, nil
}
