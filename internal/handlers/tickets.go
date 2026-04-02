package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/tmjpugh/househero/internal/database"
	"github.com/tmjpugh/househero/internal/models"
	"github.com/tmjpugh/househero/internal/mqttservice"
)

type TicketHandler struct {
	db   *database.DB
	mqtt *mqttservice.Service
}

func NewTicketHandler(db *database.DB, mqttSvc *mqttservice.Service) *TicketHandler {
	return &TicketHandler{db: db, mqtt: mqttSvc}
}

func (h *TicketHandler) GetTickets(w http.ResponseWriter, r *http.Request) {
	homeID := r.URL.Query().Get("home_id")

	rows, err := h.db.Query(
		`SELECT id, ticket_number, home_id, title, description, type, priority, status, requester, room, 
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
			&ticket.ID, &ticket.TicketNumber, &ticket.HomeID, &ticket.Title, &ticket.Description, &ticket.Type,
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
		`SELECT id, ticket_number, home_id, title, description, type, priority, status, requester, room,
		        inventory_item_id, inventory_item, estimated_cost, closer, created_at, updated_at, closed_at 
		 FROM tickets WHERE id = $1`,
		ticketID,
	).Scan(
		&ticket.ID, &ticket.TicketNumber, &ticket.HomeID, &ticket.Title, &ticket.Description, &ticket.Type,
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
		`INSERT INTO tickets (home_id, ticket_number, title, description, type, priority, status, requester, room, 
		                       inventory_item_id, inventory_item, estimated_cost) 
		 VALUES ($1, (SELECT COALESCE(MAX(ticket_number), 0) + 1 FROM tickets WHERE home_id = $1),
		         $2, $3, $4, $5, $6, $7, $8, $9,
		         (SELECT name FROM inventory_items WHERE id = $9),
		         $10) 
		 RETURNING id, ticket_number, inventory_item, created_at, updated_at`,
		ticket.HomeID, ticket.Title, ticket.Description, ticket.Type, ticket.Priority,
		ticket.Status, ticket.Requester, ticket.Room, ticket.InventoryItemID, ticket.EstimatedCost,
	).Scan(&ticket.ID, &ticket.TicketNumber, &ticket.InventoryItem, &ticket.CreatedAt, &ticket.UpdatedAt)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if h.mqtt != nil {
		h.mqtt.Publish(mqttservice.TopicTicketCreated, mqttservice.TicketEvent{
			ID:              ticket.ID,
			TicketNumber:    ticket.TicketNumber,
			HomeID:          ticket.HomeID,
			Title:           ticket.Title,
			Description:     ticket.Description,
			Type:            ticket.Type,
			Priority:        ticket.Priority,
			Status:          ticket.Status,
			Requester:       ticket.Requester,
			Room:            ticket.Room,
			InventoryItemID: ticket.InventoryItemID,
			InventoryItem:   ticket.InventoryItem,
			EstimatedCost:   ticket.EstimatedCost,
			CreatedAt:       ticket.CreatedAt,
			UpdatedAt:       ticket.UpdatedAt,
		})
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

	// Fetch current state before updating so we can detect status changes for MQTT.
	var oldStatus string
	var oldHomeID, oldTicketNumber int64
	var oldCreatedAt time.Time
	prefetchErr := h.db.QueryRow(
		`SELECT status, home_id, ticket_number, created_at FROM tickets WHERE id = $1`, ticketID,
	).Scan(&oldStatus, &oldHomeID, &oldTicketNumber, &oldCreatedAt)

	var updatedAt time.Time
	err := h.db.QueryRow(
		`UPDATE tickets SET title = $1, description = $2, type = $3, priority = $4, 
		                    status = $5, requester = $6, room = $7, inventory_item_id = $8,
		                    inventory_item = $9, estimated_cost = $10, closer = $11, closed_at = $12,
		                    updated_at = CURRENT_TIMESTAMP 
		 WHERE id = $13 RETURNING updated_at`,
		ticket.Title, ticket.Description, ticket.Type, ticket.Priority,
		ticket.Status, ticket.Requester, ticket.Room, ticket.InventoryItemID,
		ticket.InventoryItem, ticket.EstimatedCost, ticket.Closer, ticket.ClosedAt, ticketID,
	).Scan(&updatedAt)

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

	if h.mqtt != nil {
		if prefetchErr != nil {
			log.Printf("MQTT: could not fetch ticket state before update (id=%s): %v", ticketID, prefetchErr)
		} else {
			ticket.ID, _ = strconv.ParseInt(ticketID, 10, 64)
			// Use values fetched from DB so home_id and ticket_number are always present.
			homeID := oldHomeID
			if ticket.HomeID != 0 {
				homeID = ticket.HomeID
			}
			ticketNumber := oldTicketNumber
			if ticket.TicketNumber != 0 {
				ticketNumber = ticket.TicketNumber
			}
			event := mqttservice.TicketEvent{
				ID:              ticket.ID,
				TicketNumber:    ticketNumber,
				HomeID:          homeID,
				Title:           ticket.Title,
				Description:     ticket.Description,
				Type:            ticket.Type,
				Priority:        ticket.Priority,
				Status:          ticket.Status,
				Requester:       ticket.Requester,
				Room:            ticket.Room,
				InventoryItemID: ticket.InventoryItemID,
				InventoryItem:   ticket.InventoryItem,
				EstimatedCost:   ticket.EstimatedCost,
				Closer:          ticket.Closer,
				CreatedAt:       oldCreatedAt,
				UpdatedAt:       updatedAt,
				ClosedAt:        ticket.ClosedAt,
			}
			if oldStatus != ticket.Status {
				event.StatusOld = oldStatus
				event.StatusNew = ticket.Status
			}
			h.mqtt.Publish(mqttservice.TopicTicketUpdated, event)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

func (h *TicketHandler) DeleteTicket(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ticketID := vars["id"]

	homeID := r.URL.Query().Get("home_id")
	if homeID == "" {
		http.Error(w, "home_id query parameter is required", http.StatusBadRequest)
		return
	}

	result, err := h.db.Exec("DELETE FROM tickets WHERE id = $1 AND home_id = $2", ticketID, homeID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil || rowsAffected == 0 {
		http.Error(w, "Ticket not found in this home", http.StatusNotFound)
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

	if h.mqtt != nil {
		// Fetch full ticket data so the MQTT event is self-contained.
		var t models.Ticket
		if ctxErr := h.db.QueryRow(
			`SELECT id, ticket_number, home_id, title, description, type, priority, status,
			        requester, room, inventory_item_id, inventory_item, estimated_cost,
			        closer, created_at, updated_at, closed_at
			 FROM tickets WHERE id = $1`, ticketID,
		).Scan(
			&t.ID, &t.TicketNumber, &t.HomeID, &t.Title, &t.Description, &t.Type,
			&t.Priority, &t.Status, &t.Requester, &t.Room, &t.InventoryItemID,
			&t.InventoryItem, &t.EstimatedCost, &t.Closer,
			&t.CreatedAt, &t.UpdatedAt, &t.ClosedAt,
		); ctxErr != nil {
			log.Printf("MQTT: could not fetch ticket context for comment (ticket_id=%s): %v", ticketID, ctxErr)
		} else {
			h.mqtt.Publish(mqttservice.TopicCommentAdded, mqttservice.CommentEvent{
				CommentID:       comment.ID,
				TicketID:        comment.TicketID,
				TicketNumber:    t.TicketNumber,
				HomeID:          t.HomeID,
				Author:          comment.Author,
				Text:            comment.Text,
				IsSystem:        comment.IsSystem,
				Timestamp:       comment.Timestamp,
				Title:           t.Title,
				Type:            t.Type,
				Priority:        t.Priority,
				Status:          t.Status,
				Requester:       t.Requester,
				Room:            t.Room,
				Description:     t.Description,
				InventoryItemID: t.InventoryItemID,
				InventoryItem:   t.InventoryItem,
				EstimatedCost:   t.EstimatedCost,
				Closer:          t.Closer,
				CreatedAt:       t.CreatedAt,
				UpdatedAt:       t.UpdatedAt,
				ClosedAt:        t.ClosedAt,
			})
		}
	}

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
