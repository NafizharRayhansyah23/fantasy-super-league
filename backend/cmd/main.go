package main

import (
	"log"
	"os"

	"github.com/fantasysuperleague/backend/internal/database"
	"github.com/fantasysuperleague/backend/internal/router"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Initialize database
	db, err := database.Connect()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := database.Migrate(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Println("Database migrations completed")

	// Setup router
	r := router.Setup(db)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Fantasy Super League API starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
