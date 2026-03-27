package api

import (
	"github.com/gorilla/mux"
	"github.com/tmjpugh/househero/internal/database"
	"github.com/tmjpugh/househero/internal/handlers"
	"github.com/tmjpugh/househero/internal/middleware"
)

func SetupRoutes(db *database.DB) *mux.Router {
	router := mux.NewRouter()

	router.Use(middleware.AuthMiddleware)

	homeHandler := handlers.NewHomeHandler(db)
	ticketHandler := handlers.NewTicketHandler(db)
	inventoryHandler := handlers.NewInventoryHandler(db)

	// Home routes
	router.HandleFunc("/api/homes", homeHandler.GetHomes).Methods("GET")
	router.HandleFunc("/api/homes/{id}", homeHandler.GetHome).Methods("GET")
	router.HandleFunc("/api/homes", homeHandler.CreateHome).Methods("POST")
	router.HandleFunc("/api/homes/{id}", homeHandler.UpdateHome).Methods("PUT")
	router.HandleFunc("/api/homes/{id}", homeHandler.DeleteHome).Methods("DELETE")

	// Ticket routes
	router.HandleFunc("/api/tickets", ticketHandler.GetTickets).Methods("GET")
	router.HandleFunc("/api/tickets/{id}", ticketHandler.GetTicket).Methods("GET")
	router.HandleFunc("/api/tickets", ticketHandler.CreateTicket).Methods("POST")
	router.HandleFunc("/api/tickets/{id}", ticketHandler.UpdateTicket).Methods("PUT")
	router.HandleFunc("/api/tickets/{id}", ticketHandler.DeleteTicket).Methods("DELETE")
	router.HandleFunc("/api/tickets/{id}/comments", ticketHandler.AddComment).Methods("POST")
	router.HandleFunc("/api/tickets/{id}/photos", ticketHandler.AddPhoto).Methods("POST")

	// Inventory routes
	router.HandleFunc("/api/inventory", inventoryHandler.GetInventory).Methods("GET")
	router.HandleFunc("/api/inventory/{id}", inventoryHandler.GetInventoryItem).Methods("GET")
	router.HandleFunc("/api/inventory", inventoryHandler.CreateInventoryItem).Methods("POST")
	router.HandleFunc("/api/inventory/{id}", inventoryHandler.UpdateInventoryItem).Methods("PUT")
	router.HandleFunc("/api/inventory/{id}", inventoryHandler.DeleteInventoryItem).Methods("DELETE")
	router.HandleFunc("/api/inventory/{id}/documents", inventoryHandler.AddDocument).Methods("POST")
	router.HandleFunc("/api/inventory/{id}/notes", inventoryHandler.AddNote).Methods("POST")

	return router
}
