package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetValue(t *testing.T) {
	// Save original state
	originalData := Data
	originalRaw := raw
	defer func() {
		Data = originalData
		raw = originalRaw
	}()
	Data = nil
	raw = nil

	// Create a test config file with extra fields not in the struct
	configContent := `
app:
  name: "test-app"
  mode: "development"
  # Extra field not in struct
  version: "1.0.0"
  buildNumber: 123
  features:
    - "feature1"
    - "feature2"
    - "feature3"

server:
  restApi:
    host: "localhost"
    port: 8080
  # Extra field not in struct  
  grpc:
    host: "localhost"
    port: 9090
    
# Completely custom section not in struct
custom:
  setting1: "value1"
  setting2: 42
  setting3: true
  nested:
    key: "nested-value"
    
database:
  sql:
    host: "db-host"
    port: 5432
  # Extra field
  connectionPool:
    size: 10
    timeout: 30
`

	tempFile, err := os.CreateTemp("", "test-getvalue-config-*.yml")
	require.NoError(t, err)
	defer os.Remove(tempFile.Name())

	_, err = tempFile.WriteString(configContent)
	require.NoError(t, err)
	tempFile.Close()

	// Load the configuration
	err = unmarshal(tempFile.Name())
	require.NoError(t, err)
	require.NotNil(t, Data)
	require.NotNil(t, raw)

	// Test GetValue function for values in struct
	t.Run("Structured values", func(t *testing.T) {
		assert.Equal(t, "test-app", GetValue("app.name"))
		assert.Equal(t, "development", GetValue("app.mode"))
		assert.Equal(t, "localhost", GetValue("server.restapi.host")) // Note: lowercase 'restapi'
		assert.Equal(t, "8080", GetValue("server.restapi.port"))      // Should be converted to string
	})

	// Test GetValue function for values NOT in struct but in config file
	t.Run("Raw config values", func(t *testing.T) {
		assert.Equal(t, "1.0.0", GetValue("app.version"))
		assert.Equal(t, "123", GetValue("app.buildnumber")) // Note: lowercase
		assert.Equal(t, "value1", GetValue("custom.setting1"))
		assert.Equal(t, "42", GetValue("custom.setting2"))   // Should be converted to string
		assert.Equal(t, "true", GetValue("custom.setting3")) // Should be converted to string
		assert.Equal(t, "nested-value", GetValue("custom.nested.key"))
		assert.Equal(t, "localhost", GetValue("server.grpc.host"))
		assert.Equal(t, "9090", GetValue("server.grpc.port"))           // Should be converted to string
		assert.Equal(t, "10", GetValue("database.connectionpool.size")) // Note: lowercase
	})

	// Test non-existent keys
	t.Run("Non-existent keys", func(t *testing.T) {
		assert.Equal(t, "", GetValue("nonexistent.key"))
		assert.Equal(t, "", GetValue("app.nonexistent"))
	})
}

func TestGetValueWithNilRaw(t *testing.T) {
	// Save original state
	originalData := Data
	originalRaw := raw
	defer func() {
		Data = originalData
		raw = originalRaw
	}()
	Data = nil
	raw = nil

	// Test with nil raw config
	assert.Equal(t, "", GetValue("app.name"))
	assert.Equal(t, "", GetValue("any.key"))
}

func TestGetValueWithEmptyKey(t *testing.T) {
	// Save original state
	originalData := Data
	originalRaw := raw
	defer func() {
		Data = originalData
		raw = originalRaw
	}()

	// Set up some dummy raw data
	raw = map[string]interface{}{
		"test": "value",
	}

	// Test with empty key
	assert.Equal(t, "", GetValue(""))
}

func TestGetVal(t *testing.T) {
	testConfig := map[string]interface{}{
		"simple": "value",
		"nested": map[string]interface{}{
			"key": "nested-value",
			"deep": map[string]interface{}{
				"key": "deep-value",
			},
		},
		"number":  42,
		"boolean": true,
	}

	tests := []struct {
		key      string
		expected interface{}
	}{
		{"simple", "value"},
		{"nested.key", "nested-value"},
		{"nested.deep.key", "deep-value"},
		{"number", 42},
		{"boolean", true},
		{"nonexistent", nil},
		{"nested.nonexistent", nil},
		{"", nil},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			result := getVal(tt.key, testConfig)
			assert.Equal(t, tt.expected, result)
		})
	}
}
