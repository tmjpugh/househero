package mqttservice

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/tmjpugh/househero/internal/database"
	"github.com/tmjpugh/househero/internal/models"
)

// DBCommandHandler implements CommandHandler using the application database.
type DBCommandHandler struct {
	db *database.DB
}

// NewDBCommandHandler creates a CommandHandler backed by the given database.
func NewDBCommandHandler(db *database.DB) *DBCommandHandler {
	return &DBCommandHandler{db: db}
}

// HandleCreateTicket creates a ticket from the MQTT payload.
// Required fields: home_id, title.
// Optional fields: type, priority, requester, room, description, estimated_cost.
func (h *DBCommandHandler) HandleCreateTicket(payload []byte) (interface{}, error) {
	var data map[string]interface{}
	if err := json.Unmarshal(payload, &data); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	homeID, err := extractInt64(data, "home_id")
	if err != nil {
		return nil, fmt.Errorf("home_id is required and must be a number")
	}

	title, ok := data["title"].(string)
	if !ok || title == "" {
		return nil, fmt.Errorf("title is required")
	}

	ticketType := stringOrDefault(data, "type", "maintenance")
	priority := stringOrDefault(data, "priority", "medium")
	requester := stringOrDefault(data, "requester", "")
	room := stringOrDefault(data, "room", "")

	var description *string
	if d, ok := data["description"].(string); ok && d != "" {
		description = &d
	}

	var estimatedCost *string
	if c, ok := data["estimated_cost"].(string); ok && c != "" {
		estimatedCost = &c
	}

	ticket := models.Ticket{
		HomeID:        homeID,
		Title:         title,
		Description:   description,
		Type:          ticketType,
		Priority:      priority,
		Status:        "open",
		Requester:     requester,
		Room:          room,
		EstimatedCost: estimatedCost,
	}

	err = h.db.QueryRow(
		`INSERT INTO tickets (home_id, ticket_number, title, description, type, priority, status, requester, room, estimated_cost)
		 VALUES ($1, (SELECT COALESCE(MAX(ticket_number), 0) + 1 FROM tickets WHERE home_id = $1),
		         $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING id, ticket_number, created_at, updated_at`,
		ticket.HomeID, ticket.Title, ticket.Description, ticket.Type, ticket.Priority,
		ticket.Status, ticket.Requester, ticket.Room, ticket.EstimatedCost,
	).Scan(&ticket.ID, &ticket.TicketNumber, &ticket.CreatedAt, &ticket.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}

	return ticket, nil
}

// HandleTicketDetail retrieves a ticket and its comments by ticket_number + home_id.
// Required fields: ticket_number, home_id.
func (h *DBCommandHandler) HandleTicketDetail(payload []byte) (interface{}, error) {
	var data map[string]interface{}
	if err := json.Unmarshal(payload, &data); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	homeID, err := extractInt64(data, "home_id")
	if err != nil {
		return nil, fmt.Errorf("home_id is required and must be a number")
	}

	ticketNumber, err := extractInt64(data, "ticket_number")
	if err != nil {
		return nil, fmt.Errorf("ticket_number is required and must be a number")
	}

	var ticket models.Ticket
	err = h.db.QueryRow(
		`SELECT id, ticket_number, home_id, title, description, type, priority, status, requester, room,
		        inventory_item_id, inventory_item, estimated_cost, closer, created_at, updated_at, closed_at
		 FROM tickets WHERE home_id = $1 AND ticket_number = $2`,
		homeID, ticketNumber,
	).Scan(
		&ticket.ID, &ticket.TicketNumber, &ticket.HomeID, &ticket.Title, &ticket.Description,
		&ticket.Type, &ticket.Priority, &ticket.Status, &ticket.Requester, &ticket.Room,
		&ticket.InventoryItemID, &ticket.InventoryItem, &ticket.EstimatedCost,
		&ticket.Closer, &ticket.CreatedAt, &ticket.UpdatedAt, &ticket.ClosedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("ticket #%d not found in home %d", ticketNumber, homeID)
	}

	commentRows, err := h.db.Query(
		`SELECT id, ticket_id, text, author, is_system, timestamp
		 FROM comments WHERE ticket_id = $1 ORDER BY timestamp`,
		ticket.ID,
	)
	if err == nil {
		defer commentRows.Close()
		for commentRows.Next() {
			var c models.Comment
			if scanErr := commentRows.Scan(&c.ID, &c.TicketID, &c.Text, &c.Author, &c.IsSystem, &c.Timestamp); scanErr == nil {
				ticket.Comments = append(ticket.Comments, c)
			}
		}
	}

	return ticket, nil
}

// extractInt64 reads a numeric value from a JSON-decoded map.
func extractInt64(data map[string]interface{}, key string) (int64, error) {
	v, ok := data[key]
	if !ok || v == nil {
		return 0, fmt.Errorf("missing key %q", key)
	}
	switch val := v.(type) {
	case float64:
		return int64(val), nil
	case string:
		n, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("key %q is not a valid integer: %w", key, err)
		}
		return n, nil
	default:
		return 0, fmt.Errorf("key %q has unexpected type %T", key, v)
	}
}

// stringOrDefault reads a string value from the map or returns the default.
func stringOrDefault(data map[string]interface{}, key, def string) string {
	if v, ok := data[key].(string); ok {
		return v
	}
	return def
}
