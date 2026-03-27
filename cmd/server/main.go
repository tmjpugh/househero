
---

## **`cmd/server/main.go`**
```go
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/tmjpugh/househero/api"
	"github.com/tmjpugh/househero/internal/config"
	"github.com/tmjpugh/househero/internal/database"
)

func main() {
	godotenv.Load()

	cfg := config.Load()

	db, err := database.New(cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.RunMigrations(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	router := api.SetupRoutes(db)

	log.Printf("Server starting on port %s", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, router))
}
