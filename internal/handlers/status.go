package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/tmjpugh/househero/internal/mqttservice"
)

// StatusHandler exposes application health/status information.
type StatusHandler struct {
	mqttSvc *mqttservice.Service
}

func NewStatusHandler(mqttSvc *mqttservice.Service) *StatusHandler {
	return &StatusHandler{mqttSvc: mqttSvc}
}

// GetStatus returns the current connection status of optional integrations.
func (h *StatusHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	mqttConnected := h.mqttSvc != nil && h.mqttSvc.IsEnabled()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"mqtt_connected": mqttConnected})
}
