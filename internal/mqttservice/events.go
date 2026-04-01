package mqttservice

import "time"

// TicketEvent is published to househero/tickets/created and househero/tickets/updated.
// For created events StatusOld and StatusNew are empty.
// For updated events StatusOld and StatusNew are populated only when the status field changed.
type TicketEvent struct {
	ID              int64      `json:"id"`
	TicketNumber    int64      `json:"ticket_number"`
	HomeID          int64      `json:"home_id"`
	Title           string     `json:"title"`
	Description     *string    `json:"description,omitempty"`
	Type            string     `json:"type"`
	Priority        string     `json:"priority"`
	Status          string     `json:"status"`
	Requester       string     `json:"requester"`
	Room            string     `json:"room"`
	InventoryItemID *int64     `json:"inventory_item_id,omitempty"`
	InventoryItem   *string    `json:"inventory_item,omitempty"`
	EstimatedCost   *string    `json:"estimated_cost,omitempty"`
	Closer          *string    `json:"closer,omitempty"`
	StatusOld       string     `json:"status_old,omitempty"`
	StatusNew       string     `json:"status_new,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	ClosedAt        *time.Time `json:"closed_at,omitempty"`
}

// CommentEvent is published to househero/tickets/comment_added.
// Includes the full ticket context so consumers never need a second lookup.
type CommentEvent struct {
	CommentID    int64     `json:"comment_id"`
	TicketID     int64     `json:"ticket_id"`
	TicketNumber int64     `json:"ticket_number"`
	HomeID       int64     `json:"home_id"`
	Author       string    `json:"author"`
	Text         string    `json:"text"`
	IsSystem     bool      `json:"is_system"`
	Timestamp    time.Time `json:"timestamp"`

	// Full ticket context so subscribers have all ticket data without a second lookup.
	TicketTitle           string     `json:"ticket_title"`
	TicketType            string     `json:"ticket_type"`
	TicketPriority        string     `json:"ticket_priority"`
	TicketStatus          string     `json:"ticket_status"`
	TicketRequester       string     `json:"ticket_requester"`
	TicketRoom            string     `json:"ticket_room"`
	TicketDescription     *string    `json:"ticket_description,omitempty"`
	TicketInventoryItemID *int64     `json:"ticket_inventory_item_id,omitempty"`
	TicketInventoryItem   *string    `json:"ticket_inventory_item,omitempty"`
	TicketEstimatedCost   *string    `json:"ticket_estimated_cost,omitempty"`
	TicketCloser          *string    `json:"ticket_closer,omitempty"`
	TicketCreatedAt       time.Time  `json:"ticket_created_at"`
	TicketUpdatedAt       time.Time  `json:"ticket_updated_at"`
	TicketClosedAt        *time.Time `json:"ticket_closed_at,omitempty"`
}

// InventoryEvent is published to househero/inventory/created and househero/inventory/updated.
type InventoryEvent struct {
	ID              int64      `json:"id"`
	HomeID          int64      `json:"home_id"`
	Name            string     `json:"name"`
	Type            string     `json:"type"`
	Make            string     `json:"make"`
	Model           *string    `json:"model,omitempty"`
	Room            string     `json:"room"`
	SerialNumber    *string    `json:"serial_number,omitempty"`
	PurchaseDate    *time.Time `json:"purchase_date,omitempty"`
	WarrantyExpires *time.Time `json:"warranty_expires,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}
