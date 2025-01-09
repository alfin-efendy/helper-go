package restapi

import (
	"github.com/gin-gonic/gin"
)

func GetPage(ctx *gin.Context) PageRequest {
	page, _ := ctx.Get(pageStr)
	return page.(PageRequest)
}
