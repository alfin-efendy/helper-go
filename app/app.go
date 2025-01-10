package app

import (
	"sync"

	"github.com/alfin-efendy/helper-go/config"
	"github.com/alfin-efendy/helper-go/database"
	"github.com/alfin-efendy/helper-go/logger"
	"github.com/alfin-efendy/helper-go/server/restapi"
	"github.com/alfin-efendy/helper-go/storage"
)

func init() {
	config.Load()
	logger.Init()
	database.Init()
	storage.Init()
}

func Start(fn func()) {
	fn()
	go restapi.Run()

	group := sync.WaitGroup{}
	group.Add(1)
	group.Wait()
}
