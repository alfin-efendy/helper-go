package main

import (
	"fmt"

	"github.com/alfin-efendy/helper-go/app"
)

// This is a example of how to use the helper-go package
func main() {
	app.Start(func() {
		// Your code here
		fmt.Println("Hello, World!")
	})
}
