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
