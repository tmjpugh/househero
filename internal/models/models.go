package models

import "time"

type Home struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Name      string    `json:"name"`
	Address   string    `json:"address"`
	Photo     *string   `json:"photo,omitempty"`  // Can be NULL
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Ticket struct {
	ID              int64     `json:"id"`
	HomeID          int64     `json:"home_id"`
	Title           string    `json:"title"`
	Description     *string   `json:"description,omitempty"`  // Can be NULL
	Type            string    `json:"type"`
	Priority        string    `json:"priority"`
	Status          string    `json:"status"`
	Requester       string    `json:"requester"`
	Room            string    `json:"room"`
	InventoryItemID *int64    `json:"inventory_item_id,omitempty"`  // Already correct
	InventoryItem   *string   `json:"inventory_item,omitempty"`  // Already correct
	EstimatedCost   *string   `json:"estimated_cost,omitempty"`  // Can be NULL
	Closer          *string   `json:"closer,omitempty"`  // Can be NULL
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	ClosedAt        *time.Time `json:"closed_at,omitempty"`  // Already correct
	BlockedBy       []int64   `json:"blocked_by,omitempty"`
	IsBlocking      []int64   `json:"is_blocking,omitempty"`
	Comments        []Comment `json:"comments,omitempty"`
	Photos          []Photo   `json:"photos,omitempty"`
}

type Comment struct {
	ID        int64     `json:"id"`
	TicketID  int64     `json:"ticket_id"`
	Text      string    `json:"text"`
	Author    string    `json:"author"`
	IsSystem  bool      `json:"is_system"`
	Timestamp time.Time `json:"timestamp"`
}

type Photo struct {
	ID         int64     `json:"id"`
	TicketID   int64     `json:"ticket_id"`
	URL        string    `json:"url"`
	Name       string    `json:"name"`
	UploadedAt time.Time `json:"uploaded_at"`
}

type InventoryItem struct {
	ID              int64      `json:"id"`
	HomeID          int64      `json:"home_id"`
	Name            string     `json:"name"`
	Type            string     `json:"type"`
	Make            string     `json:"make"`
	Model           *string    `json:"model,omitempty"`  // Can be NULL
	Room            string     `json:"room"`
	SerialNumber    *string    `json:"serial_number,omitempty"`  // Can be NULL
	PurchaseDate    *time.Time `json:"purchase_date,omitempty"`  // Already correct
	WarrantyExpires *time.Time `json:"warranty_expires,omitempty"`  // Already correct
	Manuals         []Document `json:"manuals,omitempty"`
	Receipts        []Document `json:"receipts,omitempty"`
	Notes           []Note     `json:"notes,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type Document struct {
	ID         int64     `json:"id"`
	Name       string    `json:"name"`
	URL        string    `json:"url"`
	UploadedAt time.Time `json:"uploaded_at"`
}

type Note struct {
	ID        int64     `json:"id"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

type User struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	Password  string    `json:"-"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
