package app

import (
	"context"
	"sync"

	"github.com/alfin-efendy/helper-go/config"
	"github.com/alfin-efendy/helper-go/database"
	"github.com/alfin-efendy/helper-go/logger"
	"github.com/alfin-efendy/helper-go/otel"
	"github.com/alfin-efendy/helper-go/server/restapi"
	"github.com/alfin-efendy/helper-go/storage"
)

var ctx context.Context

func init() {
	ctx = context.Background()

	config.Load()
	logger.Init()
	otel.Init()

	ctx, span := otel.Trace(ctx)
	defer span.End()

	database.Init(ctx)
	storage.Init(ctx)
	restapi.Init(ctx)
}

func Start(fn func()) {
	ctx, span := otel.Trace(ctx)
	defer span.End()

	fn()
	go restapi.Run(ctx)

	defer func() {
		err := otel.Shutdown(ctx)
		if err != nil {
			logger.Error(ctx, err)
		}
	}()

	group := sync.WaitGroup{}
	group.Add(1)
	group.Wait()
}
