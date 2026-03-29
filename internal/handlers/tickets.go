package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/tmjpugh/househero/internal/database"
	"github.com/tmjpugh/househero/internal/models"
)

type TicketHandler struct {
	db *database.DB
}

func NewTicketHandler(db *database.DB) *TicketHandler {
	return &TicketHandler{db: db}
}

func (h *TicketHandler) GetTickets(w http.ResponseWriter, r *http.Request) {
	homeID := r.URL.Query().Get("home_id")

	rows, err := h.db.Query(
		`SELECT id, home_id, title, description, type, priority, status, requester, room, 
		        inventory_item_id, inventory_item, estimated_cost, closer, created_at, updated_at, closed_at 
		 FROM tickets WHERE home_id = $1 ORDER BY created_at DESC`,
		homeID,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var tickets []models.Ticket
	ticketIndex := map[int64]int{}
	for rows.Next() {
		var ticket models.Ticket
		if err := rows.Scan(
			&ticket.ID, &ticket.HomeID, &ticket.Title, &ticket.Description, &ticket.Type,
			&ticket.Priority, &ticket.Status, &ticket.Requester, &ticket.Room,
			&ticket.InventoryItemID, &ticket.InventoryItem, &ticket.EstimatedCost,
			&ticket.Closer, &ticket.CreatedAt, &ticket.UpdatedAt, &ticket.ClosedAt,
		); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		ticketIndex[ticket.ID] = len(tickets)
		tickets = append(tickets, ticket)
	}

	if tickets == nil {
		tickets = []models.Ticket{}
	}

	// Load comments for all tickets in a single query
	commentRows, err := h.db.Query(
		`SELECT id, ticket_id, text, author, is_system, timestamp 
		 FROM comments 
		 WHERE ticket_id IN (SELECT id FROM tickets WHERE home_id = $1) 
		 ORDER BY timestamp`,
		homeID,
	)
	if err == nil {
		defer commentRows.Close()
		for commentRows.Next() {
			var comment models.Comment
			if err := commentRows.Scan(&comment.ID, &comment.TicketID, &comment.Text, &comment.Author, &comment.IsSystem, &comment.Timestamp); err == nil {
				if idx, ok := ticketIndex[comment.TicketID]; ok {
					tickets[idx].Comments = append(tickets[idx].Comments, comment)
				}
			}
		}
	} else {
		log.Printf("Warning: failed to load comments for home %s: %v", homeID, err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tickets)
}

func (h *TicketHandler) GetTicket(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ticketID := vars["id"]

	var ticket models.Ticket
	err := h.db.QueryRow(
		`SELECT id, home_id, title, description, type, priority, status, requester, room,
		        inventory_item_id, inventory_item, estimated_cost, closer, created_at, updated_at, closed_at 
		 FROM tickets WHERE id = $1`,
		ticketID,
	).Scan(
		&ticket.ID, &ticket.HomeID, &ticket.Title, &ticket.Description, &ticket.Type,
		&ticket.Priority, &ticket.Status, &ticket.Requester, &ticket.Room,
		&ticket.InventoryItemID, &ticket.InventoryItem, &ticket.EstimatedCost,
		&ticket.Closer, &ticket.CreatedAt, &ticket.UpdatedAt, &ticket.ClosedAt,
	)

	if err != nil {
		http.Error(w, "Ticket not found", http.StatusNotFound)
		return
	}

	// Load comments
	commentRows, _ := h.db.Query("SELECT id, ticket_id, text, author, is_system, timestamp FROM comments WHERE ticket_id = $1 ORDER BY timestamp", ticketID)
	defer commentRows.Close()
	for commentRows.Next() {
		var comment models.Comment
		commentRows.Scan(&comment.ID, &comment.TicketID, &comment.Text, &comment.Author, &comment.IsSystem, &comment.Timestamp)
		ticket.Comments = append(ticket.Comments, comment)
	}

	// Load photos
	photoRows, _ := h.db.Query("SELECT id, ticket_id, url, name, uploaded_at FROM photos WHERE ticket_id = $1", ticketID)
	defer photoRows.Close()
	for photoRows.Next() {
		var photo models.Photo
		photoRows.Scan(&photo.ID, &photo.TicketID, &photo.URL, &photo.Name, &photo.UploadedAt)
		ticket.Photos = append(ticket.Photos, photo)
	}

	// Load dependencies
	depRows, _ := h.db.Query("SELECT blocked_by_id, is_blocking_id FROM ticket_dependencies WHERE ticket_id = $1", ticketID)
	defer depRows.Close()
	for depRows.Next() {
		var blockedByID, isBlockingID *int64
		depRows.Scan(&blockedByID, &isBlockingID)
		if blockedByID != nil {
			ticket.BlockedBy = append(ticket.BlockedBy, *blockedByID)
		}
		if isBlockingID != nil {
			ticket.IsBlocking = append(ticket.IsBlocking, *isBlockingID)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ticket)
}

func (h *TicketHandler) CreateTicket(w http.ResponseWriter, r *http.Request) {
	// Decode as map first to handle string/empty values
	var data map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Convert inventory_item_id properly
	var invItemID *int64
	if invID, ok := data["inventory_item_id"]; ok && invID != nil {
		switch v := invID.(type) {
		case string:
			if v != "" && v != "0" {
				parsed, err := strconv.ParseInt(v, 10, 64)
				if err == nil {
					invItemID = &parsed
				}
			}
		case float64:
			if v != 0 {
				id := int64(v)
				invItemID = &id
			}
		}
	}

	// Convert string fields to pointers
	var description *string
	if desc, ok := data["description"].(string); ok && desc != "" {
		description = &desc
	}

	var estimatedCost *string
	if cost, ok := data["estimated_cost"].(string); ok && cost != "" {
		estimatedCost = &cost
	}

	// Build the ticket
	ticket := models.Ticket{
		HomeID:          int64(data["home_id"].(float64)),
		Title:           data["title"].(string),
		Description:     description,
		Type:            data["type"].(string),
		Priority:        data["priority"].(string),
		Status:          "open",
		Requester:       data["requester"].(string),
		Room:            data["room"].(string),
		InventoryItemID: invItemID,
		EstimatedCost:   estimatedCost,
	}

	err := h.db.QueryRow(
		`INSERT INTO tickets (home_id, title, description, type, priority, status, requester, room, 
		                       inventory_item_id, estimated_cost) 
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) 
		 RETURNING id, created_at, updated_at`,
		ticket.HomeID, ticket.Title, ticket.Description, ticket.Type, ticket.Priority,
		ticket.Status, ticket.Requester, ticket.Room, ticket.InventoryItemID, ticket.EstimatedCost,
	).Scan(&ticket.ID, &ticket.CreatedAt, &ticket.UpdatedAt)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(ticket)
}

// Helper function
func getStringOrEmpty(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func (h *TicketHandler) UpdateTicket(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ticketID := vars["id"]

	var ticket models.Ticket
	if err := json.NewDecoder(r.Body).Decode(&ticket); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, err := h.db.Exec(
		`UPDATE tickets SET title = $1, description = $2, type = $3, priority = $4, 
		                    status = $5, requester = $6, room = $7, inventory_item_id = $8,
		                    inventory_item = $9, estimated_cost = $10, closer = $11, closed_at = $12,
		                    updated_at = CURRENT_TIMESTAMP 
		 WHERE id = $13`,
		ticket.Title, ticket.Description, ticket.Type, ticket.Priority,
		ticket.Status, ticket.Requester, ticket.Room, ticket.InventoryItemID,
		ticket.InventoryItem, ticket.EstimatedCost, ticket.Closer, ticket.ClosedAt, ticketID,
	)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Update dependencies
	h.db.Exec("DELETE FROM ticket_dependencies WHERE ticket_id = $1", ticketID)
	for _, blockedByID := range ticket.BlockedBy {
		h.db.Exec("INSERT INTO ticket_dependencies (ticket_id, blocked_by_id) VALUES ($1, $2)", ticketID, blockedByID)
	}
	for _, isBlockingID := range ticket.IsBlocking {
		h.db.Exec("INSERT INTO ticket_dependencies (ticket_id, is_blocking_id) VALUES ($1, $2)", ticketID, isBlockingID)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

func (h *TicketHandler) DeleteTicket(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ticketID := vars["id"]

	_, err := h.db.Exec("DELETE FROM tickets WHERE id = $1", ticketID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

func (h *TicketHandler) AddComment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ticketID := vars["id"]

	var comment models.Comment
	if err := json.NewDecoder(r.Body).Decode(&comment); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err := h.db.QueryRow(
		"INSERT INTO comments (ticket_id, text, author, is_system) VALUES ($1, $2, $3, $4) RETURNING id, timestamp",
		ticketID, comment.Text, comment.Author, comment.IsSystem,
	).Scan(&comment.ID, &comment.Timestamp)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	comment.TicketID, _ = strconv.ParseInt(ticketID, 10, 64)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(comment)
}

func (h *TicketHandler) AddPhoto(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ticketID := vars["id"]

	var photo models.Photo
	if err := json.NewDecoder(r.Body).Decode(&photo); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err := h.db.QueryRow(
		"INSERT INTO photos (ticket_id, url, name) VALUES ($1, $2, $3) RETURNING id, uploaded_at",
		ticketID, photo.URL, photo.Name,
	).Scan(&photo.ID, &photo.UploadedAt)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	photo.TicketID, _ = strconv.ParseInt(ticketID, 10, 64)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(photo)
}
