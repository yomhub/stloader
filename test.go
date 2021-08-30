package main

import (
	"fmt"
	"stloader/modules/secloader"
)

// "modules\secloader"
func main() {
	alllst := secloader.GetList()
	for _, tobj := range alllst {
		fmt.Printf("%s: %s\n", tobj.Name, tobj.Address)
		secloader.Read(tobj)
	}
	// r := gin.Default()
	// r.GET("/ping", func(c *gin.Context) {
	// 	c.JSON(200, gin.H{
	// 		"message": "pong",
	// 	})
	// })
	// r.Run() // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}
