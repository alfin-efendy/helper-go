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
	"github.com/alfin-efendy/helper-go/otel"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

var (
	Server  *gin.Engine
	options []healthcheck.Option
)

func Init(ctx context.Context) {
	ctx, span := otel.Trace(ctx)
	defer span.End()

	Server = gin.Default()

	Server.Use(
		otelgin.Middleware(config.Config.App.Name),
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

func Run(ctx context.Context) {
	conf := config.Config

	if conf.Server.RestAPI == nil {
		logger.Warn(ctx, "REST API is disabled")
		return
	}

	if conf.Database.Redis != nil {
		redis := database.GetRedisClient()

		addChecker("redis", func(ctx context.Context) error {
			if _, err := redis.Ping(ctx).Result(); err != nil {
				return err
			}
			return nil
		})
	}

	if conf.Database.Sql != nil {
		sqlClient := database.GetSqlClient()
		dbsql, _ := sqlClient.DB()

		addChecker("sql", func(ctx context.Context) error {
			return dbsql.Ping()
		})
	}

	host := conf.Server.RestAPI.Host
	port := conf.Server.RestAPI.Port

	err := Server.Run(fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		logger.Fatal(ctx, err, fmt.Sprintf("Failed to run REST server, port=%d", port))
	}
}
