package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/alfin-efendy/helper-go/config"
	"github.com/alfin-efendy/helper-go/config/schema"
	log "github.com/alfin-efendy/helper-go/logger"
	"github.com/rs/zerolog"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	gormSchema "gorm.io/gorm/schema"
)

var (
	sqlClient       *gorm.DB
	sqlClientMutex  sync.RWMutex
	isConnected     bool
	connectionStats ConnectionStats
	zerologLevelMap = map[zerolog.Level]logger.LogLevel{
		zerolog.PanicLevel: logger.Error,
		zerolog.FatalLevel: logger.Error,
		zerolog.ErrorLevel: logger.Error,
		zerolog.WarnLevel:  logger.Warn,
		zerolog.InfoLevel:  logger.Info,
		zerolog.DebugLevel: logger.Info,
	}
)

// ConnectionStats holds database connection statistics
type ConnectionStats struct {
	OpenConnections int
	InUse           int
	Idle            int
	MaxOpen         int
	MaxIdle         int
	ConnectedAt     time.Time
	LastPingAt      time.Time
	PingDuration    time.Duration
}

func InitSQL(ctx context.Context) error {
	dbConfig := config.Data.Database.SQL
	if dbConfig == nil {
		err := fmt.Errorf("database configuration is not found")
		log.Warn(ctx, "❌ "+err.Error())
		return err
	}

	// Validate configuration
	if err := validateSQLConfig(dbConfig); err != nil {
		log.Error(ctx, err, "❌ Invalid database configuration")
		return err
	}

	// Determine SSL mode
	sslMode := "disable"
	if dbConfig.Host != "localhost" && dbConfig.Host != "127.0.0.1" {
		sslMode = "require"
	}

	dialector := postgres.Open(
		fmt.Sprintf(
			"host=%s port=%d user=%s dbname=%s password=%s sslmode=%s",
			dbConfig.Host,
			dbConfig.Port,
			dbConfig.Username,
			dbConfig.Database,
			dbConfig.Password,
			sslMode,
		),
	)

	// Get log level from config or default to info
	logLevel := zerolog.InfoLevel
	if config.Data.Log.Level != "" {
		if level, err := zerolog.ParseLevel(config.Data.Log.Level); err == nil {
			logLevel = level
		}
	}

	// Create Zerolog-based GORM logger
	gormLogger := log.NewGormLogger(
		log.WithLogLevel(zerologLevelMap[logLevel]),
		log.WithSlowThreshold(3*time.Second),
		log.WithIgnoreRecordNotFoundError(true),
		log.WithCustomFields(
			log.StringField("component", "database"),
			log.StringField("driver", "postgres"),
		),
	)

	db, err := gorm.Open(dialector, &gorm.Config{
		SkipDefaultTransaction: true,
		Logger:                 gormLogger,
		PrepareStmt:            true,
		NamingStrategy: gormSchema.NamingStrategy{
			SingularTable: true,
			NameReplacer:  strings.NewReplacer("-", "_", " ", "_"),
			NoLowerCase:   false,
		},
	})
	if err != nil {
		log.Error(ctx, err, "❌ Failed to open sql database connection")
		return err
	}

	// Get underlying database connection
	sqlDB, err := db.DB()
	if err != nil {
		log.Error(ctx, err, "❌ Failed to get underlying sql database connection")
		return err
	}

	// Configure connection pool with validation
	poolConfig := dbConfig.PoolingConnection
	if poolConfig.MaxIdle <= 0 {
		poolConfig.MaxIdle = 10
	}
	if poolConfig.MaxOpen <= 0 {
		poolConfig.MaxOpen = 100
	}
	if poolConfig.MaxLifetime <= 0 {
		poolConfig.MaxLifetime = 3600 // 1 hour
	}

	sqlDB.SetMaxIdleConns(poolConfig.MaxIdle)
	sqlDB.SetMaxOpenConns(poolConfig.MaxOpen)
	sqlDB.SetConnMaxLifetime(time.Duration(poolConfig.MaxLifetime) * time.Second)
	sqlDB.SetConnMaxIdleTime(15 * time.Minute)

	// Test connection with ping
	if err := pingDatabase(ctx, sqlDB); err != nil {
		log.Error(ctx, err, "❌ Failed to ping database")
		return err
	}

	// Thread-safe assignment
	sqlClientMutex.Lock()
	sqlClient = db
	isConnected = true
	connectionStats.ConnectedAt = time.Now()
	connectionStats.MaxOpen = poolConfig.MaxOpen
	connectionStats.MaxIdle = poolConfig.MaxIdle
	sqlClientMutex.Unlock()

	log.Info(ctx, "✅ Database connection established",
		"host", dbConfig.Host,
		"database", dbConfig.Database,
		"ssl_mode", sslMode,
		"max_open", poolConfig.MaxOpen,
		"max_idle", poolConfig.MaxIdle,
	)

	return nil
}

func GetSQLClient() *gorm.DB {
	sqlClientMutex.RLock()
	defer sqlClientMutex.RUnlock()
	return sqlClient
}

// IsSQLConnected checks if the database connection is active
func IsSQLConnected() bool {
	sqlClientMutex.RLock()
	defer sqlClientMutex.RUnlock()
	return isConnected && sqlClient != nil
}

// GetSQLStats returns database connection statistics
func GetSQLStats() ConnectionStats {
	sqlClientMutex.RLock()
	defer sqlClientMutex.RUnlock()

	if sqlClient == nil {
		return ConnectionStats{}
	}

	sqlDB, err := sqlClient.DB()
	if err != nil {
		return connectionStats
	}

	stats := sqlDB.Stats()
	connectionStats.OpenConnections = stats.OpenConnections
	connectionStats.InUse = stats.InUse
	connectionStats.Idle = stats.Idle

	return connectionStats
}

// PingSQLDatabase pings the database to check connectivity
func PingSQLDatabase(ctx context.Context) error {
	sqlClientMutex.RLock()
	client := sqlClient
	sqlClientMutex.RUnlock()

	if client == nil {
		return fmt.Errorf("database client is not initialized")
	}

	sqlDB, err := client.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying database: %w", err)
	}

	return pingDatabase(ctx, sqlDB)
}

// CloseSQL gracefully closes the database connection
func CloseSQL() error {
	sqlClientMutex.Lock()
	defer sqlClientMutex.Unlock()

	if sqlClient == nil {
		return nil
	}

	sqlDB, err := sqlClient.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying database for closing: %w", err)
	}

	if err := sqlDB.Close(); err != nil {
		return fmt.Errorf("failed to close database connection: %w", err)
	}

	sqlClient = nil
	isConnected = false
	connectionStats = ConnectionStats{}

	return nil
}

// validateSQLConfig validates the SQL configuration
func validateSQLConfig(config *schema.SQL) error {
	if config == nil {
		return fmt.Errorf("database config is nil")
	}
	if config.Host == "" {
		return fmt.Errorf("database host is required")
	}
	if config.Port <= 0 || config.Port > 65535 {
		return fmt.Errorf("database port must be between 1 and 65535")
	}
	if config.Username == "" {
		return fmt.Errorf("database username is required")
	}
	if config.Database == "" {
		return fmt.Errorf("database name is required")
	}
	if config.Password == "" {
		return fmt.Errorf("database password is required")
	}
	return nil
}

// pingDatabase pings the database with timeout and updates statistics
func pingDatabase(ctx context.Context, sqlDB *sql.DB) error {
	start := time.Now()

	// Create context with timeout
	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(pingCtx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	// Update ping statistics
	sqlClientMutex.Lock()
	connectionStats.LastPingAt = time.Now()
	connectionStats.PingDuration = time.Since(start)
	sqlClientMutex.Unlock()

	return nil
}
