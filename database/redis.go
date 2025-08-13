package database

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/alfin-efendy/helper-go/config"
	"github.com/alfin-efendy/helper-go/config/schema"
	"github.com/alfin-efendy/helper-go/logger"
	"github.com/redis/go-redis/v9"
)

var (
	redisClient *redis.Client
	redisMutex  sync.RWMutex
	redisOnce   sync.Once
)

// RedisConfig holds configuration for Redis connection
type RedisConfig struct {
	MaxRetries     int
	ConnectTimeout time.Duration
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	PoolSize       int
	MinIdleConns   int
	MaxIdleConns   int
	PoolTimeout    time.Duration
	IdleTimeout    time.Duration
	MaxConnAge     time.Duration
}

// DefaultRedisConfig returns sensible defaults for Redis configuration
func DefaultRedisConfig() *RedisConfig {
	return &RedisConfig{
		MaxRetries:     3,
		ConnectTimeout: 5 * time.Second,
		ReadTimeout:    3 * time.Second,
		WriteTimeout:   3 * time.Second,
		PoolSize:       10,
		MinIdleConns:   5,
		MaxIdleConns:   10,
		PoolTimeout:    4 * time.Second,
		IdleTimeout:    5 * time.Minute,
		MaxConnAge:     30 * time.Minute,
	}
}

// validateRedisConfig validates the Redis configuration
func validateRedisConfig(cfg *schema.Redis) error {
	if cfg == nil {
		return fmt.Errorf("redis configuration is nil")
	}

	if cfg.Address == "" {
		return fmt.Errorf("redis address is required")
	}

	if cfg.Mode != "single" && cfg.Mode != "sentinel" {
		return fmt.Errorf("unsupported Redis mode: %s (supported: single, sentinel)", cfg.Mode)
	}

	if cfg.Mode == "sentinel" {
		if cfg.MasterName == "" {
			return fmt.Errorf("master name is required for sentinel mode")
		}
		if len(cfg.SentinelAddress) == 0 {
			return fmt.Errorf("sentinel addresses are required for sentinel mode")
		}
	}

	return nil
}

func InitRedis(ctx context.Context) error {
	config := config.Data
	if config.Database.Redis == nil {
		logger.Warn(ctx, "❌ Redis configuration is not found")
		return fmt.Errorf("redis configuration not found")
	}

	// Validate configuration
	if err := validateRedisConfig(config.Database.Redis); err != nil {
		logger.Error(ctx, err, "❌ Invalid Redis configuration")
		return fmt.Errorf("invalid Redis configuration: %w", err)
	}

	redisOnce.Do(func() {
		var err error
		var client *redis.Client

		switch config.Database.Redis.Mode {
		case "single":
			client = initSingleMode(config)
		case "sentinel":
			client = initSentinelMode(config)
		default:
			err = fmt.Errorf("redis mode %s is not supported", config.Database.Redis.Mode)
		}

		if err != nil {
			logger.Error(ctx, err, "❌ Failed to initialize Redis client")
			return
		}

		// Test connection
		if err := testRedisConnection(ctx, client); err != nil {
			logger.Error(ctx, err, "❌ Redis connection test failed")
			if closeErr := client.Close(); closeErr != nil {
				logger.Error(ctx, closeErr, "❌ Failed to close Redis client")
			}
			return
		}

		redisMutex.Lock()
		redisClient = client
		redisMutex.Unlock()

		logger.Info(ctx, "✅ Redis client connected successfully")
	})

	return nil
}

// testRedisConnection tests if the Redis connection is working
func testRedisConnection(ctx context.Context, client *redis.Client) error {
	// Test basic connectivity
	pong, err := client.Ping(ctx).Result()
	if err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}

	if pong != "PONG" {
		return fmt.Errorf("unexpected ping response: %s", pong)
	}

	// Test basic operations
	testKey := "__health_check__"
	testValue := "ok"

	// Set a test value
	err = client.Set(ctx, testKey, testValue, time.Second*10).Err()
	if err != nil {
		return fmt.Errorf("test set operation failed: %w", err)
	}

	// Get the test value
	val, err := client.Get(ctx, testKey).Result()
	if err != nil {
		return fmt.Errorf("test get operation failed: %w", err)
	}

	if val != testValue {
		return fmt.Errorf("test value mismatch: expected %s, got %s", testValue, val)
	}

	// Clean up test key
	client.Del(ctx, testKey)

	return nil
}

// applyRedisOptions applies configuration options to Redis client options
func applyRedisOptions(options interface{}, configRedis *schema.Redis, defaults *RedisConfig) {
	switch opts := options.(type) {
	case *redis.Options:
		applyRedisBaseOptions(opts, configRedis, defaults)

		if configRedis.Username != nil {
			opts.Username = *configRedis.Username
		}
		if configRedis.Password != nil {
			opts.Password = *configRedis.Password
		}
		if configRedis.DB != nil {
			opts.DB = *configRedis.DB
		}

	case *redis.FailoverOptions:
		applyFailoverOptions(opts, configRedis, defaults)

		if configRedis.Username != nil {
			opts.Username = *configRedis.Username
		}
		if configRedis.Password != nil {
			opts.Password = *configRedis.Password
		}
		if configRedis.DB != nil {
			opts.DB = *configRedis.DB
		}
	}
}

// applyRedisBaseOptions applies configuration to basic Redis options
func applyRedisBaseOptions(opts *redis.Options, configRedis *schema.Redis, defaults *RedisConfig) {
	// Set defaults first
	opts.MaxRetries = defaults.MaxRetries
	opts.MinRetryBackoff = defaults.ConnectTimeout / 4
	opts.MaxRetryBackoff = defaults.ConnectTimeout
	opts.DialTimeout = defaults.ConnectTimeout
	opts.ReadTimeout = defaults.ReadTimeout
	opts.WriteTimeout = defaults.WriteTimeout
	opts.PoolSize = defaults.PoolSize
	opts.MinIdleConns = defaults.MinIdleConns
	opts.MaxIdleConns = defaults.MaxIdleConns
	opts.PoolTimeout = defaults.PoolTimeout
	opts.ConnMaxIdleTime = defaults.IdleTimeout
	opts.ConnMaxLifetime = defaults.MaxConnAge

	// Apply configuration overrides
	if configRedis.MaxRetries != nil {
		opts.MaxRetries = *configRedis.MaxRetries
	}
	if configRedis.MinRetryBackoff != nil {
		opts.MinRetryBackoff = time.Duration(*configRedis.MinRetryBackoff) * time.Second
	}
	if configRedis.MaxRetryBackoff != nil {
		opts.MaxRetryBackoff = time.Duration(*configRedis.MaxRetryBackoff) * time.Second
	}
	if configRedis.DialTimeout != nil {
		opts.DialTimeout = time.Duration(*configRedis.DialTimeout) * time.Second
	}
	if configRedis.ReadTimeout != nil {
		opts.ReadTimeout = time.Duration(*configRedis.ReadTimeout) * time.Second
	}
	if configRedis.WriteTimeout != nil {
		opts.WriteTimeout = time.Duration(*configRedis.WriteTimeout) * time.Second
	}
	if configRedis.PoolFIFO != nil {
		opts.PoolFIFO = *configRedis.PoolFIFO
	}
	if configRedis.PoolSize != nil {
		opts.PoolSize = *configRedis.PoolSize
	}
	if configRedis.PoolTimeout != nil {
		opts.PoolTimeout = time.Duration(*configRedis.PoolTimeout) * time.Second
	}
	if configRedis.MinIdleConns != nil {
		opts.MinIdleConns = *configRedis.MinIdleConns
	}
	if configRedis.MaxIdleConns != nil {
		opts.MaxIdleConns = *configRedis.MaxIdleConns
	}
}

// applyFailoverOptions applies configuration to Redis failover options
func applyFailoverOptions(opts *redis.FailoverOptions, configRedis *schema.Redis, defaults *RedisConfig) {
	// Set defaults first
	opts.MaxRetries = defaults.MaxRetries
	opts.MinRetryBackoff = defaults.ConnectTimeout / 4
	opts.MaxRetryBackoff = defaults.ConnectTimeout
	opts.DialTimeout = defaults.ConnectTimeout
	opts.ReadTimeout = defaults.ReadTimeout
	opts.WriteTimeout = defaults.WriteTimeout
	opts.PoolSize = defaults.PoolSize
	opts.MinIdleConns = defaults.MinIdleConns
	opts.MaxIdleConns = defaults.MaxIdleConns
	opts.PoolTimeout = defaults.PoolTimeout
	opts.ConnMaxIdleTime = defaults.IdleTimeout
	opts.ConnMaxLifetime = defaults.MaxConnAge

	// Apply configuration overrides
	if configRedis.MaxRetries != nil {
		opts.MaxRetries = *configRedis.MaxRetries
	}
	if configRedis.MinRetryBackoff != nil {
		opts.MinRetryBackoff = time.Duration(*configRedis.MinRetryBackoff) * time.Second
	}
	if configRedis.MaxRetryBackoff != nil {
		opts.MaxRetryBackoff = time.Duration(*configRedis.MaxRetryBackoff) * time.Second
	}
	if configRedis.DialTimeout != nil {
		opts.DialTimeout = time.Duration(*configRedis.DialTimeout) * time.Second
	}
	if configRedis.ReadTimeout != nil {
		opts.ReadTimeout = time.Duration(*configRedis.ReadTimeout) * time.Second
	}
	if configRedis.WriteTimeout != nil {
		opts.WriteTimeout = time.Duration(*configRedis.WriteTimeout) * time.Second
	}
	if configRedis.PoolFIFO != nil {
		opts.PoolFIFO = *configRedis.PoolFIFO
	}
	if configRedis.PoolSize != nil {
		opts.PoolSize = *configRedis.PoolSize
	}
	if configRedis.PoolTimeout != nil {
		opts.PoolTimeout = time.Duration(*configRedis.PoolTimeout) * time.Second
	}
	if configRedis.MinIdleConns != nil {
		opts.MinIdleConns = *configRedis.MinIdleConns
	}
	if configRedis.MaxIdleConns != nil {
		opts.MaxIdleConns = *configRedis.MaxIdleConns
	}

	// Failover-specific options
	if configRedis.RouteByLatency != nil {
		opts.RouteByLatency = *configRedis.RouteByLatency
	}
	if configRedis.RouteRandomly != nil {
		opts.RouteRandomly = *configRedis.RouteRandomly
	}
	if configRedis.ReplicaOnly != nil {
		opts.ReplicaOnly = *configRedis.ReplicaOnly
	}
	if configRedis.UseDisconnectedReplicas != nil {
		opts.UseDisconnectedReplicas = *configRedis.UseDisconnectedReplicas
	}
}

func initSingleMode(config *schema.Config) *redis.Client {
	configRedis := config.Database.Redis
	defaults := DefaultRedisConfig()

	option := &redis.Options{
		Addr: configRedis.Address,
	}

	// Apply all configuration options
	applyRedisOptions(option, configRedis, defaults)

	client := redis.NewClient(option)
	return client
}

func initSentinelMode(config *schema.Config) *redis.Client {
	configRedis := config.Database.Redis
	defaults := DefaultRedisConfig()

	option := &redis.FailoverOptions{
		MasterName:    configRedis.MasterName,
		SentinelAddrs: configRedis.SentinelAddress,
	}

	// Apply all configuration options
	applyRedisOptions(option, configRedis, defaults)

	client := redis.NewFailoverClient(option)
	return client
}

// GetRedisClient returns the Redis client instance thread-safely
func GetRedisClient() *redis.Client {
	redisMutex.RLock()
	defer redisMutex.RUnlock()
	return redisClient
}

// IsRedisConnected checks if Redis client is connected
func IsRedisConnected(ctx context.Context) bool {
	client := GetRedisClient()
	if client == nil {
		return false
	}

	err := client.Ping(ctx).Err()
	return err == nil
}

// CloseRedis closes the Redis connection and cleans up resources
func CloseRedis() error {
	redisMutex.Lock()
	defer redisMutex.Unlock()

	if redisClient != nil {
		err := redisClient.Close()
		redisClient = nil
		return err
	}
	return nil
}

// GetRedisStats returns Redis connection statistics
func GetRedisStats() *redis.PoolStats {
	client := GetRedisClient()
	if client == nil {
		return nil
	}
	return client.PoolStats()
}
