package api

import (
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/tmjpugh/househero/internal/config"
	"github.com/tmjpugh/househero/internal/database"
	"github.com/tmjpugh/househero/internal/handlers"
)

func SetupRoutes(db *database.DB, cfg *config.Config) *mux.Router {
	router := mux.NewRouter()

	homeHandler := handlers.NewHomeHandler(db)
	ticketHandler := handlers.NewTicketHandler(db)
	inventoryHandler := handlers.NewInventoryHandler(db)
	
	// Create uploads directory
	uploadDir := "/app/uploads"
	os.MkdirAll(uploadDir, os.ModePerm)
	uploadHandler := handlers.NewUploadHandler(db, uploadDir)

	// Serve static files with no-cache headers
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		http.ServeFile(w, r, "/app/index.html")
	})

	// Serve uploaded files
	router.PathPrefix("/uploads/").Handler(http.StripPrefix("/uploads/", http.FileServer(http.Dir(uploadDir))))

	// API routes - no auth required
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

	// Upload routes - TICKET FILES
	router.HandleFunc("/api/tickets/{id}/photos", uploadHandler.UploadTicketPhoto).Methods("POST")
	router.HandleFunc("/api/tickets/{id}/documents", uploadHandler.UploadTicketDocument).Methods("POST")
	router.HandleFunc("/api/uploads/{type}/{filename}", uploadHandler.DeleteFile).Methods("DELETE")

	// Inventory routes
	router.HandleFunc("/api/inventory", inventoryHandler.GetInventory).Methods("GET")
	router.HandleFunc("/api/inventory/{id}", inventoryHandler.GetInventoryItem).Methods("GET")
	router.HandleFunc("/api/inventory", inventoryHandler.CreateInventoryItem).Methods("POST")
	router.HandleFunc("/api/inventory/{id}", inventoryHandler.UpdateInventoryItem).Methods("PUT")
	router.HandleFunc("/api/inventory/{id}", inventoryHandler.DeleteInventoryItem).Methods("DELETE")

	// Upload routes - INVENTORY FILES
	router.HandleFunc("/api/inventory/{id}/receipts", uploadHandler.UploadInventoryReceipt).Methods("POST")
	router.HandleFunc("/api/inventory/{id}/manuals", uploadHandler.UploadInventoryManual).Methods("POST")

	return router
}
