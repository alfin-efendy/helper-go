package config

import (
	"os"
	"testing"

	"github.com/alfin-efendy/helper-go/config/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerConfiguration(t *testing.T) {
	// Save and reset global state
	originalData := Data
	defer func() { Data = originalData }()
	Data = nil

	configContent := `
server:
  restApi:
    host: "0.0.0.0"
    port: 9090`

	tempFile, err := os.CreateTemp("", "server-config-*.yml")
	require.NoError(t, err)
	defer os.Remove(tempFile.Name())

	_, err = tempFile.WriteString(configContent)
	require.NoError(t, err)
	tempFile.Close()

	var result schema.Config
	err = Unmarshal(tempFile.Name(), &result)
	require.NoError(t, err)
	require.NotNil(t, Data)

	assert.Equal(t, "0.0.0.0", Data.Server.RestAPI.Host)
	assert.Equal(t, 9090, Data.Server.RestAPI.Port)
}

func TestMinimalValidConfiguration(t *testing.T) {
	// Save and reset global state
	originalData := Data
	defer func() { Data = originalData }()
	Data = nil

	// Test with just app configuration (minimum viable config)
	configContent := `
app:
  name: "minimal-app"
  mode: "test"`

	tempFile, err := os.CreateTemp("", "minimal-config-*.yml")
	require.NoError(t, err)
	defer os.Remove(tempFile.Name())

	_, err = tempFile.WriteString(configContent)
	require.NoError(t, err)
	tempFile.Close()

	var result schema.Config
	err = Unmarshal(tempFile.Name(), &result)
	require.NoError(t, err)
	require.NotNil(t, Data)

	assert.Equal(t, "minimal-app", Data.App.Name)
	assert.Equal(t, "test", Data.App.Mode)

	// Verify other sections are initialized to zero values
	assert.Empty(t, Data.Server.RestAPI.Host)
	assert.Equal(t, 0, Data.Server.RestAPI.Port)
	assert.Nil(t, Data.Database.SQL)
	assert.Nil(t, Data.Database.Redis)
}

func TestOtelConfiguration(t *testing.T) {
	// Save and reset global state
	originalData := Data
	defer func() { Data = originalData }()
	Data = nil

	configContent := `
otel:
  address: "jaeger:14268"
  timeout: 60
  trace: true
  metric: false`

	tempFile, err := os.CreateTemp("", "otel-config-*.yml")
	require.NoError(t, err)
	defer os.Remove(tempFile.Name())

	_, err = tempFile.WriteString(configContent)
	require.NoError(t, err)
	tempFile.Close()

	var result schema.Config
	err = Unmarshal(tempFile.Name(), &result)
	require.NoError(t, err)
	require.NotNil(t, Data)

	assert.Equal(t, "jaeger:14268", Data.Otel.Address)
	assert.Equal(t, 60, Data.Otel.Timeout)
	assert.True(t, Data.Otel.Trace)
	assert.False(t, Data.Otel.Metric)
}

func TestLogConfiguration(t *testing.T) {
	// Save and reset global state
	originalData := Data
	defer func() { Data = originalData }()
	Data = nil

	configContent := `
log:
  level: "debug"`

	tempFile, err := os.CreateTemp("", "log-config-*.yml")
	require.NoError(t, err)
	defer os.Remove(tempFile.Name())

	_, err = tempFile.WriteString(configContent)
	require.NoError(t, err)
	tempFile.Close()

	var result schema.Config
	err = Unmarshal(tempFile.Name(), &result)
	require.NoError(t, err)
	require.NotNil(t, Data)

	assert.Equal(t, "debug", Data.Log.Level)
}

func TestMessageBrokerConfiguration(t *testing.T) {
	// Save and reset global state
	originalData := Data
	defer func() { Data = originalData }()
	Data = nil

	configContent := `
messageBroker:
  rabbitmq:
    host: "localhost"
    port: 5672
    username: "guest"
    password: "guest"`

	tempFile, err := os.CreateTemp("", "messagebroker-config-*.yml")
	require.NoError(t, err)
	defer os.Remove(tempFile.Name())

	_, err = tempFile.WriteString(configContent)
	require.NoError(t, err)
	tempFile.Close()

	var result schema.Config
	err = Unmarshal(tempFile.Name(), &result)
	require.NoError(t, err)
	require.NotNil(t, Data)

	assert.Equal(t, "localhost", Data.MessageBroker.RabbitMQ.Host)
	assert.Equal(t, 5672, Data.MessageBroker.RabbitMQ.Port)
	assert.Equal(t, "guest", Data.MessageBroker.RabbitMQ.Username)
	assert.Equal(t, "guest", Data.MessageBroker.RabbitMQ.Password)
}

func TestDatabaseSqlConfiguration(t *testing.T) {
	// Save and reset global state
	originalData := Data
	defer func() { Data = originalData }()
	Data = nil

	configContent := `
database:
  sql:
    host: "localhost"
    port: 5432
    database: "testdb"
    username: "testuser"
    password: "testpass"
    poolingConnection:
      maxIdle: 5
      maxOpen: 25
      maxLifetime: 1800`

	tempFile, err := os.CreateTemp("", "sql-config-*.yml")
	require.NoError(t, err)
	defer os.Remove(tempFile.Name())

	_, err = tempFile.WriteString(configContent)
	require.NoError(t, err)
	tempFile.Close()

	var result schema.Config
	err = Unmarshal(tempFile.Name(), &result)
	require.NoError(t, err)
	require.NotNil(t, Data)
	require.NotNil(t, Data.Database.SQL)

	assert.Equal(t, "localhost", Data.Database.SQL.Host)
	assert.Equal(t, 5432, Data.Database.SQL.Port)
	assert.Equal(t, "testdb", Data.Database.SQL.Database)
	assert.Equal(t, "testuser", Data.Database.SQL.Username)
	assert.Equal(t, "testpass", Data.Database.SQL.Password)

	require.NotNil(t, Data.Database.SQL.PoolingConnection)
	assert.Equal(t, 5, Data.Database.SQL.PoolingConnection.MaxIdle)
	assert.Equal(t, 25, Data.Database.SQL.PoolingConnection.MaxOpen)
	assert.Equal(t, int64(1800), Data.Database.SQL.PoolingConnection.MaxLifetime)
}
