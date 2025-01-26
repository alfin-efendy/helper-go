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

func init() {
	config.Load()
	logger.Init()
	otel.Init()
	database.Init()
	storage.Init()
	restapi.Init()
}

func Start(fn func()) {
	fn()
	go restapi.Run()

	defer func() {
		ctx := context.Background()

		err := otel.Shutdown(ctx)
		if err != nil {
			logger.Error(ctx, err)
		}
	}()

	group := sync.WaitGroup{}
	group.Add(1)
	group.Wait()
}
