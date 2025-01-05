package main

import (
	"github.com/alfin-efendy/helper-go/app"
	"github.com/alfin-efendy/helper-go/config"
)

// This is a example of how to use the helper-go package
func main() {
	app.Start(func() {
		// Your code here
		email, _ := config.GetValue("superuser.email")
		password, _ := config.GetValue("superuser.password")

		println(email, password)
	})
}
