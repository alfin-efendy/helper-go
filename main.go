package main

import (
	"context"
	"net/http"

	"github.com/alfin-efendy/helper-go/app"
	"github.com/alfin-efendy/helper-go/config"
	"github.com/alfin-efendy/helper-go/otel"
	"github.com/alfin-efendy/helper-go/server"
	"github.com/alfin-efendy/helper-go/server/restapi"
	"github.com/gin-gonic/gin"
)

type CreateUserRequest struct {
	Type string `json:"type" binding:"required,enum=employee/customer/vendor"`
}

// This is a example of how to use the helper-go package
func main() {
	app.Start(func() {
		// Your code here
		email := config.GetValue("superuser.email")
		password := config.GetValue("superuser.password")

		println(email, password)

		restapi.Server.GET("/hello", restapi.AuthMiddleware(""), HelloHandler)

		restapi.Server.POST("/user", CreateUserHandler)

	})
}

type User struct {
	FullName string `json:"fullName" binding:"required"`
	Type     string `json:"type" binding:"required,enum=employee/customer/vendor"`
	Status   string `json:"status" binding:"required,enum=active/inactive"`
}

func HelloHandler(c *gin.Context) {
	ctx, span := otel.Trace(c.Request.Context())
	defer span.End()

	Test(ctx)

	c.JSON(http.StatusOK, gin.H{
		"message": "Hello, World!",
	})
}

func Test(c context.Context) {
	_, span := otel.Trace(c)
	defer span.End()
}

func CreateUserHandler(c *gin.Context) {
	var user User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.Error(err)
		return
	}

	pagination := restapi.GetPage(c)

	restapi.SetData(c, user)
	restapi.SetPaggination(c, server.PageResponse{
		TotalPage:   pagination.PageSize,
		TotalRecord: int64(pagination.Page),
	})
}
