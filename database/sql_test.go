package database

import (
	"context"
	"testing"
	"time"

	"github.com/alfin-efendy/helper-go/config"
	"github.com/alfin-efendy/helper-go/config/schema"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// SqlTestSuite is the test suite for SQL database functionality
type SqlTestSuite struct {
	suite.Suite
	ctx context.Context
}

func (suite *SqlTestSuite) SetupTest() {
	suite.ctx = context.Background()

	// Reset global state
	sqlClientMutex.Lock()
	sqlClient = nil
	isConnected = false
	connectionStats = ConnectionStats{}
	sqlClientMutex.Unlock()
}

func (suite *SqlTestSuite) TearDownTest() {
	// Clean up any connections
	CloseSQL()
}

func (suite *SqlTestSuite) TestValidateSqlConfig() {
	tests := []struct {
		name    string
		config  *schema.SQL
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: &schema.SQL{
				Host:     "localhost",
				Port:     5432,
				Database: "testdb",
				Username: "testuser",
				Password: "testpass",
				PoolingConnection: &schema.PoolingConnection{
					MaxIdle:     10,
					MaxOpen:     100,
					MaxLifetime: 3600,
				},
			},
			wantErr: false,
		},
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
			errMsg:  "database config is nil",
		},
		{
			name: "missing host",
			config: &schema.SQL{
				Port:     5432,
				Database: "testdb",
				Username: "testuser",
				Password: "testpass",
			},
			wantErr: true,
			errMsg:  "database host is required",
		},
		{
			name: "invalid port - zero",
			config: &schema.SQL{
				Host:     "localhost",
				Port:     0,
				Database: "testdb",
				Username: "testuser",
				Password: "testpass",
			},
			wantErr: true,
			errMsg:  "database port must be between 1 and 65535",
		},
		{
			name: "invalid port - too high",
			config: &schema.SQL{
				Host:     "localhost",
				Port:     70000,
				Database: "testdb",
				Username: "testuser",
				Password: "testpass",
			},
			wantErr: true,
			errMsg:  "database port must be between 1 and 65535",
		},
		{
			name: "missing username",
			config: &schema.SQL{
				Host:     "localhost",
				Port:     5432,
				Database: "testdb",
				Password: "testpass",
			},
			wantErr: true,
			errMsg:  "database username is required",
		},
		{
			name: "missing database",
			config: &schema.SQL{
				Host:     "localhost",
				Port:     5432,
				Username: "testuser",
				Password: "testpass",
			},
			wantErr: true,
			errMsg:  "database name is required",
		},
		{
			name: "missing password",
			config: &schema.SQL{
				Host:     "localhost",
				Port:     5432,
				Database: "testdb",
				Username: "testuser",
			},
			wantErr: true,
			errMsg:  "database password is required",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			err := validateSQLConfig(tt.config)
			if tt.wantErr {
				suite.Error(err)
				if tt.errMsg != "" {
					suite.Contains(err.Error(), tt.errMsg)
				}
			} else {
				suite.NoError(err)
			}
		})
	}
}

func (suite *SqlTestSuite) TestGetSQLClient() {
	// Test when client is nil
	client := GetSQLClient()
	suite.Nil(client)

	// Test thread safety - this would catch race conditions if run with -race
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			GetSQLClient()
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func (suite *SqlTestSuite) TestIsSQLConnected() {
	// Initially should be false
	suite.False(IsSQLConnected())

	// Mock connected state
	sqlClientMutex.Lock()
	isConnected = true
	sqlClientMutex.Unlock()

	// Should still be false because sqlClient is nil
	suite.False(IsSQLConnected())

	// Set both flags for true result
	sqlClientMutex.Lock()
	isConnected = true
	// We can't create a real client here, but the logic checks for both
	sqlClientMutex.Unlock()

	// Reset
	sqlClientMutex.Lock()
	isConnected = false
	sqlClient = nil
	sqlClientMutex.Unlock()

	suite.False(IsSQLConnected())
}

func (suite *SqlTestSuite) TestGetSQLStats() {
	// Test with no connection - should return empty stats
	stats := GetSQLStats()
	suite.Zero(stats.OpenConnections)
	suite.Zero(stats.ConnectedAt)

	// When sqlClient is nil, GetSQLStats returns empty ConnectionStats{}
	// So let's test that the mock stats don't affect the return when client is nil
	now := time.Now()
	sqlClientMutex.Lock()
	connectionStats = ConnectionStats{
		OpenConnections: 5,
		InUse:           2,
		Idle:            3,
		MaxOpen:         100,
		MaxIdle:         10,
		ConnectedAt:     now,
		LastPingAt:      now,
		PingDuration:    50 * time.Millisecond,
	}
	sqlClientMutex.Unlock()

	// Since sqlClient is still nil, this should return empty stats
	stats = GetSQLStats()
	suite.Zero(stats.OpenConnections)
	suite.Zero(stats.InUse)
	suite.Zero(stats.Idle)
	suite.Zero(stats.MaxOpen)
	suite.Zero(stats.MaxIdle)
	suite.Zero(stats.ConnectedAt)
}

func (suite *SqlTestSuite) TestPingSQLDatabase() {
	// Test with no client
	err := PingSQLDatabase(suite.ctx)
	suite.Error(err)
	suite.Contains(err.Error(), "database client is not initialized")
}

func (suite *SqlTestSuite) TestCloseSql() {
	// Test closing when no connection
	err := CloseSQL()
	suite.NoError(err)

	// Mock a connection state
	sqlClientMutex.Lock()
	isConnected = true
	connectionStats = ConnectionStats{ConnectedAt: time.Now()}
	sqlClientMutex.Unlock()

	// Close should reset state
	err = CloseSQL()
	suite.NoError(err)
	suite.False(IsSQLConnected())

	stats := GetSQLStats()
	suite.Zero(stats.ConnectedAt)
}

func (suite *SqlTestSuite) TestInitSQLWithInvalidConfig() {
	// Test with nil config
	config.Data = &schema.Config{}
	err := InitSQL(suite.ctx)
	suite.Error(err)
	suite.Contains(err.Error(), "database configuration is not found")

	// Test with invalid config
	config.Data.Database.SQL = &schema.SQL{
		Host: "", // Invalid: empty host
		Port: 5432,
	}
	err = InitSQL(suite.ctx)
	suite.Error(err)
	suite.Contains(err.Error(), "database host is required")
}

func (suite *SqlTestSuite) TestInitSQLConfigDefaults() {
	// Test that missing pooling config gets defaults
	config.Data = &schema.Config{
		Database: schema.Database{
			SQL: &schema.SQL{
				Host:     "nonexistent-host-for-testing",
				Port:     5432,
				Database: "testdb",
				Username: "testuser",
				Password: "testpass",
				PoolingConnection: &schema.PoolingConnection{
					MaxIdle:     0, // Should get default
					MaxOpen:     0, // Should get default
					MaxLifetime: 0, // Should get default
				},
			},
		},
		Log: schema.Log{
			Level: "info",
		},
	}

	// This will fail to connect but should validate our defaults logic
	err := InitSQL(suite.ctx)
	suite.Error(err) // Expected to fail since host doesn't exist

	// The defaults should have been applied to the pooling config
	// We can't easily test this without a more complex setup, but the code path is covered
}

func (suite *SqlTestSuite) TestZerologLevelMapping() {
	// Test that our level mapping contains all expected levels
	expectedLevels := []string{"panic", "fatal", "error", "warn", "info", "debug"}

	for _, levelStr := range expectedLevels {
		level, err := zerolog.ParseLevel(levelStr)
		suite.NoError(err)

		gormLevel, exists := zerologLevelMap[level]
		suite.True(exists, "Level %s should exist in mapping", levelStr)
		suite.NotNil(gormLevel)
	}
}

func (suite *SqlTestSuite) TestConnectionStats() {
	stats := ConnectionStats{
		OpenConnections: 10,
		InUse:           5,
		Idle:            5,
		MaxOpen:         100,
		MaxIdle:         20,
		ConnectedAt:     time.Now(),
		LastPingAt:      time.Now(),
		PingDuration:    100 * time.Millisecond,
	}

	suite.Equal(10, stats.OpenConnections)
	suite.Equal(5, stats.InUse)
	suite.Equal(5, stats.Idle)
	suite.Equal(100, stats.MaxOpen)
	suite.Equal(20, stats.MaxIdle)
	suite.NotZero(stats.ConnectedAt)
	suite.NotZero(stats.LastPingAt)
	suite.Equal(100*time.Millisecond, stats.PingDuration)
}

// Test suite runner
func TestSqlTestSuite(t *testing.T) {
	suite.Run(t, new(SqlTestSuite))
}

// Benchmark tests
func BenchmarkGetSQLClient(b *testing.B) {
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			GetSQLClient()
		}
	})
}

func BenchmarkIsSQLConnected(b *testing.B) {
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			IsSQLConnected()
		}
	})
}

func BenchmarkGetSQLStats(b *testing.B) {
	// Setup some mock stats
	sqlClientMutex.Lock()
	connectionStats = ConnectionStats{
		OpenConnections: 10,
		InUse:           5,
		Idle:            5,
	}
	sqlClientMutex.Unlock()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			GetSQLStats()
		}
	})
}

// Individual function tests
func TestValidateSqlConfig(t *testing.T) {
	validConfig := &schema.SQL{
		Host:     "localhost",
		Port:     5432,
		Database: "testdb",
		Username: "testuser",
		Password: "testpass",
	}

	err := validateSQLConfig(validConfig)
	assert.NoError(t, err)

	invalidConfig := &schema.SQL{
		Host: "", // Invalid
	}

	err = validateSQLConfig(invalidConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database host is required")
}

func TestConnectionStatsStruct(t *testing.T) {
	stats := ConnectionStats{}
	assert.Zero(t, stats.OpenConnections)
	assert.Zero(t, stats.ConnectedAt)

	now := time.Now()
	stats.ConnectedAt = now
	stats.OpenConnections = 5

	assert.Equal(t, now, stats.ConnectedAt)
	assert.Equal(t, 5, stats.OpenConnections)
}
