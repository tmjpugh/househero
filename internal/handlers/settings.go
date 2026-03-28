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

// customSettings holds the per-user configurable lists stored as JSON text.
type customSettings struct {
	People      []string `json:"people"`
	Rooms       []string `json:"rooms"`
	TicketTypes []string `json:"ticketTypes"`
	Makes       []string `json:"makes"`
	Types       []string `json:"types"`
}

// UserSettings is the combined settings object exchanged with the frontend.
type UserSettings struct {
	SettingsPassword string   `json:"settings_password"`
	People           []string `json:"people"`
	Rooms            []string `json:"rooms"`
	TicketTypes      []string `json:"ticketTypes"`
	Makes            []string `json:"makes"`
	Types            []string `json:"types"`
}

func (h *SettingsHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		userID = "1"
	}

	var nullablePassword sql.NullString
	var nullableCustom sql.NullString
	err := h.db.QueryRow(
		"SELECT settings_password, custom_settings FROM user_settings WHERE user_id = $1",
		userID,
	).Scan(&nullablePassword, &nullableCustom)

	var settings UserSettings
	if err != nil || !nullablePassword.Valid || nullablePassword.String == "" {
		settings.SettingsPassword = "1234"
	} else {
		settings.SettingsPassword = nullablePassword.String
	}

	if nullableCustom.Valid && nullableCustom.String != "" {
		var cs customSettings
		if jsonErr := json.Unmarshal([]byte(nullableCustom.String), &cs); jsonErr == nil {
			settings.People = cs.People
			settings.Rooms = cs.Rooms
			settings.TicketTypes = cs.TicketTypes
			settings.Makes = cs.Makes
			settings.Types = cs.Types
		}
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

	cs := customSettings{
		People:      settings.People,
		Rooms:       settings.Rooms,
		TicketTypes: settings.TicketTypes,
		Makes:       settings.Makes,
		Types:       settings.Types,
	}
	customJSON, err := json.Marshal(cs)
	if err != nil {
		http.Error(w, "Failed to encode settings", http.StatusInternalServerError)
		return
	}

	_, err = h.db.Exec(
		`INSERT INTO user_settings (user_id, settings_password, custom_settings)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (user_id) DO UPDATE
		   SET settings_password = $2,
		       custom_settings   = $3,
		       updated_at        = CURRENT_TIMESTAMP`,
		userID, settings.SettingsPassword, string(customJSON),
	)
	if err != nil {
		http.Error(w, "Failed to update settings", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}
