// cmd/server/main.go
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/tmjpugh/househero/internal/database"
	"github.com/tmjpugh/househero/api"
)

func main() {
	godotenv.Load()

	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}
	dbPort := os.Getenv("DB_PORT")
	if dbPort == "" {
		dbPort = "5432"
	}
	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		dbUser = "househero"
	}
	dbPassword := os.Getenv("DB_PASSWORD")
	if dbPassword == "" {
		dbPassword = "househero_dev"
	}
	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = "househero_db"
	}

	db, err := database.New(dbHost, dbPort, dbUser, dbPassword, dbName)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.RunMigrations(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	router := api.SetupRoutes(db)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, router))
}
