package database

import (
	"context"
	"testing"
	"time"

	"github.com/alfin-efendy/helper-go/config/schema"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// RedisTestSuite provides a test suite for Redis functionality
type RedisTestSuite struct {
	suite.Suite
	ctx context.Context
}

func (suite *RedisTestSuite) SetupTest() {
	suite.ctx = context.Background()
	// Reset global state
	redisMutex.Lock()
	redisClient = nil
	redisMutex.Unlock()
}

func (suite *RedisTestSuite) TearDownTest() {
	CloseRedis()
}

func TestRedisTestSuite(t *testing.T) {
	suite.Run(t, new(RedisTestSuite))
}

func TestValidateRedisConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *schema.Redis
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
			errMsg:  "redis configuration is nil",
		},
		{
			name: "missing address",
			config: &schema.Redis{
				Mode: "single",
			},
			wantErr: true,
			errMsg:  "redis address is required",
		},
		{
			name: "invalid mode",
			config: &schema.Redis{
				RedisCluster: schema.RedisCluster{
					RedisSingle: schema.RedisSingle{
						Address: "localhost:6379",
					},
				},
				Mode: "cluster",
			},
			wantErr: true,
			errMsg:  "unsupported Redis mode",
		},
		{
			name: "sentinel mode missing master name",
			config: &schema.Redis{
				RedisCluster: schema.RedisCluster{
					RedisSingle: schema.RedisSingle{
						Address: "localhost:6379",
					},
				},
				Mode: "sentinel",
			},
			wantErr: true,
			errMsg:  "master name is required",
		},
		{
			name: "sentinel mode missing sentinel addresses",
			config: &schema.Redis{
				RedisCluster: schema.RedisCluster{
					RedisSingle: schema.RedisSingle{
						Address: "localhost:6379",
					},
					MasterName: "mymaster",
				},
				Mode: "sentinel",
			},
			wantErr: true,
			errMsg:  "sentinel addresses are required",
		},
		{
			name: "valid single mode config",
			config: &schema.Redis{
				RedisCluster: schema.RedisCluster{
					RedisSingle: schema.RedisSingle{
						Address: "localhost:6379",
					},
				},
				Mode: "single",
			},
			wantErr: false,
		},
		{
			name: "valid sentinel mode config",
			config: &schema.Redis{
				RedisCluster: schema.RedisCluster{
					RedisSingle: schema.RedisSingle{
						Address: "localhost:6379",
					},
					MasterName:      "mymaster",
					SentinelAddress: []string{"localhost:26379"},
				},
				Mode: "sentinel",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRedisConfig(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDefaultRedisConfig(t *testing.T) {
	config := DefaultRedisConfig()

	require.NotNil(t, config)
	assert.Equal(t, 3, config.MaxRetries)
	assert.Equal(t, 5*time.Second, config.ConnectTimeout)
	assert.Equal(t, 3*time.Second, config.ReadTimeout)
	assert.Equal(t, 3*time.Second, config.WriteTimeout)
	assert.Equal(t, 10, config.PoolSize)
	assert.Equal(t, 5, config.MinIdleConns)
	assert.Equal(t, 10, config.MaxIdleConns)
	assert.Equal(t, 4*time.Second, config.PoolTimeout)
	assert.Equal(t, 5*time.Minute, config.IdleTimeout)
	assert.Equal(t, 30*time.Minute, config.MaxConnAge)
}

func TestApplyRedisOptions(t *testing.T) {
	defaults := DefaultRedisConfig()

	t.Run("single mode options", func(t *testing.T) {
		config := &schema.Redis{
			RedisCluster: schema.RedisCluster{
				RedisSingle: schema.RedisSingle{
					Address:  "localhost:6379",
					Username: stringPtr("testuser"),
					Password: stringPtr("testpass"),
					DB:       intPtr(1),
					PoolSize: intPtr(20),
				},
			},
			Mode: "single",
		}

		opts := &redis.Options{
			Addr: config.Address,
		}

		applyRedisOptions(opts, config, defaults)

		assert.Equal(t, "testuser", opts.Username)
		assert.Equal(t, "testpass", opts.Password)
		assert.Equal(t, 1, opts.DB)
		assert.Equal(t, 20, opts.PoolSize)
		assert.Equal(t, defaults.MaxRetries, opts.MaxRetries)
		assert.Equal(t, defaults.ConnectTimeout, opts.DialTimeout)
	})

	t.Run("failover mode options", func(t *testing.T) {
		config := &schema.Redis{
			RedisCluster: schema.RedisCluster{
				RedisSingle: schema.RedisSingle{
					Address:  "localhost:6379",
					Username: stringPtr("testuser"),
					Password: stringPtr("testpass"),
					DB:       intPtr(2),
				},
				MasterName:      "mymaster",
				SentinelAddress: []string{"localhost:26379"},
				RouteByLatency:  boolPtr(true),
			},
			Mode: "sentinel",
		}

		opts := &redis.FailoverOptions{
			MasterName:    config.MasterName,
			SentinelAddrs: config.SentinelAddress,
		}

		applyRedisOptions(opts, config, defaults)

		assert.Equal(t, "testuser", opts.Username)
		assert.Equal(t, "testpass", opts.Password)
		assert.Equal(t, 2, opts.DB)
		assert.True(t, opts.RouteByLatency)
		assert.Equal(t, defaults.MaxRetries, opts.MaxRetries)
	})
}

func TestGetRedisClient(t *testing.T) {
	// Initially should be nil
	client := GetRedisClient()
	assert.Nil(t, client)

	// Set a mock client
	mockClient := &redis.Client{}
	redisMutex.Lock()
	redisClient = mockClient
	redisMutex.Unlock()

	// Should return the mock client
	client = GetRedisClient()
	assert.Equal(t, mockClient, client)

	// Clean up
	redisMutex.Lock()
	redisClient = nil
	redisMutex.Unlock()
}

func TestIsRedisConnected(t *testing.T) {
	ctx := context.Background()

	// No client - should return false
	connected := IsRedisConnected(ctx)
	assert.False(t, connected)
}

func TestCloseRedis(t *testing.T) {
	// Test with nil client
	err := CloseRedis()
	assert.NoError(t, err)

	// Test that client is set to nil after close
	redisMutex.Lock()
	redisClient = nil // Ensure it's nil
	redisMutex.Unlock()

	// Call close again - should not panic
	err = CloseRedis()
	assert.NoError(t, err)

	client := GetRedisClient()
	assert.Nil(t, client)
}

func TestGetRedisStats(t *testing.T) {
	// With nil client
	stats := GetRedisStats()
	assert.Nil(t, stats)
}

func (suite *RedisTestSuite) TestConfigurationValidation() {
	// Test various configuration scenarios
	testCases := []struct {
		name   string
		config *schema.Redis
		valid  bool
	}{
		{
			name: "valid single mode",
			config: &schema.Redis{
				RedisCluster: schema.RedisCluster{
					RedisSingle: schema.RedisSingle{
						Address: "localhost:6379",
					},
				},
				Mode: "single",
			},
			valid: true,
		},
		{
			name: "valid sentinel mode",
			config: &schema.Redis{
				RedisCluster: schema.RedisCluster{
					RedisSingle: schema.RedisSingle{
						Address: "localhost:6379",
					},
					MasterName:      "mymaster",
					SentinelAddress: []string{"localhost:26379", "localhost:26380"},
				},
				Mode: "sentinel",
			},
			valid: true,
		},
		{
			name: "invalid mode",
			config: &schema.Redis{
				RedisCluster: schema.RedisCluster{
					RedisSingle: schema.RedisSingle{
						Address: "localhost:6379",
					},
				},
				Mode: "invalid",
			},
			valid: false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			err := validateRedisConfig(tc.config)
			if tc.valid {
				suite.NoError(err)
			} else {
				suite.Error(err)
			}
		})
	}
}

func (suite *RedisTestSuite) TestTimeoutConversions() {
	config := &schema.Redis{
		RedisCluster: schema.RedisCluster{
			RedisSingle: schema.RedisSingle{
				Address:         "localhost:6379",
				DialTimeout:     intPtr(10), // 10 seconds
				ReadTimeout:     intPtr(5),  // 5 seconds
				WriteTimeout:    intPtr(3),  // 3 seconds
				MinRetryBackoff: intPtr(1),  // 1 second
				MaxRetryBackoff: intPtr(5),  // 5 seconds
			},
		},
		Mode: "single",
	}

	defaults := DefaultRedisConfig()
	opts := &redis.Options{Addr: config.Address}

	applyRedisOptions(opts, config, defaults)

	suite.Equal(10*time.Second, opts.DialTimeout)
	suite.Equal(5*time.Second, opts.ReadTimeout)
	suite.Equal(3*time.Second, opts.WriteTimeout)
	suite.Equal(1*time.Second, opts.MinRetryBackoff)
	suite.Equal(5*time.Second, opts.MaxRetryBackoff)
}

// Helper functions for creating pointers
func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

func boolPtr(b bool) *bool {
	return &b
}

// Benchmark tests
func BenchmarkValidateRedisConfig(b *testing.B) {
	config := &schema.Redis{
		RedisCluster: schema.RedisCluster{
			RedisSingle: schema.RedisSingle{
				Address: "localhost:6379",
			},
		},
		Mode: "single",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validateRedisConfig(config)
	}
}

func BenchmarkApplyRedisOptions(b *testing.B) {
	config := &schema.Redis{
		RedisCluster: schema.RedisCluster{
			RedisSingle: schema.RedisSingle{
				Address:     "localhost:6379",
				Username:    stringPtr("user"),
				Password:    stringPtr("pass"),
				DB:          intPtr(1),
				PoolSize:    intPtr(20),
				DialTimeout: intPtr(5),
			},
		},
		Mode: "single",
	}
	defaults := DefaultRedisConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		opts := &redis.Options{Addr: config.Address}
		applyRedisOptions(opts, config, defaults)
	}
}
