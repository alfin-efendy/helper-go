package restapi

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/etherlabsio/healthcheck/v2"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"

	"github.com/alfin-efendy/helper-go/config"
	"github.com/alfin-efendy/helper-go/database"
	"github.com/alfin-efendy/helper-go/logger"
)

var (
	Server  *gin.Engine
	options []healthcheck.Option
)

func init() {
	Server = gin.Default()

	Server.Use(
		traceRequest(),
		gin.Recovery(),
		gzip.Gzip(gzip.DefaultCompression),
		loggerMiddleware(),
		corsMiddleware(),
		helmetMiddleware(),
		paginationRequest(),
		errorResponse(),
		successResponse(),
	)

	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("isUrl", isUrl)
		v.RegisterValidation("isActiveEmail", isActiveEmail)
		_ = v.RegisterValidation("enum", enum)
	}

	Server.GET("/_health", gin.WrapH(healthz()))
}

func addChecker(name string, f func(ctx context.Context) error) {
	options = append(
		options,
		healthcheck.WithChecker(
			name,
			healthcheck.CheckerFunc(f),
		))
}

// @Summary		Health Check
// @Description	Perform a health check
// @Produce		json
// @Success		200
// @Failure		503
// @Router			/healthz [get]
func healthz() http.Handler {
	options = append(options, healthcheck.WithTimeout(5*time.Second))
	return healthcheck.Handler(options...)
}

func Run() {
	ctx := context.Background()

	if config.Config.Server.RestAPI == nil {
		logger.Warn(ctx, "REST API is disabled")
		return
	}

	redis := database.GetRedisClient()

	addChecker("redis", func(ctx context.Context) error {
		if _, err := redis.Ping(ctx).Result(); err != nil {
			return err
		}
		return nil
	})

	sqlClient := database.GetSqlClient()
	dbsql, _ := sqlClient.DB()

	addChecker("sql", func(ctx context.Context) error {
		return dbsql.Ping()
	})

	host := config.Config.Server.RestAPI.Host
	port := config.Config.Server.RestAPI.Port

	err := Server.Run(fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		logger.Fatal(ctx, err, fmt.Sprintf("Failed to run REST server, port=%d", port))
	}
}
