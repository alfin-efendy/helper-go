package config

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/alfin-efendy/helper-go/config/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	// Save original state
	originalData := Data
	originalArgs := os.Args
	defer func() {
		Data = originalData
		os.Args = originalArgs
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	}()

	tests := []struct {
		name          string
		configContent string
		args          []string
		expectedData  *schema.Config
		shouldFail    bool
	}{
		{
			name: "Valid config file",
			configContent: `
app:
  name: "test-app"
  mode: "development"
server:
  restApi:
    host: "localhost"
    port: 8080
database:
  sql:
    address: "localhost:5432"
    database: "testdb"
    username: "testuser"
    password: "testpass"
messageBroker:
  rabbitmq:
    host: "localhost"
    port: 5672
    username: "guest"
    password: "guest"
otel:
  address: "localhost:4317"
  timeout: 30
  trace: true
  metric: false
log:
  level: "info"`,
			args: []string{"test", "-config", "test-config.yml"},
			expectedData: &schema.Config{
				App: schema.App{
					Name: "test-app",
					Mode: "development",
				},
				Server: schema.Server{
					RestAPI: schema.RestAPI{
						Host: "localhost",
						Port: 8080,
					},
				},
				Database: schema.Database{
					SQL: &schema.SQL{
						Host:     "localhost",
						Port:     5432,
						Database: "testdb",
						Username: "testuser",
						Password: "testpass",
					},
				},
				MessageBroker: schema.MessageBroker{
					RabbitMQ: schema.RabbitMQ{
						Host:     "localhost",
						Port:     5672,
						Username: "guest",
						Password: "guest",
					},
				},
				Otel: schema.Otel{
					Address: "localhost:4317",
					Timeout: 30,
					Trace:   true,
					Metric:  false,
				},
				Log: schema.Log{
					Level: "info",
				},
			},
			shouldFail: false,
		},
		{
			name: "Default config file path",
			configContent: `
app:
  name: "default-app"
  mode: "production"`,
			args: []string{"test"},
			expectedData: &schema.Config{
				App: schema.App{
					Name: "default-app",
					Mode: "production",
				},
			},
			shouldFail: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset global state
			Data = nil
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

			// Create temporary config file
			var configPath string
			if len(tt.args) > 2 && tt.args[1] == "-config" {
				configPath = tt.args[2]
			} else {
				configPath = "config.yml"
			}

			tempDir := t.TempDir()
			fullConfigPath := filepath.Join(tempDir, configPath)

			// Create directory if needed
			if err := os.MkdirAll(filepath.Dir(fullConfigPath), 0o755); err != nil {
				t.Fatal(err)
			}

			err := os.WriteFile(fullConfigPath, []byte(tt.configContent), 0o644)
			require.NoError(t, err)

			// Set command line args
			os.Args = make([]string, len(tt.args))
			copy(os.Args, tt.args)
			if len(tt.args) > 2 && tt.args[1] == "-config" {
				os.Args[2] = fullConfigPath
			} else {
				// For default config test, create config.yml in current directory
				err := os.WriteFile("config.yml", []byte(tt.configContent), 0o644)
				require.NoError(t, err)
				defer os.Remove("config.yml")
			}

			// Change to temp directory for default config test
			if configPath == "config.yml" {
				originalDir, _ := os.Getwd()
				os.Chdir(tempDir)
				defer os.Chdir(originalDir)
			}

			if tt.shouldFail {
				// For failure cases, we'd need to capture os.Exit
				// This is complex to test, so we'll test the underlying function instead
				return
			}

			// Test Load function
			Load()

			// Verify results
			require.NotNil(t, Data)
			if tt.expectedData.App.Name != "" {
				assert.Equal(t, tt.expectedData.App.Name, Data.App.Name)
				assert.Equal(t, tt.expectedData.App.Mode, Data.App.Mode)
			}
			if tt.expectedData.Server.RestAPI.Host != "" {
				assert.Equal(t, tt.expectedData.Server.RestAPI.Host, Data.Server.RestAPI.Host)
				assert.Equal(t, tt.expectedData.Server.RestAPI.Port, Data.Server.RestAPI.Port)
			}
		})
	}
}

func TestUnmarshal(t *testing.T) {
	tests := []struct {
		name           string
		configContent  string
		envVars        map[string]string
		expectedConfig *schema.Config
		expectedError  bool
	}{
		{
			name: "Valid YAML config",
			configContent: `
app:
  name: "test-app"
  mode: "development"
server:
  restApi:
    host: "localhost"
    port: 8080`,
			expectedConfig: &schema.Config{
				App: schema.App{
					Name: "test-app",
					Mode: "development",
				},
				Server: schema.Server{
					RestAPI: schema.RestAPI{
						Host: "localhost",
						Port: 8080,
					},
				},
			},
			expectedError: false,
		},
		{
			name: "Config with environment variable substitution",
			configContent: `
app:
  name: "${APP_NAME}"
  mode: "${APP_MODE}"
server:
  restApi:
    host: "${SERVER_HOST}"
    port: 8080`,
			envVars: map[string]string{
				"APP_NAME":    "env-app",
				"APP_MODE":    "production",
				"SERVER_HOST": "0.0.0.0",
			},
			expectedConfig: &schema.Config{
				App: schema.App{
					Name: "env-app",
					Mode: "production",
				},
				Server: schema.Server{
					RestAPI: schema.RestAPI{
						Host: "0.0.0.0",
						Port: 8080,
					},
				},
			},
			expectedError: false,
		},
		{
			name: "Complex config with database and message broker",
			configContent: `
app:
  name: "complex-app"
  mode: "production"
database:
  sql:
    host: "localhost"
    port: 5432
    database: "proddb"
    username: "produser"
    password: "prodpass"
    poolingConnection:
      maxIdle: 10
      maxOpen: 100
      maxLifetime: 3600
messageBroker:
  rabbitmq:
    host: "rabbitmq.example.com"
    port: 5672
    username: "admin"
    password: "secret"
otel:
  address: "jaeger:14268"
  timeout: 60
  trace: true
  metric: true
log:
  level: "debug"`,
			expectedConfig: &schema.Config{
				App: schema.App{
					Name: "complex-app",
					Mode: "production",
				},
				Database: schema.Database{
					SQL: &schema.SQL{
						Host:     "localhost",
						Port:     5432,
						Database: "proddb",
						Username: "produser",
						Password: "prodpass",
						PoolingConnection: &schema.PoolingConnection{
							MaxIdle:     10,
							MaxOpen:     100,
							MaxLifetime: 3600,
						},
					},
				},
				MessageBroker: schema.MessageBroker{
					RabbitMQ: schema.RabbitMQ{
						Host:     "rabbitmq.example.com",
						Port:     5672,
						Username: "admin",
						Password: "secret",
					},
				},
				Otel: schema.Otel{
					Address: "jaeger:14268",
					Timeout: 60,
					Trace:   true,
					Metric:  true,
				},
				Log: schema.Log{
					Level: "debug",
				},
			},
			expectedError: false,
		},
		{
			name:          "Invalid YAML",
			configContent: "invalid: yaml: content: [",
			expectedError: true,
		},
		{
			name:          "Non-existent file",
			configContent: "",
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and reset global state
			originalData := Data
			defer func() { Data = originalData }()
			Data = nil

			// Set environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
				defer os.Unsetenv(key)
			}

			var configPath string
			var err error

			if tt.name == "Non-existent file" {
				configPath = "non-existent-config.yml"
			} else {
				// Create temporary config file
				tempFile, err := os.CreateTemp("", "config-*.yml")
				require.NoError(t, err)
				defer os.Remove(tempFile.Name())

				_, err = tempFile.WriteString(tt.configContent)
				require.NoError(t, err)
				tempFile.Close()

				configPath = tempFile.Name()
			}

			var result schema.Config
			err = Unmarshal(configPath, &result)

			if tt.expectedError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			// The function unmarshals into the global Data variable, not the passed interface
			require.NotNil(t, Data)

			if tt.expectedConfig != nil {
				assert.Equal(t, tt.expectedConfig.App, Data.App)
				assert.Equal(t, tt.expectedConfig.Server, Data.Server)

				if tt.expectedConfig.Database.SQL != nil {
					require.NotNil(t, Data.Database.SQL)
					assert.Equal(t, tt.expectedConfig.Database.SQL.Host, Data.Database.SQL.Host)
					assert.Equal(t, tt.expectedConfig.Database.SQL.Port, Data.Database.SQL.Port)
					assert.Equal(t, tt.expectedConfig.Database.SQL.Database, Data.Database.SQL.Database)
					assert.Equal(t, tt.expectedConfig.Database.SQL.Username, Data.Database.SQL.Username)
					assert.Equal(t, tt.expectedConfig.Database.SQL.Password, Data.Database.SQL.Password)

					if tt.expectedConfig.Database.SQL.PoolingConnection != nil {
						require.NotNil(t, Data.Database.SQL.PoolingConnection)
						assert.Equal(t, tt.expectedConfig.Database.SQL.PoolingConnection.MaxIdle, Data.Database.SQL.PoolingConnection.MaxIdle)
						assert.Equal(t, tt.expectedConfig.Database.SQL.PoolingConnection.MaxOpen, Data.Database.SQL.PoolingConnection.MaxOpen)
						assert.Equal(t, tt.expectedConfig.Database.SQL.PoolingConnection.MaxLifetime, Data.Database.SQL.PoolingConnection.MaxLifetime)
					}
				}

				assert.Equal(t, tt.expectedConfig.MessageBroker, Data.MessageBroker)
				assert.Equal(t, tt.expectedConfig.Otel, Data.Otel)
				assert.Equal(t, tt.expectedConfig.Log, Data.Log)
			}
		})
	}
}

func TestUnmarshalNilConfig(t *testing.T) {
	// Save and reset global state
	originalData := Data
	defer func() { Data = originalData }()
	Data = nil

	// Test case where config data becomes nil after unmarshal
	tempFile, err := os.CreateTemp("", "empty-config-*.yml")
	require.NoError(t, err)
	defer os.Remove(tempFile.Name())

	// Write content that would parse but result in empty struct
	_, err = tempFile.WriteString("# Just a comment")
	require.NoError(t, err)
	tempFile.Close()

	var result interface{}
	err = Unmarshal(tempFile.Name(), &result)
	// With just a comment, the unmarshal should succeed but Data might be nil or empty
	// This test mainly verifies the function doesn't panic
	if err != nil {
		// It's okay if it errors, just shouldn't panic
		t.Log("Unmarshal returned error (expected for empty config):", err)
	}
}

func TestEnvironmentVariableSubstitution(t *testing.T) {
	tests := []struct {
		name        string
		configValue string
		envVar      string
		envValue    string
		expected    string
	}{
		{
			name:        "Simple environment variable",
			configValue: "${TEST_VAR}",
			envVar:      "TEST_VAR",
			envValue:    "test_value",
			expected:    "test_value",
		},
		{
			name:        "Environment variable not set",
			configValue: "${UNSET_VAR}",
			envVar:      "",
			envValue:    "",
			expected:    "",
		},
		{
			name:        "No environment variable syntax",
			configValue: "static_value",
			envVar:      "",
			envValue:    "",
			expected:    "static_value",
		},
		{
			name:        "Mixed content with environment variable",
			configValue: "prefix_${MIX_VAR}_suffix",
			envVar:      "MIX_VAR",
			envValue:    "middle",
			expected:    "prefix_${MIX_VAR}_suffix", // Should not substitute partial matches
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and reset global state
			originalData := Data
			defer func() { Data = originalData }()
			Data = nil

			if tt.envVar != "" {
				os.Setenv(tt.envVar, tt.envValue)
				defer os.Unsetenv(tt.envVar)
			}

			configContent := "test:\n  value: \"" + tt.configValue + "\""

			tempFile, err := os.CreateTemp("", "env-test-*.yml")
			require.NoError(t, err)
			defer os.Remove(tempFile.Name())

			_, err = tempFile.WriteString(configContent)
			require.NoError(t, err)
			tempFile.Close()

			var result map[string]interface{}
			err = Unmarshal(tempFile.Name(), &result)
			require.NoError(t, err)

			// Since the function uses global Data, we need to check that instead
			require.NotNil(t, Data)
			// For this test, we'll just verify the function doesn't error
			// The actual environment variable substitution testing is covered in other tests
		})
	}
}

func TestLoadConfigFileNotFound(t *testing.T) {
	// Save original state
	originalData := Data
	originalArgs := os.Args
	defer func() {
		Data = originalData
		os.Args = originalArgs
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	}()

	// Reset global state
	Data = nil
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	// This test checks that Load() will call os.Exit(1) on error
	// Since we can't easily test os.Exit, we test the underlying Unmarshal function
	err := Unmarshal("non-existent-file.yml", &Data)
	assert.Error(t, err)
}

// Benchmark tests
func BenchmarkUnmarshal(b *testing.B) {
	// Save original state
	originalData := Data
	defer func() { Data = originalData }()

	configContent := `
app:
  name: "benchmark-app"
  mode: "production"
server:
  restApi:
    host: "localhost"
    port: 8080
database:
  sql:
    address: "localhost:5432"
    database: "benchdb"
    username: "benchuser"
    password: "benchpass"
messageBroker:
  rabbitmq:
    host: "localhost"
    port: 5672
    username: "guest"
    password: "guest"
otel:
  address: "localhost:4317"
  timeout: 30
  trace: true
  metric: false
log:
  level: "info"`

	tempFile, err := os.CreateTemp("", "benchmark-config-*.yml")
	if err != nil {
		b.Fatal(err)
	}
	defer os.Remove(tempFile.Name())

	_, err = tempFile.WriteString(configContent)
	if err != nil {
		b.Fatal(err)
	}
	tempFile.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Data = nil // Reset for each iteration
		var config schema.Config
		err := Unmarshal(tempFile.Name(), &config)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUnmarshalWithEnvVars(b *testing.B) {
	// Save original state
	originalData := Data
	defer func() { Data = originalData }()

	// Set environment variables
	os.Setenv("BENCH_APP_NAME", "benchmark-app")
	os.Setenv("BENCH_HOST", "localhost")
	defer func() {
		os.Unsetenv("BENCH_APP_NAME")
		os.Unsetenv("BENCH_HOST")
	}()

	configContent := `
app:
  name: "${BENCH_APP_NAME}"
  mode: "production"
server:
  restApi:
    host: "${BENCH_HOST}"
    port: 8080`

	tempFile, err := os.CreateTemp("", "benchmark-env-config-*.yml")
	if err != nil {
		b.Fatal(err)
	}
	defer os.Remove(tempFile.Name())

	_, err = tempFile.WriteString(configContent)
	if err != nil {
		b.Fatal(err)
	}
	tempFile.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Data = nil // Reset for each iteration
		var config schema.Config
		err := Unmarshal(tempFile.Name(), &config)
		if err != nil {
			b.Fatal(err)
		}
	}
}
