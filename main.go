package main

import (
	"fmt"
	"log"

	"github.com/kennyniu2/booker/api"
	config "github.com/kennyniu2/booker/configs"
	database "github.com/kennyniu2/booker/db"

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
	r := api.SetupRouter(db)

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
