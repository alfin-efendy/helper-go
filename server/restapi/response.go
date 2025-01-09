package restapi

import (
	"github.com/gin-gonic/gin"
)

func SetData(ctx *gin.Context, data interface{}) {
	ctx.Set(dataStr, data)
}

func SetPaggination(ctx *gin.Context, page PageResponse) {
	ctx.Set(pageStr, page)
}

func SetRawResponse(ctx *gin.Context, httpCode int, message string) {
	ctx.JSON(httpCode, gin.H{
		"message": message,
	})
}
