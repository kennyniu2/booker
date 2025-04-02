package main

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	config "github.com/kennyniu2/bcj/configs"
	database "github.com/kennyniu2/bcj/db"

	_ "github.com/lib/pq"
)

func main() {
	// Initialize configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Print database configuration
	fmt.Printf("Database Config: Host=%s, Port=%s, User=%s, DBName=%s\n",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBName)

	// Initialize database
	db, err := database.InitDB(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Check database connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	fmt.Println("✅ Database connection successful!")

	// Verify connection
	var version string
	err = db.QueryRow("SELECT version()").Scan(&version)
	if err != nil {
		log.Fatalf("Failed to query database version: %v", err)
	}
	fmt.Printf("PostgreSQL Version: %s\n", version)

	// Database Unit test
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS connection_test (
        id SERIAL PRIMARY KEY,
        test_message TEXT,
        created_at TIMESTAMP DEFAULT NOW()
    )`)
	if err != nil {
		log.Printf("Failed to create test table: %v", err)
	} else {
		// Insert a test row
		_, err = db.Exec("INSERT INTO connection_test(test_message) VALUES($1)", "Connection successful!")
		if err != nil {
			log.Printf("Failed to insert test data: %v", err)
		} else {
			fmt.Println("✅ Successfully wrote test data to database!")
		}
	}

	// Setup router
	r := gin.Default()

	// Add a database check endpoint
	r.GET("/ping", func(c *gin.Context) {
		// Check if DB is still connected
		err := db.Ping()
		if err != nil {
			c.JSON(500, gin.H{
				"message":   "pong",
				"db_status": "disconnected",
				"error":     err.Error(),
			})
			return
		}

		c.JSON(200, gin.H{
			"message":   "pong",
			"db_status": "connected",
		})
	})

	// Add a database stats endpoint
	r.GET("/db-info", func(c *gin.Context) {
		var (
			version     string
			numTables   int
			currentDB   string
			currentUser string
		)

		db.QueryRow("SELECT version()").Scan(&version)
		db.QueryRow("SELECT current_database()").Scan(&currentDB)
		db.QueryRow("SELECT current_user").Scan(&currentUser)
		db.QueryRow("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public'").Scan(&numTables)

		c.JSON(200, gin.H{
			"version":      version,
			"database":     currentDB,
			"user":         currentUser,
			"tables_count": numTables,
		})
	})

	// Start the server
	port := cfg.Port
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Server starting on port %s...\n", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
