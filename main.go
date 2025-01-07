package main

import (
	"net/http"

	"github.com/alfin-efendy/helper-go/app"
	"github.com/alfin-efendy/helper-go/config"
	"github.com/alfin-efendy/helper-go/server/restapi"
	"github.com/gin-gonic/gin"
)

// This is a example of how to use the helper-go package
func main() {
	app.Start(func() {
		// Your code here
		email := config.GetValue("superuser.email")
		password := config.GetValue("superuser.password")

		println(email, password)

		restapi.Server.GET("/hello", restapi.AuthMiddleware(""), func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "Hello, World!",
			})
		})

	})
}
