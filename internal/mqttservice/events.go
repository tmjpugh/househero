package mqttservice

import "time"

// TicketEvent is published to househero/tickets/created and househero/tickets/updated.
// For created events StatusOld and StatusNew are empty.
// For updated events StatusOld and StatusNew are populated only when the status field changed.
type TicketEvent struct {
	ID            int64      `json:"id"`
	TicketNumber  int64      `json:"ticket_number"`
	HomeID        int64      `json:"home_id"`
	Title         string     `json:"title"`
	Type          string     `json:"type"`
	Priority      string     `json:"priority"`
	Status        string     `json:"status"`
	Requester     string     `json:"requester"`
	Room          string     `json:"room"`
	EstimatedCost *string    `json:"estimated_cost,omitempty"`
	Closer        *string    `json:"closer,omitempty"`
	StatusOld     string     `json:"status_old,omitempty"`
	StatusNew     string     `json:"status_new,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// CommentEvent is published to househero/tickets/comment_added.
// Includes home_id and ticket_number so consumers never need a second lookup.
type CommentEvent struct {
	CommentID    int64     `json:"comment_id"`
	TicketID     int64     `json:"ticket_id"`
	TicketNumber int64     `json:"ticket_number"`
	HomeID       int64     `json:"home_id"`
	Author       string    `json:"author"`
	Text         string    `json:"text"`
	IsSystem     bool      `json:"is_system"`
	Timestamp    time.Time `json:"timestamp"`
}

// InventoryEvent is published to househero/inventory/created and househero/inventory/updated.
type InventoryEvent struct {
	ID        int64     `json:"id"`
	HomeID    int64     `json:"home_id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Make      string    `json:"make"`
	Room      string    `json:"room"`
	UpdatedAt time.Time `json:"updated_at"`
}
