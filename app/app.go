package app

import (
	"sync"

	"github.com/alfin-efendy/helper-go/config"
	"github.com/alfin-efendy/helper-go/database"
	"github.com/alfin-efendy/helper-go/logger"
	"github.com/alfin-efendy/helper-go/server/restapi"
)

func init() {
	config.Load()
	logger.Init()
	database.Init()
}

func Start() {
	go restapi.Run()

	group := sync.WaitGroup{}
	group.Add(1)
	group.Wait()
}
