package main

import (
	"log"
	"net/http"

	"github.com/joho/godotenv"
	"github.com/tmjpugh/househero/api"
	"github.com/tmjpugh/househero/internal/config"
	"github.com/tmjpugh/househero/internal/database"
	"github.com/tmjpugh/househero/internal/mqttservice"
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

	// Initialize MQTT service (optional — disabled when MQTT_BROKER is not set).
	cmdHandler := mqttservice.NewDBCommandHandler(db)
	mqttSvc, err := mqttservice.New(cfg.MQTTBroker, cfg.MQTTClientID, cfg.MQTTUsername, cfg.MQTTPassword, cmdHandler)
	if err != nil {
		log.Printf("Warning: MQTT initialization failed: %v — continuing without MQTT", err)
		mqttSvc = nil
	}
	if mqttSvc != nil {
		defer mqttSvc.Close()
	}

	router := api.SetupRoutes(db, cfg, mqttSvc)

	log.Printf("Server starting on port %s", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, router))
}
