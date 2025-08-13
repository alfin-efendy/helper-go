package app

import (
	"context"

	"github.com/alfin-efendy/helper-go/config"
	"github.com/alfin-efendy/helper-go/logger"
	"github.com/alfin-efendy/helper-go/otel"
)

func Start(fn func()) context.Context {
	var ctx context.Context

	// Initialize configuration
	ctx, span := otel.Trace(ctx)
	defer span.End()

	// Load the configuration at application startup
	config.Load()
	logger, err := logger.NewLogger(
		logger.WithConfig(),
	)
	if err != nil {
		panic("Failed to initialize logger: " + err.Error())
	}
	defer func() {
		if err := logger.Close(); err != nil {
			panic("Failed to close logger: " + err.Error())
		}
	}()

	otel.Init()
	fn()

	return ctx
}
