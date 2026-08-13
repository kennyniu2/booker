package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/kennyniu2/booker/api"
	"github.com/kennyniu2/booker/bot"
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

	// Start the Discord bot if a token is configured
	if cfg.DiscordToken != "" {
		b, err := bot.New(cfg.DiscordToken, cfg.DiscordGuildID, db)
		if err != nil {
			log.Fatalf("Failed to create Discord bot: %v", err)
		}
		if err := b.Start(); err != nil {
			log.Fatalf("Failed to start Discord bot: %v", err)
		}
		defer b.Close()
	} else {
		fmt.Println("DISCORD_TOKEN not set, Discord bot disabled")
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	if !cfg.APIEnabled {
		fmt.Println("API disabled, running Discord bot only")
		<-stop
		return
	}

	// Setup router
	r := api.SetupRouter(db)

	port := cfg.Port
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Server starting on port %s...\n", port)
	serverErr := make(chan error, 1)
	go func() { serverErr <- r.Run(":" + port) }()

	select {
	case err := <-serverErr:
		log.Fatalf("Failed to start server: %v", err)
	case <-stop:
		fmt.Println("Shutting down...")
	}
}
