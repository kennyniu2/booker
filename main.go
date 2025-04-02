package main

import (
	"github.com/gin-gonic/gin"
	config "github.com/kennyniu2/bcj/configs"
	database "github.com/kennyniu2/bcj/db"

	_ "github.com/lib/pq"
)

func main() {
	// Initialize configuration
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	// Initialize database
	db, err := database.InitDB(cfg)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// Setup router
	r := gin.Default()

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

}
