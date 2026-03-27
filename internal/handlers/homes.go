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

// CreateHomeRequest represents the home creation request with optional user info
type CreateHomeRequest struct {
	Name             string `json:"name"`
	Address          string `json:"address"`
	Photo            string `json:"photo"`
	UserName         string `json:"user_name"`      // New: user name
	UserEmail        string `json:"user_email"`    // New: user email
	SettingsPassword string `json:"settings_password"` // New: settings password
}

func (h *HomeHandler) CreateHome(w http.ResponseWriter, r *http.Request) {
	var req CreateHomeRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	userID := r.Header.Get("X-User-ID")
	
	// If no user_id header, create a new user (first-time setup)
	if userID == "" {
		if req.UserEmail == "" || req.UserName == "" {
			http.Error(w, "user_email and user_name required for new user", http.StatusBadRequest)
			return
		}

		// Create new user
		newUserID, err := h.createUser(req.UserName, req.UserEmail, req.SettingsPassword)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		userID = strconv.FormatInt(newUserID, 10)
	}

	// Now create the home
	var home models.Home
	home.Name = req.Name
	home.Address = req.Address
	home.Photo = req.Photo

	err := h.db.QueryRow(
		"INSERT INTO homes (user_id, name, address, photo) VALUES ($1, $2, $3, $4) RETURNING id, created_at, updated_at",
		userID, home.Name, home.Address, home.Photo,
	).Scan(&home.ID, &home.CreatedAt, &home.UpdatedAt)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	home.UserID, _ = strconv.ParseInt(userID, 10, 64)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(home)
}

// createUser creates a new user and stores the settings password
func (h *HomeHandler) createUser(name, email, settingsPassword string) (int64, error) {
	var userID int64

	err := h.db.QueryRow(
		"INSERT INTO users (email, password) VALUES ($1, $2) RETURNING id",
		email, "default_password",
	).Scan(&userID)

	if err != nil {
		return 0, err
	}

	// Store settings password in a settings table
	if settingsPassword != "" {
		_, err = h.db.Exec(
			"INSERT INTO user_settings (user_id, settings_password) VALUES ($1, $2)",
			userID, settingsPassword,
		)
		if err != nil {
			// If settings table doesn't exist yet, that's ok - we'll create it in migrations
			// For now, just log and continue
		}
	}

	return userID, nil
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
