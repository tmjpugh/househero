package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/tmjpugh/househero/internal/database"
)

type SettingsHandler struct {
	db *database.DB
}

func NewSettingsHandler(db *database.DB) *SettingsHandler {
	return &SettingsHandler{db: db}
}

type UserSettings struct {
	SettingsPassword string `json:"settings_password"`
}

func (h *SettingsHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		userID = "1"
	}

	var nullablePassword sql.NullString
	err := h.db.QueryRow(
		"SELECT settings_password FROM user_settings WHERE user_id = $1",
		userID,
	).Scan(&nullablePassword)

	var settings UserSettings
	if err != nil || !nullablePassword.Valid || nullablePassword.String == "" {
		// Row missing or password NULL/empty — return the '1234' default
		settings.SettingsPassword = "1234"
	} else {
		settings.SettingsPassword = nullablePassword.String
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(settings)
}

func (h *SettingsHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		userID = "1"
	}

	var settings UserSettings
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	_, err := h.db.Exec(
		`INSERT INTO user_settings (user_id, settings_password)
		 VALUES ($1, $2)
		 ON CONFLICT (user_id) DO UPDATE SET settings_password = $2, updated_at = CURRENT_TIMESTAMP`,
		userID, settings.SettingsPassword,
	)
	if err != nil {
		http.Error(w, "Failed to update settings", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}
