package restapi

import (
	"github.com/alfin-efendy/helper-go/server"
	"github.com/gin-gonic/gin"
)

func SetData(ctx *gin.Context, data interface{}) {
	ctx.Set(dataStr, data)
}

func SetPaggination(ctx *gin.Context, page server.PageResponse) {
	ctx.Set(pageStr, page)
}

func SetRawResponse(ctx *gin.Context, httpCode int, message string) {
	ctx.JSON(httpCode, gin.H{
		"message": message,
	})
}
