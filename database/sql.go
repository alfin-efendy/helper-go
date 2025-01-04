package database

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/alfin-efendy/helper-go/config"
	log "github.com/alfin-efendy/helper-go/logger"
	"github.com/alfin-efendy/helper-go/server/restapi"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

var (
	sqlClient      *gorm.DB
	logrusLevelMap = map[logrus.Level]logger.LogLevel{
		logrus.PanicLevel: logger.Error,
		logrus.FatalLevel: logger.Error,
		logrus.ErrorLevel: logger.Error,
		logrus.WarnLevel:  logger.Warn,
		logrus.InfoLevel:  logger.Warn,
		logrus.DebugLevel: logger.Info,
		logrus.TraceLevel: logger.Info,
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
		LogLevel:                  logrusLevelMap[logLevel],
		Colorful:                  true,
		IgnoreRecordNotFoundError: true,
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		SkipDefaultTransaction: true,
		Logger:                 logger.New(log.GetLogrusLogger(), loggerConfig),
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

	restapi.AddChecker("sql", func(ctx context.Context) error {
		return dbSql.Ping()
	})

	sqlClient = db
}

func GetSqlClient() *gorm.DB {
	return sqlClient
}
