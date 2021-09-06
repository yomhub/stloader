package main

import (
	"fmt"
	"stloader/modules/secloader"
)

// "modules\secloader"
func main() {
	alllst := secloader.GetList()
	fmt.Printf("%s: %s\n", alllst[0].Name, alllst[0].Address)
	secloader.Update(alllst[0])
	// r := gin.Default()
	// r.GET("/ping", func(c *gin.Context) {
	// 	c.JSON(200, gin.H{
	// 		"message": "pong",
	// 	})
	// })
	// r.Run() // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}
