package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/tmjpugh/househero/internal/database"
	"github.com/tmjpugh/househero/internal/models"
)

type HomeHandler struct {
	db *database.DB
}

func NewHomeHandler(db *database.DB) *HomeHandler {
	return &HomeHandler{db: db}
}

func (h *HomeHandler) GetHomes(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		userID = "1"
	}

	rows, err := h.db.Query("SELECT id, user_id, name, address, photo, created_at, updated_at FROM homes WHERE user_id = $1", userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var homes []models.Home
	for rows.Next() {
		var home models.Home
		if err := rows.Scan(&home.ID, &home.UserID, &home.Name, &home.Address, &home.Photo, &home.CreatedAt, &home.UpdatedAt); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		homes = append(homes, home)
	}

	if homes == nil {
		homes = []models.Home{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(homes)
}

func (h *HomeHandler) GetHome(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	homeID := vars["id"]

	var home models.Home
	err := h.db.QueryRow("SELECT id, user_id, name, address, photo, created_at, updated_at FROM homes WHERE id = $1", homeID).
		Scan(&home.ID, &home.UserID, &home.Name, &home.Address, &home.Photo, &home.CreatedAt, &home.UpdatedAt)

	if err != nil {
		http.Error(w, "Home not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(home)
}

type CreateHomeRequest struct {
	Name             string `json:"name"`
	Address          string `json:"address"`
	Photo            string `json:"photo"`
	UserName         string `json:"user_name"`
	UserEmail        string `json:"user_email"`
	SettingsPassword string `json:"settings_password"`
}

func (h *HomeHandler) CreateHome(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		userID = "1"
	}

	var req CreateHomeRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var home models.Home
	home.Name = req.Name
	home.Address = req.Address
	
	// Convert string photo to *string
	if req.Photo != "" {
		home.Photo = &req.Photo
	}

	err := h.db.QueryRow(
		"INSERT INTO homes (user_id, name, address, photo) VALUES ($1, $2, $3, $4) RETURNING id, created_at, updated_at",
		userID, home.Name, home.Address, home.Photo,
	).Scan(&home.ID, &home.CreatedAt, &home.UpdatedAt)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Save settings_password if provided
	if req.SettingsPassword != "" {
		if _, err := h.db.Exec(
			`INSERT INTO user_settings (user_id, settings_password)
			 VALUES ($1, $2)
			 ON CONFLICT (user_id) DO UPDATE SET settings_password = $2, updated_at = CURRENT_TIMESTAMP`,
			userID, req.SettingsPassword,
		); err != nil {
			http.Error(w, "Failed to save settings password", http.StatusInternalServerError)
			return
		}
	}

	home.UserID, _ = strconv.ParseInt(userID, 10, 64)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(home)
}

func (h *HomeHandler) UpdateHome(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	homeID := vars["id"]

	var home models.Home
	if err := json.NewDecoder(r.Body).Decode(&home); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, err := h.db.Exec(
		"UPDATE homes SET name = $1, address = $2, photo = $3, updated_at = CURRENT_TIMESTAMP WHERE id = $4",
		home.Name, home.Address, home.Photo, homeID,
	)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

func (h *HomeHandler) DeleteHome(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	homeID := vars["id"]

	_, err := h.db.Exec("DELETE FROM homes WHERE id = $1", homeID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}
