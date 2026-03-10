package template

import (
	"context"
	"testing"
	"time"

	"github.com/athena/platform-lib/pkg/config"
	"github.com/athena/platform-lib/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockRepository is a mock implementation of the Repository interface
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) CreateTemplate(ctx context.Context, template *Template) error {
	args := m.Called(ctx, template)
	return args.Error(0)
}

func (m *MockRepository) GetTemplate(ctx context.Context, id, version string) (*Template, error) {
	args := m.Called(ctx, id, version)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Template), args.Error(1)
}

func (m *MockRepository) UpdateTemplate(ctx context.Context, template *Template) error {
	args := m.Called(ctx, template)
	return args.Error(0)
}

func (m *MockRepository) DeleteTemplate(ctx context.Context, id, version string) error {
	args := m.Called(ctx, id, version)
	return args.Error(0)
}

func (m *MockRepository) ListTemplates(ctx context.Context, filters *TemplateFilters) ([]*Template, error) {
	args := m.Called(ctx, filters)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Template), args.Error(1)
}

func (m *MockRepository) SearchTemplates(ctx context.Context, query string, filters *TemplateFilters) ([]*Template, error) {
	args := m.Called(ctx, query, filters)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Template), args.Error(1)
}

func (m *MockRepository) GetTemplateVersions(ctx context.Context, id string) ([]string, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockRepository) CreateAsset(ctx context.Context, templateID, templateVersion string, asset *Asset) error {
	args := m.Called(ctx, templateID, templateVersion, asset)
	return args.Error(0)
}

func (m *MockRepository) GetAssets(ctx context.Context, templateID, templateVersion string) ([]*Asset, error) {
	args := m.Called(ctx, templateID, templateVersion)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Asset), args.Error(1)
}

func (m *MockRepository) DeleteAsset(ctx context.Context, templateID, templateVersion, assetType, assetPath string) error {
	args := m.Called(ctx, templateID, templateVersion, assetType, assetPath)
	return args.Error(0)
}

func (m *MockRepository) TemplateExists(ctx context.Context, id, version string) (bool, error) {
	args := m.Called(ctx, id, version)
	return args.Bool(0), args.Error(1)
}

func (m *MockRepository) GetTemplateCount(ctx context.Context, filters *TemplateFilters) (int64, error) {
	args := m.Called(ctx, filters)
	return args.Get(0).(int64), args.Error(1)
}

// setupTestService creates a test service with mocked dependencies
func setupTestService() (*Service, *MockRepository) {
	cfg := &config.Config{
		LogLevel:    "debug",
		ServiceName: "test-template-service",
	}
	logger := logger.New("debug", "test")
	mockRepo := new(MockRepository)

	service, err := NewService(cfg, logger, mockRepo)
	require.NoError(nil, err)

	return service, mockRepo
}

// createTestTemplate creates a sample template for testing
func createTestTemplate() *Template {
	now := time.Now()
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"sensorPin": map[string]interface{}{
				"type":    "integer",
				"minimum": 0,
				"maximum": 13,
			},
			"interval": map[string]interface{}{
				"type":    "integer",
				"default": 1000,
			},
		},
		"required": []string{"sensorPin"},
	}

	parameters := map[string]interface{}{
		"sensorPin": 2,
		"interval":  1000,
	}

	return &Template{
		ID:              "test-template-1",
		Name:            "Temperature Sensor",
		Version:         "1.0.0",
		Category:        "sensing",
		Description:     "A simple temperature sensor template",
		BoardsSupported: []string{"arduino-uno", "arduino-nano"},
		Schema:          schema,
		Parameters:      parameters,
		Libraries: []LibraryDependency{
			{Name: "DHT sensor library", Version: "1.4.4"},
			{Name: "Adafruit Unified Sensor", Version: "1.1.9"},
		},
		Assets: []Asset{
			{Type: "wiring_diagram", Path: "/diagrams/temp-sensor.png"},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func TestService_ListTemplates(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()

	filters := &TemplateFilters{
		Category:  "sensing",
		BoardType: "arduino-uno",
		Limit:     10,
		Offset:    0,
	}

	expectedTemplates := []*Template{createTestTemplate()}

	mockRepo.On("ListTemplates", ctx, filters).Return(expectedTemplates, nil)

	result, err := service.ListTemplates(ctx, filters)

	assert.NoError(t, err)
	assert.Equal(t, expectedTemplates, result)
	mockRepo.AssertExpectations(t)
}

func TestService_GetTemplate(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()

	template := createTestTemplate()
	templateID := "test-template-1"
	version := "1.0.0"

	mockRepo.On("GetTemplate", ctx, templateID, version).Return(template, nil)

	result, err := service.GetTemplate(ctx, templateID, version)

	assert.NoError(t, err)
	assert.Equal(t, template, result)
	mockRepo.AssertExpectations(t)
}

func TestService_GetTemplate_NotFound(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()

	templateID := "non-existent"
	version := "1.0.0"

	mockRepo.On("GetTemplate", ctx, templateID, version).Return(nil, assert.AnError)

	result, err := service.GetTemplate(ctx, templateID, version)

	assert.Error(t, err)
	assert.Nil(t, result)
	mockRepo.AssertExpectations(t)
}

func TestService_CreateTemplate(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()

	template := createTestTemplate()

	mockRepo.On("CreateTemplate", ctx, template).Return(nil)
	mockRepo.On("GetTemplateVersions", ctx, template.ID).Return([]string{"1.0.0"}, nil)

	err := service.CreateTemplate(ctx, template)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestService_CreateTemplate_ValidationError(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()

	template := createTestTemplate()
	// Make template invalid by removing required fields
	template.ID = ""

	err := service.CreateTemplate(ctx, template)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "template ID is required")
	mockRepo.AssertNotCalled(t, "CreateTemplate")
}

func TestService_UpdateTemplate(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()

	template := createTestTemplate()
	template.Description = "Updated description"

	// First, create the template so it exists for update
	mockRepo.On("GetTemplateVersions", ctx, template.ID).Return([]string{}, nil)
	mockRepo.On("UpdateTemplate", ctx, template).Return(nil)

	err := service.UpdateTemplate(ctx, template)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestService_DeleteTemplate(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()

	templateID := "test-template-1"
	version := "1.0.0"

	mockRepo.On("DeleteTemplate", ctx, templateID, version).Return(nil)

	err := service.DeleteTemplate(ctx, templateID, version)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestService_SearchTemplates(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()

	query := "temperature sensor"
	filters := &TemplateFilters{Category: "sensing"}
	expectedTemplates := []*Template{createTestTemplate()}

	mockRepo.On("SearchTemplates", ctx, query, filters).Return(expectedTemplates, nil)

	result, err := service.SearchTemplates(ctx, query, filters)

	assert.NoError(t, err)
	assert.Equal(t, expectedTemplates, result)
	mockRepo.AssertExpectations(t)
}

func TestService_GetTemplateVersions(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()

	templateID := "test-template-1"
	versions := []string{"1.0.0", "1.1.0", "2.0.0"}

	mockRepo.On("GetTemplateVersions", ctx, templateID).Return(versions, nil)

	result, err := service.GetTemplateVersions(ctx, templateID)

	assert.NoError(t, err)
	assert.Equal(t, versions, result)
	mockRepo.AssertExpectations(t)
}

func TestService_GetTemplateCount(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()

	filters := &TemplateFilters{Category: "sensing"}
	expectedCount := int64(5)

	mockRepo.On("GetTemplateCount", ctx, filters).Return(expectedCount, nil)

	result, err := service.GetTemplateCount(ctx, filters)

	assert.NoError(t, err)
	assert.Equal(t, expectedCount, result)
	mockRepo.AssertExpectations(t)
}

func TestService_CreateAsset(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()

	asset := &Asset{
		Type: "wiring_diagram",
		Path: "/diagrams/sensor.png",
		Metadata: map[string]interface{}{
			"width":  800,
			"height": 600,
		},
	}

	templateID := "test-template-1"
	templateVersion := "1.0.0"

	mockRepo.On("CreateAsset", ctx, templateID, templateVersion, asset).Return(nil)

	err := service.CreateAsset(ctx, templateID, templateVersion, asset)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestService_GetAssets(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()

	templateID := "test-template-1"
	templateVersion := "1.0.0"
	expectedAssets := []*Asset{
		{Type: "wiring_diagram", Path: "/diagrams/sensor.png"},
		{Type: "documentation", Path: "/docs/README.md"},
	}

	mockRepo.On("GetAssets", ctx, templateID, templateVersion).Return(expectedAssets, nil)

	result, err := service.GetAssets(ctx, templateID, templateVersion)

	assert.NoError(t, err)
	assert.Equal(t, expectedAssets, result)
	mockRepo.AssertExpectations(t)
}

func TestService_DeleteAsset(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()

	templateID := "test-template-1"
	templateVersion := "1.0.0"
	assetType := "wiring_diagram"
	assetPath := "/diagrams/sensor.png"

	mockRepo.On("DeleteAsset", ctx, templateID, templateVersion, assetType, assetPath).Return(nil)

	err := service.DeleteAsset(ctx, templateID, templateVersion, assetType, assetPath)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

// Additional tests for edge cases and comprehensive coverage

func TestService_CreateTemplate_EdgeCases(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()

	t.Run("Template with complex nested schema", func(t *testing.T) {
		template := &Template{
			ID:              "complex-template",
			Name:            "Complex Template",
			Version:         "1.0.0",
			Category:        "sensing",
			Description:     "Template with complex nested schema",
			BoardsSupported: []string{"arduino-uno", "arduino-nano"},
			Schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"sensor": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"type": map[string]interface{}{
								"type": "string",
								"enum": []string{"DHT22", "BME280", "DS18B20"},
							},
							"pin": map[string]interface{}{
								"type":    "integer",
								"minimum": 2,
								"maximum": 13,
							},
						},
						"required": []string{"type", "pin"},
					},
					"interval": map[string]interface{}{
						"type":    "integer",
						"default": 1000,
						"minimum": 100,
					},
				},
				"required": []string{"sensor"},
			},
			Parameters: map[string]interface{}{
				"sensor": map[string]interface{}{
					"type": "DHT22",
					"pin":  2,
				},
				"interval": 2000,
			},
		}

		mockRepo.On("CreateTemplate", ctx, template).Return(nil)
		mockRepo.On("GetTemplateVersions", ctx, template.ID).Return([]string{"1.0.0"}, nil)

		err := service.CreateTemplate(ctx, template)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Template with empty libraries array", func(t *testing.T) {
		template := &Template{
			ID:              "no-libs-template",
			Name:            "No Libraries Template",
			Version:         "1.0.0",
			Category:        "basic",
			Description:     "Template with no external libraries",
			BoardsSupported: []string{"arduino-uno"},
			Schema:          map[string]interface{}{"type": "object"},
			Parameters:      map[string]interface{}{},
			Libraries:       []LibraryDependency{},
		}

		mockRepo.On("CreateTemplate", ctx, template).Return(nil)
		mockRepo.On("GetTemplateVersions", ctx, template.ID).Return([]string{"1.0.0"}, nil)

		err := service.CreateTemplate(ctx, template)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})
}

func TestService_SearchTemplates_EdgeCases(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()

	t.Run("Empty search query", func(t *testing.T) {
		filters := &TemplateFilters{Category: "sensing"}
		expectedTemplates := []*Template{createTestTemplate()}

		mockRepo.On("SearchTemplates", ctx, "", filters).Return(expectedTemplates, nil)

		result, err := service.SearchTemplates(ctx, "", filters)

		assert.NoError(t, err)
		assert.Equal(t, expectedTemplates, result)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Search with special characters", func(t *testing.T) {
		query := "temperature+humidity DHT22!"
		filters := &TemplateFilters{}
		expectedTemplates := []*Template{createTestTemplate()}

		mockRepo.On("SearchTemplates", ctx, query, filters).Return(expectedTemplates, nil)

		result, err := service.SearchTemplates(ctx, query, filters)

		assert.NoError(t, err)
		assert.Equal(t, expectedTemplates, result)
		mockRepo.AssertExpectations(t)
	})
}

func TestService_ListTemplates_Pagination(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()

	t.Run("List with limit and offset", func(t *testing.T) {
		filters := &TemplateFilters{
			Limit:  5,
			Offset: 10,
		}
		expectedTemplates := []*Template{createTestTemplate()}

		mockRepo.On("ListTemplates", ctx, filters).Return(expectedTemplates, nil)

		result, err := service.ListTemplates(ctx, filters)

		assert.NoError(t, err)
		assert.Equal(t, expectedTemplates, result)
		mockRepo.AssertExpectations(t)
	})

	t.Run("List with all filter options", func(t *testing.T) {
		filters := &TemplateFilters{
			Category:        "sensing",
			BoardType:       "arduino-uno",
			SupportedBoards: []string{"arduino-uno", "arduino-nano"},
			Limit:           20,
			Offset:          0,
		}
		expectedTemplates := []*Template{createTestTemplate()}

		mockRepo.On("ListTemplates", ctx, filters).Return(expectedTemplates, nil)

		result, err := service.ListTemplates(ctx, filters)

		assert.NoError(t, err)
		assert.Equal(t, expectedTemplates, result)
		mockRepo.AssertExpectations(t)
	})
}
