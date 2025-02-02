package database

import (
	"context"
	"fmt"
	"time"

	"github.com/alfin-efendy/helper-go/config"
	"github.com/alfin-efendy/helper-go/config/model"
	"github.com/alfin-efendy/helper-go/logger"
	"github.com/redis/go-redis/v9"
)

var redisClient *redis.Client

func initRedis(ctx context.Context) {
	config := config.Config
	if config.Database.Redis == nil {
		logger.Warn(ctx, "❌ Redis configuration is not found")
		return
	}

	switch config.Database.Redis.Mode {
	case "single":
		redisClient = initSingleMode(ctx, config)
	case "sentinel":
		redisClient = initSentinelMode(ctx, config)
	default:
		logger.Fatal(ctx, fmt.Errorf("redis mode %s is not supported", config.Database.Redis.Mode), "❌ Failed unsupported redis mode")
		return
	}

	logger.Info(ctx, "✅ Redis client connected")
}

func initSingleMode(ctx context.Context, config *model.Config) *redis.Client {
	configRedis := config.Database.Redis
	option := &redis.Options{
		Addr: configRedis.Address,
	}

	if configRedis.Username != nil {
		option.Username = *configRedis.Username
	}
	if configRedis.Password != nil {
		option.Password = *configRedis.Password
	}
	if configRedis.DB != nil {
		option.DB = *configRedis.DB
	}
	if configRedis.MinRetryBackoff != nil {
		option.MinRetryBackoff = time.Duration(*configRedis.MinRetryBackoff) * time.Minute
	}
	if configRedis.MaxRetryBackoff != nil {
		option.MaxRetryBackoff = time.Duration(*configRedis.MaxRetryBackoff) * time.Minute
	}
	if configRedis.DialTimeout != nil {
		option.DialTimeout = time.Duration(*configRedis.DialTimeout) * time.Minute
	}
	if configRedis.ReadTimeout != nil {
		option.ReadTimeout = time.Duration(*configRedis.ReadTimeout) * time.Minute
	}
	if configRedis.WriteTimeout != nil {
		option.WriteTimeout = time.Duration(*configRedis.WriteTimeout) * time.Minute
	}
	if configRedis.PoolFIFO != nil {
		option.PoolFIFO = *configRedis.PoolFIFO
	}
	if configRedis.PoolSize != nil {
		option.PoolSize = *configRedis.PoolSize
	}
	if configRedis.PoolTimeout != nil {
		option.PoolTimeout = time.Duration(*configRedis.PoolTimeout) * time.Minute
	}
	if configRedis.MinIdleConns != nil {
		option.MinIdleConns = *configRedis.MinIdleConns
	}
	if configRedis.MaxIdleConns != nil {
		option.MaxIdleConns = *configRedis.MaxIdleConns
	}

	RedisClient := redis.NewClient(option)

	if _, err := RedisClient.Ping(ctx).Result(); err != nil {
		logger.Fatal(ctx, err, "❌ Redis client failed to connect")
	}
	return RedisClient
}

func initSentinelMode(ctx context.Context, config *model.Config) *redis.Client {
	configRedis := config.Database.Redis
	option := &redis.FailoverOptions{
		MasterName:    configRedis.MasterName,
		SentinelAddrs: configRedis.SentinelAddress,
	}

	if configRedis.Username != nil {
		option.Username = *configRedis.Username
	}
	if configRedis.Password != nil {
		option.Password = *configRedis.Password
	}
	if configRedis.DB != nil {
		option.DB = *configRedis.DB
	}
	if configRedis.MinRetryBackoff != nil {
		option.MinRetryBackoff = time.Duration(*configRedis.MinRetryBackoff) * time.Minute
	}
	if configRedis.MaxRetryBackoff != nil {
		option.MaxRetryBackoff = time.Duration(*configRedis.MaxRetryBackoff) * time.Minute
	}
	if configRedis.DialTimeout != nil {
		option.DialTimeout = time.Duration(*configRedis.DialTimeout) * time.Minute
	}
	if configRedis.ReadTimeout != nil {
		option.ReadTimeout = time.Duration(*configRedis.ReadTimeout) * time.Minute
	}
	if configRedis.WriteTimeout != nil {
		option.WriteTimeout = time.Duration(*configRedis.WriteTimeout) * time.Minute
	}
	if configRedis.PoolFIFO != nil {
		option.PoolFIFO = *configRedis.PoolFIFO
	}
	if configRedis.PoolSize != nil {
		option.PoolSize = *configRedis.PoolSize
	}
	if configRedis.PoolTimeout != nil {
		option.PoolTimeout = time.Duration(*configRedis.PoolTimeout) * time.Minute
	}
	if configRedis.MinIdleConns != nil {
		option.MinIdleConns = *configRedis.MinIdleConns
	}
	if configRedis.MaxIdleConns != nil {
		option.MaxIdleConns = *configRedis.MaxIdleConns
	}

	RedisClient := redis.NewFailoverClient(option)

	if _, err := RedisClient.Ping(ctx).Result(); err != nil {
		logger.Fatal(ctx, err, "❌ Redis client failed to connect")
	}

	return RedisClient
}

func GetRedisClient() *redis.Client {
	return redisClient
}
