package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/autoslides/video-proxy/internal/mapping"
)

type ConfigHandler struct {
	mapper *mapping.IntranetMapper
}

func NewConfigHandler(mapper *mapping.IntranetMapper) *ConfigHandler {
	return &ConfigHandler{mapper: mapper}
}

// ServeHTTP routes to the appropriate config endpoint
func (h *ConfigHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	switch {
	case r.URL.Path == "/api/v1/config/mappings" && r.Method == "GET":
		h.getMappings(w, r)
	case r.URL.Path == "/api/v1/config/reload" && r.Method == "POST":
		h.reloadMappings(w, r)
	default:
		http.Error(w, "Not found", http.StatusNotFound)
	}
}

func (h *ConfigHandler) getMappings(w http.ResponseWriter, r *http.Request) {
	mappings := h.mapper.GetMappings()
	json.NewEncoder(w).Encode(mappings)
}

func (h *ConfigHandler) reloadMappings(w http.ResponseWriter, r *http.Request) {
	if err := h.mapper.Reload(); err != nil {
		log.Printf("Failed to reload mappings: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "error",
			"error":  err.Error(),
		})
		return
	}

	log.Println("Mappings reloaded successfully")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}
