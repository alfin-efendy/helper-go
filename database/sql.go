package database

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/alfin-efendy/helper-go/config"
	log "github.com/alfin-efendy/helper-go/logger"
	"go.uber.org/zap/zapcore"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

var (
	sqlClient   *gorm.DB
	zapLevelMap = map[zapcore.Level]logger.LogLevel{
		zapcore.PanicLevel: logger.Error,
		zapcore.FatalLevel: logger.Error,
		zapcore.ErrorLevel: logger.Error,
		zapcore.WarnLevel:  logger.Warn,
		zapcore.InfoLevel:  logger.Warn,
		zapcore.DebugLevel: logger.Info,
	}
)

func initSql(ctx context.Context) {
	config := config.Config.Database.Sql
	logLevel := log.GetLevel()

	dialector := postgres.Open(
		fmt.Sprintf(
			"host=%s port=%d user=%s dbname=%s password=%s sslmode=disable",
			config.Host,
			config.Port,
			config.Username,
			config.Database,
			config.Password,
		),
	)

	loggerConfig := logger.Config{
		SlowThreshold:             3 * time.Second,
		LogLevel:                  zapLevelMap[logLevel],
		Colorful:                  true,
		IgnoreRecordNotFoundError: true,
	}

	logging := log.New(log.WithConfig(loggerConfig))

	db, err := gorm.Open(dialector, &gorm.Config{
		SkipDefaultTransaction: true,
		Logger:                 logging,
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
			NameReplacer:  strings.NewReplacer("-", "_", " ", "_"),
			NoLowerCase:   false,
		},
	})

	if err != nil {
		log.Fatal(ctx, err, "❌ Failed to open sql database connection")
		return
	}

	log.Info(ctx, "✅ Database connection established")

	db.Session(&gorm.Session{
		FullSaveAssociations: true,
		PrepareStmt:          true,
	})

	dbSql, err := db.DB()
	if err != nil {
		log.Fatal(ctx, err, "❌ Failed to get sql database connection")
		return
	}

	dbSql.SetMaxIdleConns(config.PoolingConnection.MaxIdle)
	dbSql.SetMaxOpenConns(config.PoolingConnection.MaxOpen)
	dbSql.SetConnMaxLifetime(time.Duration(config.PoolingConnection.MaxLifetime) * time.Second)
	db.Config.NamingStrategy = schema.NamingStrategy{}

	sqlClient = db
}

func GetSqlClient() *gorm.DB {
	return sqlClient
}
