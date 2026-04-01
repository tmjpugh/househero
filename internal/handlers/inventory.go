package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/tmjpugh/househero/internal/database"
	"github.com/tmjpugh/househero/internal/models"
	"github.com/tmjpugh/househero/internal/mqttservice"
)

type InventoryHandler struct {
	db   *database.DB
	mqtt *mqttservice.Service
}

func NewInventoryHandler(db *database.DB, mqttSvc *mqttservice.Service) *InventoryHandler {
	return &InventoryHandler{db: db, mqtt: mqttSvc}
}

func (h *InventoryHandler) GetInventory(w http.ResponseWriter, r *http.Request) {
	homeID := r.URL.Query().Get("home_id")

	rows, err := h.db.Query(
		`SELECT id, home_id, name, type, make, model, room, serial_number, purchase_date, warranty_expires, 
		        created_at, updated_at FROM inventory_items WHERE home_id = $1 ORDER BY created_at DESC`,
		homeID,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var items []models.InventoryItem
	for rows.Next() {
		var item models.InventoryItem
		if err := rows.Scan(
			&item.ID, &item.HomeID, &item.Name, &item.Type, &item.Make, &item.Model,
			&item.Room, &item.SerialNumber, &item.PurchaseDate, &item.WarrantyExpires,
			&item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		items = append(items, item)
	}

	// Batch-load documents for all items so the card can show counts/links.
	if len(items) > 0 {
		// Build a plain integer list (safe: all values come from the DB scan above).
		ids := make([]string, len(items))
		for i, it := range items {
			ids[i] = strconv.FormatInt(it.ID, 10)
		}
		inClause := strings.Join(ids, ",")

		// Build an index map for fast lookup.
		itemIndex := make(map[int64]int, len(items))
		for i, it := range items {
			itemIndex[it.ID] = i
		}

		docRows, docErr := h.db.Query(
			`SELECT inventory_item_id, id, doc_type, name, url, uploaded_at
			 FROM documents
			 WHERE inventory_item_id IN (` + inClause + `)
			 ORDER BY uploaded_at`,
		)
		if docErr == nil {
			defer docRows.Close()
			for docRows.Next() {
				var parentID int64
				var doc models.Document
				var docType string
				if scanErr := docRows.Scan(&parentID, &doc.ID, &docType, &doc.Name, &doc.URL, &doc.UploadedAt); scanErr != nil {
					continue
				}
				if idx, ok := itemIndex[parentID]; ok {
					if docType == "manual" {
						items[idx].Manuals = append(items[idx].Manuals, doc)
					} else if docType == "receipt" {
						items[idx].Receipts = append(items[idx].Receipts, doc)
					}
				}
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

func (h *InventoryHandler) GetInventoryItem(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	itemID := vars["id"]

	var item models.InventoryItem
	err := h.db.QueryRow(
		`SELECT id, home_id, name, type, make, model, room, serial_number, purchase_date, warranty_expires,
		        created_at, updated_at FROM inventory_items WHERE id = $1`,
		itemID,
	).Scan(
		&item.ID, &item.HomeID, &item.Name, &item.Type, &item.Make, &item.Model,
		&item.Room, &item.SerialNumber, &item.PurchaseDate, &item.WarrantyExpires,
		&item.CreatedAt, &item.UpdatedAt,
	)

	if err != nil {
		http.Error(w, "Item not found", http.StatusNotFound)
		return
	}

	// Load documents (manuals and receipts)
	docRows, _ := h.db.Query("SELECT id, doc_type, name, url, uploaded_at FROM documents WHERE inventory_item_id = $1", itemID)
	defer docRows.Close()
	for docRows.Next() {
		var doc models.Document
		var docType string
		docRows.Scan(&doc.ID, &docType, &doc.Name, &doc.URL, &doc.UploadedAt)
		if docType == "manual" {
			item.Manuals = append(item.Manuals, doc)
		} else if docType == "receipt" {
			item.Receipts = append(item.Receipts, doc)
		}
	}

	// Load notes
	noteRows, _ := h.db.Query("SELECT id, text, created_at FROM notes WHERE inventory_item_id = $1", itemID)
	defer noteRows.Close()
	for noteRows.Next() {
		var note models.Note
		noteRows.Scan(&note.ID, &note.Text, &note.CreatedAt)
		item.Notes = append(item.Notes, note)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(item)
}

func (h *InventoryHandler) CreateInventoryItem(w http.ResponseWriter, r *http.Request) {
	var item models.InventoryItem
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err := h.db.QueryRow(
		`INSERT INTO inventory_items (home_id, name, type, make, model, room, serial_number, purchase_date, warranty_expires)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING id, created_at, updated_at`,
		item.HomeID, item.Name, item.Type, item.Make, item.Model, item.Room,
		item.SerialNumber, item.PurchaseDate, item.WarrantyExpires,
	).Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if h.mqtt != nil {
		h.mqtt.Publish(mqttservice.TopicInventoryCreated, item)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(item)
}

func (h *InventoryHandler) UpdateInventoryItem(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	itemID := vars["id"]

	var item models.InventoryItem
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, err := h.db.Exec(
		`UPDATE inventory_items SET name = $1, type = $2, make = $3, model = $4, room = $5,
		                             serial_number = $6, purchase_date = $7, warranty_expires = $8,
		                             updated_at = CURRENT_TIMESTAMP
		 WHERE id = $9`,
		item.Name, item.Type, item.Make, item.Model, item.Room,
		item.SerialNumber, item.PurchaseDate, item.WarrantyExpires, itemID,
	)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if h.mqtt != nil {
		item.ID, _ = strconv.ParseInt(itemID, 10, 64)
		h.mqtt.Publish(mqttservice.TopicInventoryUpdated, item)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

func (h *InventoryHandler) DeleteInventoryItem(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	itemID := vars["id"]

	_, err := h.db.Exec("DELETE FROM inventory_items WHERE id = $1", itemID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

func (h *InventoryHandler) AddDocument(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	itemID := vars["id"]

	var doc models.Document
	docType := r.URL.Query().Get("type")

	if err := json.NewDecoder(r.Body).Decode(&doc); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err := h.db.QueryRow(
		"INSERT INTO documents (inventory_item_id, doc_type, name, url) VALUES ($1, $2, $3, $4) RETURNING id, uploaded_at",
		itemID, docType, doc.Name, doc.URL,
	).Scan(&doc.ID, &doc.UploadedAt)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(doc)
}

func (h *InventoryHandler) AddNote(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	itemID := vars["id"]

	var note models.Note
	if err := json.NewDecoder(r.Body).Decode(&note); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err := h.db.QueryRow(
		"INSERT INTO notes (inventory_item_id, text) VALUES ($1, $2) RETURNING id, created_at",
		itemID, note.Text,
	).Scan(&note.ID, &note.CreatedAt)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(note)
}
