package template

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplate_ToEntity(t *testing.T) {
	template := createTestTemplate()

	entity, err := template.ToEntity()

	assert.NoError(t, err)
	assert.NotNil(t, entity)

	// Verify basic fields
	assert.Equal(t, template.ID, entity.ID)
	assert.Equal(t, template.Name, entity.Name)
	assert.Equal(t, template.Version, entity.Version)
	assert.Equal(t, template.Category, entity.Category)
	assert.Equal(t, template.Description, entity.Description)
	assert.Equal(t, template.BoardsSupported, entity.BoardsSupported)
	assert.Equal(t, template.CreatedAt, entity.CreatedAt)
	assert.Equal(t, template.UpdatedAt, entity.UpdatedAt)

	// Verify JSON fields
	var schema map[string]interface{}
	err = json.Unmarshal([]byte(entity.SchemaJSON), &schema)
	assert.NoError(t, err)
	assert.Equal(t, template.Schema, schema)

	var parameters map[string]interface{}
	err = json.Unmarshal([]byte(entity.ParametersJSON), &parameters)
	assert.NoError(t, err)
	assert.Equal(t, template.Parameters, parameters)

	var libraries []LibraryDependency
	err = json.Unmarshal([]byte(entity.LibrariesJSON), &libraries)
	assert.NoError(t, err)
	assert.Equal(t, template.Libraries, libraries)
}

func TestTemplate_ToEntity_EmptyFields(t *testing.T) {
	template := &Template{
		ID:              "test",
		Name:            "Test Template",
		Version:         "1.0.0",
		Category:        "test",
		Description:     "Test description",
		BoardsSupported: []string{},
		Schema:          nil,
		Parameters:      nil,
		Libraries:       []LibraryDependency{},
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	entity, err := template.ToEntity()

	assert.NoError(t, err)
	assert.NotNil(t, entity)

	// Empty JSON fields should be empty strings, not null
	assert.Equal(t, "", entity.SchemaJSON)
	assert.Equal(t, "", entity.ParametersJSON)
	assert.Equal(t, "[]", entity.LibrariesJSON)
}

func TestTemplate_ToEntity_InvalidSchema(t *testing.T) {
	template := createTestTemplate()
	// Create an invalid JSON value
	template.Schema = map[string]interface{}{
		"invalid": make(chan int), // Channels cannot be marshaled to JSON
	}

	entity, err := template.ToEntity()

	assert.Error(t, err)
	assert.Nil(t, entity)
	assert.Contains(t, err.Error(), "json: unsupported type")
}

func TestTemplateEntity_FromEntity(t *testing.T) {
	entity := &TemplateEntity{
		ID:              "test-template-1",
		Name:            "Temperature Sensor",
		Version:         "1.0.0",
		Category:        "sensing",
		Description:     "A simple temperature sensor template",
		BoardsSupported: []string{"arduino-uno", "arduino-nano"},
		SchemaJSON:      `{"type":"object","properties":{"sensorPin":{"type":"integer"}}}`,
		ParametersJSON:  `{"sensorPin":2,"interval":1000}`,
		LibrariesJSON:   `[{"name":"DHT sensor library","version":"1.4.4"}]`,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	template, err := entity.FromEntity()

	assert.NoError(t, err)
	assert.NotNil(t, template)

	// Verify basic fields
	assert.Equal(t, entity.ID, template.ID)
	assert.Equal(t, entity.Name, template.Name)
	assert.Equal(t, entity.Version, template.Version)
	assert.Equal(t, entity.Category, template.Category)
	assert.Equal(t, entity.Description, template.Description)
	assert.Equal(t, entity.BoardsSupported, template.BoardsSupported)
	assert.Equal(t, entity.CreatedAt, template.CreatedAt)
	assert.Equal(t, entity.UpdatedAt, template.UpdatedAt)

	// Verify JSON fields
	assert.Equal(t, "object", template.Schema["type"])
	assert.Equal(t, float64(2), template.Parameters["sensorPin"])
	assert.Len(t, template.Libraries, 1)
	assert.Equal(t, "DHT sensor library", template.Libraries[0].Name)

	// Assets should be empty slice
	assert.Empty(t, template.Assets)
}

func TestTemplateEntity_FromEntity_EmptyJSON(t *testing.T) {
	entity := &TemplateEntity{
		ID:              "test",
		Name:            "Test Template",
		Version:         "1.0.0",
		Category:        "test",
		Description:     "Test description",
		BoardsSupported: []string{},
		SchemaJSON:      "",
		ParametersJSON:  "",
		LibrariesJSON:   "",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	template, err := entity.FromEntity()

	assert.NoError(t, err)
	assert.NotNil(t, template)

	// Empty JSON fields should result in nil values
	assert.Nil(t, template.Schema)
	assert.Nil(t, template.Parameters)
	assert.Empty(t, template.Libraries)
}

func TestTemplateEntity_FromEntity_InvalidJSON(t *testing.T) {
	entity := &TemplateEntity{
		ID:              "test",
		Name:            "Test Template",
		Version:         "1.0.0",
		Category:        "test",
		Description:     "Test description",
		BoardsSupported: []string{},
		SchemaJSON:      `{"invalid": json}`, // Invalid JSON syntax
		ParametersJSON:  "",
		LibrariesJSON:   "",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	template, err := entity.FromEntity()

	assert.Error(t, err)
	assert.Nil(t, template)
	assert.Contains(t, err.Error(), "invalid character")
}

func TestAsset_ToAssetEntity(t *testing.T) {
	asset := &Asset{
		Type: "wiring_diagram",
		Path: "/diagrams/sensor.png",
		Metadata: map[string]interface{}{
			"width":  800,
			"height": 600,
			"format": "PNG",
		},
	}

	templateID := "test-template-1"
	templateVersion := "1.0.0"

	entity, err := asset.ToAssetEntity(templateID, templateVersion)

	assert.NoError(t, err)
	assert.NotNil(t, entity)

	assert.Equal(t, templateID, entity.TemplateID)
	assert.Equal(t, templateVersion, entity.TemplateVersion)
	assert.Equal(t, asset.Type, entity.AssetType)
	assert.Equal(t, asset.Path, entity.AssetPath)

	// Verify metadata JSON
	var metadata map[string]interface{}
	err = json.Unmarshal([]byte(entity.MetadataJSON), &metadata)
	assert.NoError(t, err)
	assert.Equal(t, asset.Metadata, metadata)
}

func TestAsset_ToAssetEntity_EmptyMetadata(t *testing.T) {
	asset := &Asset{
		Type:     "documentation",
		Path:     "/docs/README.md",
		Metadata: nil,
	}

	templateID := "test-template-1"
	templateVersion := "1.0.0"

	entity, err := asset.ToAssetEntity(templateID, templateVersion)

	assert.NoError(t, err)
	assert.NotNil(t, entity)

	// Nil metadata should result in empty JSON object
	assert.Equal(t, "{}", entity.MetadataJSON)
}

func TestAsset_ToAssetEntity_InvalidMetadata(t *testing.T) {
	asset := &Asset{
		Type: "image",
		Path: "/images/photo.jpg",
		Metadata: map[string]interface{}{
			"invalid": make(chan int), // Cannot be marshaled to JSON
		},
	}

	templateID := "test-template-1"
	templateVersion := "1.0.0"

	entity, err := asset.ToAssetEntity(templateID, templateVersion)

	assert.Error(t, err)
	assert.Nil(t, entity)
	assert.Contains(t, err.Error(), "json: unsupported type")
}

func TestTemplateAssetEntity_FromAssetEntity(t *testing.T) {
	entity := &TemplateAssetEntity{
		TemplateID:      "test-template-1",
		TemplateVersion: "1.0.0",
		AssetType:       "wiring_diagram",
		AssetPath:       "/diagrams/sensor.png",
		MetadataJSON:    `{"width":800,"height":600,"format":"PNG"}`,
		CreatedAt:       time.Now(),
	}

	asset, err := entity.FromAssetEntity()

	assert.NoError(t, err)
	assert.NotNil(t, asset)

	assert.Equal(t, entity.AssetType, asset.Type)
	assert.Equal(t, entity.AssetPath, asset.Path)

	// Verify metadata
	assert.Equal(t, float64(800), asset.Metadata["width"])
	assert.Equal(t, float64(600), asset.Metadata["height"])
	assert.Equal(t, "PNG", asset.Metadata["format"])
}

func TestTemplateAssetEntity_FromAssetEntity_EmptyMetadata(t *testing.T) {
	entity := &TemplateAssetEntity{
		TemplateID:      "test-template-1",
		TemplateVersion: "1.0.0",
		AssetType:       "documentation",
		AssetPath:       "/docs/README.md",
		MetadataJSON:    "",
		CreatedAt:       time.Now(),
	}

	asset, err := entity.FromAssetEntity()

	assert.NoError(t, err)
	assert.NotNil(t, asset)

	// Empty metadata JSON should result in nil metadata
	assert.Nil(t, asset.Metadata)
}

func TestTemplateAssetEntity_FromAssetEntity_InvalidMetadata(t *testing.T) {
	entity := &TemplateAssetEntity{
		TemplateID:      "test-template-1",
		TemplateVersion: "1.0.0",
		AssetType:       "image",
		AssetPath:       "/images/photo.jpg",
		MetadataJSON:    `{"invalid": json}`, // Invalid JSON
		CreatedAt:       time.Now(),
	}

	asset, err := entity.FromAssetEntity()

	assert.Error(t, err)
	assert.Nil(t, asset)
	assert.Contains(t, err.Error(), "invalid character")
}

func TestTemplate_RoundTripConversion(t *testing.T) {
	// Test that template -> entity -> template preserves data
	originalTemplate := createTestTemplate()

	entity, err := originalTemplate.ToEntity()
	require.NoError(t, err)

	convertedTemplate, err := entity.FromEntity()
	require.NoError(t, err)

	// Most fields should match
	assert.Equal(t, originalTemplate.ID, convertedTemplate.ID)
	assert.Equal(t, originalTemplate.Name, convertedTemplate.Name)
	assert.Equal(t, originalTemplate.Version, convertedTemplate.Version)
	assert.Equal(t, originalTemplate.Category, convertedTemplate.Category)
	assert.Equal(t, originalTemplate.Description, convertedTemplate.Description)
	assert.Equal(t, originalTemplate.BoardsSupported, convertedTemplate.BoardsSupported)
	assert.Equal(t, originalTemplate.CreatedAt.Unix(), convertedTemplate.CreatedAt.Unix())
	assert.Equal(t, originalTemplate.UpdatedAt.Unix(), convertedTemplate.UpdatedAt.Unix())

	// JSON fields should match (note: JSON unmarshaling converts numbers to float64)
	assert.Equal(t, originalTemplate.Schema["type"], convertedTemplate.Schema["type"])
	assert.Equal(t, float64(originalTemplate.Parameters["sensorPin"].(int)), convertedTemplate.Parameters["sensorPin"])
	assert.Equal(t, float64(originalTemplate.Parameters["interval"].(int)), convertedTemplate.Parameters["interval"])
	assert.Equal(t, originalTemplate.Libraries, convertedTemplate.Libraries)

	// Assets should be empty in converted template (loaded separately)
	assert.Empty(t, convertedTemplate.Assets)
}

func TestAsset_RoundTripConversion(t *testing.T) {
	// Test that asset -> entity -> asset preserves data
	originalAsset := &Asset{
		Type: "wiring_diagram",
		Path: "/diagrams/sensor.png",
		Metadata: map[string]interface{}{
			"width":       800,
			"height":      600,
			"format":      "PNG",
			"description": "Temperature sensor wiring diagram",
		},
	}

	templateID := "test-template-1"
	templateVersion := "1.0.0"

	entity, err := originalAsset.ToAssetEntity(templateID, templateVersion)
	require.NoError(t, err)

	convertedAsset, err := entity.FromAssetEntity()
	require.NoError(t, err)

	// Asset fields should match
	assert.Equal(t, originalAsset.Type, convertedAsset.Type)
	assert.Equal(t, originalAsset.Path, convertedAsset.Path)

	// Metadata should match (note: JSON unmarshaling converts numbers to float64)
	assert.Equal(t, originalAsset.Metadata["format"], convertedAsset.Metadata["format"])
	assert.Equal(t, originalAsset.Metadata["description"], convertedAsset.Metadata["description"])
	assert.Equal(t, float64(originalAsset.Metadata["width"].(int)), convertedAsset.Metadata["width"])
	assert.Equal(t, float64(originalAsset.Metadata["height"].(int)), convertedAsset.Metadata["height"])
}

func TestTemplateFilters_Validation(t *testing.T) {
	t.Run("Valid filters", func(t *testing.T) {
		filters := &TemplateFilters{
			Category:        "sensing",
			BoardType:       "arduino-uno",
			SupportedBoards: []string{"arduino-uno", "arduino-nano"},
			Limit:           10,
			Offset:          0,
		}

		// This should not cause any issues when used
		assert.NotNil(t, filters)
		assert.Equal(t, "sensing", filters.Category)
		assert.Equal(t, "arduino-uno", filters.BoardType)
		assert.Len(t, filters.SupportedBoards, 2)
		assert.Equal(t, 10, filters.Limit)
		assert.Equal(t, 0, filters.Offset)
	})

	t.Run("Empty filters", func(t *testing.T) {
		filters := &TemplateFilters{}

		assert.NotNil(t, filters)
		assert.Empty(t, filters.Category)
		assert.Empty(t, filters.BoardType)
		assert.Empty(t, filters.SupportedBoards)
		assert.Equal(t, 0, filters.Limit)
		assert.Equal(t, 0, filters.Offset)
	})

	t.Run("Nil filters", func(t *testing.T) {
		var filters *TemplateFilters = nil

		assert.Nil(t, filters)
	})
}

func TestValidationResult_Creation(t *testing.T) {
	t.Run("Valid result", func(t *testing.T) {
		result := &ValidationResult{
			Valid:    true,
			Errors:   []string{},
			Warnings: []string{},
		}

		assert.True(t, result.Valid)
		assert.Empty(t, result.Errors)
		assert.Empty(t, result.Warnings)
	})

	t.Run("Invalid result with errors", func(t *testing.T) {
		result := &ValidationResult{
			Valid:  false,
			Errors: []string{"Missing required field: ID", "Invalid schema format"},
			Warnings: []string{
				"Library version not specified",
				"Consider adding more documentation",
			},
		}

		assert.False(t, result.Valid)
		assert.Len(t, result.Errors, 2)
		assert.Len(t, result.Warnings, 2)
		assert.Contains(t, result.Errors, "Missing required field: ID")
		assert.Contains(t, result.Warnings, "Library version not specified")
	})
}

func TestRenderedTemplate_Creation(t *testing.T) {
	template := createTestTemplate()
	parameters := map[string]interface{}{"sensorPin": 2}
	renderedCode := "// Rendered Arduino code"

	renderedTemplate := &RenderedTemplate{
		Template:     template,
		Parameters:   parameters,
		RenderedCode: renderedCode,
		Assets:       template.Assets,
	}

	assert.Equal(t, template, renderedTemplate.Template)
	assert.Equal(t, parameters, renderedTemplate.Parameters)
	assert.Equal(t, renderedCode, renderedTemplate.RenderedCode)
	assert.Equal(t, template.Assets, renderedTemplate.Assets)
}

func TestWiringDiagram_Creation(t *testing.T) {
	diagram := &WiringDiagram{
		MermaidSyntax: "graph TD\nA[Arduino] --> B[Sensor]",
		Components: []Component{
			{
				ID:   "arduino",
				Type: "board",
				Name: "Arduino Uno",
				Pins: []Pin{
					{Number: "2", Name: "D2", Type: "digital"},
				},
			},
		},
		Connections: []Connection{
			{
				FromComponent: "arduino",
				FromPin:       "2",
				ToComponent:   "sensor",
				ToPin:         "data",
				WireColor:     "red",
			},
		},
		Metadata: map[string]interface{}{
			"created_by": "ATHENA",
			"version":    "1.0",
		},
	}

	assert.NotEmpty(t, diagram.MermaidSyntax)
	assert.Len(t, diagram.Components, 1)
	assert.Len(t, diagram.Connections, 1)
	assert.Equal(t, "ATHENA", diagram.Metadata["created_by"])

	component := diagram.Components[0]
	assert.Equal(t, "arduino", component.ID)
	assert.Equal(t, "board", component.Type)
	assert.Len(t, component.Pins, 1)

	pin := component.Pins[0]
	assert.Equal(t, "2", pin.Number)
	assert.Equal(t, "D2", pin.Name)
	assert.Equal(t, "digital", pin.Type)

	connection := diagram.Connections[0]
	assert.Equal(t, "arduino", connection.FromComponent)
	assert.Equal(t, "sensor", connection.ToComponent)
	assert.Equal(t, "red", connection.WireColor)
}
