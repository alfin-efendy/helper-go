package restapi

import (
	"github.com/alfin-efendy/helper-go/server"
	"github.com/gin-gonic/gin"
)

func GetPage(ctx *gin.Context) server.PageRequest {
	page, _ := ctx.Get(pageStr)
	return page.(server.PageRequest)
}
