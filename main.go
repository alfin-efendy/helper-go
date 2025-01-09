package main

import (
	"net/http"

	"github.com/alfin-efendy/helper-go/app"
	"github.com/alfin-efendy/helper-go/config"
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
	// Gunakan comma untuk memisahkan nilai enum
	Type   string `json:"type" binding:"required,enum=employee/customer/vendor"`
	Status string `json:"status" binding:"required,enum=active/inactive"`
}

func HelloHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "Hello, World!",
	})
}

func CreateUserHandler(c *gin.Context) {
	var user User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.Error(err) // Tambahkan error ke context
		return
	}

	pagination := restapi.GetPage(c)

	// Set data untuk success response
	restapi.SetData(c, user)
	restapi.SetPaggination(c, restapi.PageResponse{
		TotalPage:   pagination.PageSize,
		TotalRecord: int64(pagination.Page),
	})
}
